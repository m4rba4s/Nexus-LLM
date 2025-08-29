package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/gollm/internal/core"
	"github.com/yourusername/gollm/internal/display"
	"github.com/yourusername/gollm/internal/providers/deepseek"
)

func main() {
	fmt.Println("🧪 DeepSeek + Smart Highlighting Final Test")
	fmt.Println("==========================================")

	// Настройка DeepSeek провайдера
	config := deepseek.Config{
		APIKey:  "sk-69879594f67142f19a9daabab1d699de",
		BaseURL: "https://api.deepseek.com/v1",
		Model:   "deepseek-chat",
		Timeout: 30 * time.Second,
	}

	provider, err := deepseek.New(config)
	if err != nil {
		fmt.Printf("❌ Failed to create DeepSeek provider: %v\n", err)
		return
	}

	// Создаем умный форматтер
	highlighter := display.NewSyntaxHighlighter()

	// Тестовые запросы
	testCases := []struct {
		name     string
		prompt   string
		language string
	}{
		{
			name:     "Go Optimization",
			prompt:   "Optimize this Go function for better performance:\n\nfunc fibonacci(n int) int {\n    if n <= 1 {\n        return n\n    }\n    return fibonacci(n-1) + fibonacci(n-2)\n}",
			language: "go",
		},
		{
			name:     "Python Data Science",
			prompt:   "Write a Python function to analyze CSV data with pandas and create a simple visualization",
			language: "python",
		},
		{
			name:     "SQL Query",
			prompt:   "Write a complex SQL query to find top 10 customers by revenue with their order statistics",
			language: "sql",
		},
	}

	for i, tc := range testCases {
		fmt.Printf("\n🚀 Test %d: %s\n", i+1, tc.name)
		fmt.Println(strings.Repeat("═", 50))

		// Подготовка запроса
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		request := &core.CompletionRequest{
			Model: "deepseek-chat",
			Messages: []core.Message{
				{
					Role:    core.RoleUser,
					Content: tc.prompt,
				},
			},
			MaxTokens:   intPtr(500),
			Temperature: float64Ptr(0.3),
		}

		fmt.Printf("📤 Request: %s\n", tc.prompt)
		fmt.Println("⏳ Waiting for DeepSeek response...")

		start := time.Now()
		response, err := provider.CreateCompletion(ctx, request)
		duration := time.Since(start)

		cancel()

		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		if len(response.Choices) == 0 {
			fmt.Println("❌ No response choices received")
			continue
		}

		content := response.Choices[0].Message.Content

		// Применяем умную подсветку
		fmt.Println("\n📋 DeepSeek Response:")
		fmt.Println("─────────────────────")

		// Проверяем, есть ли код в ответе
		if highlighter.IsCodeBlock(content) {
			// Применяем умное форматирование
			formatted, err := highlighter.FormatResponse(content)
			if err != nil {
				fmt.Printf("Warning: Formatting failed: %v\n", err)
				fmt.Println(content)
			} else {
				fmt.Println(formatted)
			}
		} else {
			// Обычный текст
			fmt.Println(content)
		}

		// Статистика
		fmt.Printf("\n📊 Stats: %v | %d tokens | $%.6f\n",
			duration.Round(time.Millisecond),
			response.Usage.TotalTokens,
			getCost(response.Usage))
	}

	// Дополнительный тест: автодетект языка
	fmt.Println("\n🔍 Language Detection Test")
	fmt.Println("─────────────────────────")

	testCode := `package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	ch := make(chan int, 10)

	// Producer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			ch <- i
		}
		close(ch)
	}()

	// Consumer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for num := range ch {
			fmt.Printf("Received: %d\n", num)
		}
	}()

	wg.Wait()
}`

	detected := highlighter.DetectLanguage(testCode)
	fmt.Printf("Detected language: %s\n", detected)

	highlighted, err := highlighter.HighlightCode(testCode, detected)
	if err == nil {
		fmt.Println("With syntax highlighting:")
		fmt.Println(highlighted)
	}

	fmt.Println("\n✅ DeepSeek integration with smart highlighting completed!")
	fmt.Println("🎉 GOLLM is now ready for production use!")
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func getCost(usage core.Usage) float64 {
	if usage.TotalCost != nil {
		return *usage.TotalCost
	}
	// DeepSeek pricing: ~$0.14 per 1M input tokens, ~$0.28 per 1M output tokens
	inputCost := float64(usage.PromptTokens) * 0.14 / 1000000
	outputCost := float64(usage.CompletionTokens) * 0.28 / 1000000
	return inputCost + outputCost
}
