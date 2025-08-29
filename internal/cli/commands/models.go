// Package commands provides model management commands for GOLLM.
package commands

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yourusername/gollm/internal/config"
	"github.com/yourusername/gollm/internal/core"

	// Import providers to trigger their init() registration
	_ "github.com/yourusername/gollm/internal/providers/anthropic"
	_ "github.com/yourusername/gollm/internal/providers/mock"
	_ "github.com/yourusername/gollm/internal/providers/openai"
)

// ModelFlags contains flags specific to model commands.
type ModelFlags struct {
	Provider     string
	OutputFormat string
	ShowPricing  bool
	ShowAll      bool
	Detailed     bool
}

// NewModelsCommand creates the models command with subcommands.
func NewModelsCommand() *cobra.Command {
	flags := &ModelFlags{}

	cmd := &cobra.Command{
		Use:   "models",
		Short: "List and manage LLM models",
		Long: `List available models from configured providers and show model information.

The models command allows you to discover available models, view their capabilities,
pricing information, and other metadata from your configured providers.

Examples:
  # List all models from all providers
  gollm models list

  # List models from specific provider
  gollm models list --provider openai

  # Show detailed model information
  gollm models list --detailed

  # Show model pricing information
  gollm models list --pricing

  # Get information about a specific model
  gollm models info gpt-4

  # List models in JSON format
  gollm models list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to list command if no subcommand specified
			return runModelsListCommand(flags)
		},
	}

	// Add flags
	addModelsFlags(cmd, flags)

	// Add subcommands
	cmd.AddCommand(newModelsListCommand(flags))
	cmd.AddCommand(newModelsInfoCommand(flags))

	return cmd
}

// addModelsFlags adds common flags for models commands.
func addModelsFlags(cmd *cobra.Command, flags *ModelFlags) {
	f := cmd.PersistentFlags()

	f.StringVarP(&flags.Provider, "provider", "p", "",
		"filter models by provider (openai, anthropic, ollama, etc.)")
	f.StringVarP(&flags.OutputFormat, "output", "o", "text",
		"output format (text, json, yaml)")
	f.BoolVar(&flags.ShowPricing, "pricing", false,
		"show pricing information")
	f.BoolVar(&flags.ShowAll, "all", false,
		"show all models including deprecated")
	f.BoolVar(&flags.Detailed, "detailed", false,
		"show detailed model information")

	// Add completion for provider flag
	cmd.RegisterFlagCompletionFunc("provider", completeProviders)
}

// newModelsListCommand creates the 'models list' subcommand.
func newModelsListCommand(flags *ModelFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available models",
		Long: `List all available models from configured providers.

This command queries each configured provider to retrieve their available models
and displays them with relevant information like capabilities and pricing.

Examples:
  # List all models
  gollm models list

  # List models from OpenAI only
  gollm models list --provider openai

  # Show detailed information
  gollm models list --detailed

  # Show with pricing
  gollm models list --pricing

  # Output as JSON
  gollm models list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelsListCommand(flags)
		},
	}
}

