package treesitter

import (
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

var (
	javaLangOnce sync.Once
	javaLang     *sitter.Language
)

// JavaLanguage returns the singleton Tree-sitter Java language.
func JavaLanguage() *sitter.Language {
	javaLangOnce.Do(func() {
		javaLang = java.GetLanguage()
	})
	return javaLang
}
