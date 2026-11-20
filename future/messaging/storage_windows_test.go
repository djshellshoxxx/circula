//go:build windows

package messaging

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func privateFixtureDirectory(t testing.TB) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "private")
	if err := CreatePrivateDirectory(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func nativeUserSID(t testing.TB) *windows.SID {
	t.Helper()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	return user.User.Sid
}

func nativeSecurity(t testing.TB, path string) *windows.SECURITY_DESCRIPTOR {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	return sd
}

func nativeSetDACL(t testing.TB, path, text string) {
	t.Helper()
	sd, err := windows.SecurityDescriptorFromString(text)
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}
	runtime.KeepAlive(sd)
}

// Read actual Windows security metadata independently of the production
// validator; assert resulting SQLite and backup files grant only the user.
func assertPrivateBackup(t testing.TB, path string) {
	t.Helper()
	sd := nativeSecurity(t, path)
	owner, _, err := sd.Owner()
	user := nativeUserSID(t)
	if err != nil || !owner.Equals(user) {
		t.Fatalf("unexpected file owner: %v", err)
	}
	dacl, _, err := sd.DACL()
	if err != nil || dacl == nil || dacl.AceCount != 1 {
		t.Fatalf("file grants are not owner-only: %s, %v", sd.String(), err)
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err = windows.GetAce(dacl, 0, &ace); err != nil {
		t.Fatal(err)
	}
	sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Mask != 0x001f01ff || ace.Header.AceFlags&^windows.INHERITED_ACE != 0 || !sid.Equals(user) {
		t.Fatalf("unexpected file access: %s", sd.String())
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("file is not regular: %v, %v", info, err)
	}
	runtime.KeepAlive(sd)
}

func nativeAllow(context.Context, *sql.Tx, Actor, string, string) error { return nil }

func TestWindowsFailedOpenReleasesSuccessfullyClosedStorage(t *testing.T) {
	directory := privateFixtureDirectory(t)
	path := filepath.Join(directory, "invalid-authorizer.sqlite")
	if s, err := Open(path, nil); s != nil || !errors.Is(err, ErrInvalid) {
		t.Fatalf("Open with missing authorizer = %v, %v", s, err)
	}
	assertPrivateBackup(t, path)
	if err := os.Rename(path, path+".closed"); err != nil {
		t.Fatalf("failed Open retained the main-file guard after successful cleanup: %v", err)
	}
	if err := os.Rename(directory, directory+".closed"); err != nil {
		t.Fatalf("failed Open retained directory guards after successful cleanup: %v", err)
	}
}

func TestWindowsPrivateLifecycleAndSidecars(t *testing.T) {
	f := setup(t)
	thread := f.thread()
	f.send(thread.ID, "private WAL content", "native-sidecars")
	for _, suffix := range []string{"", "-wal", "-shm"} {
		assertPrivateBackup(t, f.path+suffix)
	}
	var tempStore int
	if err := f.s.db.QueryRow("PRAGMA temp_store").Scan(&tempStore); err != nil || tempStore != 2 {
		t.Fatalf("SQLite can spill private temporary data outside the directory: %d, %v", tempStore, err)
	}
	if err := f.s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.s.Close(); err != nil {
		t.Fatalf("repeated close: %v", err)
	}
	f.open()
	items, err := f.s.Messages(ctx, bob, thread.ID, 0, 10)
	if err != nil || len(items) != 1 || items[0].Body != "private WAL content" {
		t.Fatalf("private lifecycle lost data: %#v, %v", items, err)
	}
	assertPrivateBackup(t, f.path)
}

func TestWindowsPrivateSettingsSurviveConnectionReplacement(t *testing.T) {
	f := setup(t)
	thread := f.thread()
	f.send(thread.ID, "survives replacement", "native-replacement")
	// Force the original physical connection to close. Every subsequent query
	// then opens a new pinned-driver connection and must retain the profile.
	f.s.db.SetMaxIdleConns(0)
	for pragma, expected := range map[string]int{"temp_store": 2, "foreign_keys": 1, "secure_delete": 1, "busy_timeout": 10000} {
		var value int
		if err := f.s.db.QueryRow("PRAGMA " + pragma).Scan(&value); err != nil || value != expected {
			t.Fatalf("replacement connection %s=%d, want %d: %v", pragma, value, expected, err)
		}
	}
	var journal string
	if err := f.s.db.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil || journal != "wal" {
		t.Fatalf("replacement connection journal=%q: %v", journal, err)
	}
	items, err := f.s.Messages(ctx, bob, thread.ID, 0, 10)
	if err != nil || len(items) != 1 || items[0].Body != "survives replacement" {
		t.Fatalf("replacement lost data: %#v, %v", items, err)
	}
}

func TestWindowsRejectsUnsafeExistingFileWithoutChanges(t *testing.T) {
	user := nativeUserSID(t).String()
	cases := map[string]string{
		"everyone read":             "D:P(A;;FA;;;" + user + ")(A;;GR;;;WD)",
		"users write":               "D:P(A;;FA;;;" + user + ")(A;;GW;;;BU)",
		"authenticated users":       "D:P(A;;FA;;;" + user + ")(A;;GR;;;AU)",
		"null DACL":                 "D:NO_ACCESS_CONTROL",
		"empty DACL":                "D:P",
		"unsupported duplicate ACE": "D:P(A;;FA;;;" + user + ")(A;;GR;;;" + user + ")",
		"unsupported deny ACE":      "D:P(D;;GW;;;WD)(A;;FA;;;" + user + ")",
	}
	for name, dacl := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(nativePrivateDirectory(t), "existing.sqlite")
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			content := "unchanged rejected caller data"
			if _, err = file.WriteString(content); err != nil {
				t.Fatal(err)
			}
			nativeSetDACL(t, path, dacl)
			before := nativeSecurity(t, path).String()
			if s, err := Open(path, nativeAllow); err == nil {
				s.Close()
				t.Fatal("unsafe existing file accepted")
			}
			if after := nativeSecurity(t, path).String(); after != before {
				t.Fatalf("rejected ACL changed: %q -> %q", before, after)
			}
			if _, err = file.Seek(0, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(file)
			if err != nil || string(data) != content {
				t.Fatalf("rejected bytes changed: %q, %v", data, err)
			}
			// Restore only our synthetic fixture after proving production did not.
			nativeSetDACL(t, path, "D:P(A;;FA;;;"+user+")")
		})
	}
}

