package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/yourusername/gollm/internal/config"
	"github.com/yourusername/gollm/internal/core"
	"github.com/yourusername/gollm/internal/providers/deepseek"
	"github.com/yourusername/gollm/internal/providers/gemini"
	"github.com/yourusername/gollm/internal/providers/openrouter"
	"github.com/yourusername/gollm/internal/providers/openai"
	"github.com/yourusername/gollm/internal/providers/anthropic"
)

func main() {
	fmt.Println("🧪 GOLLM Provider Testing Tool")
	fmt.Println(strings.Repeat("=", 50))

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("📋 Loaded config with %d providers\n", len(cfg.Providers))
	fmt.Printf("🎯 Default provider: %s\n\n", cfg.DefaultProvider)

	// Test each configured provider
	testProviders(cfg)
}

func testProviders(cfg *config.Config) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testMessage := []core.Message{
		{
			Role:    "user",
			Content: "Say 'Hello from " + "PROVIDER_NAME" + "!' in exactly 5 words.",
		},
	}

	for name, providerConfig := range cfg.Providers {
		fmt.Printf("🔧 Testing provider: %s (%s)\n", name, providerConfig.Type)

		// Create provider instance
		provider, err := createProvider(name, providerConfig)
		if err != nil {
			fmt.Printf("❌ Failed to create %s provider: %v\n\n", name, err)
			continue
		}

		// Validate configuration
		if err := provider.ValidateConfig(); err != nil {
			fmt.Printf("❌ Invalid config for %s: %v\n\n", name, err)
			continue
		}

		// Prepare test request
		testReq := &core.CompletionRequest{
			Model:    providerConfig.DefaultModel,
			Messages: replaceProviderName(testMessage, provider.Name()),
		}

		// Test the provider
		fmt.Printf("   📤 Sending test request...\n")

		start := time.Now()
		resp, err := provider.CreateCompletion(ctx, testReq)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("❌ Request failed: %v\n", err)
		} else if len(resp.Choices) == 0 {
			fmt.Printf("❌ No response choices received\n")
		} else {
			fmt.Printf("✅ Success! Response in %v\n", duration)
			fmt.Printf("   📝 Response: %s\n", resp.Choices[0].Message.Content)
			fmt.Printf("   📊 Tokens: %d prompt + %d completion = %d total\n",
				resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
		}

		// Test model listing if supported
		fmt.Printf("   📋 Testing model listing...\n")
		models, err := provider.GetModels(ctx)
		if err != nil {
			fmt.Printf("   ⚠️  Model listing failed: %v\n", err)
		} else {
			fmt.Printf("   📚 Found %d models\n", len(models))
			if len(models) > 0 {
				fmt.Printf("   🎯 Sample models: ")
				for i, model := range models {
					if i >= 3 { // Show max 3 models
						fmt.Printf("...")
						break
					}
					if i > 0 {
						fmt.Printf(", ")
					}
					fmt.Printf("%s", model.ID)
				}
				fmt.Printf("\n")
			}
		}

		fmt.Println()
	}
}

func createProvider(name string, providerConfig config.ProviderConfig) (core.Provider, error) {
	switch providerConfig.Type {
	case "openai":
		provider, err := openai.New(openai.Config{
			APIKey:     providerConfig.APIKey.Value(),
			BaseURL:    providerConfig.BaseURL,
			MaxRetries: providerConfig.MaxRetries,
			Timeout:    providerConfig.Timeout,
		})
		return provider, err

	case "anthropic":
		provider, err := anthropic.New(anthropic.Config{
			APIKey:     providerConfig.APIKey.Value(),
			BaseURL:    providerConfig.BaseURL,
			MaxRetries: providerConfig.MaxRetries,
			Timeout:    providerConfig.Timeout,
		})
		return provider, err

	case "gemini":
		provider, err := gemini.New(gemini.Config{
			APIKey:     providerConfig.APIKey.Value(),
			BaseURL:    providerConfig.BaseURL,
			MaxRetries: providerConfig.MaxRetries,
			Timeout:    providerConfig.Timeout,
			Headers:    providerConfig.CustomHeaders,
		})
		return provider, err

	case "deepseek":
		provider, err := deepseek.New(deepseek.Config{
			APIKey:     providerConfig.APIKey.Value(),
			BaseURL:    providerConfig.BaseURL,
			MaxRetries: providerConfig.MaxRetries,
			Timeout:    providerConfig.Timeout,
			Headers:    providerConfig.CustomHeaders,
		})
		return provider, err

	case "openrouter":
		provider, err := openrouter.New(openrouter.Config{
			APIKey:     providerConfig.APIKey.Value(),
			BaseURL:    providerConfig.BaseURL,
			MaxRetries: providerConfig.MaxRetries,
			Timeout:    providerConfig.Timeout,
			Headers:    providerConfig.CustomHeaders,
			SiteURL:    getExtraString(providerConfig.Extra, "site_url"),
			SiteName:   getExtraString(providerConfig.Extra, "site_name"),
		})
		return provider, err

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerConfig.Type)
	}
}

func replaceProviderName(messages []core.Message, providerName string) []core.Message {
	result := make([]core.Message, len(messages))
	for i, msg := range messages {
		result[i] = core.Message{
			Role:    msg.Role,
			Content: replaceString(msg.Content, "PROVIDER_NAME", providerName),
		}
	}
	return result
}

func replaceString(text, old, new string) string {
	// Simple string replacement
	result := ""
	for i := 0; i < len(text); i++ {
		if i+len(old) <= len(text) && text[i:i+len(old)] == old {
			result += new
			i += len(old) - 1
		} else {
			result += string(text[i])
		}
	}
	return result
}

func getExtraString(extra map[string]interface{}, key string) string {
	if extra == nil {
		return ""
	}
	if val, ok := extra[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
