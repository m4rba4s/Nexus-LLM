package main

import (
	"fmt"
	"time"

	"github.com/yourusername/gollm/internal/display"
)

func main() {
	fmt.Println("🎨 GOLLM Smart Syntax Highlighting Demo")
	fmt.Println("=======================================")
	fmt.Println()

	// Создаем умный подсвечиватель
	highlighter := display.NewSyntaxHighlighter()

	// Test case 1: DeepSeek оптимизированный Go код
	fmt.Println("🚀 1. DeepSeek Optimized Go Function")
	fmt.Println("─────────────────────────────────────")

	goCode := `// Optimized fibonacci with memoization
package main

import (
	"fmt"
	"sync"
)

type FibCache struct {
	cache map[int]int
	mu    sync.RWMutex
}

func NewFibCache() *FibCache {
	return &FibCache{
		cache: make(map[int]int),
	}
}

func (fc *FibCache) Fibonacci(n int) int {
	if n <= 1 {
		return n
	}

	// Check cache first
	fc.mu.RLock()
	if val, exists := fc.cache[n]; exists {
		fc.mu.RUnlock()
		return val
	}
	fc.mu.RUnlock()

	// Calculate and cache
	result := fc.Fibonacci(n-1) + fc.Fibonacci(n-2)

	fc.mu.Lock()
	fc.cache[n] = result
	fc.mu.Unlock()

	return result
}

func main() {
	cache := NewFibCache()
	fmt.Printf("Fibonacci(40) = %d\n", cache.Fibonacci(40))
}`

	// Автодетект + подсветка
	detectedLang := highlighter.DetectLanguage(goCode)
	fmt.Printf("🔍 Detected language: %s\n", detectedLang)

	highlighted, err := highlighter.HighlightCode(goCode, detectedLang)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println(goCode)
	} else {
		fmt.Println(highlighted)
	}

	fmt.Println()

	// Test case 2: Python машинное обучение
	fmt.Println("🐍 2. Python ML Code")
	fmt.Println("────────────────────")

	pythonCode := `import numpy as np
import pandas as pd
from sklearn.model_selection import train_test_split
from sklearn.ensemble import RandomForestClassifier
from sklearn.metrics import accuracy_score, classification_report

def train_model(data_path):
    """Train a RandomForest model on dataset"""
    # Load and preprocess data
    df = pd.read_csv(data_path)
    X = df.drop('target', axis=1)
    y = df['target']

    # Split data
    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=0.2, random_state=42
    )

    # Train model
    model = RandomForestClassifier(n_estimators=100, random_state=42)
    model.fit(X_train, y_train)

    # Evaluate
    predictions = model.predict(X_test)
    accuracy = accuracy_score(y_test, predictions)

    print(f"Model accuracy: {accuracy:.4f}")
    print(classification_report(y_test, predictions))

    return model

if __name__ == "__main__":
    model = train_model("dataset.csv")`

	detectedLang = highlighter.DetectLanguage(pythonCode)
	fmt.Printf("🔍 Detected language: %s\n", detectedLang)

	highlighted, err = highlighter.HighlightCode(pythonCode, detectedLang)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println(pythonCode)
	} else {
		fmt.Println(highlighted)
	}

	fmt.Println()

	// Test case 3: Смешанный ответ LLM с кодом
	fmt.Println("🤖 3. LLM Response with Mixed Content")
	fmt.Println("─────────────────────────────────────")

	llmResponse := `Here's how to create a high-performance HTTP server in Go:

` + "```go" + `
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// API endpoint
	mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, ` + "`{\"status\":\"success\",\"data\":\"Hello World!\"}`" + `)
	})

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Server starting on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed:", err)
	}
}
` + "```" + `

You can also use it with Docker:

` + "```dockerfile" + `
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o server .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
` + "```" + `

This provides enterprise-grade performance with proper timeouts and resource management! 🚀`

	formatted, err := highlighter.FormatResponse(llmResponse)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println(llmResponse)
	} else {
		fmt.Println(formatted)
	}

	fmt.Println()

	// Test case 4: JSON API Response
	fmt.Println("📋 4. JSON API Response")
	fmt.Println("───────────────────────")

	jsonCode := `{
  "status": "success",
  "data": {
    "user": {
      "id": 12345,
      "name": "Alice Johnson",
      "email": "alice@example.com",
      "preferences": {
        "theme": "dark",
        "notifications": true,
        "language": "en"
      }
    },
    "permissions": [
      "read:profile",
      "write:profile",
      "read:data"
    ],
    "metadata": {
      "last_login": "2024-01-15T10:30:00Z",
      "session_id": "abc123def456",
      "expires_at": "2024-01-15T18:30:00Z"
    }
  },
  "timestamp": "2024-01-15T14:15:00Z",
  "version": "2.1.0"
}`

	detectedLang = highlighter.DetectLanguage(jsonCode)
	fmt.Printf("🔍 Detected language: %s\n", detectedLang)

	highlighted, err = highlighter.HighlightCode(jsonCode, detectedLang)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println(jsonCode)
	} else {
		fmt.Println(highlighted)
	}

	fmt.Println()

	// Test case 5: SQL Query
	fmt.Println("🗃️  5. Complex SQL Query")
	fmt.Println("─────────────────────")

	sqlCode := `-- Advanced analytics query for user engagement
WITH user_stats AS (
    SELECT
        u.id,
        u.name,
        COUNT(DISTINCT s.id) as session_count,
        AVG(s.duration) as avg_session_duration,
        MAX(s.created_at) as last_activity
    FROM users u
    LEFT JOIN sessions s ON u.id = s.user_id
    WHERE u.created_at >= '2024-01-01'
    GROUP BY u.id, u.name
),
engagement_tiers AS (
    SELECT *,
        CASE
            WHEN session_count >= 20 AND avg_session_duration > 300 THEN 'High'
            WHEN session_count >= 10 OR avg_session_duration > 180 THEN 'Medium'
            ELSE 'Low'
        END as engagement_level
    FROM user_stats
)
SELECT
    engagement_level,
    COUNT(*) as user_count,
    AVG(session_count) as avg_sessions,
    AVG(avg_session_duration) as avg_duration
FROM engagement_tiers
GROUP BY engagement_level
ORDER BY
    CASE engagement_level
        WHEN 'High' THEN 1
        WHEN 'Medium' THEN 2
        WHEN 'Low' THEN 3
    END;`

	detectedLang = highlighter.DetectLanguage(sqlCode)
	fmt.Printf("🔍 Detected language: %s\n", detectedLang)

	highlighted, err = highlighter.HighlightCode(sqlCode, detectedLang)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println(sqlCode)
	} else {
		fmt.Println(highlighted)
	}

	fmt.Println()

	// Performance benchmark
	fmt.Println("⚡ 6. Performance Benchmark")
	fmt.Println("─────────────────────────")

	start := time.Now()
	for i := 0; i < 100; i++ {
		_, _ = highlighter.HighlightCode(goCode, "go")
	}
	duration := time.Since(start)

	fmt.Printf("✅ Processed 100 Go files in %v (%.2fms per file)\n",
		duration, float64(duration.Nanoseconds())/1000000/100)

	fmt.Println()
	fmt.Println("🎉 Demo completed! GOLLM smart syntax highlighting is ready!")
}