func TestWindowsRejectsUnsafeDirectoryAndSidecars(t *testing.T) {
	user := nativeUserSID(t).String()
	for _, name := range []string{"directory broad", "directory no child inheritance", "-wal", "-shm", "-journal"} {
		t.Run(name, func(t *testing.T) {
			dir := nativePrivateDirectory(t)
			path := filepath.Join(dir, "messages.sqlite")
			unsafePath := dir
			if name[0] == '-' {
				unsafePath = path + name
				if err := os.WriteFile(unsafePath, []byte("existing sidecar"), 0600); err != nil {
					t.Fatal(err)
				}
				nativeSetDACL(t, unsafePath, "D:P(A;;FA;;;"+user+")(A;;GR;;;WD)")
			} else if name == "directory broad" {
				nativeSetDACL(t, dir, "D:P(A;OICI;FA;;;"+user+")(A;OICI;GR;;;WD)")
			} else {
				nativeSetDACL(t, dir, "D:P(A;;FA;;;"+user+")")
			}
			before := nativeSecurity(t, unsafePath).String()
			if s, err := Open(path, nativeAllow); err == nil {
				s.Close()
				t.Fatal("unsafe directory/sidecar accepted")
			}
			if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("database created before unsafe-state rejection: %v", err)
			}
			if nativeSecurity(t, unsafePath).String() != before {
				t.Fatal("unsafe caller ACL silently changed")
			}
			if unsafePath != dir {
				data, err := os.ReadFile(unsafePath)
				if err != nil || string(data) != "existing sidecar" {
					t.Fatalf("sidecar changed: %q, %v", data, err)
				}
			}
		})
	}
}

func TestWindowsPrivateDirectoryCreationIsExplicit(t *testing.T) {
	dir := t.TempDir()
	before := nativeSecurity(t, dir).String()
	if err := CreatePrivateDirectory(dir); err == nil {
		t.Fatal("existing directory silently accepted")
	}
	if nativeSecurity(t, dir).String() != before {
		t.Fatal("existing caller directory ACL changed")
	}
	path := filepath.Join(dir, "private")
	start := make(chan struct{})
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { <-start; results <- CreatePrivateDirectory(path) }()
	}
	close(start)
	passed := 0
	for i := 0; i < 2; i++ {
		if err := <-results; err == nil {
			passed++
		}
	}
	if passed != 1 {
		t.Fatalf("atomic creation successes: %d", passed)
	}
	sd := nativeSecurity(t, path)
	control, _, err := sd.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatal("new private directory does not block inherited grants", err)
	}
	file := filepath.Join(path, "inherited-backup")
	if err = os.WriteFile(file, []byte("inherited private backup"), 0600); err != nil {
		t.Fatal(err)
	}
	assertPrivateBackup(t, file)
}

func nativeOtherAssignableOwner(t testing.TB) *windows.SID {
	t.Helper()
	user := nativeUserSID(t)
	if nativeOriginalOwner != nil && !nativeOriginalOwner.Equals(user) {
		return nativeOriginalOwner
	}
	groups, err := windows.GetCurrentProcessToken().GetTokenGroups()
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range groups.AllGroups() {
		if group.Attributes&windows.SE_GROUP_OWNER != 0 && group.Attributes&windows.SE_GROUP_USE_FOR_DENY_ONLY == 0 && !group.Sid.Equals(user) {
			return group.Sid
		}
	}
	return nil
}

