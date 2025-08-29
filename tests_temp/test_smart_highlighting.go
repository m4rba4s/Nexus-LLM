package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/gollm/internal/core"
	"github.com/yourusername/gollm/internal/display"
)

func main() {
	fmt.Println("🎨 GOLLM Smart Syntax Highlighting Test")
	fmt.Println("========================================")

	// Test 1: Basic syntax highlighter
	fmt.Println("\n📋 Test 1: Basic Syntax Highlighting")
	testBasicHighlighting()

	// Test 2: Language auto-detection
	fmt.Println("\n📋 Test 2: Language Auto-Detection")
	testLanguageDetection()

	// Test 3: Smart response formatting
	fmt.Println("\n📋 Test 3: Smart Response Formatting")
	testResponseFormatting()

	// Test 4: Different themes
	fmt.Println("\n📋 Test 4: Theme Showcase")
	testThemes()

	// Test 5: Response formatter integration
	fmt.Println("\n📋 Test 5: Response Formatter")
	testResponseFormatter()

	fmt.Println("\n✅ All highlighting tests completed!")
}

func testBasicHighlighting() {
	highlighter := display.NewSyntaxHighlighter()

	// Go code example
	goCode := `package main

import "fmt"

func main() {
    message := "Hello, World! 🚀"
    fmt.Println(message)

    // Loop example
    for i := 0; i < 5; i++ {
        fmt.Printf("Count: %d\n", i+1)
    }
}`

	fmt.Println("Original Go code:")
	fmt.Println("─────────────────")
	fmt.Println(goCode)

	fmt.Println("\nWith syntax highlighting:")
	fmt.Println("─────────────────────────")
	highlighted, err := highlighter.HighlightCode(goCode, "go")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(highlighted)
}

func testLanguageDetection() {
	highlighter := display.NewSyntaxHighlighter()

	testCases := map[string]string{
		"Go": `func fibonacci(n int) int {
    if n <= 1 {
        return n
    }
    return fibonacci(n-1) + fibonacci(n-2)
}`,
		"Python": `def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)

if __name__ == "__main__":
    print(fibonacci(10))`,
		"JavaScript": `function fibonacci(n) {
    if (n <= 1) {
        return n;
    }
    return fibonacci(n-1) + fibonacci(n-2);
}

console.log(fibonacci(10));`,
		"JSON": `{
    "name": "GOLLM",
    "version": "1.0.0",
    "description": "Advanced LLM client with smart highlighting",
    "features": [
        "syntax-highlighting",
        "auto-detection",
        "streaming"
    ]
}`,
		"SQL": `SELECT users.name, COUNT(orders.id) as order_count
FROM users
LEFT JOIN orders ON users.id = orders.user_id
WHERE users.created_at > '2024-01-01'
GROUP BY users.id, users.name
HAVING COUNT(orders.id) > 5
ORDER BY order_count DESC;`,
	}

	for expectedLang, code := range testCases {
		detected := highlighter.DetectLanguage(code)
		status := "✅"
		if strings.ToLower(detected) != strings.ToLower(expectedLang) {
			status = "⚠️"
		}

		fmt.Printf("%s Expected: %-10s | Detected: %-10s\n", status, expectedLang, detected)

		// Show highlighted code
		highlighted, err := highlighter.HighlightCode(code, detected)
		if err == nil {
			fmt.Println("Preview:")
			// Show first 2 lines of highlighted code
			lines := strings.Split(highlighted, "\n")
			maxLines := 2
			if len(lines) < maxLines {
				maxLines = len(lines)
			}
			for i := 0; i < maxLines; i++ {
				fmt.Printf("  %s\n", lines[i])
			}
		}
		fmt.Println()
	}
}

