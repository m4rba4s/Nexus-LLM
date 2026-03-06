// Package display provides intelligent response formatting and display capabilities.
package display

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/m4rba4s/Nexus-LLM/internal/core"
)

// Additional format types extending the base Format type
const (
	FormatAuto     Format = "auto"
	FormatPlain    Format = "plain"
	FormatMarkdown Format = "markdown"
)

// DisplayMode represents different display modes
type DisplayMode string

const (
	ModeCompact DisplayMode = "compact"
	ModeVerbose DisplayMode = "verbose"
	ModeQuiet   DisplayMode = "quiet"
)

// ResponseFormatter provides intelligent formatting for LLM responses
type ResponseFormatter struct {
	highlighter   *SyntaxHighlighter
	format        Format
	mode          DisplayMode
	showMetadata  bool
	showTokens    bool
	showTiming    bool
	maxWidth      int
	colorEnabled  bool
	streamingMode bool
}

// FormatterConfig holds configuration for the response formatter
type FormatterConfig struct {
	Format       Format
	Mode         DisplayMode
	Theme        string
	ShowMetadata bool
	ShowTokens   bool
	ShowTiming   bool
	MaxWidth     int
	ColorEnabled bool
}

// DefaultFormatterConfig returns default formatter configuration
func DefaultFormatterConfig() FormatterConfig {
	return FormatterConfig{
		Format:       FormatAuto,
		Mode:         ModeCompact,
		Theme:        "github",
		ShowMetadata: false,
		ShowTokens:   true,
		ShowTiming:   true,
		MaxWidth:     100,
		ColorEnabled: HasColorSupport(),
	}
}

// NewResponseFormatter creates a new response formatter with default config
func NewResponseFormatter() *ResponseFormatter {
	config := DefaultFormatterConfig()
	return NewResponseFormatterWithConfig(config)
}

// NewResponseFormatterWithConfig creates a response formatter with custom config
func NewResponseFormatterWithConfig(config FormatterConfig) *ResponseFormatter {
	highlighter := NewSyntaxHighlighterWithTheme(config.Theme)
	highlighter.SetColorEnabled(config.ColorEnabled)

	return &ResponseFormatter{
		highlighter:   highlighter,
		format:        config.Format,
		mode:          config.Mode,
		showMetadata:  config.ShowMetadata,
		showTokens:    config.ShowTokens,
		showTiming:    config.ShowTiming,
		maxWidth:      config.MaxWidth,
		colorEnabled:  config.ColorEnabled,
		streamingMode: false,
	}
}

// SetFormat sets the output format
func (rf *ResponseFormatter) SetFormat(format Format) {
	rf.format = format
}

// SetMode sets the display mode
func (rf *ResponseFormatter) SetMode(mode DisplayMode) {
	rf.mode = mode
}

// SetTheme sets the syntax highlighting theme
func (rf *ResponseFormatter) SetTheme(theme string) {
	rf.highlighter.SetTheme(theme)
}

// SetStreamingMode enables or disables streaming mode
func (rf *ResponseFormatter) SetStreamingMode(enabled bool) {
	rf.streamingMode = enabled
}

// FormatResponse formats a complete LLM response for display
func (rf *ResponseFormatter) FormatResponse(response *core.CompletionResponse, duration time.Duration) (string, error) {
	if response == nil {
		return "", fmt.Errorf("response is nil")
	}

	switch rf.format {
	case FormatJSON:
		return rf.formatAsJSON(response, duration)
	case FormatRaw:
		return rf.formatAsRaw(response)
	case FormatMarkdown:
		return rf.formatAsMarkdown(response, duration)
	case FormatPlain:
		return rf.formatAsPlain(response, duration)
	case FormatAuto:
		return rf.formatAuto(response, duration)
	default:
		return rf.formatAuto(response, duration)
	}
}

// FormatStreamChunk formats a streaming response chunk
func (rf *ResponseFormatter) FormatStreamChunk(chunk core.StreamChunk) (string, error) {
	if chunk.Error != nil {
		return rf.colorize(fmt.Sprintf("❌ Error: %v\n", chunk.Error), "red"), nil
	}

	if chunk.Done {
		return rf.colorize("\n✅ Stream completed\n", "green"), nil
	}

	if len(chunk.Choices) > 0 && chunk.Choices[0].Message.Content != "" {
		content := chunk.Choices[0].Message.Content

		// Apply syntax highlighting if content looks like code
		if rf.highlighter.IsCodeBlock(content) {
			highlighted, err := rf.highlighter.HighlightCode(content, "")
			if err == nil {
				content = highlighted
			}
		}

		return content, nil
	}

	return "", nil
}

// FormatError formats an error for display
func (rf *ResponseFormatter) FormatError(err error) string {
	if err == nil {
		return ""
	}

	switch rf.mode {
	case ModeQuiet:
		return rf.colorize(fmt.Sprintf("❌ %s", err.Error()), "red")
	case ModeVerbose:
		return rf.colorize(fmt.Sprintf("❌ Error Details:\n%+v", err), "red")
	default:
		return rf.colorize(fmt.Sprintf("❌ Error: %s", err.Error()), "red")
	}
}

