// Package display provides syntax highlighting and code formatting capabilities.
package display

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/mattn/go-isatty"
)

// SyntaxHighlighter provides intelligent code highlighting capabilities
type SyntaxHighlighter struct {
	theme         string
	formatter     chroma.Formatter
	colorEnabled  bool
	tabWidth      int
	lineNumbers   bool
	autoDetect    bool
	fallbackLexer chroma.Lexer
}

// Theme represents a color scheme for syntax highlighting
type Theme struct {
	Name        string
	Background  string
	Foreground  string
	Keyword     string
	String      string
	Comment     string
	Number      string
	Function    string
	Variable    string
	Type        string
	Operator    string
	Punctuation string
}

// Predefined themes
var (
	// DarkTheme - Modern dark theme optimized for terminals
	DarkTheme = Theme{
		Name:        "dark",
		Background:  "#1e1e1e",
		Foreground:  "#d4d4d4",
		Keyword:     "#569cd6",
		String:      "#ce9178",
		Comment:     "#6a9955",
		Number:      "#b5cea8",
		Function:    "#dcdcaa",
		Variable:    "#9cdcfe",
		Type:        "#4ec9b0",
		Operator:    "#d4d4d4",
		Punctuation: "#d4d4d4",
	}

	// LightTheme - Clean light theme
	LightTheme = Theme{
		Name:        "light",
		Background:  "#ffffff",
		Foreground:  "#000000",
		Keyword:     "#0000ff",
		String:      "#a31515",
		Comment:     "#008000",
		Number:      "#098658",
		Function:    "#795e26",
		Variable:    "#001080",
		Type:        "#267f99",
		Operator:    "#000000",
		Punctuation: "#000000",
	}

	// SolarizedTheme - Popular solarized color scheme
	SolarizedTheme = Theme{
		Name:        "solarized",
		Background:  "#002b36",
		Foreground:  "#839496",
		Keyword:     "#268bd2",
		String:      "#2aa198",
		Comment:     "#586e75",
		Number:      "#d33682",
		Function:    "#b58900",
		Variable:    "#268bd2",
		Type:        "#dc322f",
		Operator:    "#859900",
		Punctuation: "#93a1a1",
	}
)

// NewSyntaxHighlighter creates a new syntax highlighter with default settings
func NewSyntaxHighlighter() *SyntaxHighlighter {
	return &SyntaxHighlighter{
		theme:         "github",
		formatter:     nil, // Will be initialized on first use
		colorEnabled:  isatty.IsTerminal(os.Stdout.Fd()),
		tabWidth:      4,
		lineNumbers:   false,
		autoDetect:    true,
		fallbackLexer: lexers.Get("text"),
	}
}

// NewSyntaxHighlighterWithTheme creates a highlighter with a specific theme
func NewSyntaxHighlighterWithTheme(theme string) *SyntaxHighlighter {
	sh := NewSyntaxHighlighter()
	sh.theme = theme
	return sh
}

// SetTheme sets the color theme for syntax highlighting
func (sh *SyntaxHighlighter) SetTheme(theme string) {
	sh.theme = theme
	sh.formatter = nil // Reset formatter to apply new theme
}

// SetColorEnabled enables or disables color output
func (sh *SyntaxHighlighter) SetColorEnabled(enabled bool) {
	sh.colorEnabled = enabled
	sh.formatter = nil // Reset formatter
}

// SetLineNumbers enables or disables line numbers
func (sh *SyntaxHighlighter) SetLineNumbers(enabled bool) {
	sh.lineNumbers = enabled
	sh.formatter = nil // Reset formatter
}

// getFormatter returns the appropriate formatter for the current settings
func (sh *SyntaxHighlighter) getFormatter() chroma.Formatter {
	if sh.formatter != nil {
		return sh.formatter
	}

	if !sh.colorEnabled {
		sh.formatter = formatters.Get("terminal")
		return sh.formatter
	}

	// Use terminal256 formatter for colored output
	sh.formatter = formatters.Get("terminal256")
	if sh.formatter == nil {
		sh.formatter = formatters.Get("terminal")
	}

	return sh.formatter
}

// DetectLanguage attempts to automatically detect the programming language
func (sh *SyntaxHighlighter) DetectLanguage(content string) string {
	if !sh.autoDetect {
		return ""
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return "text"
	}

	// Try lexer analysis first
	lexer := lexers.Analyse(content)
	if lexer != nil {
		return lexer.Config().Name
	}

	// Fallback to pattern-based detection
	return sh.detectByPatterns(content)
}

