package cli

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

// AnimatedLogo represents an animated ASCII logo
type AnimatedLogo struct {
	frames     []string
	colors     []lipgloss.Color
	frameIndex int
	colorIndex int
	animSpeed  time.Duration
}

// NewAnimatedLogo creates a new animated logo
func NewAnimatedLogo() *AnimatedLogo {
	return &AnimatedLogo{
		frames: []string{
			`

    NexusLLMAGI simulation`,
		},
		colors: []lipgloss.Color{
			lipgloss.Color("#ff0000"), // Red
			lipgloss.Color("#ff7f00"), // Orange
			lipgloss.Color("#ffff00"), // Yellow
			lipgloss.Color("#00ff00"), // Green
			lipgloss.Color("#00ffff"), // Cyan
			lipgloss.Color("#0000ff"), // Blue
			lipgloss.Color("#8b00ff"), // Violet
			lipgloss.Color("#ff00ff"), // Magenta
		},
		frameIndex: 0,
		colorIndex: 0,
		animSpeed:  100 * time.Millisecond,
	}
}

// GetFrame returns the current frame with color applied
func (l *AnimatedLogo) GetFrame() string {
	frame := l.frames[l.frameIndex%len(l.frames)]
	style := lipgloss.NewStyle().Foreground(l.colors[l.colorIndex%len(l.colors)])
	return style.Render(frame)
}

// NextFrame advances to the next animation frame
func (l *AnimatedLogo) NextFrame() {
	l.frameIndex++
	if l.frameIndex%2 == 0 {
		l.colorIndex++
	}
}

// Animate displays an animated logo
func (l *AnimatedLogo) Animate(duration time.Duration) {
	start := time.Now()
	for time.Since(start) < duration {
		fmt.Print("\033[H\033[2J") // Clear screen
		fmt.Println(l.GetFrame())
		l.NextFrame()
		time.Sleep(l.animSpeed)
	}
}

// GetStaticLogo returns a static version of the logo with gradient
func GetStaticLogo() string {
	logo := `
╔══════════════════════════════════════════════════════════════════════╗
║   ▄████████    ▄██████▄   ▄█        ▄█        ▄▄▄▄███▄▄▄▄           ║
║  ███    ███   ███    ███ ███       ███      ▄██▀▀▀███▀▀▀██▄         ║
║  ███    █▀    ███    ███ ███       ███      ███   ███   ███         ║
║  ███          ███    ███ ███       ███      ███   ███   ███         ║
║  ███    ▄███▄ ███    ███ ███       ███      ███   ███   ███         ║
║  ███    ███   ███    ███ ███       ███      ███   ███   ███         ║
║  ███    ███   ███    ███ ███▌    ▄ ███▌    ▄███   ███   ███         ║
║  ████████▀     ▀██████▀  █████▄▄██ █████▄▄██ ▀█   ███   █▀          ║
║                          ▀         ▀                                 ║
║              🚀 Advanced AI Terminal Interface 🤖                    ║
╚══════════════════════════════════════════════════════════════════════╝`

	return ApplyGradient(logo)
}

