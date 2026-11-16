package main

import (
 "os"
 "path/filepath"
 "runtime"
 "testing"
)

func TestWriteJSONFileDoesNotFollowPredictableTempSymlink(t *testing.T) {
 if runtime.GOOS == "windows" { t.Skip("symlink permissions differ on Windows") }
 dir := t.TempDir()
 target := filepath.Join(dir, "protected")
 if err := os.WriteFile(target, []byte("unchanged"), 0o600); err != nil { t.Fatal(err) }
 path := filepath.Join(dir, "record.json")
 if err := os.Symlink(target, path+".tmp"); err != nil { t.Fatal(err) }
 if err := writeJSONFile(path, map[string]string{"ok":"yes"}); err != nil { t.Fatal(err) }
 b, err := os.ReadFile(target); if err != nil { t.Fatal(err) }
 if string(b) != "unchanged" { t.Fatalf("symlink target overwritten: %q", b) }
 b, err = os.ReadFile(path); if err != nil { t.Fatal(err) }
 if string(b) != "{\n  \"ok\": \"yes\"\n}\n" { t.Fatalf("unexpected JSON: %q", b) }
}

func TestManifestAndAnchorWritersDoNotFollowTempSymlinks(t *testing.T) {
 if runtime.GOOS == "windows" { t.Skip("symlink permissions differ on Windows") }
 writers := []struct{name string; write func(string) error}{
  {"manifest", func(p string) error { return writeManifest(p, &Manifest{}) }},
  {"anchor", func(p string) error { return writeAnchorFile(p, &auditAnchor{}) }},
 }
 for _, tc := range writers {
  t.Run(tc.name, func(t *testing.T) {
   dir := t.TempDir()
   protected := filepath.Join(dir, "protected")
   if err := os.WriteFile(protected, []byte("safe"), 0o600); err != nil { t.Fatal(err) }
   dest := filepath.Join(dir, "out.json")
   if err := os.Symlink(protected, dest+".tmp"); err != nil { t.Fatal(err) }
   if err := tc.write(dest); err != nil { t.Fatal(err) }
   b, err := os.ReadFile(protected); if err != nil { t.Fatal(err) }
   if string(b) != "safe" { t.Fatalf("overwrote protected file: %q", b) }
  })
 }
}
