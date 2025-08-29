// Package commands provides the profile command for managing configuration profiles.
//
// The profile command allows users to create, manage, and switch between different
// configuration profiles optimized for specific use cases. Profiles contain
// provider settings, model parameters, and other configuration options.
//
// Usage:
//   gollm profile list
//   gollm profile show coding
//   gollm profile create my-profile --provider deepseek --model deepseek-chat
//   gollm profile switch creative
//   gollm profile delete old-profile
package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yourusername/gollm/internal/config"
	"github.com/yourusername/gollm/internal/display"
)

// ProfileFlags holds all profile command configuration.
type ProfileFlags struct {
	// Creation/editing flags
	Provider          string
	Model             string
	Description       string
	Temperature       float64
	MaxTokens         int
	TopP              float64
	FrequencyPenalty  float64
	PresencePenalty   float64
	SystemMessage     string
	Stream            bool
	NoStream          bool
	Timeout           string
	Tags              []string
	Inherits          string

	// Display flags
	OutputFormat      string
	Quiet             bool
	Verbose           bool
	ShowInheritance   bool
	ShowTimestamps    bool

	// Filter flags
	Tag               string
	ProviderFilter    string
}

// NewProfileCommand creates the profile management command.
func NewProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage configuration profiles",
		Long: `Manage configuration profiles for different use cases.

Profiles allow you to save and switch between different configurations
optimized for specific scenarios like coding, creative writing, or analysis.

Each profile can specify provider settings, model parameters, system messages,
and other configuration options. Profiles support inheritance, allowing you
to create specialized profiles based on existing ones.

Examples:
  gollm profile list
  gollm profile show coding
  gollm profile create my-coding --provider deepseek --model deepseek-coder --temperature 0.2
  gollm profile switch creative
  gollm profile delete old-profile`,
	}

	// Add subcommands
	cmd.AddCommand(newProfileListCommand())
	cmd.AddCommand(newProfileShowCommand())
	cmd.AddCommand(newProfileCreateCommand())
	cmd.AddCommand(newProfileEditCommand())
	cmd.AddCommand(newProfileDeleteCommand())
	cmd.AddCommand(newProfileSwitchCommand())
	cmd.AddCommand(newProfileCurrentCommand())
	cmd.AddCommand(newProfileSearchCommand())
	cmd.AddCommand(newProfileValidateCommand())
	cmd.AddCommand(newProfileExportCommand())
	cmd.AddCommand(newProfileImportCommand())

	return cmd
}

// newProfileListCommand creates the profile list subcommand.
func newProfileListCommand() *cobra.Command {
	flags := &ProfileFlags{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration profiles",
		Long: `List all available configuration profiles.

Shows profile names, descriptions, and basic information. Use --verbose
to display additional details like creation dates and inheritance.

Examples:
  gollm profile list
  gollm profile list --verbose
  gollm profile list --tag coding
  gollm profile list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileListCommand(cmd.Context(), flags)
		},
	}

	cmd.Flags().StringVarP(&flags.OutputFormat, "output", "o", "pretty", "Output format (pretty, json, table)")
	cmd.Flags().BoolVarP(&flags.Verbose, "verbose", "v", false, "Show detailed information")
	cmd.Flags().StringVar(&flags.Tag, "tag", "", "Filter by tag")
	cmd.Flags().StringVar(&flags.ProviderFilter, "provider", "", "Filter by provider")
	cmd.Flags().BoolVar(&flags.ShowInheritance, "show-inheritance", false, "Show inheritance relationships")
	cmd.Flags().BoolVar(&flags.ShowTimestamps, "show-timestamps", false, "Show creation/update timestamps")

	return cmd
}

// newProfileShowCommand creates the profile show subcommand.
func newProfileShowCommand() *cobra.Command {
	flags := &ProfileFlags{}

	cmd := &cobra.Command{
		Use:   "show [name]",
		Short: "Show detailed information about a profile",
		Long: `Display detailed information about a specific profile.

Shows all configuration parameters, inheritance relationships,
and metadata for the specified profile. If no name is provided,
shows the currently active profile.

Examples:
  gollm profile show coding
  gollm profile show --show-inheritance
  gollm profile show my-profile --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var profileName string
			if len(args) > 0 {
				profileName = args[0]
			}
			return runProfileShowCommand(cmd.Context(), profileName, flags)
		},
	}

	cmd.Flags().StringVarP(&flags.OutputFormat, "output", "o", "pretty", "Output format (pretty, json)")
	cmd.Flags().BoolVar(&flags.ShowInheritance, "show-inheritance", true, "Show inheritance chain")
	cmd.Flags().BoolVar(&flags.ShowTimestamps, "show-timestamps", true, "Show creation/update timestamps")

	return cmd
}

