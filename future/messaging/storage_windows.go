//go:build windows

package messaging

import (
	"errors"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/sys/windows"
)

const privateFileAccess = windows.ACCESS_MASK(0x001f01ff) // FILE_ALL_ACCESS

// Keep every directory component and the main database open without
// FILE_SHARE_DELETE while SQLite uses the validated absolute path. Sidecars
// are checked but not retained: SQLite must be able to delete/recreate them in
// the private directory. Only the trusted storage identity can create there.
type windowsStorageGuard struct {
	handles []windows.Handle
	once    sync.Once
	err     error
}

func (g *windowsStorageGuard) Close() error {
	g.once.Do(func() {
		for i := len(g.handles) - 1; i >= 0; i-- {
			g.err = errors.Join(g.err, windows.CloseHandle(g.handles[i]))
		}
	})
	return g.err
}

func privateWindowsPath(path string) (string, error) {
	if path == "" || !utf8.ValidString(path) || strings.ContainsAny(path, "?\x00") {
		return "", errPrivateStorage
	}
	if filepath.VolumeName(path) != "" && !filepath.IsAbs(path) {
		return "", errPrivateStorage
	}
	full, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	volume := filepath.VolumeName(full)
	if len(volume) != 2 || volume[1] != ':' || len(full) <= 3 || len(full) > 32760 {
		return "", errPrivateStorage
	}
	parts := strings.Split(full[3:], `\`)
	if len(parts) > 128 {
		return "", errPrivateStorage
	}
	for _, part := range parts {
		if !filepath.IsLocal(part) || strings.ContainsRune(part, ':') || strings.TrimRight(part, ". ") != part {
			return "", errPrivateStorage
		}
	}
	root, err := windows.UTF16PtrFromString(full[:3])
	if err != nil || windows.GetDriveType(root) != windows.DRIVE_FIXED {
		return "", errPrivateStorage
	}
	return full, nil
}

func windowsStorageIdentity() (*windows.SID, error) {
	// Storage initialization is a process-identity operation. An impersonating
	// host thread could otherwise make SQLite create children under another token.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	thread, err := windows.GetCurrentThread()
	if err != nil {
		return nil, err
	}
	var impersonation windows.Token
	err = windows.OpenThreadToken(thread, windows.TOKEN_QUERY, true, &impersonation)
	if err == nil {
		impersonation.Close()
		return nil, errPrivateStorage
	}
	if !errors.Is(err, windows.ERROR_NO_TOKEN) {
		return nil, err
	}
	token := windows.GetCurrentProcessToken()
	user, err := token.GetTokenUser()
	if err != nil {
		return nil, err
	}
	var needed uint32
	err = windows.GetTokenInformation(token, windows.TokenOwner, nil, 0, &needed)
	if !errors.Is(err, windows.ERROR_INSUFFICIENT_BUFFER) || needed < uint32(unsafe.Sizeof(uintptr(0))) || needed > 4096 {
		return nil, errPrivateStorage
	}
	data := make([]byte, needed)
	if err = windows.GetTokenInformation(token, windows.TokenOwner, &data[0], needed, &needed); err != nil {
		return nil, err
	}
	owner := *(**windows.SID)(unsafe.Pointer(&data[0]))
	if err = sameWindowsStorageOwner(user.User.Sid, owner); err != nil {
		return nil, err
	}
	return user.User.Sid.Copy()
}

func sameWindowsStorageOwner(user, owner *windows.SID) error {
	if user == nil || owner == nil || !user.IsValid() || !owner.IsValid() || !user.Equals(owner) {
		return errPrivateStorage
	}
	return nil
}

func storageSecurityDescriptor(sid *windows.SID, directory bool) (*windows.SECURITY_DESCRIPTOR, error) {
	inherit := ""
	if directory {
		inherit = "OICI"
	}
	return windows.SecurityDescriptorFromString("O:" + sid.String() + "D:P(A;" + inherit + ";FA;;;" + sid.String() + ")")
}

func checkWindowsStorageACL(handle windows.Handle, sid *windows.SID, directory bool) error {
	sd, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	if sd == nil || !sd.IsValid() {
		return errPrivateStorage
	}
	owner, _, err := sd.Owner()
	if err != nil || sameWindowsStorageOwner(sid, owner) != nil {
		return errPrivateStorage
	}
	control, _, err := sd.Control()
	if err != nil || control&windows.SE_DACL_PRESENT == 0 || (directory && control&windows.SE_DACL_PROTECTED == 0) {
		return errPrivateStorage
	}
	dacl, _, err := sd.DACL()
	if err != nil || dacl == nil || dacl.AceCount != 1 {
		return errPrivateStorage
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err = windows.GetAce(dacl, 0, &ace); err != nil {
		return err
	}
	wantFlags := uint8(0)
	if directory {
		wantFlags = windows.OBJECT_INHERIT_ACE | windows.CONTAINER_INHERIT_ACE
	}
	if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Mask != privateFileAccess || ace.Header.AceFlags&^windows.INHERITED_ACE != wantFlags {
		return errPrivateStorage
	}
	if ace.Header.AceSize < uint16(unsafe.Offsetof(ace.SidStart))+8 {
		return errPrivateStorage
	}
	aceSID := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	if !aceSID.IsValid() || int(unsafe.Offsetof(ace.SidStart))+aceSID.Len() != int(ace.Header.AceSize) || !aceSID.Equals(sid) {
		return errPrivateStorage
	}
	runtime.KeepAlive(sd)
	return nil
}

func checkWindowsStorageType(handle windows.Handle, directory bool) error {
	typ, err := windows.GetFileType(handle)
	if err != nil || typ != windows.FILE_TYPE_DISK {
		return errPrivateStorage
	}
	var info windows.ByHandleFileInformation
	if err = windows.GetFileInformationByHandle(handle, &info); err != nil {
		return err
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 || (info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0) != directory || (!directory && info.NumberOfLinks != 1) {
		return errPrivateStorage
	}
	return nil
}

func openWindowsStorageHandle(path string, directory, shareDelete bool) (windows.Handle, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return windows.InvalidHandle, err
	}
	flags := uint32(windows.FILE_FLAG_OPEN_REPARSE_POINT)
	if directory {
		flags |= windows.FILE_FLAG_BACKUP_SEMANTICS
	}
	share := uint32(windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE)
	if shareDelete {
		share |= windows.FILE_SHARE_DELETE
	}
	return windows.CreateFile(name, windows.READ_CONTROL|windows.FILE_READ_ATTRIBUTES, share, nil, windows.OPEN_EXISTING, flags, 0)
}

func lockWindowsStorageDirectories(path string) (*windowsStorageGuard, windows.Handle, error) {
	g := &windowsStorageGuard{}
	keep := false
	defer func() {
		if !keep {
			g.Close()
		}
	}()
	current := path[:3]
	parts := []string{current}
	if len(path) > 3 {
		for _, part := range strings.Split(path[3:], `\`) {
			current = filepath.Join(current, part)
			parts = append(parts, current)
		}
	}
	for _, part := range parts {
		handle, err := openWindowsStorageHandle(part, true, false)
		if err != nil {
			return nil, 0, err
		}
		g.handles = append(g.handles, handle)
		if err = checkWindowsStorageType(handle, true); err != nil {
			return nil, 0, err
		}
	}
	var flags uint32
	last := g.handles[len(g.handles)-1]
	if err := windows.GetVolumeInformationByHandle(last, nil, 0, nil, nil, &flags, nil, 0); err != nil || flags&windows.FILE_PERSISTENT_ACLS == 0 {
		return nil, 0, errPrivateStorage
	}
	keep = true
	return g, last, nil
}

func createPrivateDirectory(path string) error {
	sid, err := windowsStorageIdentity()
	if err != nil {
		return err
	}
	full, err := privateWindowsPath(path)
	if err != nil {
		return err
	}
	guard, _, err := lockWindowsStorageDirectories(filepath.Dir(full))
	if err != nil {
		return err
	}
	defer guard.Close()
	sd, err := storageSecurityDescriptor(sid, true)
	if err != nil {
		return err
	}
	name, err := windows.UTF16PtrFromString(full)
	if err != nil {
		return err
	}
	sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	err = windows.CreateDirectory(name, &sa)
	runtime.KeepAlive(sd)
	if err != nil {
		return err
	}
	handle, err := openWindowsStorageHandle(full, true, false)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	if err = checkWindowsStorageType(handle, true); err != nil {
		return err
	}
	return checkWindowsStorageACL(handle, sid, true)
}

func preparePrivateDatabase(path string) (string, io.Closer, error) {
	sid, err := windowsStorageIdentity()
	if err != nil {
		return "", nil, err
	}
	full, err := privateWindowsPath(path)
	if err != nil {
		return "", nil, err
	}
	guard, directory, err := lockWindowsStorageDirectories(filepath.Dir(full))
	if err != nil {
		return "", nil, err
	}
	keep := false
	defer func() {
		if !keep {
			guard.Close()
		}
	}()
	if err = checkWindowsStorageACL(directory, sid, true); err != nil {
		return "", nil, err
	}
	// Validate existing sidecars before creating or opening the database. Only
	// this private directory can supply subsequently created owner-only children.
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		handle, err := openWindowsStorageHandle(full+suffix, false, true)
		if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
			continue
		}
		if err != nil {
			return "", nil, err
		}
		err = errors.Join(checkWindowsStorageType(handle, false), checkWindowsStorageACL(handle, sid, false))
		err = errors.Join(err, windows.CloseHandle(handle))
		if err != nil {
			return "", nil, err
		}
	}
	handle, err := openWindowsStorageHandle(full, false, false)
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		sd, e := storageSecurityDescriptor(sid, false)
		if e != nil {
			return "", nil, e
		}
		name, e := windows.UTF16PtrFromString(full)
		if e != nil {
			return "", nil, e
		}
		sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
		handle, err = windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE|windows.READ_CONTROL, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, &sa, windows.CREATE_NEW, windows.FILE_ATTRIBUTE_NORMAL, 0)
		runtime.KeepAlive(sd)
		if errors.Is(err, windows.ERROR_FILE_EXISTS) || errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
			// A concurrent trusted creator may win; reopen without truncation and
			// apply the same existing-file checks rather than changing its ACL.
			handle, err = openWindowsStorageHandle(full, false, false)
		}
	}
	if err != nil {
		return "", nil, err
	}
	guard.handles = append(guard.handles, handle)
	if err = checkWindowsStorageType(handle, false); err != nil {
		return "", nil, err
	}
	if err = checkWindowsStorageACL(handle, sid, false); err != nil {
		return "", nil, err
	}
	keep = true
	return full, guard, nil
}

func privateDatabaseDSN(path string) string {
	// SQLite's Windows temporary files otherwise use the process temp directory,
	// outside the validated database directory and its inheritable private ACL.
	// These trusted driver options also initialize replacement pool connections.
	// Input paths containing '?' were rejected before filesystem access; the
	// pinned driver strips this query from ordinary (non-URI) filenames.
	return path + "?_pragma=busy_timeout(10000)&_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=secure_delete(ON)&_pragma=temp_store(MEMORY)"
}
