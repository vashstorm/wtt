package cli

import "strings"

// FormatCDCommand returns a shell command string that changes the current
// directory to the given path. The path is wrapped in POSIX single quotes
// with any embedded single quotes escaped using the '\” pattern.
//
// Example: FormatCDCommand("/tmp/a'b") returns `cd '/tmp/a'\”b'`
func FormatCDCommand(path string) string {
	return "cd '" + escapeSingleQuotes(path) + "'"
}

// escapeSingleQuotes replaces each ' in s with '\” (end current single-quoted
// segment, add an escaped literal single quote, start a new single-quoted segment).
func escapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", `'\''`)
}