// newModelsInfoCommand creates the 'models info' subcommand.
func newModelsInfoCommand(flags *ModelFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "info <model-id>",
		Short: "Show detailed information about a specific model",
		Long: `Show detailed information about a specific model including its capabilities,
pricing, context window, and other metadata.

Examples:
  # Get info about GPT-4
  gollm models info gpt-4

  # Get info with specific provider context
  gollm models info gpt-3.5-turbo --provider openai

  # Output as JSON
  gollm models info claude-3-sonnet --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelsInfoCommand(args[0], flags)
		},
	}
}

// runModelsListCommand executes the models list command.
func runModelsListCommand(flags *ModelFlags) error {
	ctx, cancel := setupContextWithTimeout(30 * time.Second)
	defer cancel()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get providers to query
	providersToQuery, err := getProvidersToQuery(flags.Provider, cfg)
	if err != nil {
		return err
	}

	if len(providersToQuery) == 0 {
		fmt.Println("No providers configured or available")
		return nil
	}

	// Collect models from all providers
	allModels := make(map[string][]core.Model)
	var errors []string

	for _, providerName := range providersToQuery {
		models, err := getModelsFromProvider(ctx, providerName, cfg)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Provider %s: %v", providerName, err))
			continue
		}
		allModels[providerName] = models
	}

	// Display errors if any (but don't fail completely)
	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "Warnings:\n")
		for _, errMsg := range errors {
			fmt.Fprintf(os.Stderr, "  • %s\n", errMsg)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Format and display results
	return formatModelsOutput(allModels, flags)
}

// runModelsInfoCommand executes the models info command.
func runModelsInfoCommand(modelID string, flags *ModelFlags) error {
	ctx, cancel := setupContextWithTimeout(30 * time.Second)
	defer cancel()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get providers to search
	providersToQuery, err := getProvidersToQuery(flags.Provider, cfg)
	if err != nil {
		return err
	}

	// Find the model across providers
	var foundModel *core.Model
	var foundProvider string

	for _, providerName := range providersToQuery {
		models, err := getModelsFromProvider(ctx, providerName, cfg)
		if err != nil {
			continue // Skip providers with errors
		}

		for _, model := range models {
			if model.ID == modelID || strings.Contains(model.ID, modelID) {
				foundModel = &model
				foundProvider = providerName
				break
			}
		}

		if foundModel != nil {
			break
		}
	}

	if foundModel == nil {
		return fmt.Errorf("model %q not found in any configured provider", modelID)
	}

	// Format and display model info
	return formatModelInfo(*foundModel, foundProvider, flags)
}

// getProvidersToQuery determines which providers to query based on flags and config.
func getProvidersToQuery(providerFlag string, cfg *config.Config) ([]string, error) {
	if providerFlag != "" {
		// Validate that the specified provider is configured
		if !cfg.HasProvider(providerFlag) {
			available := cfg.ListProviders()
			return nil, fmt.Errorf("provider %q not configured; available providers: %s",
				providerFlag, strings.Join(available, ", "))
		}
		return []string{providerFlag}, nil
	}

	// Return all configured providers
	providers := cfg.ListProviders()
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers configured")
	}

	return providers, nil
}

// getModelsFromProvider retrieves models from a specific provider.
func getModelsFromProvider(ctx context.Context, providerName string, cfg *config.Config) ([]core.Model, error) {
	// Create provider instance
	provider, err := createProvider(providerName, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Check if provider supports model listing
	modelLister, ok := provider.(core.ModelLister)
	if !ok {
		// Return some default models for providers that don't support listing
		return getDefaultModelsForProvider(providerName), nil
	}

	// Get models from provider
	models, err := modelLister.GetModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}

	// Sort models by ID for consistent output
	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})

	return models, nil
}

// getDefaultModelsForProvider returns default models for providers that don't support listing.
func getDefaultModelsForProvider(providerName string) []core.Model {
	switch strings.ToLower(providerName) {
	case "openai":
		return []core.Model{
			{ID: "gpt-3.5-turbo", Provider: providerName, SupportsFunctions: true, SupportsStreaming: true},
			{ID: "gpt-4", Provider: providerName, SupportsFunctions: true, SupportsStreaming: true},
			{ID: "gpt-4-turbo", Provider: providerName, SupportsFunctions: true, SupportsStreaming: true, SupportsVision: true},
		}
	case "anthropic":
		return []core.Model{
			{ID: "claude-3-haiku-20240307", Provider: providerName, SupportsStreaming: true},
			{ID: "claude-3-sonnet-20240229", Provider: providerName, SupportsStreaming: true},
			{ID: "claude-3-opus-20240229", Provider: providerName, SupportsStreaming: true},
		}
	case "mock":
		return []core.Model{
			{ID: "mock-gpt-3.5-turbo", Provider: providerName, SupportsFunctions: true, SupportsStreaming: true, MaxTokens: intPtr(4096)},
			{ID: "mock-gpt-4", Provider: providerName, SupportsFunctions: true, SupportsStreaming: true, MaxTokens: intPtr(8192)},
			{ID: "mock-claude-3-sonnet", Provider: providerName, SupportsStreaming: true, MaxTokens: intPtr(200000)},
		}
	default:
		return []core.Model{}
	}
}

// formatModelsOutput formats and displays models based on output format.
func formatModelsOutput(allModels map[string][]core.Model, flags *ModelFlags) error {
	switch flags.OutputFormat {
	case "json":
		return outputModelsJSON(allModels)
	case "yaml":
		return outputModelsYAML(allModels)
	default:
		return outputModelsText(allModels, flags)
	}
}

// outputModelsText outputs models in human-readable text format.
func outputModelsText(allModels map[string][]core.Model, flags *ModelFlags) error {
	totalModels := 0
	for _, models := range allModels {
		totalModels += len(models)
	}

	fmt.Printf("Available Models (%d total)\n\n", totalModels)

	// Sort providers for consistent output
	var providers []string
	for provider := range allModels {
		providers = append(providers, provider)
	}
	sort.Strings(providers)

	for _, provider := range providers {
		models := allModels[provider]
		if len(models) == 0 {
			continue
		}

		fmt.Printf("Provider: %s (%d models)\n", strings.ToUpper(provider), len(models))
		fmt.Printf(strings.Repeat("─", 50) + "\n")

		for _, model := range models {
			fmt.Printf("  %s\n", model.ID)

			if flags.Detailed {
				if model.Description != "" {
					fmt.Printf("    Description: %s\n", model.Description)
				}
				if model.MaxTokens != nil {
					fmt.Printf("    Max Tokens: %s\n", formatNumber(*model.MaxTokens))
				}

				// Show capabilities
				var capabilities []string
				if model.SupportsFunctions {
					capabilities = append(capabilities, "functions")
				}
				if model.SupportsStreaming {
					capabilities = append(capabilities, "streaming")
				}
				if model.SupportsVision {
					capabilities = append(capabilities, "vision")
				}
				if len(capabilities) > 0 {
					fmt.Printf("    Capabilities: %s\n", strings.Join(capabilities, ", "))
				}
			}

			if flags.ShowPricing {
				if model.InputCostPer1K != nil {
					fmt.Printf("    Input: $%.4f/1K tokens", *model.InputCostPer1K)
				}
				if model.OutputCostPer1K != nil {
					fmt.Printf("    Output: $%.4f/1K tokens", *model.OutputCostPer1K)
				}
				if model.InputCostPer1K != nil || model.OutputCostPer1K != nil {
					fmt.Println()
				}
			}

			if flags.Detailed || flags.ShowPricing {
				fmt.Println()
			}
		}
		fmt.Println()
	}

	return nil
}

// formatModelInfo formats and displays detailed information about a specific model.
func formatModelInfo(model core.Model, provider string, flags *ModelFlags) error {
	switch flags.OutputFormat {
	case "json":
		return outputModelJSON(model)
	case "yaml":
		return outputModelYAML(model)
	default:
		return outputModelText(model, provider)
	}
}

// outputModelText outputs model info in text format.
func outputModelText(model core.Model, provider string) error {
	fmt.Printf("Model Information\n")
	fmt.Printf("=================\n\n")

	fmt.Printf("ID: %s\n", model.ID)
	fmt.Printf("Provider: %s\n", provider)

	if model.Description != "" {
		fmt.Printf("Description: %s\n", model.Description)
	}

	if model.MaxTokens != nil {
		fmt.Printf("Max Tokens: %s\n", formatNumber(*model.MaxTokens))
	}

	if model.OwnedBy != "" {
		fmt.Printf("Owned By: %s\n", model.OwnedBy)
	}

	// Capabilities
	fmt.Printf("\nCapabilities:\n")
	fmt.Printf("  Streaming: %s\n", formatBool(model.SupportsStreaming))
	fmt.Printf("  Functions: %s\n", formatBool(model.SupportsFunctions))
	if model.SupportsVision {
		fmt.Printf("  Vision: %s\n", formatBool(model.SupportsVision))
	}

	// Pricing
	if model.InputCostPer1K != nil || model.OutputCostPer1K != nil {
		fmt.Printf("\nPricing:\n")
		if model.InputCostPer1K != nil {
			fmt.Printf("  Input: $%.4f per 1K tokens\n", *model.InputCostPer1K)
		}
		if model.OutputCostPer1K != nil {
			fmt.Printf("  Output: $%.4f per 1K tokens\n", *model.OutputCostPer1K)
		}
	}

	// Tags
	if len(model.Tags) > 0 {
		fmt.Printf("\nTags: %s\n", strings.Join(model.Tags, ", "))
	}

	// Metadata
	if len(model.Metadata) > 0 {
		fmt.Printf("\nMetadata:\n")
		for key, value := range model.Metadata {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	return nil
}

// outputModelsJSON outputs models in JSON format.
func outputModelsJSON(allModels map[string][]core.Model) error {
	// Implementation would use json.NewEncoder(os.Stdout).Encode(allModels)
	fmt.Println("JSON output not yet implemented")
	return nil
}

// outputModelsYAML outputs models in YAML format.
func outputModelsYAML(allModels map[string][]core.Model) error {
	// Implementation would use yaml.NewEncoder(os.Stdout).Encode(allModels)
	fmt.Println("YAML output not yet implemented")
	return nil
}

// outputModelJSON outputs a single model in JSON format.
func outputModelJSON(model core.Model) error {
	// Implementation would use json.NewEncoder(os.Stdout).Encode(model)
	fmt.Println("JSON output not yet implemented")
	return nil
}

// outputModelYAML outputs a single model in YAML format.
func outputModelYAML(model core.Model) error {
	// Implementation would use yaml.NewEncoder(os.Stdout).Encode(model)
	fmt.Println("YAML output not yet implemented")
	return nil
}

// Helper functions

// setupContextWithTimeout creates a context with timeout.
func setupContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// formatNumber formats large numbers with commas.
func formatNumber(n int) string {
	str := strconv.Itoa(n)
	if len(str) <= 3 {
		return str
	}

	var result strings.Builder
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(digit)
	}

	return result.String()
}

// formatBool formats boolean values for display.
func formatBool(b bool) string {
	if b {
		return "✓ Yes"
	}
	return "✗ No"
}

// intPtr returns a pointer to an int value.
func intPtr(i int) *int {
	return &i
}
