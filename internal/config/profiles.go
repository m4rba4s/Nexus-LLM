// Package config/profiles provides profile management for GOLLM configurations.
//
// Profiles allow users to define and switch between different configuration sets
// optimized for specific use cases such as coding, creative writing, analysis, etc.
//
// The profile system supports:
// - Predefined profiles (default, coding, creative)
// - Custom user-defined profiles
// - Profile inheritance and overrides
// - Context-aware profile selection
// - Validation and migration
//
// Example usage:
//
//	manager, err := profiles.NewManager("~/.gollm/profiles.yaml")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	profile, err := manager.GetProfile("coding")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	config := profile.ToConfig()
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// ProfileManager manages configuration profiles and their persistence.
type ProfileManager struct {
	filepath  string
	profiles  map[string]*Profile
	active    string
	validator *validator.Validate
}

// Profile represents a named configuration profile with specific settings.
type Profile struct {
	// Name is the unique identifier for this profile
	Name string `yaml:"name" json:"name" validate:"required,min=1,max=50"`

	// Description provides a human-readable description of the profile's purpose
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Provider specifies the default LLM provider for this profile
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty" validate:"omitempty"`

	// Model specifies the default model for this profile
	Model string `yaml:"model,omitempty" json:"model,omitempty" validate:"omitempty"`

	// Temperature controls randomness in responses (0.0-2.0)
	Temperature *float64 `yaml:"temperature,omitempty" json:"temperature,omitempty" validate:"omitempty,min=0,max=2"`

	// MaxTokens limits the maximum response length
	MaxTokens *int `yaml:"max_tokens,omitempty" json:"max_tokens,omitempty" validate:"omitempty,min=1,max=100000"`

	// TopP controls nucleus sampling (0.0-1.0)
	TopP *float64 `yaml:"top_p,omitempty" json:"top_p,omitempty" validate:"omitempty,min=0,max=1"`

	// FrequencyPenalty reduces repetition (-2.0-2.0)
	FrequencyPenalty *float64 `yaml:"frequency_penalty,omitempty" json:"frequency_penalty,omitempty" validate:"omitempty,min=-2,max=2"`

	// PresencePenalty encourages topic diversity (-2.0-2.0)
	PresencePenalty *float64 `yaml:"presence_penalty,omitempty" json:"presence_penalty,omitempty" validate:"omitempty,min=-2,max=2"`

	// SystemMessage provides default system instructions
	SystemMessage string `yaml:"system_message,omitempty" json:"system_message,omitempty"`

	// StopSequences defines custom stop sequences
	StopSequences []string `yaml:"stop_sequences,omitempty" json:"stop_sequences,omitempty"`

	// Stream enables streaming responses by default
	Stream *bool `yaml:"stream,omitempty" json:"stream,omitempty"`

	// Timeout specifies request timeout duration
	Timeout *time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// Tags for profile categorization and search
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`

	// Inherits allows profile inheritance from another profile
	Inherits string `yaml:"inherits,omitempty" json:"inherits,omitempty"`

	// CreatedAt timestamp
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`

	// UpdatedAt timestamp
	UpdatedAt time.Time `yaml:"updated_at" json:"updated_at"`
}

// ProfilesConfig represents the complete profiles configuration file.
type ProfilesConfig struct {
	// Version of the profiles configuration format
	Version string `yaml:"version" json:"version"`

	// ActiveProfile specifies the currently active profile
	ActiveProfile string `yaml:"active_profile" json:"active_profile"`

	// Profiles contains all defined profiles
	Profiles map[string]*Profile `yaml:"profiles" json:"profiles"`

	// UpdatedAt timestamp for the configuration file
	UpdatedAt time.Time `yaml:"updated_at" json:"updated_at"`
}

// NewManager creates a new profile manager with the specified configuration file path.
func NewProfileManager(filepath string) (*ProfileManager, error) {
	manager := &ProfileManager{
		filepath:  filepath,
		profiles:  make(map[string]*Profile),
		validator: validator.New(),
	}

	// Initialize with default profiles if file doesn't exist
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		if err := manager.initializeDefaults(); err != nil {
			return nil, fmt.Errorf("failed to initialize default profiles: %w", err)
		}
		if err := manager.Save(); err != nil {
			return nil, fmt.Errorf("failed to save default profiles: %w", err)
		}
	} else {
		// Load existing profiles
		if err := manager.Load(); err != nil {
			return nil, fmt.Errorf("failed to load profiles: %w", err)
		}
	}

	return manager, nil
}