// newProfileCreateCommand creates the profile create subcommand.
func newProfileCreateCommand() *cobra.Command {
	flags := &ProfileFlags{}

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new configuration profile",
		Long: `Create a new configuration profile with the specified settings.

You can specify all configuration parameters via command line flags,
or create a basic profile and edit it later. Profiles can inherit
from existing profiles using the --inherits flag.

Examples:
  gollm profile create my-coding --provider deepseek --model deepseek-coder
  gollm profile create creative --inherits default --temperature 0.9
  gollm profile create analysis --provider openai --model gpt-4 --temperature 0.3`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileCreateCommand(cmd.Context(), args[0], flags)
		},
	}

	addProfileModificationFlags(cmd, flags)
	return cmd
}

// newProfileEditCommand creates the profile edit subcommand.
func newProfileEditCommand() *cobra.Command {
	flags := &ProfileFlags{}

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit an existing configuration profile",
		Long: `Edit an existing configuration profile.

Only specified flags will be updated; unspecified settings will remain
unchanged. Use empty string values to clear optional settings.

Examples:
  gollm profile edit coding --temperature 0.1
  gollm profile edit my-profile --description "Updated description"
  gollm profile edit creative --system-message ""`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileEditCommand(cmd.Context(), args[0], flags)
		},
	}

	addProfileModificationFlags(cmd, flags)
	return cmd
}

// newProfileDeleteCommand creates the profile delete subcommand.
func newProfileDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a configuration profile",
		Long: `Delete a configuration profile.

This action cannot be undone. If the profile is currently active,
the active profile will be reset to 'default'. You cannot delete
profiles that are inherited by other profiles.

Examples:
  gollm profile delete old-profile
  gollm profile delete test-profile`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileDeleteCommand(cmd.Context(), args[0])
		},
	}

	return cmd
}

// newProfileSwitchCommand creates the profile switch subcommand.
func newProfileSwitchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch <name>",
		Short: "Switch to a different configuration profile",
		Long: `Switch to a different configuration profile.

The specified profile becomes the active profile and will be used
for subsequent commands that don't explicitly specify a profile.

Examples:
  gollm profile switch coding
  gollm profile switch creative
  gollm profile switch default`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileSwitchCommand(cmd.Context(), args[0])
		},
	}

	return cmd
}

