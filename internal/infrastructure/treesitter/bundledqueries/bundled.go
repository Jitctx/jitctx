// Package bundledqueries embeds the Tree-sitter query (.scm) sets that ship
// with the jitctx binary, indexed by canonical language id. The embed root
// holds one subdirectory per supported language; declaring a language whose
// directory is absent yields a domerr.LanguageUnsupportedError at load time
// (EP04US-005).
package bundledqueries

import "embed"

// bundledFS owns the on-binary representation of every shipped language's
// query set. The all: prefix preserves dot-prefixed entries (e.g. .gitkeep)
// that other packages rely on for keeping otherwise-empty trees committed.
//
//go:embed all:java
var bundledFS embed.FS