func TestWindowsRejectsDifferentDefaultOwner(t *testing.T) {
	user := nativeUserSID(t)
	world, err := windows.CreateWellKnownSid(windows.WinWorldSid)
	if err != nil {
		t.Fatal(err)
	}
	if sameWindowsStorageOwner(user, world) == nil || sameWindowsStorageOwner(user, nil) == nil || sameWindowsStorageOwner(nil, user) == nil {
		t.Fatal("production identity comparison accepted mismatched or missing owner")
	}
	if err = sameWindowsStorageOwner(user, user); err != nil {
		t.Fatal(err)
	}
	other := nativeOtherAssignableOwner(t)
	if other == nil {
		t.Log("token has no alternate legally assignable owner; production mismatch predicate exercised, public-entry token-change case unavailable")
		return
	}
	dir := nativePrivateDirectory(t)
	path := filepath.Join(dir, "must-not-exist.sqlite")
	var token windows.Token
	if err = windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY|windows.TOKEN_ADJUST_DEFAULT, &token); err != nil {
		t.Fatal(err)
	}
	defer token.Close()
	if err = nativeSetOwner(token, other); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := nativeSetOwner(token, user); err != nil {
			t.Fatal("restore test process owner", err)
		}
	}()
	if s, err := Open(path, nativeAllow); !errors.Is(err, errPrivateStorage) {
		if s != nil {
			s.Close()
		}
		t.Fatalf("public Open did not reject different process default owner: %v", err)
	}
	if _, err = os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owner mismatch created a database: %v", err)
	}
	if err = CreatePrivateDirectory(filepath.Join(dir, "must-not-exist")); !errors.Is(err, errPrivateStorage) {
		t.Fatalf("directory creation accepted different process default owner: %v", err)
	}
	t.Log("actual public Open and directory creation rejected alternate TokenOwner", other.String())
}

