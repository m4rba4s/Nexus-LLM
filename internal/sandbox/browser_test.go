package sandbox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowserEngine_JavascriptEvaluation(t *testing.T) {
	if os.Getenv("CI_SANDBOX") == "1" {
		t.Skip("skipping browser tests in CI sandbox")
	}

	engine, err := NewBrowserEngine()
	if err != nil {
		t.Skip("Skipping test: Playwright not fully installed or chromium absent in CI", err)
	}
	defer engine.Close()

	ctx := context.Background()

	t.Run("Basic Math", func(t *testing.T) {
		res, err := engine.EvaluateJS(ctx, "5 + 10")
		require.NoError(t, err)
		assert.Equal(t, "15", res)
	})

	t.Run("String Manipulation", func(t *testing.T) {
		res, err := engine.EvaluateJS(ctx, "'openclaw' + '_' + 'agent'.toUpperCase()")
		require.NoError(t, err)
		assert.Equal(t, "openclaw_AGENT", res)
	})

	t.Run("Timeout Protection", func(t *testing.T) {
		// Infinite loop to test sandboxing limit (5 seconds)
		_, err := engine.EvaluateJS(ctx, "while(true) {}")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "execution timeout")
	})
}

func TestBrowserEngine_StealthScraping(t *testing.T) {
	if os.Getenv("CI_SANDBOX") == "1" {
		t.Skip("skipping browser tests in CI sandbox")
	}

	engine, err := NewBrowserEngine()
	if err != nil {
		t.Skip("Skipping test: Playwright not fully installed", err)
	}
	defer engine.Close()

	// Create a mock Single Page Application (SPA) server
	mockSPA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head><title>OpenClaw Mock</title></head>
			<body>
				<div id="loading">Loading network...</div>
				<script>
					// Simulate React/Vue network mounting delay
					setTimeout(() => {
						document.getElementById("loading").outerHTML = "<div id='agent-feed'>Agent Alice posted a $500 bounty for a code review</div>";
					}, 200);
				</script>
			</body>
			</html>
		`))
	}))
	defer mockSPA.Close()

	t.Run("Extract Dynamic DOM", func(t *testing.T) {
		// Wait for network idle and extract the dynamically rendered #agent-feed
		content, err := engine.NavigateAndExtract(mockSPA.URL, "#agent-feed")
		require.NoError(t, err)

		// Ensure the JS successfully fully mounted instead of returning "Loading network..."
		assert.Contains(t, content, "Agent Alice posted a $500 bounty")
		assert.NotContains(t, content, "Loading network...")
	})
}