// Load reads profiles from the configuration file.
func (pm *ProfileManager) Load() error {
	data, err := os.ReadFile(pm.filepath)
	if err != nil {
		return fmt.Errorf("failed to read profiles file: %w", err)
	}

	var config ProfilesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse profiles file: %w", err)
	}

	pm.profiles = config.Profiles
	pm.active = config.ActiveProfile

	// Validate all profiles
	for name, profile := range pm.profiles {
		if err := pm.validator.Struct(profile); err != nil {
			return fmt.Errorf("invalid profile '%s': %w", name, err)
		}
	}

	return nil
}

// Save writes the current profiles to the configuration file.
func (pm *ProfileManager) Save() error {
	// Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(pm.filepath), 0700); err != nil {
        return fmt.Errorf("failed to create profiles directory: %w", err)
    }

	config := ProfilesConfig{
		Version:       "1.0",
		ActiveProfile: pm.active,
		Profiles:      pm.profiles,
		UpdatedAt:     time.Now(),
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal profiles: %w", err)
	}

    if err := os.WriteFile(pm.filepath, data, 0600); err != nil {
        return fmt.Errorf("failed to write profiles file: %w", err)
    }

    return nil
}

// GetProfile retrieves a profile by name with inheritance resolution.
func (pm *ProfileManager) GetProfile(name string) (*Profile, error) {
	profile, exists := pm.profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}

	// Resolve inheritance
	resolved, err := pm.resolveInheritance(profile, make(map[string]bool))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve inheritance for profile '%s': %w", name, err)
	}

	return resolved, nil
}

// SetProfile creates or updates a profile.
func (pm *ProfileManager) SetProfile(profile *Profile) error {
	if err := pm.validator.Struct(profile); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}

	// Check for circular inheritance
	if profile.Inherits != "" {
		if err := pm.checkCircularInheritance(profile.Name, profile.Inherits, make(map[string]bool)); err != nil {
			return fmt.Errorf("circular inheritance detected: %w", err)
		}
	}

	now := time.Now()
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now

	pm.profiles[profile.Name] = profile
	return pm.Save()
}

// DeleteProfile removes a profile by name.
func (pm *ProfileManager) DeleteProfile(name string) error {
	if _, exists := pm.profiles[name]; !exists {
		return fmt.Errorf("profile '%s' not found", name)
	}

	// Check if any other profile inherits from this one
	for _, profile := range pm.profiles {
		if profile.Inherits == name {
			return fmt.Errorf("cannot delete profile '%s': profile '%s' inherits from it", name, profile.Name)
		}
	}

	delete(pm.profiles, name)

	// If this was the active profile, reset to default
	if pm.active == name {
		pm.active = "default"
	}

	return pm.Save()
}

// ListProfiles returns a list of all profile names sorted alphabetically.
func (pm *ProfileManager) ListProfiles() []string {
	names := make([]string, 0, len(pm.profiles))
	for name := range pm.profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetActiveProfile returns the currently active profile.
func (pm *ProfileManager) GetActiveProfile() (*Profile, error) {
	if pm.active == "" {
		pm.active = "default"
	}
	return pm.GetProfile(pm.active)
}

// SetActiveProfile sets the active profile.
func (pm *ProfileManager) SetActiveProfile(name string) error {
	if _, exists := pm.profiles[name]; !exists {
		return fmt.Errorf("profile '%s' not found", name)
	}

	pm.active = name
	return pm.Save()
}

// SearchProfiles searches for profiles by name, description, or tags.
func (pm *ProfileManager) SearchProfiles(query string) []*Profile {
	query = strings.ToLower(query)
	var results []*Profile

	for _, profile := range pm.profiles {
		// Search in name
		if strings.Contains(strings.ToLower(profile.Name), query) {
			results = append(results, profile)
			continue
		}

		// Search in description
		if strings.Contains(strings.ToLower(profile.Description), query) {
			results = append(results, profile)
			continue
		}

		// Search in tags
		for _, tag := range profile.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				results = append(results, profile)
				break
			}
		}
	}

	// Sort results by relevance (exact matches first, then partial matches)
	sort.Slice(results, func(i, j int) bool {
		iExact := strings.EqualFold(results[i].Name, query)
		jExact := strings.EqualFold(results[j].Name, query)
		if iExact != jExact {
			return iExact
		}
		return results[i].Name < results[j].Name
	})

	return results
}

