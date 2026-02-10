package ui

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

func HighlightCode(code, language string) (string, error) {
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	style := styles.Get("catppuccin-mocha")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code, err
	}

	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return code, err
	}

	return buf.String(), nil
}

func GetLanguageFromExtension(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))

	languageMap := map[string]string{
		"go":         "go",
		"js":         "javascript",
		"ts":         "typescript",
		"tsx":        "tsx",
		"jsx":        "jsx",
		"py":         "python",
		"java":       "java",
		"c":          "c",
		"cpp":        "cpp",
		"cc":         "cpp",
		"cxx":        "cpp",
		"rs":         "rust",
		"rb":         "ruby",
		"php":        "php",
		"sql":        "sql",
		"sh":         "bash",
		"bash":       "bash",
		"yaml":       "yaml",
		"yml":        "yaml",
		"json":       "json",
		"xml":        "xml",
		"html":       "html",
		"css":        "css",
		"scss":       "scss",
		"md":         "markdown",
		"txt":        "text",
		"toml":       "toml",
		"ini":        "ini",
		"dockerfile": "dockerfile",
		"makefile":   "makefile",
	}

	if lang, ok := languageMap[ext]; ok {
		return lang
	}

	return "text"
}
