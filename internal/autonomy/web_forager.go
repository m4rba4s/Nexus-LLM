package autonomy

import (
	"context"
	"fmt"
	"strings"

	"github.com/yourusername/gollm/internal/sandbox"
)

// BrowserForager implements WebForager by delegating to the Phase 15 BrowserEngine.
type BrowserForager struct {
	engine *sandbox.BrowserEngine
}

func NewBrowserForager(engine *sandbox.BrowserEngine) *BrowserForager {
	return &BrowserForager{engine: engine}
}

// Forage visits the target location using a headless browser, avoiding basic bot detections,
// and extracts the DOM body text to find IOCs or threat bulletins.
func (bf *BrowserForager) Forage(ctx context.Context, targetURL string) (string, error) {
	if bf.engine == nil {
		return "", fmt.Errorf("browser engine is not initialized")
	}

	text, err := bf.engine.NavigateAndExtract(targetURL, "body")
	if err != nil {
		return "", err
	}

	// Truncate to avoid context window explosion
	if len(text) > 40000 {
		text = text[:40000] + "\n...[TRUNCATED]"
	}

	// Minify slightly
	text = strings.ReplaceAll(text, "\t", " ")
	return "TARGET_URL: " + targetURL + "\n\n" + text, nil
}