// newProfileCurrentCommand creates the profile current subcommand.
func newProfileCurrentCommand() *cobra.Command {
	flags := &ProfileFlags{}

	cmd := &cobra.Command{
		Use:   "current",
		Short: "Show the currently active profile",
		Long: `Display information about the currently active profile.

Shows which profile is currently active and its configuration details.

Examples:
  gollm profile current
  gollm profile current --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileCurrentCommand(cmd.Context(), flags)
		},
	}

	cmd.Flags().StringVarP(&flags.OutputFormat, "output", "o", "pretty", "Output format (pretty, json)")
	cmd.Flags().BoolVar(&flags.ShowInheritance, "show-inheritance", true, "Show inheritance chain")

	return cmd
}

// newProfileSearchCommand creates the profile search subcommand.
func newProfileSearchCommand() *cobra.Command {
	flags := &ProfileFlags{}

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for profiles by name, description, or tags",
		Long: `Search for profiles by name, description, or tags.

The search is case-insensitive and matches partial strings.
Results are sorted by relevance (exact matches first).

Examples:
  gollm profile search coding
  gollm profile search "creative writing"
  gollm profile search openai`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileSearchCommand(cmd.Context(), args[0], flags)
		},
	}

	cmd.Flags().StringVarP(&flags.OutputFormat, "output", "o", "pretty", "Output format (pretty, json, table)")
	cmd.Flags().BoolVarP(&flags.Verbose, "verbose", "v", false, "Show detailed information")

	return cmd
}

// newProfileValidateCommand creates the profile validate subcommand.
func newProfileValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [name...]",
		Short: "Validate one or more profiles",
		Long: `Validate profile configurations for correctness.

Checks for validation errors, circular inheritance, and missing
dependencies. If no profile names are specified, validates all profiles.

Examples:
  gollm profile validate
  gollm profile validate coding creative
  gollm profile validate my-profile`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileValidateCommand(cmd.Context(), args)
		},
	}

	return cmd
}

// newProfileExportCommand creates the profile export subcommand.
func newProfileExportCommand() *cobra.Command {
	flags := &ProfileFlags{}

	cmd := &cobra.Command{
		Use:   "export [name...] --output <file>",
		Short: "Export profiles to a file",
		Long: `Export one or more profiles to a YAML or JSON file.

If no profile names are specified, exports all profiles.
The output format is determined by the file extension.

Examples:
  gollm profile export --output my-profiles.yaml
  gollm profile export coding creative --output work-profiles.json
  gollm profile export my-profile --output single-profile.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileExportCommand(cmd.Context(), args, flags)
		},
	}

	cmd.Flags().StringVarP(&flags.OutputFormat, "output", "o", "", "Output file path (required)")
	cmd.MarkFlagRequired("output")

	return cmd
}

