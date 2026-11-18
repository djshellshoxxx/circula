//go:build !unix

package main

import "os"

// privateParent has no portable owner/mode test outside Unix; Windows
// directories are protected by ACLs, which this program does not change.
func privateParent(fi os.FileInfo) bool { return fi.IsDir() }

// checkStagingParent accepts every directory where Unix permission bits do
// not describe who may rename entries.
func checkStagingParent(string) error { return nil }

// singleLink cannot read a link count portably; the identity check in
// secureRemove still refuses a file replaced after inspection.
func singleLink(os.FileInfo) bool { return true }

// overwriteOpenFlags adds nothing where O_NOFOLLOW/O_NONBLOCK do not exist;
// the identity check in zeroFile still applies.
const overwriteOpenFlags = 0