func testResponseFormatting() {
	highlighter := display.NewSyntaxHighlighter()

	// Simulate LLM response with mixed content
	mockResponse := `Here's a simple Go function to reverse a string:

` + "```go" + `
func reverseString(s string) string {
    runes := []rune(s)
    for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
        runes[i], runes[j] = runes[j], runes[i]
    }
    return string(runes)
}
` + "```" + `

You can also do this with Python:

` + "```python" + `
def reverse_string(s):
    return s[::-1]

# Example usage
result = reverse_string("Hello, World!")
print(result)  # Output: !dlroW ,olleH
` + "```" + `

Both approaches work well, but Go's approach handles Unicode better.`

	fmt.Println("Original response:")
	fmt.Println("─────────────────")
	fmt.Println(mockResponse)

	fmt.Println("\nFormatted with smart highlighting:")
	fmt.Println("─────────────────────────────────")
	formatted, err := highlighter.FormatResponse(mockResponse)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(formatted)
}

func testThemes() {
	highlighter := display.NewSyntaxHighlighter()

	sampleCode := `// Sample code for theme testing
package main

import (
    "fmt"
    "time"
)

func main() {
    message := "Testing themes! 🎨"
    fmt.Println(message)

    time.Sleep(100 * time.Millisecond)
}`

	themes := []string{"github", "monokai", "solarized-dark", "vim"}

	for _, theme := range themes {
		fmt.Printf("\n🎨 Theme: %s\n", theme)
		fmt.Println(strings.Repeat("─", 30))

		highlighter.SetTheme(theme)
		highlighted, err := highlighter.HighlightCode(sampleCode, "go")
		if err != nil {
			fmt.Printf("Error with theme %s: %v\n", theme, err)
			continue
		}

		// Show first few lines
		lines := strings.Split(highlighted, "\n")
		maxLines := 4
		if len(lines) < maxLines {
			maxLines = len(lines)
		}
		for i := 0; i < maxLines; i++ {
			fmt.Println(lines[i])
		}
	}
}

func testResponseFormatter() {
	// Create a mock completion response
	mockResponse := &core.CompletionResponse{
		ID:     "test-response-123",
		Object: "chat.completion",
		Model:  "deepseek-chat",
		Choices: []core.Choice{
			{
				Index: 0,
				Message: core.Message{
					Role: core.RoleAssistant,
					Content: `Here's a DeepSeek optimized function:

` + "```go" + `
// OptimizedSort performs efficient sorting with benchmarks
func OptimizedSort(data []int) []int {
    if len(data) <= 1 {
        return data
    }

    // Use quicksort for large datasets
    result := make([]int, len(data))
    copy(result, data)
    quickSort(result, 0, len(result)-1)
    return result
}

func quickSort(arr []int, low, high int) {
    if low < high {
        pi := partition(arr, low, high)
        quickSort(arr, low, pi-1)
        quickSort(arr, pi+1, high)
    }
}
` + "```" + `

This implementation provides O(n log n) average performance! 🚀`,
				},
				FinishReason: "stop",
			},
		},
		Usage: core.Usage{
			PromptTokens:     45,
			CompletionTokens: 128,
			TotalTokens:      173,
			TotalCost:        func() *float64 { cost := 0.00024; return &cost }(),
		},
	}

	// Test different formatter configurations
	configs := []struct {
		name   string
		config display.FormatterConfig
	}{
		{
			name: "Auto Format (Default)",
			config: display.DefaultFormatterConfig(),
		},
		{
			name: "Compact Mode",
			config: display.FormatterConfig{
				Format:       display.FormatAuto,
				Mode:         display.ModeCompact,
				Theme:        "github",
				ShowMetadata: false,
				ShowTokens:   true,
				ShowTiming:   true,
				ColorEnabled: display.HasColorSupport(),
			},
		},
		{
			name: "Verbose Mode",
			config: display.FormatterConfig{
				Format:       display.FormatMarkdown,
				Mode:         display.ModeVerbose,
				Theme:        "monokai",
				ShowMetadata: true,
				ShowTokens:   true,
				ShowTiming:   true,
				ColorEnabled: display.HasColorSupport(),
			},
		},
	}

	duration := 1250 * time.Millisecond

	for _, tc := range configs {
		fmt.Printf("\n🎛️  %s\n", tc.name)
		fmt.Println(strings.Repeat("═", len(tc.name)+4))

		formatter := display.NewResponseFormatterWithConfig(tc.config)
		formatted, err := formatter.FormatResponse(mockResponse, duration)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Println(formatted)
		fmt.Println()
	}
}