// detectByPatterns uses regex patterns to detect programming languages
func (sh *SyntaxHighlighter) detectByPatterns(content string) string {
	// Define language patterns
	patterns := map[string][]*regexp.Regexp{
		"go": {
			regexp.MustCompile(`^package\s+\w+`),
			regexp.MustCompile(`func\s+\w+\s*\(`),
			regexp.MustCompile(`import\s*\(`),
			regexp.MustCompile(`type\s+\w+\s+(struct|interface)`),
			regexp.MustCompile(`\bgo\s+(mod|run|build|test)\b`),
		},
		"python": {
			regexp.MustCompile(`^#!/usr/bin/env\s+python`),
			regexp.MustCompile(`^import\s+\w+`),
			regexp.MustCompile(`^from\s+\w+\s+import`),
			regexp.MustCompile(`def\s+\w+\s*\(`),
			regexp.MustCompile(`class\s+\w+\s*(\(.*\))?\s*:`),
			regexp.MustCompile(`if\s+__name__\s*==\s*["']__main__["']`),
		},
		"javascript": {
			regexp.MustCompile(`function\s+\w+\s*\(`),
			regexp.MustCompile(`const\s+\w+\s*=`),
			regexp.MustCompile(`let\s+\w+\s*=`),
			regexp.MustCompile(`var\s+\w+\s*=`),
			regexp.MustCompile(`=>\s*{`),
			regexp.MustCompile(`require\s*\(\s*['"].+['"]\s*\)`),
			regexp.MustCompile(`import\s+.*from\s+['"].+['"]`),
		},
		"java": {
			regexp.MustCompile(`^package\s+[\w\.]+;`),
			regexp.MustCompile(`public\s+(class|interface)\s+\w+`),
			regexp.MustCompile(`public\s+static\s+void\s+main`),
			regexp.MustCompile(`import\s+[\w\.]+;`),
			regexp.MustCompile(`@\w+(\(.*\))?`), // Annotations
		},
		"rust": {
			regexp.MustCompile(`fn\s+\w+\s*\(`),
			regexp.MustCompile(`use\s+\w+::`),
			regexp.MustCompile(`pub\s+(fn|struct|enum)`),
			regexp.MustCompile(`let\s+(mut\s+)?\w+\s*=`),
			regexp.MustCompile(`match\s+\w+\s*{`),
		},
		"cpp": {
			regexp.MustCompile(`#include\s*<.*>`),
			regexp.MustCompile(`using\s+namespace\s+std`),
			regexp.MustCompile(`int\s+main\s*\(`),
			regexp.MustCompile(`std::\w+`),
			regexp.MustCompile(`template\s*<.*>`),
		},
		"c": {
			regexp.MustCompile(`#include\s*<.*\.h>`),
			regexp.MustCompile(`int\s+main\s*\(`),
			regexp.MustCompile(`printf\s*\(`),
			regexp.MustCompile(`malloc\s*\(`),
		},
		"html": {
			regexp.MustCompile(`<!DOCTYPE\s+html>`),
			regexp.MustCompile(`<html.*>`),
			regexp.MustCompile(`<head.*>.*</head>`),
			regexp.MustCompile(`<body.*>.*</body>`),
		},
		"css": {
			regexp.MustCompile(`\w+\s*{[^}]*}`),
			regexp.MustCompile(`@media\s*\([^)]*\)`),
			regexp.MustCompile(`#\w+\s*{`),
			regexp.MustCompile(`\.\w+\s*{`),
		},
		"sql": {
			regexp.MustCompile(`(?i)SELECT\s+.*FROM`),
			regexp.MustCompile(`(?i)INSERT\s+INTO`),
			regexp.MustCompile(`(?i)UPDATE\s+.*SET`),
			regexp.MustCompile(`(?i)CREATE\s+(TABLE|DATABASE|INDEX)`),
			regexp.MustCompile(`(?i)ALTER\s+TABLE`),
		},
		"yaml": {
			regexp.MustCompile(`^\w+:\s*$`),
			regexp.MustCompile(`^\s*-\s+\w+:`),
			regexp.MustCompile(`^---\s*$`),
		},
		"json": {
			regexp.MustCompile(`^\s*{.*}\s*$`),
			regexp.MustCompile(`^\s*\[.*\]\s*$`),
			regexp.MustCompile(`"\w+":\s*".*"`),
		},
		"xml": {
			regexp.MustCompile(`<\?xml\s+version`),
			regexp.MustCompile(`<\w+.*>.*</\w+>`),
		},
		"markdown": {
			regexp.MustCompile(`^#+\s+.*`),
			regexp.MustCompile(`^\*{1,3}.*\*{1,3}`),
			regexp.MustCompile(`^\[.*\]\(.*\)`),
			regexp.MustCompile("^```\\w*"),
		},
		"bash": {
			regexp.MustCompile(`^#!/bin/(bash|sh)`),
			regexp.MustCompile(`\$\{?\w+\}?`),
			regexp.MustCompile(`if\s+\[.*\];\s*then`),
			regexp.MustCompile(`for\s+\w+\s+in`),
		},
		"dockerfile": {
			regexp.MustCompile(`^FROM\s+\w+`),
			regexp.MustCompile(`^RUN\s+`),
			regexp.MustCompile(`^COPY\s+`),
			regexp.MustCompile(`^WORKDIR\s+`),
			regexp.MustCompile(`^EXPOSE\s+\d+`),
		},
	}

	// Score each language based on pattern matches
	scores := make(map[string]int)
	lines := strings.Split(content, "\n")

	for lang, regexes := range patterns {
		score := 0
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			for _, regex := range regexes {
				if regex.MatchString(line) {
					score++
					break // Only count one match per line
				}
			}
		}
		if score > 0 {
			scores[lang] = score
		}
	}

	// Find the language with the highest score
	maxScore := 0
	detectedLang := "text"
	for lang, score := range scores {
		if score > maxScore {
			maxScore = score
			detectedLang = lang
		}
	}

	return detectedLang
}

