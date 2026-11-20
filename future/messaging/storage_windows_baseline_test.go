//go:build windows

package messaging

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The fixture changes only its own test process. Production never changes a
// token. SQLite's default-created sidecars must have the same owner as TokenUser.
var nativeOriginalOwner *windows.SID

func TestMain(m *testing.M) {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY|windows.TOKEN_ADJUST_DEFAULT, &token); err != nil {
		fmt.Fprintln(os.Stderr, "native private-storage fixture token:", err)
		os.Exit(1)
	}
	user, err := token.GetTokenUser()
	if err != nil {
		fmt.Fprintln(os.Stderr, "native private-storage fixture user:", err)
		token.Close()
		os.Exit(1)
	}
	var needed uint32
	windows.GetTokenInformation(token, windows.TokenOwner, nil, 0, &needed)
	if needed < uint32(unsafe.Sizeof(uintptr(0))) || needed > 4096 {
		fmt.Fprintln(os.Stderr, "invalid native fixture token-owner size")
		token.Close()
		os.Exit(1)
	}
	data := make([]byte, needed)
	if err = windows.GetTokenInformation(token, windows.TokenOwner, &data[0], needed, &needed); err == nil {
		owner := *(**windows.SID)(unsafe.Pointer(&data[0]))
		nativeOriginalOwner, err = owner.Copy()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "native private-storage fixture original owner:", err)
		token.Close()
		os.Exit(1)
	}
	if err = nativeSetOwner(token, user.User.Sid); err != nil {
		fmt.Fprintln(os.Stderr, "native private-storage fixture set owner:", err)
		token.Close()
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "native private-storage fixture: process TokenOwner %s -> TokenUser %s; restore after suite\n", nativeOriginalOwner.String(), user.User.Sid.String())
	code := m.Run()
	if err = nativeSetOwner(token, nativeOriginalOwner); err != nil {
		fmt.Fprintln(os.Stderr, "native private-storage fixture restore owner:", err)
		code = 1
	}
	if err = token.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "native private-storage fixture close token:", err)
		code = 1
	}
	os.Exit(code)
}

func nativeSetOwner(token windows.Token, sid *windows.SID) error {
	owner := struct{ Owner *windows.SID }{sid}
	err := windows.SetTokenInformation(token, windows.TokenOwner, (*byte)(unsafe.Pointer(&owner)), uint32(unsafe.Sizeof(owner)))
	runtime.KeepAlive(sid)
	return err
}

// This fixture uses Win32 directly, independently of the production helper.
// It can be copied unchanged onto the archived implementation for a native red
// run: Open rejects this legitimate owner-private file's synthesized mode bits.
func nativePrivateDirectory(t testing.TB) string {
	t.Helper()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sid := user.User.Sid.String()
	sd, err := windows.SecurityDescriptorFromString("O:" + sid + "D:P(A;OICI;FA;;;" + sid + ")")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "private")
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	if err := windows.CreateDirectory(name, &sa); err != nil {
		t.Fatal(err)
	}
	runtime.KeepAlive(sd)
	return path
}

func TestWindowsOpenOwnerPrivateDatabase(t *testing.T) {
	path := filepath.Join(nativePrivateDirectory(t), "messages.sqlite")
	s, err := Open(path, func(context.Context, *sql.Tx, Actor, string, string) error { return nil })
	if err != nil {
		t.Fatalf("owner-private native SQLite file rejected: %v", err)
	}
	defer s.Close()
	if err = s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	thread, err := s.Create(context.Background(), Actor{"network", "alice"}, "bob", "", "windows-create", 30)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Send(context.Background(), Actor{"network", "alice"}, thread.ID, "native private data", "windows-send"); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path, func(context.Context, *sql.Tx, Actor, string, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	messages, err := s.Messages(context.Background(), Actor{"network", "bob"}, thread.ID, 0, 10)
	if err != nil || len(messages) != 1 || messages[0].Body != "native private data" {
		t.Fatalf("native restart: %#v, %v", messages, err)
	}
}
