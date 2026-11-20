package store

import "strings"

// uriEscaper percent-encodes the three characters that end or alter the path
// of a SQLite URI filename: '?' starts the query (and the driver's own
// _pragma parameters), '#' starts a fragment and '%' introduces an escape.
// SQLite decodes %HH in the path, so the requested file is opened exactly.
// All other characters, including Windows drive letters and separators,
// are passed unchanged, as before (AUD4-D01).
var uriEscaper = strings.NewReplacer("%", "%25", "?", "%3F", "#", "%23")

// fileURI returns the SQLite "file:" URI for path. Without escaping, a
// directory named "backups?x" or "backups#x" would make SQLite open a
// different file and the driver drop or replace the pragmas that follow,
// including trusted_schema(0) and foreign_keys(1).
func fileURI(path string) string { return "file:" + uriEscaper.Replace(path) }