// HighlightCode applies syntax highlighting to code content
func (sh *SyntaxHighlighter) HighlightCode(content, language string) (string, error) {
	if !sh.colorEnabled {
		return content, nil
	}

	// Auto-detect language if not specified
	if language == "" {
		language = sh.DetectLanguage(content)
	}

	// Get lexer for the language
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = sh.fallbackLexer
	}

	// Get style
	style := styles.Get(sh.theme)
	if style == nil {
		style = styles.Get("github")
	}

	// Get formatter
	formatter := sh.getFormatter()

	// Tokenize and format
	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content, err
	}

	var buf strings.Builder
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return content, err
	}

	return buf.String(), nil
}

// IsCodeBlock determines if content appears to be a code block
func (sh *SyntaxHighlighter) IsCodeBlock(content string) bool {
	content = strings.TrimSpace(content)

	// Check for common code block indicators
	codeIndicators := []string{
		"```",                                  // Markdown code blocks
		"func ", "function ", "def ", "class ", // Function definitions
		"import ", "from ", "package ", "use ", // Import statements
		"public class", "private class", // Java classes
		"SELECT ", "INSERT ", "UPDATE ", // SQL
		"<html", "<!DOCTYPE", // HTML
		"#include", "using namespace", // C/C++
	}

	lowerContent := strings.ToLower(content)
	for _, indicator := range codeIndicators {
		if strings.Contains(lowerContent, strings.ToLower(indicator)) {
			return true
		}
	}

	// Check for code-like patterns
	lines := strings.Split(content, "\n")
	codeLines := 0
	totalLines := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		totalLines++

		// Look for code-like patterns
		if sh.looksLikeCode(line) {
			codeLines++
		}
	}

	// If more than 30% of non-empty lines look like code, consider it a code block
	if totalLines > 0 && float64(codeLines)/float64(totalLines) > 0.3 {
		return true
	}

	return false
}

// looksLikeCode checks if a line looks like code
func (sh *SyntaxHighlighter) looksLikeCode(line string) bool {
	// Patterns that suggest code
	codePatterns := []string{
		"{", "}", "[", "]", "()", // Brackets and braces
		";", "::", "->", "=>", // Programming punctuation
		"func", "function", "def", "class", "struct", // Keywords
		"if", "else", "for", "while", "switch", // Control flow
		"var", "let", "const", "int", "string", "bool", // Variable declarations
		"#include", "import", "from", "package", "use", // Imports
		"public", "private", "protected", "static", // Modifiers
	}

	lowerLine := strings.ToLower(line)
	for _, pattern := range codePatterns {
		if strings.Contains(lowerLine, pattern) {
			return true
		}
	}

	// Check for indentation (common in code)
	if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
		return true
	}

	// Check for assignment operators
	assignmentOps := []string{"=", "==", "!=", "<=", ">=", "+=", "-=", "*=", "/="}
	for _, op := range assignmentOps {
		if strings.Contains(line, op) {
			return true
		}
	}

	return false
}

