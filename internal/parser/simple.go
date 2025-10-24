package parser

import (
	"bufio"
	"regexp"
	"strings"
)

// Symbol represents a code symbol (function, class, variable, etc.)
type Symbol struct {
	Name      string
	Kind      SymbolKind
	Line      int
	Column    int
	EndLine   int
	Signature string // Full signature/declaration
	DocString string // Documentation if available
}

// SymbolKind represents the kind of symbol
type SymbolKind string

const (
	KindFunction   SymbolKind = "function"
	KindMethod     SymbolKind = "method"
	KindClass      SymbolKind = "class"
	KindInterface  SymbolKind = "interface"
	KindStruct     SymbolKind = "struct"
	KindVariable   SymbolKind = "variable"
	KindConstant   SymbolKind = "constant"
	KindImport     SymbolKind = "import"
	KindPackage    SymbolKind = "package"
	KindType       SymbolKind = "type"
)

// SimpleParser is a simple regex-based parser for extracting symbols
// Used as a fallback when LSP is not available
type SimpleParser struct {
	language string
	patterns map[SymbolKind]*regexp.Regexp
}

// NewSimpleParser creates a new simple parser for a language
func NewSimpleParser(language string) *SimpleParser {
	patterns := getPatterns(language)
	return &SimpleParser{
		language: language,
		patterns: patterns,
	}
}

// Parse parses source code and extracts symbols
func (p *SimpleParser) Parse(source string) []Symbol {
	symbols := []Symbol{}
	scanner := bufio.NewScanner(strings.NewReader(source))

	lineNum := 0
	var currentDoc strings.Builder

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Collect documentation comments
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			currentDoc.WriteString(strings.TrimSpace(trimmed[2:]))
			currentDoc.WriteString(" ")
			continue
		}

		// Try to match symbols
		for kind, pattern := range p.patterns {
			if matches := pattern.FindStringSubmatch(line); matches != nil {
				symbol := Symbol{
					Kind:      kind,
					Line:      lineNum,
					Column:    strings.Index(line, matches[0]),
					Signature: strings.TrimSpace(line),
					DocString: strings.TrimSpace(currentDoc.String()),
				}

				// Extract name based on pattern
				if len(matches) > 1 {
					symbol.Name = matches[1]
				}

				symbols = append(symbols, symbol)
				currentDoc.Reset()
			}
		}
	}

	return symbols
}

// getPatterns returns regex patterns for a language
func getPatterns(language string) map[SymbolKind]*regexp.Regexp {
	switch language {
	case "go":
		return getGoPatterns()
	case "python":
		return getPythonPatterns()
	case "javascript", "typescript":
		return getJavaScriptPatterns()
	default:
		return make(map[SymbolKind]*regexp.Regexp)
	}
}

// getGoPatterns returns patterns for Go
func getGoPatterns() map[SymbolKind]*regexp.Regexp {
	return map[SymbolKind]*regexp.Regexp{
		KindPackage:   regexp.MustCompile(`^package\s+(\w+)`),
		KindImport:    regexp.MustCompile(`^\s*import\s+.*"(.+)"`),
		KindFunction:  regexp.MustCompile(`^func\s+(\w+)\s*\(`),
		KindMethod:    regexp.MustCompile(`^func\s+\([^)]+\)\s+(\w+)\s*\(`),
		KindStruct:    regexp.MustCompile(`^type\s+(\w+)\s+struct`),
		KindInterface: regexp.MustCompile(`^type\s+(\w+)\s+interface`),
		KindType:      regexp.MustCompile(`^type\s+(\w+)\s+`),
		KindConstant:  regexp.MustCompile(`^const\s+(\w+)`),
		KindVariable:  regexp.MustCompile(`^var\s+(\w+)`),
	}
}

// getPythonPatterns returns patterns for Python
func getPythonPatterns() map[SymbolKind]*regexp.Regexp {
	return map[SymbolKind]*regexp.Regexp{
		KindImport:   regexp.MustCompile(`^import\s+(\w+)`),
		KindClass:    regexp.MustCompile(`^class\s+(\w+)`),
		KindFunction: regexp.MustCompile(`^def\s+(\w+)\s*\(`),
		KindMethod:   regexp.MustCompile(`^\s+def\s+(\w+)\s*\(`),
		KindVariable: regexp.MustCompile(`^(\w+)\s*=`),
	}
}

// getJavaScriptPatterns returns patterns for JavaScript/TypeScript
func getJavaScriptPatterns() map[SymbolKind]*regexp.Regexp {
	return map[SymbolKind]*regexp.Regexp{
		KindImport:   regexp.MustCompile(`^import\s+.*from\s+['"](.+)['"]`),
		KindClass:    regexp.MustCompile(`^class\s+(\w+)`),
		KindFunction: regexp.MustCompile(`^(?:function\s+(\w+)\s*\(|(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?(?:function|\([^)]*\)\s*=>))`),
		KindMethod:   regexp.MustCompile(`^\s+(\w+)\s*\([^)]*\)\s*{`),
	}
}

// FindSymbolByName finds a symbol by name in the parsed symbols
func FindSymbolByName(symbols []Symbol, name string) *Symbol {
	for i := range symbols {
		if symbols[i].Name == name {
			return &symbols[i]
		}
	}
	return nil
}

// FindSymbolsAtLine finds all symbols at a given line
func FindSymbolsAtLine(symbols []Symbol, line int) []Symbol {
	result := []Symbol{}
	for _, symbol := range symbols {
		if symbol.Line == line || (symbol.EndLine > 0 && line >= symbol.Line && line <= symbol.EndLine) {
			result = append(result, symbol)
		}
	}
	return result
}

// FindSymbolsByKind finds all symbols of a given kind
func FindSymbolsByKind(symbols []Symbol, kind SymbolKind) []Symbol {
	result := []Symbol{}
	for _, symbol := range symbols {
		if symbol.Kind == kind {
			result = append(result, symbol)
		}
	}
	return result
}