// ApplyGradient applies a color gradient to text
func ApplyGradient(text string) string {
	lines := strings.Split(text, "\n")
	var result strings.Builder

	colors := []lipgloss.Color{
		lipgloss.Color("#00ffff"), // Cyan
		lipgloss.Color("#00ccff"), // Light Blue
		lipgloss.Color("#0099ff"), // Blue
		lipgloss.Color("#0066ff"), // Dark Blue
		lipgloss.Color("#0033ff"), // Deeper Blue
		lipgloss.Color("#3300ff"), // Blue-Purple
		lipgloss.Color("#6600ff"), // Purple
		lipgloss.Color("#9900ff"), // Violet
		lipgloss.Color("#cc00ff"), // Magenta-Purple
		lipgloss.Color("#ff00ff"), // Magenta
	}

	for i, line := range lines {
		if line == "" {
			result.WriteString("\n")
			continue
		}

		colorIndex := i % len(colors)
		style := lipgloss.NewStyle().Foreground(colors[colorIndex])
		result.WriteString(style.Render(line))
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// NeonGlow creates a neon glow effect for text
func NeonGlow(text string, baseColor lipgloss.Color) string {
	// Create multiple layers with different intensities
	glow := lipgloss.NewStyle().
		Foreground(baseColor).
		Bold(true).
		Render(text)

	return glow
}

// MatrixRain creates a Matrix-style rain effect
func MatrixRain(width, height int) string {
	chars := "ｱｲｳｴｵｶｷｸｹｺｻｼｽｾｿﾀﾁﾂﾃﾄﾅﾆﾇﾈﾉﾊﾋﾌﾍﾎﾏﾐﾑﾒﾓﾔﾕﾖﾗﾘﾙﾚﾛﾜﾝ01"
	var result strings.Builder

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if math.Mod(float64(x+y), 3) == 0 {
				charIndex := (x + y) % len(chars)
				intensity := 255 - (y * 10)
				if intensity < 50 {
					intensity = 50
				}

				greenColor := fmt.Sprintf("#00%02x00", intensity)
				style := lipgloss.NewStyle().Foreground(lipgloss.Color(greenColor))
				result.WriteString(style.Render(string(chars[charIndex])))
			} else {
				result.WriteString(" ")
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}

// PulsingText creates a pulsing text effect
type PulsingText struct {
	text      string
	baseColor lipgloss.Color
	phase     float64
}

// NewPulsingText creates a new pulsing text effect
func NewPulsingText(text string, color lipgloss.Color) *PulsingText {
	return &PulsingText{
		text:      text,
		baseColor: color,
		phase:     0,
	}
}

// Render renders the pulsing text
func (p *PulsingText) Render() string {
	// Calculate brightness based on sine wave
	brightness := (math.Sin(p.phase) + 1) / 2

	// Adjust color based on intensity
	style := lipgloss.NewStyle().
		Foreground(p.baseColor).
		Bold(brightness > 0.7)

	p.phase += 0.1
	if p.phase > 2*math.Pi {
		p.phase = 0
	}

	return style.Render(p.text)
}

// TypewriterEffect creates a typewriter effect for text
func TypewriterEffect(text string, delay time.Duration) {
	cyan := color.New(color.FgCyan)
	for _, char := range text {
		cyan.Print(string(char))
		time.Sleep(delay)
	}
	fmt.Println()
}

// GlitchEffect creates a glitch effect for text
func GlitchEffect(text string) string {
	glitchChars := "!@#$%^&*()_+-=[]{}|;:,.<>?"
	lines := strings.Split(text, "\n")
	var result strings.Builder

	for _, line := range lines {
		if line == "" {
			result.WriteString("\n")
			continue
		}

		// Randomly glitch some characters
		runes := []rune(line)
		for i := range runes {
			if i%7 == 0 && runes[i] != ' ' {
				glitchIndex := i % len(glitchChars)
				runes[i] = rune(glitchChars[glitchIndex])
			}
		}

		result.WriteString(string(runes))
		result.WriteString("\n")
	}

	return result.String()
}

// RainbowText creates rainbow colored text
func RainbowText(text string) string {
	colors := []lipgloss.Color{
		lipgloss.Color("#ff0000"), // Red
		lipgloss.Color("#ff7f00"), // Orange
		lipgloss.Color("#ffff00"), // Yellow
		lipgloss.Color("#00ff00"), // Green
		lipgloss.Color("#0000ff"), // Blue
		lipgloss.Color("#4b0082"), // Indigo
		lipgloss.Color("#9400d3"), // Violet
	}

	var result strings.Builder
	colorIndex := 0

	for _, char := range text {
		if char == ' ' || char == '\n' {
			result.WriteRune(char)
			continue
		}

		style := lipgloss.NewStyle().Foreground(colors[colorIndex%len(colors)])
		result.WriteString(style.Render(string(char)))
		colorIndex++
	}

	return result.String()
}