// FormatResponse intelligently formats an LLM response with syntax highlighting
func (sh *SyntaxHighlighter) FormatResponse(response string) (string, error) {
	if !sh.colorEnabled || response == "" {
		return response, nil
	}

	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(response))

	var currentBlock strings.Builder
	var inCodeBlock bool
	var codeBlockLanguage string

	for scanner.Scan() {
		line := scanner.Text()

		// Check for markdown code block start
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if inCodeBlock {
				// End of code block
				if currentBlock.Len() > 0 {
					highlighted, err := sh.HighlightCode(currentBlock.String(), codeBlockLanguage)
					if err != nil {
						result.WriteString(currentBlock.String())
					} else {
						result.WriteString(highlighted)
					}
				}
				result.WriteString("\n")
				currentBlock.Reset()
				inCodeBlock = false
				codeBlockLanguage = ""
			} else {
				// Start of code block
				inCodeBlock = true
				// Extract language hint
				parts := strings.Fields(strings.TrimSpace(line))
				if len(parts) > 1 {
					codeBlockLanguage = parts[1]
				}
			}
			continue
		}

		if inCodeBlock {
			currentBlock.WriteString(line + "\n")
		} else {
			// Regular text - check if it looks like inline code
			if sh.IsCodeBlock(line) && !strings.Contains(line, " ") {
				// Single line that looks like code
				highlighted, err := sh.HighlightCode(line, "")
				if err != nil {
					result.WriteString(line + "\n")
				} else {
					result.WriteString(highlighted + "\n")
				}
			} else {
				result.WriteString(line + "\n")
			}
		}
	}

	// Handle case where response ends while in a code block
	if inCodeBlock && currentBlock.Len() > 0 {
		highlighted, err := sh.HighlightCode(currentBlock.String(), codeBlockLanguage)
		if err != nil {
			result.WriteString(currentBlock.String())
		} else {
			result.WriteString(highlighted)
		}
	}

	return strings.TrimRight(result.String(), "\n"), scanner.Err()
}

// GetAvailableThemes returns a list of available color themes
func (sh *SyntaxHighlighter) GetAvailableThemes() []string {
	return []string{
		"github", "monokai", "solarized-dark", "solarized-light",
		"vim", "emacs", "vs", "xcode", "autumn", "borland",
		"bw", "colorful", "default", "friendly", "fruity",
		"manni", "murphy", "native", "pastie", "perldoc",
		"rainbow_dash", "rrt", "tango", "trac", "dracula",
	}
}

// PrintThemePreview shows a preview of a color theme
func (sh *SyntaxHighlighter) PrintThemePreview(theme string) error {
	sampleCode := `// Sample Go code
package main

import (
	"fmt"
	"time"
)

func main() {
	message := "Hello, World!"
	fmt.Println(message)

	// This is a comment
	for i := 0; i < 5; i++ {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("Count: %d\n", i+1)
	}
}
`

	originalTheme := sh.theme
	sh.SetTheme(theme)
	defer sh.SetTheme(originalTheme)

	highlighted, err := sh.HighlightCode(sampleCode, "go")
	if err != nil {
		return err
	}

	fmt.Printf("Theme: %s\n", theme)
	fmt.Println("─────────────────────")
	fmt.Print(highlighted)
	fmt.Println("─────────────────────")

	return nil
}

// GetSupportedLanguages returns a list of supported programming languages
func (sh *SyntaxHighlighter) GetSupportedLanguages() []string {
	return []string{
		"go", "python", "javascript", "typescript", "java", "cpp", "c", "rust",
		"ruby", "php", "swift", "kotlin", "scala", "bash", "sh", "sql", "html",
		"css", "xml", "json", "yaml", "markdown", "dockerfile", "toml",
	}
}

// ProcessContent intelligently processes content for display
func (sh *SyntaxHighlighter) ProcessContent(content, contentType string) (string, error) {
	if content == "" {
		return "", nil
	}

	// Handle different content types
	switch contentType {
	case "code", "text/plain":
		return sh.FormatResponse(content)
	case "json", "application/json":
		return sh.HighlightCode(content, "json")
	case "xml", "application/xml", "text/xml":
		return sh.HighlightCode(content, "xml")
	case "html", "text/html":
		return sh.HighlightCode(content, "html")
	case "yaml", "application/yaml", "text/yaml":
		return sh.HighlightCode(content, "yaml")
	default:
		// Auto-detect and format
		return sh.FormatResponse(content)
	}
}

// HasColorSupport checks if the current terminal supports color output
func HasColorSupport() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) && os.Getenv("NO_COLOR") == ""
}

// StripANSIColors removes ANSI color codes from text
func StripANSIColors(text string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(text, "")
}

// WrapText wraps text to fit within specified width
func WrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(text))

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) <= width {
			result.WriteString(line + "\n")
			continue
		}

		// Wrap long lines
		words := strings.Fields(line)
		currentLine := ""

		for _, word := range words {
			if len(currentLine)+len(word)+1 > width {
				if currentLine != "" {
					result.WriteString(currentLine + "\n")
					currentLine = word
				} else {
					// Word is longer than width, break it
					for len(word) > width {
						result.WriteString(word[:width] + "\n")
						word = word[width:]
					}
					currentLine = word
				}
			} else {
				if currentLine != "" {
					currentLine += " " + word
				} else {
					currentLine = word
				}
			}
		}

		if currentLine != "" {
			result.WriteString(currentLine + "\n")
		}
	}

	return strings.TrimRight(result.String(), "\n")
}