// formatAsJSON formats response as JSON
func (rf *ResponseFormatter) formatAsJSON(response *core.CompletionResponse, duration time.Duration) (string, error) {
	// Add timing metadata if enabled
	data := map[string]interface{}{
		"response": response,
	}

	if rf.showTiming {
		data["timing"] = map[string]interface{}{
			"duration":    duration.String(),
			"duration_ms": duration.Milliseconds(),
		}
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	// Apply JSON syntax highlighting
	highlighted, err := rf.highlighter.HighlightCode(string(jsonBytes), "json")
	if err != nil {
		return string(jsonBytes), nil
	}

	return highlighted, nil
}

// formatAsRaw formats response as raw content only
func (rf *ResponseFormatter) formatAsRaw(response *core.CompletionResponse) (string, error) {
	if len(response.Choices) == 0 {
		return "", nil
	}
	return response.Choices[0].Message.Content, nil
}

// formatAsMarkdown formats response with markdown styling
func (rf *ResponseFormatter) formatAsMarkdown(response *core.CompletionResponse, duration time.Duration) (string, error) {
	var result strings.Builder

	// Header
	result.WriteString(rf.colorize("# LLM Response\n\n", "blue"))

	// Content
	if len(response.Choices) > 0 {
		content := response.Choices[0].Message.Content
		highlighted, err := rf.highlighter.FormatResponse(content)
		if err != nil {
			result.WriteString(content)
		} else {
			result.WriteString(highlighted)
		}
		result.WriteString("\n\n")
	}

	// Metadata
	if rf.showMetadata {
		result.WriteString(rf.formatMetadata(response, duration))
	}

	return result.String(), nil
}

// formatAsPlain formats response with plain text styling
func (rf *ResponseFormatter) formatAsPlain(response *core.CompletionResponse, duration time.Duration) (string, error) {
	var result strings.Builder

	if len(response.Choices) > 0 {
		content := response.Choices[0].Message.Content

		// Apply intelligent formatting
		highlighted, err := rf.highlighter.FormatResponse(content)
		if err != nil {
			result.WriteString(content)
		} else {
			result.WriteString(highlighted)
		}
	}

	if rf.mode != ModeQuiet {
		result.WriteString("\n")
		result.WriteString(rf.formatStatusLine(response, duration))
	}

	return result.String(), nil
}

// formatAuto automatically chooses the best format based on content
func (rf *ResponseFormatter) formatAuto(response *core.CompletionResponse, duration time.Duration) (string, error) {
	if len(response.Choices) == 0 {
		return rf.formatAsJSON(response, duration)
	}

	content := response.Choices[0].Message.Content

	// Check if content is structured data
	if rf.looksLikeJSON(content) {
		rf.format = FormatJSON
		return rf.formatAsJSON(response, duration)
	}

	// Default to plain format with smart highlighting
	return rf.formatAsPlain(response, duration)
}

// formatMetadata formats response metadata
func (rf *ResponseFormatter) formatMetadata(response *core.CompletionResponse, duration time.Duration) string {
	var result strings.Builder

	result.WriteString(rf.colorize("## Metadata\n", "cyan"))
	result.WriteString(fmt.Sprintf("- **Model**: %s\n", response.Model))
	result.WriteString(fmt.Sprintf("- **ID**: %s\n", response.ID))

	if rf.showTokens && response.Usage.TotalTokens > 0 {
		result.WriteString(fmt.Sprintf("- **Tokens**: %d total (%d prompt + %d completion)\n",
			response.Usage.TotalTokens, response.Usage.PromptTokens, response.Usage.CompletionTokens))
	}

	if rf.showTiming {
		result.WriteString(fmt.Sprintf("- **Duration**: %v\n", duration.Round(time.Millisecond)))
	}

	if len(response.Choices) > 0 && response.Choices[0].FinishReason != "" {
		result.WriteString(fmt.Sprintf("- **Finish Reason**: %s\n", response.Choices[0].FinishReason))
	}

	if response.Usage.TotalCost != nil && *response.Usage.TotalCost > 0 {
		result.WriteString(fmt.Sprintf("- **Cost**: $%.6f\n", *response.Usage.TotalCost))
	}

	return result.String()
}

// formatStatusLine formats a compact status line
func (rf *ResponseFormatter) formatStatusLine(response *core.CompletionResponse, duration time.Duration) string {
	var parts []string

	if rf.showTokens && response.Usage.TotalTokens > 0 {
		parts = append(parts, fmt.Sprintf("%dt", response.Usage.TotalTokens))
	}

	if rf.showTiming {
		parts = append(parts, duration.Round(time.Millisecond).String())
	}

	if response.Usage.TotalCost != nil && *response.Usage.TotalCost > 0 {
		parts = append(parts, fmt.Sprintf("$%.4f", *response.Usage.TotalCost))
	}

	if len(parts) == 0 {
		return ""
	}

	status := strings.Join(parts, " • ")
	return rf.colorize(fmt.Sprintf("─── %s", status), "dim")
}

// FormatTable formats data as a table
func (rf *ResponseFormatter) FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}

	// Calculate column widths
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = utf8.RuneCountInString(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				cellWidth := utf8.RuneCountInString(StripANSIColors(cell))
				if cellWidth > colWidths[i] {
					colWidths[i] = cellWidth
				}
			}
		}
	}

	var result strings.Builder

	// Format header
	result.WriteString("┌")
	for i, width := range colWidths {
		result.WriteString(strings.Repeat("─", width+2))
		if i < len(colWidths)-1 {
			result.WriteString("┬")
		}
	}
	result.WriteString("┐\n")

	// Header row
	result.WriteString("│")
	for i, header := range headers {
		padding := colWidths[i] - utf8.RuneCountInString(header)
		result.WriteString(fmt.Sprintf(" %s%s ",
			rf.colorize(header, "bold"),
			strings.Repeat(" ", padding)))
		result.WriteString("│")
	}
	result.WriteString("\n")

	// Header separator
	result.WriteString("├")
	for i, width := range colWidths {
		result.WriteString(strings.Repeat("─", width+2))
		if i < len(colWidths)-1 {
			result.WriteString("┼")
		}
	}
	result.WriteString("┤\n")

	// Data rows
	for _, row := range rows {
		result.WriteString("│")
		for i, cell := range row {
			if i < len(colWidths) {
				cellWidth := utf8.RuneCountInString(StripANSIColors(cell))
				padding := colWidths[i] - cellWidth
				result.WriteString(fmt.Sprintf(" %s%s ", cell, strings.Repeat(" ", padding)))
			}
			result.WriteString("│")
		}
		result.WriteString("\n")
	}

	// Bottom border
	result.WriteString("└")
	for i, width := range colWidths {
		result.WriteString(strings.Repeat("─", width+2))
		if i < len(colWidths)-1 {
			result.WriteString("┴")
		}
	}
	result.WriteString("┘")

	return result.String()
}