// ToConfig converts a profile to a Config object.
func (p *Profile) ToConfig() *Config {
	config := &Config{
		DefaultProvider: p.Provider,
		Providers:       make(map[string]ProviderConfig),
	}

	// Create a default provider configuration if specified
	if p.Provider != "" {
		providerConfig := ProviderConfig{
			Type:      p.Provider,
			TLSVerify: true, // Default to secure
		}

		// Set timeout if configured
		if p.Timeout != nil {
			providerConfig.Timeout = *p.Timeout
		}

		// Add model to models list if specified
		if p.Model != "" {
			providerConfig.Models = []string{p.Model}
		}

		config.Providers[p.Provider] = providerConfig
	}

	return config
}

// Clone creates a deep copy of the profile.
func (p *Profile) Clone() *Profile {
	clone := &Profile{
		Name:          p.Name,
		Description:   p.Description,
		Provider:      p.Provider,
		Model:         p.Model,
		SystemMessage: p.SystemMessage,
		Inherits:      p.Inherits,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}

	if p.Temperature != nil {
		temp := *p.Temperature
		clone.Temperature = &temp
	}
	if p.MaxTokens != nil {
		tokens := *p.MaxTokens
		clone.MaxTokens = &tokens
	}
	if p.TopP != nil {
		topP := *p.TopP
		clone.TopP = &topP
	}
	if p.FrequencyPenalty != nil {
		freq := *p.FrequencyPenalty
		clone.FrequencyPenalty = &freq
	}
	if p.PresencePenalty != nil {
		pres := *p.PresencePenalty
		clone.PresencePenalty = &pres
	}
	if p.Stream != nil {
		stream := *p.Stream
		clone.Stream = &stream
	}
	if p.Timeout != nil {
		timeout := *p.Timeout
		clone.Timeout = &timeout
	}

	// Copy slices
	clone.StopSequences = make([]string, len(p.StopSequences))
	copy(clone.StopSequences, p.StopSequences)

	clone.Tags = make([]string, len(p.Tags))
	copy(clone.Tags, p.Tags)

	return clone
}

