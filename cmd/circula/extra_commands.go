package main

import (
	"context"
	"io"
)

// commandFunc is the signature of a top-level command.
type commandFunc func(ctx context.Context, args []string, stdout, stderr io.Writer) int

// extraCommands lets feature files register commands from init() without
// editing the dispatch switch in main.go. Their usage text goes in
// extraUsage, printed by `circula help`.
var (
	extraCommands = map[string]commandFunc{}
	extraUsage    []string
)
