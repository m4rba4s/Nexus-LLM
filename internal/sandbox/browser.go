package sandbox

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/playwright-community/playwright-go"
)

// BrowserEngine manages headless browser interactions and local JS execution.
type BrowserEngine struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	jsvm    *goja.Runtime
}

// NewBrowserEngine initializes Playwright and a Goja JS VM.
func NewBrowserEngine() (*BrowserEngine, error) {
	// Initialize local JS Runtime (Goja)
	vm := goja.New()

	// Initialize Playwright
	err := playwright.Install()
	if err != nil {
		return nil, fmt.Errorf("failed to install playwright dependencies: %w", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("could not start playwright: %w", err)
	}

	// Launch Chromium in stealth mode (no-sandbox for root safety in containers)
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args: []string{
			"--disable-blink-features=AutomationControlled", // Antibot evasion
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-gpu",
		},
	})
	if err != nil {
		pw.Stop()
		return nil, fmt.Errorf("could not launch browser: %w", err)
	}

	return &BrowserEngine{
		pw:      pw,
		browser: browser,
		jsvm:    vm,
	}, nil
}

// Close terminates the browser and Playwright runtime safely.
func (b *BrowserEngine) Close() error {
	var errs []error
	if b.browser != nil {
		if err := b.browser.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if b.pw != nil {
		if err := b.pw.Stop(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing browser engine: %v", errs)
	}
	return nil
}

// NavigateAndExtract visits a URL, waits for network idle, and extracts the DOM text.
func (b *BrowserEngine) NavigateAndExtract(url string, selector string) (string, error) {
	page, err := b.browser.NewPage(playwright.BrowserNewPageOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Add stealth scripts before navigation
	stealthScript := `
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined
		});
	`
	err = page.AddInitScript(playwright.Script{
		Content: playwright.String(stealthScript),
	})
	if err != nil {
		return "", fmt.Errorf("failed to inject stealth script: %w", err)
	}

	// Navigate and wait for network idle to ensure SPA frameworks (React/Vue) mount
	_, err = page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30000), // 30s timeout
	})
	if err != nil {
		return "", fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	// If a specific selector is provided, wait for and extract it
	if selector != "" {
		element, err := page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(10000),
		})
		if err != nil {
			return "", fmt.Errorf("selector %s not found: %w", selector, err)
		}
		return element.InnerText()
	}

	// Extract full visible page text (basic markdown-like parse)
	body, err := page.Locator("body").InnerText()
	if err != nil {
		return "", fmt.Errorf("failed to extract body text: %w", err)
	}
	return body, nil
}

// EvaluateJS runs a pure JavaScript snippet locally via Goja without a browser.
func (b *BrowserEngine) EvaluateJS(ctx context.Context, script string) (string, error) {
	// Set execution timeout to prevent infinite loops (sandbox protection)
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Handle timeout via goroutine interrupt
	done := make(chan error, 1)
	var result goja.Value

	go func() {
		var err error
		result, err = b.jsvm.RunString(script)
		done <- err
	}()

	select {
	case <-timeoutCtx.Done():
		b.jsvm.Interrupt("execution timeout")
		return "", fmt.Errorf("javascript execution timed out (5s limit)")
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("js execution error: %w", err)
		}
		if result == nil {
			return "undefined", nil
		}
		return result.String(), nil
	}
}
