//go:build !unix

package main

// restrictUmask is a no-op where there is no umask; file modes are set
// explicitly where they matter.
func restrictUmask() {}