// initializeDefaults creates the default profiles.
func (pm *ProfileManager) initializeDefaults() error {
	now := time.Now()

	// Default profile for general use
	defaultProfile := &Profile{
		Name:        "default",
		Description: "General purpose configuration with balanced settings",
		Provider:    "deepseek",
		Model:       "deepseek-chat",
		Temperature: floatPtr(0.7),
		MaxTokens:   intPtr(2000),
		Stream:      boolPtr(true),
		Tags:        []string{"general", "balanced"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Coding profile optimized for programming tasks
	codingProfile := &Profile{
		Name:          "coding",
		Description:   "Optimized for programming and technical tasks",
		Provider:      "deepseek",
		Model:         "deepseek-coder",
		Temperature:   floatPtr(0.2),
		MaxTokens:     intPtr(4000),
		Stream:        boolPtr(true),
		SystemMessage: "You are an expert software engineer. Provide precise, well-documented code solutions with clear explanations.",
		Tags:          []string{"programming", "technical", "precise"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Creative profile for creative writing and brainstorming
	creativeProfile := &Profile{
		Name:          "creative",
		Description:   "Optimized for creative writing and brainstorming",
		Provider:      "claude",
		Model:         "claude-3-sonnet",
		Temperature:   floatPtr(0.9),
		MaxTokens:     intPtr(3000),
		Stream:        boolPtr(true),
		SystemMessage: "You are a creative writing assistant. Be imaginative, expressive, and help develop unique ideas.",
		Tags:          []string{"creative", "writing", "imaginative"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Analysis profile for analytical and research tasks
	analysisProfile := &Profile{
		Name:          "analysis",
		Description:   "Optimized for analytical thinking and research tasks",
		Provider:      "openai",
		Model:         "gpt-4",
		Temperature:   floatPtr(0.3),
		MaxTokens:     intPtr(3500),
		Stream:        boolPtr(true),
		SystemMessage: "You are a research analyst. Provide thorough, well-reasoned analysis with supporting evidence.",
		Tags:          []string{"analysis", "research", "thorough"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	pm.profiles["default"] = defaultProfile
	pm.profiles["coding"] = codingProfile
	pm.profiles["creative"] = creativeProfile
	pm.profiles["analysis"] = analysisProfile
	pm.active = "default"

	return nil
}

// resolveInheritance resolves profile inheritance, merging settings from parent profiles.
func (pm *ProfileManager) resolveInheritance(profile *Profile, visited map[string]bool) (*Profile, error) {
	if profile.Inherits == "" {
		return profile.Clone(), nil
	}

	// Check for circular inheritance
	if visited[profile.Name] {
		return nil, fmt.Errorf("circular inheritance detected involving profile '%s'", profile.Name)
	}
	visited[profile.Name] = true

	// Get parent profile
	parent, exists := pm.profiles[profile.Inherits]
	if !exists {
		return nil, fmt.Errorf("parent profile '%s' not found", profile.Inherits)
	}

	// Recursively resolve parent's inheritance
	resolvedParent, err := pm.resolveInheritance(parent, visited)
	if err != nil {
		return nil, err
	}

	// Merge current profile with resolved parent
	merged := resolvedParent.Clone()
	pm.mergeProfiles(merged, profile)

	return merged, nil
}

// mergeProfiles merges child profile settings into parent profile.
func (pm *ProfileManager) mergeProfiles(parent, child *Profile) {
	// Override non-empty string fields
	if child.Name != "" {
		parent.Name = child.Name
	}
	if child.Description != "" {
		parent.Description = child.Description
	}
	if child.Provider != "" {
		parent.Provider = child.Provider
	}
	if child.Model != "" {
		parent.Model = child.Model
	}
	if child.SystemMessage != "" {
		parent.SystemMessage = child.SystemMessage
	}

	// Override non-nil pointer fields
	if child.Temperature != nil {
		parent.Temperature = child.Temperature
	}
	if child.MaxTokens != nil {
		parent.MaxTokens = child.MaxTokens
	}
	if child.TopP != nil {
		parent.TopP = child.TopP
	}
	if child.FrequencyPenalty != nil {
		parent.FrequencyPenalty = child.FrequencyPenalty
	}
	if child.PresencePenalty != nil {
		parent.PresencePenalty = child.PresencePenalty
	}
	if child.Stream != nil {
		parent.Stream = child.Stream
	}
	if child.Timeout != nil {
		parent.Timeout = child.Timeout
	}

	// Override slice fields if non-empty
	if len(child.StopSequences) > 0 {
		parent.StopSequences = make([]string, len(child.StopSequences))
		copy(parent.StopSequences, child.StopSequences)
	}
	if len(child.Tags) > 0 {
		parent.Tags = make([]string, len(child.Tags))
		copy(parent.Tags, child.Tags)
	}

	// Always update timestamps
	parent.CreatedAt = child.CreatedAt
	parent.UpdatedAt = child.UpdatedAt
}

// checkCircularInheritance checks for circular inheritance patterns.
func (pm *ProfileManager) checkCircularInheritance(profileName, parentName string, visited map[string]bool) error {
	if parentName == "" {
		return nil
	}

	if visited[parentName] {
		return fmt.Errorf("circular inheritance: '%s' -> '%s'", profileName, parentName)
	}

	parent, exists := pm.profiles[parentName]
	if !exists {
		return fmt.Errorf("parent profile '%s' not found", parentName)
	}

	visited[parentName] = true
	return pm.checkCircularInheritance(parentName, parent.Inherits, visited)
}

// Helper functions for creating pointers to basic types
func floatPtr(f float64) *float64 { return &f }
func intPtr(i int) *int           { return &i }
func boolPtr(b bool) *bool        { return &b }