// newProfileImportCommand creates the profile import subcommand.
func newProfileImportCommand() *cobra.Command {
	flags := &ProfileFlags{}

	cmd := &cobra.Command{
		Use:   "import <file>",
		Short: "Import profiles from a file",
		Long: `Import profiles from a YAML or JSON file.

Existing profiles with the same names will be overwritten.
The file format is auto-detected from the extension.

Examples:
  gollm profile import my-profiles.yaml
  gollm profile import exported-profiles.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileImportCommand(cmd.Context(), args[0], flags)
		},
	}

	cmd.Flags().BoolVar(&flags.Quiet, "force", false, "Overwrite existing profiles without confirmation")

	return cmd
}

// addProfileModificationFlags adds common flags for profile creation and editing.
func addProfileModificationFlags(cmd *cobra.Command, flags *ProfileFlags) {
	// Core settings
	cmd.Flags().StringVarP(&flags.Provider, "provider", "p", "", "LLM provider")
	cmd.Flags().StringVarP(&flags.Model, "model", "m", "", "Model name")
	cmd.Flags().StringVarP(&flags.Description, "description", "d", "", "Profile description")

	// Model parameters
	cmd.Flags().Float64VarP(&flags.Temperature, "temperature", "t", -1, "Temperature (0.0-2.0)")
	cmd.Flags().IntVar(&flags.MaxTokens, "max-tokens", 0, "Maximum tokens")
	cmd.Flags().Float64Var(&flags.TopP, "top-p", -1, "Top-p value (0.0-1.0)")
	cmd.Flags().Float64Var(&flags.FrequencyPenalty, "frequency-penalty", -999, "Frequency penalty (-2.0-2.0)")
	cmd.Flags().Float64Var(&flags.PresencePenalty, "presence-penalty", -999, "Presence penalty (-2.0-2.0)")

	// Advanced settings
	cmd.Flags().StringVar(&flags.SystemMessage, "system-message", "", "System message")
	cmd.Flags().BoolVar(&flags.Stream, "stream", false, "Enable streaming")
	cmd.Flags().BoolVar(&flags.NoStream, "no-stream", false, "Disable streaming")
	cmd.Flags().StringVar(&flags.Timeout, "timeout", "", "Request timeout (e.g., 30s, 1m)")

	// Metadata
	cmd.Flags().StringSliceVar(&flags.Tags, "tags", nil, "Profile tags")
	cmd.Flags().StringVar(&flags.Inherits, "inherits", "", "Parent profile to inherit from")
}

// Command implementation functions

func runProfileListCommand(ctx context.Context, flags *ProfileFlags) error {
	renderer := display.NewRenderer(display.Options{
		Colors:      true,
		Interactive: true,
		Format:      display.Format(flags.OutputFormat),
		Verbose:     flags.Verbose,
	})

	// Get profile manager
	manager, err := getProfileManager()
	if err != nil {
		renderer.Error(fmt.Sprintf("Failed to load profiles: %v", err))
		return err
	}

	// Get all profiles
	profileNames := manager.ListProfiles()
	if len(profileNames) == 0 {
		renderer.Info("No profiles found")
		return nil
	}

	var profiles []*config.Profile
	for _, name := range profileNames {
		profile, err := manager.GetProfile(name)
		if err != nil {
			renderer.Warning(fmt.Sprintf("Failed to load profile '%s': %v", name, err))
			continue
		}

		// Apply filters
		if flags.Tag != "" && !containsTag(profile.Tags, flags.Tag) {
			continue
		}
		if flags.ProviderFilter != "" && profile.Provider != flags.ProviderFilter {
			continue
		}

		profiles = append(profiles, profile)
	}

	// Display profiles
	displayProfileList(profiles, flags, renderer)
	return nil
}

func runProfileShowCommand(ctx context.Context, profileName string, flags *ProfileFlags) error {
	renderer := display.NewRenderer(display.Options{
		Colors:      true,
		Interactive: true,
		Format:      display.Format(flags.OutputFormat),
	})

	manager, err := getProfileManager()
	if err != nil {
		renderer.Error(fmt.Sprintf("Failed to load profiles: %v", err))
		return err
	}

	// Use active profile if no name specified
	var profile *config.Profile
	if profileName == "" {
		profile, err = manager.GetActiveProfile()
		if err != nil {
			renderer.Error(fmt.Sprintf("Failed to get active profile: %v", err))
			return err
		}
	} else {
		profile, err = manager.GetProfile(profileName)
		if err != nil {
			renderer.Error(fmt.Sprintf("Profile '%s' not found: %v", profileName, err))
			return err
		}
	}

	// Display profile details
	displayProfileDetails(profile, flags, renderer)
	return nil
}

func runProfileCreateCommand(ctx context.Context, name string, flags *ProfileFlags) error {
	renderer := display.NewRenderer(display.Options{
		Colors:      true,
		Interactive: true,
	})

	manager, err := getProfileManager()
	if err != nil {
		renderer.Error(fmt.Sprintf("Failed to load profiles: %v", err))
		return err
	}

	// Create new profile
	profile := &config.Profile{
		Name:        name,
		Description: flags.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Apply settings from flags
	applyFlagsToProfile(profile, flags)

	// Validate and save
	if err := manager.SetProfile(profile); err != nil {
		renderer.Error(fmt.Sprintf("Failed to create profile: %v", err))
		return err
	}

	renderer.Success(fmt.Sprintf("Profile '%s' created successfully", name))
	return nil
}

func runProfileEditCommand(ctx context.Context, name string, flags *ProfileFlags) error {
	renderer := display.NewRenderer(display.Options{
		Colors:      true,
		Interactive: true,
	})

	manager, err := getProfileManager()
	if err != nil {
		renderer.Error(fmt.Sprintf("Failed to load profiles: %v", err))
		return err
	}

	// Load existing profile
	profile, err := manager.GetProfile(name)
	if err != nil {
		renderer.Error(fmt.Sprintf("Profile '%s' not found: %v", name, err))
		return err
	}

	// Apply changes from flags
	applyFlagsToProfile(profile, flags)
	profile.UpdatedAt = time.Now()

	// Save changes
	if err := manager.SetProfile(profile); err != nil {
		renderer.Error(fmt.Sprintf("Failed to update profile: %v", err))
		return err
	}

	renderer.Success(fmt.Sprintf("Profile '%s' updated successfully", name))
	return nil
}

func runProfileDeleteCommand(ctx context.Context, name string) error {
	renderer := display.NewRenderer(display.Options{
		Colors:      true,
		Interactive: true,
	})

	manager, err := getProfileManager()
	if err != nil {
		renderer.Error(fmt.Sprintf("Failed to load profiles: %v", err))
		return err
	}

	// Delete profile
	if err := manager.DeleteProfile(name); err != nil {
		renderer.Error(fmt.Sprintf("Failed to delete profile: %v", err))
		return err
	}

	renderer.Success(fmt.Sprintf("Profile '%s' deleted successfully", name))
	return nil
}

func runProfileSwitchCommand(ctx context.Context, name string) error {
	renderer := display.NewRenderer(display.Options{
		Colors:      true,
		Interactive: true,
	})

	manager, err := getProfileManager()
	if err != nil {
		renderer.Error(fmt.Sprintf("Failed to load profiles: %v", err))
		return err
	}

	// Switch to profile
	if err := manager.SetActiveProfile(name); err != nil {
		renderer.Error(fmt.Sprintf("Failed to switch to profile: %v", err))
		return err
	}

	renderer.Success(fmt.Sprintf("Switched to profile '%s'", name))
	return nil
}

func runProfileCurrentCommand(ctx context.Context, flags *ProfileFlags) error {
	renderer := display.NewRenderer(display.Options{
		Colors:      true,
		Interactive: true,
		Format:      display.Format(flags.OutputFormat),
	})

	manager, err := getProfileManager()
	if err != nil {
		renderer.Error(fmt.Sprintf("Failed to load profiles: %v", err))
		return err
	}

	profile, err := manager.GetActiveProfile()
	if err != nil {
		renderer.Error(fmt.Sprintf("Failed to get active profile: %v", err))
		return err
	}

	renderer.Info(fmt.Sprintf("Active profile: %s", profile.Name))
	displayProfileDetails(profile, flags, renderer)
	return nil
}

func runProfileSearchCommand(ctx context.Context, query string, flags *ProfileFlags) error {
	renderer := display.NewRenderer(display.Options{
		Colors:      true,
		Interactive: true,
		Format:      display.Format(flags.OutputFormat),
		Verbose:     flags.Verbose,
	})

	manager, err := getProfileManager()
	if err != nil {
		renderer.Error(fmt.Sprintf("Failed to load profiles: %v", err))
		return err
	}

	results := manager.SearchProfiles(query)
	if len(results) == 0 {
		renderer.Info(fmt.Sprintf("No profiles found matching '%s'", query))
		return nil
	}

	renderer.Info(fmt.Sprintf("Found %d profile(s) matching '%s':", len(results), query))
	displayProfileList(results, flags, renderer)
	return nil
}

func runProfileValidateCommand(ctx context.Context, profileNames []string) error {
	renderer := display.NewRenderer(display.Options{
		Colors:      true,
		Interactive: true,
	})

	manager, err := getProfileManager()
	if err != nil {
		renderer.Error(fmt.Sprintf("Failed to load profiles: %v", err))
		return err
	}

	// Validate specified profiles or all if none specified
	var namesToValidate []string
	if len(profileNames) > 0 {
		namesToValidate = profileNames
	} else {
		namesToValidate = manager.ListProfiles()
	}

	allValid := true
	for _, name := range namesToValidate {
		profile, err := manager.GetProfile(name)
		if err != nil {
			renderer.Error(fmt.Sprintf("Profile '%s': %v", name, err))
			allValid = false
			continue
		}

		// Profile validation is done during loading, so if we got here it's valid
		renderer.Success(fmt.Sprintf("Profile '%s' is valid", profile.Name))
	}

	if allValid {
		renderer.Success("All profiles are valid")
	} else {
		renderer.Error("Some profiles have validation errors")
		return fmt.Errorf("validation failed")
	}

	return nil
}

func runProfileExportCommand(ctx context.Context, profileNames []string, flags *ProfileFlags) error {
	renderer := display.NewRenderer(display.Options{
		Colors:      true,
		Interactive: true,
	})

	// Implementation would export profiles to file
	renderer.Info("Export functionality not yet implemented")
	return nil
}

func runProfileImportCommand(ctx context.Context, filename string, flags *ProfileFlags) error {
	renderer := display.NewRenderer(display.Options{
		Colors:      true,
		Interactive: true,
	})

	// Implementation would import profiles from file
	renderer.Info("Import functionality not yet implemented")
	return nil
}

// Helper functions

func getProfileManager() (*config.ProfileManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	profilesPath := filepath.Join(homeDir, ".gollm", "profiles.yaml")
	return config.NewProfileManager(profilesPath)
}

func displayProfileList(profiles []*config.Profile, flags *ProfileFlags, renderer *display.Renderer) {
	switch flags.OutputFormat {
	case "json":
		renderer.ConfigInfo(profiles)
	case "table":
		// Table format implementation
		displayProfileTable(profiles, flags, renderer)
	default:
		// Pretty format
		displayProfilePretty(profiles, flags, renderer)
	}
}

func displayProfileDetails(profile *config.Profile, flags *ProfileFlags, renderer *display.Renderer) {
	switch flags.OutputFormat {
	case "json":
		renderer.ConfigInfo(profile)
	default:
		// Pretty format with detailed view
		renderer.ConfigInfo(profile)
	}
}

func displayProfileTable(profiles []*config.Profile, flags *ProfileFlags, renderer *display.Renderer) {
	// Simple table implementation
	fmt.Printf("%-20s %-15s %-15s %s\n", "NAME", "PROVIDER", "MODEL", "DESCRIPTION")
	fmt.Printf("%-20s %-15s %-15s %s\n", strings.Repeat("-", 20), strings.Repeat("-", 15), strings.Repeat("-", 15), strings.Repeat("-", 30))

	for _, profile := range profiles {
		desc := profile.Description
		if len(desc) > 30 {
			desc = desc[:27] + "..."
		}
		fmt.Printf("%-20s %-15s %-15s %s\n", profile.Name, profile.Provider, profile.Model, desc)
	}
}

func displayProfilePretty(profiles []*config.Profile, flags *ProfileFlags, renderer *display.Renderer) {
	for i, profile := range profiles {
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("📋 %s\n", profile.Name)
		if profile.Description != "" {
			fmt.Printf("   %s\n", profile.Description)
		}
		if profile.Provider != "" {
			fmt.Printf("   Provider: %s", profile.Provider)
			if profile.Model != "" {
				fmt.Printf(" (%s)", profile.Model)
			}
			fmt.Println()
		}

		if flags.ShowTimestamps && !profile.CreatedAt.IsZero() {
			fmt.Printf("   Created: %s\n", profile.CreatedAt.Format("2006-01-02 15:04"))
		}

		if flags.Verbose {
			if len(profile.Tags) > 0 {
				fmt.Printf("   Tags: %s\n", strings.Join(profile.Tags, ", "))
			}
			if profile.Inherits != "" {
				fmt.Printf("   Inherits: %s\n", profile.Inherits)
			}
		}
	}
}

func applyFlagsToProfile(profile *config.Profile, flags *ProfileFlags) {
	if flags.Provider != "" {
		profile.Provider = flags.Provider
	}
	if flags.Model != "" {
		profile.Model = flags.Model
	}
	if flags.Description != "" {
		profile.Description = flags.Description
	}
	if flags.Temperature >= 0 {
		profile.Temperature = &flags.Temperature
	}
	if flags.MaxTokens > 0 {
		profile.MaxTokens = &flags.MaxTokens
	}
	if flags.TopP >= 0 {
		profile.TopP = &flags.TopP
	}
	if flags.FrequencyPenalty > -999 {
		profile.FrequencyPenalty = &flags.FrequencyPenalty
	}
	if flags.PresencePenalty > -999 {
		profile.PresencePenalty = &flags.PresencePenalty
	}
	if flags.SystemMessage != "" {
		profile.SystemMessage = flags.SystemMessage
	}
	if flags.Stream {
		stream := true
		profile.Stream = &stream
	}
	if flags.NoStream {
		stream := false
		profile.Stream = &stream
	}
	if flags.Timeout != "" {
		if timeout, err := time.ParseDuration(flags.Timeout); err == nil {
			profile.Timeout = &timeout
		}
	}
	if len(flags.Tags) > 0 {
		profile.Tags = flags.Tags
	}
	if flags.Inherits != "" {
		profile.Inherits = flags.Inherits
	}
}

func containsTag(tags []string, target string) bool {
	for _, tag := range tags {
		if strings.EqualFold(tag, target) {
			return true
		}
	}
	return false
}
