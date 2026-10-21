//go:build unix

package main

import "syscall"

// restrictUmask makes every file the process creates owner-only by default:
// the database and its WAL files, backups, manifests and restore records hold
// password hashes and member data (SA-06).
func restrictUmask() { syscall.Umask(0o077) }