// FormatProgressBar creates a progress bar for streaming responses
func (rf *ResponseFormatter) FormatProgressBar(current, total int, message string) string {
	if total <= 0 {
		return rf.colorize(fmt.Sprintf("⏳ %s", message), "yellow")
	}

	percentage := float64(current) / float64(total)
	if percentage > 1.0 {
		percentage = 1.0
	}

	barWidth := 30
	filled := int(percentage * float64(barWidth))
	empty := barWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	return rf.colorize(fmt.Sprintf("⏳ [%s] %.1f%% %s",
		bar, percentage*100, message), "yellow")
}

// looksLikeJSON checks if content appears to be JSON
func (rf *ResponseFormatter) looksLikeJSON(content string) bool {
	content = strings.TrimSpace(content)
	if len(content) == 0 {
		return false
	}

	// Quick JSON detection
	if (strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}")) ||
		(strings.HasPrefix(content, "[") && strings.HasSuffix(content, "]")) {
		// Try to parse as JSON
		var dummy interface{}
		return json.Unmarshal([]byte(content), &dummy) == nil
	}

	return false
}

// colorize applies color to text if color is enabled
func (rf *ResponseFormatter) colorize(text, color string) string {
	if !rf.colorEnabled {
		return text
	}

	colors := map[string]string{
		"red":    "\033[31m",
		"green":  "\033[32m",
		"yellow": "\033[33m",
		"blue":   "\033[34m",
		"purple": "\033[35m",
		"cyan":   "\033[36m",
		"white":  "\033[37m",
		"bold":   "\033[1m",
		"dim":    "\033[2m",
		"reset":  "\033[0m",
	}

	if colorCode, exists := colors[color]; exists {
		return colorCode + text + colors["reset"]
	}

	return text
}

// PrintSeparator prints a visual separator
func (rf *ResponseFormatter) PrintSeparator(title string) {
	if rf.mode == ModeQuiet {
		return
	}

	width := rf.maxWidth
	if width <= 0 {
		width = 80
	}

	if title == "" {
		fmt.Println(rf.colorize(strings.Repeat("─", width), "dim"))
		return
	}

	titleLen := utf8.RuneCountInString(title)
	if titleLen >= width-4 {
		fmt.Println(rf.colorize(title, "bold"))
		return
	}

	leftPadding := (width - titleLen - 2) / 2
	rightPadding := width - titleLen - 2 - leftPadding

	separator := strings.Repeat("─", leftPadding) + " " +
		rf.colorize(title, "bold") + " " +
		strings.Repeat("─", rightPadding)

	fmt.Println(rf.colorize(separator, "dim"))
}

// GetFormatterStats returns statistics about the formatter
func (rf *ResponseFormatter) GetFormatterStats() map[string]interface{} {
	return map[string]interface{}{
		"format":        rf.format,
		"mode":          rf.mode,
		"color_enabled": rf.colorEnabled,
		"streaming":     rf.streamingMode,
		"theme":         rf.highlighter.theme,
		"max_width":     rf.maxWidth,
	}
}