func TestWindowsRejectsForeignExistingOwner(t *testing.T) {
	other := nativeOtherAssignableOwner(t)
	if other == nil {
		t.Log("token cannot assign a foreign existing owner; default-owner mismatch predicate remains covered")
		return
	}
	path := filepath.Join(nativePrivateDirectory(t), "foreign.sqlite")
	if err := os.WriteFile(path, []byte("foreign-owner fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION, other, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	before := nativeSecurity(t, path).String()
	if s, err := Open(path, nativeAllow); err == nil {
		s.Close()
		t.Fatal("foreign-owned existing file accepted")
	}
	if nativeSecurity(t, path).String() != before {
		t.Fatal("foreign owner was silently changed")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "foreign-owner fixture" {
		t.Fatalf("foreign file changed: %q, %v", data, err)
	}
}

func TestWindowsRejectsPathAliasesBeforeCreatingFiles(t *testing.T) {
	dir := nativePrivateDirectory(t)
	for _, path := range []string{filepath.Join(dir, "message.sqlite:stream"), filepath.Join(dir, "NUL"), filepath.Join(dir, "trailing."), filepath.Join(dir, "trailing "), filepath.Join(dir, "bad\xff"), `\\localhost\c$\messages.sqlite`, `\\?\C:\messages.sqlite`, `C:relative.sqlite`} {
		if s, err := Open(path, nativeAllow); err == nil {
			s.Close()
			t.Fatalf("unsupported path accepted: %q", path)
		}
	}
	items, err := os.ReadDir(dir)
	if err != nil || len(items) != 0 {
		t.Fatalf("rejected paths created files: %v, %v", items, err)
	}
}

func TestWindowsPreservesValidUnicodePath(t *testing.T) {
	name := "資料-e\u0301-\ufffd-#%&.sqlite"
	path := filepath.Join(nativePrivateDirectory(t), name)
	s, err := Open(path, nativeAllow)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	info, err := os.Stat(path)
	if err != nil || info.Name() != name {
		t.Fatalf("valid Unicode filename changed: %v, %v", info, err)
	}
	assertPrivateBackup(t, path)
}

func TestWindowsPathPreservesOrdinaryNormalization(t *testing.T) {
	dir := t.TempDir()
	want := filepath.Join(dir, "messages.sqlite")
	for _, path := range []string{
		dir + `\.\messages.sqlite`,
		dir + `\unused\..\messages.sqlite`,
		filepath.ToSlash(dir) + "/./messages.sqlite",
	} {
		got, err := privateWindowsPath(path)
		if err != nil || got != want {
			t.Fatalf("ordinary path %q resolved to %q, %v; want %q", path, got, err, want)
		}
	}
}

func TestWindowsRejectsImpersonatingStorageInitialization(t *testing.T) {
	dir := nativePrivateDirectory(t)
	path := filepath.Join(dir, "must-not-exist.sqlite")
	var process, impersonation windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY|windows.TOKEN_DUPLICATE, &process); err != nil {
		t.Fatal(err)
	}
	defer process.Close()
	if err := windows.DuplicateTokenEx(process, windows.TOKEN_QUERY|windows.TOKEN_IMPERSONATE, nil, windows.SecurityImpersonation, windows.TokenImpersonation, &impersonation); err != nil {
		t.Fatal(err)
	}
	defer impersonation.Close()
	runtime.LockOSThread()
	if err := windows.SetThreadToken(nil, impersonation); err != nil {
		runtime.UnlockOSThread()
		t.Fatal(err)
	}
	defer func() {
		if err := windows.RevertToSelf(); err != nil {
			// Do not return an impersonating thread to the runtime.
			t.Fatal("restore native test thread", err)
		}
		runtime.UnlockOSThread()
	}()
	if s, err := Open(path, nativeAllow); !errors.Is(err, errPrivateStorage) {
		if s != nil {
			s.Close()
		}
		t.Fatalf("impersonating initialization accepted: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("impersonating initialization created a file: %v", err)
	}
}

func TestWindowsStorageHandlesPreventReplacement(t *testing.T) {
	dir := nativePrivateDirectory(t)
	path := filepath.Join(dir, "messages.sqlite")
	s, err := Open(path, nativeAllow)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, old := range []string{path, dir, filepath.Dir(dir)} {
		moved := old + ".moved"
		if err := os.Rename(old, moved); err == nil {
			s.Close()
			os.Rename(moved, old)
			t.Fatalf("validated path was replaceable while open: %s", old)
		}
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(path, path+".closed"); err != nil {
		t.Fatalf("metadata handles leaked after Close: %v", err)
	}
}

func TestWindowsConcurrentPrivateFilePreparation(t *testing.T) {
	path := filepath.Join(nativePrivateDirectory(t), "messages.sqlite")
	start := make(chan struct{})
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, guard, err := preparePrivateDatabase(path)
			if guard != nil {
				if closeErr := guard.Close(); err == nil {
					err = closeErr
				}
			}
			errors <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	assertPrivateBackup(t, path)
}

// A directory junction does not require the symbolic-link creation privilege.
// Build a real mount-point reparse fixture, using only our temporary paths.
func nativeJunction(t testing.TB, path, target string) {
	t.Helper()
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)
	substitute, err := windows.UTF16FromString(`\??\` + target)
	if err != nil {
		t.Fatal(err)
	}
	printName, err := windows.UTF16FromString(target)
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 16+2*(len(substitute)+len(printName)))
	binary.LittleEndian.PutUint32(data, windows.IO_REPARSE_TAG_MOUNT_POINT)
	binary.LittleEndian.PutUint16(data[4:], uint16(len(data)-8))
	binary.LittleEndian.PutUint16(data[10:], uint16(2*(len(substitute)-1)))
	binary.LittleEndian.PutUint16(data[12:], uint16(2*len(substitute)))
	binary.LittleEndian.PutUint16(data[14:], uint16(2*(len(printName)-1)))
	for i, unit := range append(substitute, printName...) {
		binary.LittleEndian.PutUint16(data[16+2*i:], unit)
	}
	var returned uint32
	if err = windows.DeviceIoControl(handle, windows.FSCTL_SET_REPARSE_POINT, &data[0], uint32(len(data)), nil, 0, &returned, nil); err != nil {
		t.Fatal(err)
	}
}

func TestWindowsRejectsReparseDirectoryAndLinkedFiles(t *testing.T) {
	outer := nativePrivateDirectory(t)
	target := nativePrivateDirectory(t)
	junction := filepath.Join(outer, "junction")
	nativeJunction(t, junction, target)
	if s, err := Open(filepath.Join(junction, "messages.sqlite"), nativeAllow); err == nil {
		s.Close()
		t.Fatal("reparse directory accepted")
	}
	if _, err := os.Stat(filepath.Join(target, "messages.sqlite")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("reparse target mutated: %v", err)
	}
	original := filepath.Join(outer, "original")
	alias := filepath.Join(outer, "alias.sqlite")
	if err := os.WriteFile(original, []byte("linked caller data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(original, alias); err != nil {
		t.Fatal(err)
	}
	if s, err := Open(alias, nativeAllow); err == nil {
		s.Close()
		t.Fatal("multiply-linked file accepted")
	}
	data, err := os.ReadFile(original)
	if err != nil || string(data) != "linked caller data" {
		t.Fatalf("linked file changed: %q, %v", data, err)
	}
}
