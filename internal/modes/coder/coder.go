package coder

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yourusername/gollm/internal/core"
)

// Operator represents a coder operation type
type Operator string

const (
	OperatorCodegen  Operator = "Codegen"
	OperatorRefactor Operator = "Refactor"
	OperatorTest     Operator = "Test"
	OperatorDocs     Operator = "Docs"
)

func (o Operator) String() string { return string(o) }

// RunFlow runs the coder workflow
func RunFlow(ctx context.Context, r *bufio.Reader, out io.Writer, provider core.Provider, model string, op Operator) error {
	// Clear screen
	fmt.Fprintf(out, "\033[2J\033[H")

	fmt.Fprintf(out, "\n=== CODER MODE: %s ===\n", op)
	fmt.Fprintf(out, "Model: %s\n\n", model)

	// Show operator-specific instructions
	showInstructions(out, op)

	// Get context/files
	context, files, err := getContext(r, out, op)
	if err != nil {
		return fmt.Errorf("failed to get context: %w", err)
	}

	// Get user prompt
	fmt.Fprintln(out, "\nDescribe what you want:")
	fmt.Fprint(out, "prompt> ")

	prompt, err := r.ReadString('\n')
	if err != nil {
		return err
	}
	prompt = strings.TrimSpace(prompt)

	if prompt == "" {
		return nil
	}

	// Build request
	systemPrompt := getSystemPrompt(op)
	userPrompt := buildUserPrompt(op, prompt, context, files)

	temp := getTemperature(op)
	maxTok := 4096

	req := &core.CompletionRequest{
		Model: model,
		Messages: []core.Message{
			{Role: core.RoleSystem, Content: systemPrompt},
			{Role: core.RoleUser, Content: userPrompt},
		},
		Temperature: &temp,
		MaxTokens:   &maxTok,
		Stream:      true,
	}

	// Show processing
	fmt.Fprint(out, "\n[PROCESSING] Generating response...\n")
	fmt.Fprintln(out, strings.Repeat("-", 60))

	// Stream response
	var fullResponse strings.Builder
	stream, err := provider.StreamCompletion(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	startTime := time.Now()
	tokenCount := 0

	for chunk := range stream {
		if chunk.Error != nil {
			return fmt.Errorf("stream error: %w", chunk.Error)
		}

		// Extract content from chunk choices
		for _, choice := range chunk.Choices {
			var content string
			if choice.Delta != nil {
				content = choice.Delta.Content
			} else {
				content = choice.Message.Content
			}
			fullResponse.WriteString(content)
			fmt.Fprint(out, content)
			tokenCount++
		}
	}

	duration := time.Since(startTime)
	fmt.Fprintln(out, "\n"+strings.Repeat("-", 60))
	fmt.Fprintf(out, "\n[COMPLETE] Generated in %.2fs (~%d tokens)\n", duration.Seconds(), tokenCount)

	// Post-process based on operator
	if err := postProcess(r, out, op, fullResponse.String(), files); err != nil {
		return fmt.Errorf("post-process failed: %w", err)
	}

	return nil
}

func showInstructions(out io.Writer, op Operator) {
	switch op {
	case OperatorCodegen:
		fmt.Fprintln(out, "📝 Code Generation Mode")
		fmt.Fprintln(out, "Generate new code based on requirements.")
		fmt.Fprintln(out, "Tip: Be specific about language, patterns, and constraints.")

	case OperatorRefactor:
		fmt.Fprintln(out, "🔧 Code Refactoring Mode")
		fmt.Fprintln(out, "Improve existing code structure and quality.")
		fmt.Fprintln(out, "Tip: Provide the code to refactor and explain goals.")

	case OperatorTest:
		fmt.Fprintln(out, "🧪 Test Generation Mode")
		fmt.Fprintln(out, "Generate comprehensive unit tests.")
		fmt.Fprintln(out, "Tip: Include the code to test and testing framework preference.")

	case OperatorDocs:
		fmt.Fprintln(out, "📚 Documentation Mode")
		fmt.Fprintln(out, "Generate or update documentation.")
		fmt.Fprintln(out, "Tip: Specify format (README, API docs, comments, etc.)")
	}
	fmt.Fprintln(out)
}

func getContext(r *bufio.Reader, out io.Writer, op Operator) (string, []string, error) {
	var contextBuilder strings.Builder
	var files []string

	// Ask for context based on operator
	switch op {
	case OperatorCodegen:
		fmt.Fprintln(out, "Context options:")
		fmt.Fprintln(out, "1) No context needed")
		fmt.Fprintln(out, "2) Reference existing file(s)")
		fmt.Fprintln(out, "3) Describe architecture/patterns")
		fmt.Fprint(out, "\nSelect > ")

		choice, _ := r.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "2":
			files = getFileList(r, out)
			for _, file := range files {
				content, err := readFileContent(file, 200) // First 200 lines
				if err == nil {
					contextBuilder.WriteString(fmt.Sprintf("\n--- File: %s ---\n%s\n", file, content))
				}
			}

		case "3":
			fmt.Fprintln(out, "Describe architecture/patterns (empty line to finish):")
			for {
				line, _ := r.ReadString('\n')
				if line == "\n" {
					break
				}
				contextBuilder.WriteString(line)
			}
		}

	case OperatorRefactor, OperatorTest:
		fmt.Fprintln(out, "Enter file(s) to process (one per line, empty to finish):")
		files = getFileList(r, out)

		for _, file := range files {
			content, err := readFileContent(file, 0) // Full file
			if err != nil {
				fmt.Fprintf(out, "[WARN] Cannot read %s: %v\n", file, err)
				continue
			}
			contextBuilder.WriteString(fmt.Sprintf("\n--- File: %s ---\n%s\n", file, content))
		}

	case OperatorDocs:
		fmt.Fprintln(out, "Documentation context:")
		fmt.Fprintln(out, "1) Document single file")
		fmt.Fprintln(out, "2) Document directory/project")
		fmt.Fprintln(out, "3) Update existing docs")
		fmt.Fprint(out, "\nSelect > ")

		choice, _ := r.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1", "3":
			files = getFileList(r, out)
			for _, file := range files {
				content, err := readFileContent(file, 500)
				if err == nil {
					contextBuilder.WriteString(fmt.Sprintf("\n--- File: %s ---\n%s\n", file, content))
				}
			}

		case "2":
			fmt.Fprint(out, "Enter directory path: ")
			dir, _ := r.ReadString('\n')
			dir = strings.TrimSpace(dir)

			if dir != "" {
				// Get project structure
				structure := getProjectStructure(dir, 3)
				contextBuilder.WriteString(fmt.Sprintf("\nProject Structure:\n%s\n", structure))
			}
		}
	}

	return contextBuilder.String(), files, nil
}

func getFileList(r *bufio.Reader, out io.Writer) []string {
	var files []string

	for {
		fmt.Fprint(out, "file> ")
		file, _ := r.ReadString('\n')
		file = strings.TrimSpace(file)

		if file == "" {
			break
		}

		// Expand home directory
		if strings.HasPrefix(file, "~/") {
			home, _ := os.UserHomeDir()
			file = filepath.Join(home, file[2:])
		}

		// Check if file exists
		if _, err := os.Stat(file); err != nil {
			fmt.Fprintf(out, "[WARN] File not found: %s\n", file)
			continue
		}

		files = append(files, file)
	}

	return files
}

func buildUserPrompt(op Operator, prompt, context string, files []string) string {
	var builder strings.Builder

	// Add main request
	builder.WriteString(prompt)
	builder.WriteString("\n\n")

	// Add context if available
	if context != "" {
		builder.WriteString("Context:\n")
		builder.WriteString(context)
		builder.WriteString("\n\n")
	}

	// Add operator-specific instructions
	switch op {
	case OperatorCodegen:
		builder.WriteString("Requirements:\n")
		builder.WriteString("- Use idiomatic Go patterns\n")
		builder.WriteString("- Include error handling\n")
		builder.WriteString("- Add concise comments\n")
		builder.WriteString("- Follow the project's existing style if context provided\n")

	case OperatorRefactor:
		builder.WriteString("Refactoring goals:\n")
		builder.WriteString("- Improve readability and maintainability\n")
		builder.WriteString("- Reduce complexity\n")
		builder.WriteString("- Apply SOLID principles where appropriate\n")
		builder.WriteString("- Preserve functionality\n")
		builder.WriteString("- Output as a diff or complete refactored code\n")

	case OperatorTest:
		builder.WriteString("Testing requirements:\n")
		builder.WriteString("- Use table-driven tests for Go\n")
		builder.WriteString("- Cover edge cases\n")
		builder.WriteString("- Include both positive and negative test cases\n")
		builder.WriteString("- Use descriptive test names\n")
		builder.WriteString("- Mock external dependencies if needed\n")

	case OperatorDocs:
		builder.WriteString("Documentation requirements:\n")
		builder.WriteString("- Be clear and concise\n")
		builder.WriteString("- Include examples where helpful\n")
		builder.WriteString("- Follow standard documentation formats\n")
		builder.WriteString("- Update existing docs if provided\n")
	}

	return builder.String()
}

func getSystemPrompt(op Operator) string {
	base := "You are an expert software engineer with deep knowledge of Go, system design, and best practices. "

	switch op {
	case OperatorCodegen:
		return base + `Your task is to generate high-quality, production-ready code.
Focus on:
- Clean, idiomatic code
- Proper error handling
- Performance considerations
- Security best practices
- Testability

Output only code with minimal explanation. Include necessary imports.`

	case OperatorRefactor:
		return base + `Your task is to refactor code for improved quality.
Focus on:
- Reducing complexity (cyclomatic and cognitive)
- Improving naming and organization
- Applying design patterns where beneficial
- Enhancing testability
- Preserving all existing functionality

Output the refactored code as a complete replacement or a diff, depending on size.`

	case OperatorTest:
		return base + `Your task is to write comprehensive test cases.
Focus on:
- High coverage of logic branches
- Edge cases and error conditions
- Clear test names that describe the scenario
- Table-driven tests for Go code
- Proper test isolation

Output only test code with minimal explanation.`

	case OperatorDocs:
		return base + `Your task is to create or improve documentation.
Focus on:
- Clarity for the target audience
- Completeness without verbosity
- Helpful examples
- Proper formatting (Markdown, godoc, etc.)
- Maintenance considerations

Output documentation in the requested format.`

	default:
		return base + "Provide concise, accurate assistance for the given task."
	}
}

func getTemperature(op Operator) float64 {
	switch op {
	case OperatorCodegen:
		return 0.3 // Lower temperature for more deterministic code
	case OperatorRefactor:
		return 0.2 // Very low for consistent refactoring
	case OperatorTest:
		return 0.2 // Low for predictable test generation
	case OperatorDocs:
		return 0.5 // Moderate for more natural documentation
	default:
		return 0.7
	}
}

func postProcess(r *bufio.Reader, out io.Writer, op Operator, response string, files []string) error {
	fmt.Fprintln(out, "\n=== POST PROCESSING ===")

	switch op {
	case OperatorCodegen:
		fmt.Fprintln(out, "\nOptions:")
		fmt.Fprintln(out, "1) Save to new file")
		fmt.Fprintln(out, "2) Copy to clipboard")
		fmt.Fprintln(out, "3) View only")
		fmt.Fprint(out, "\nSelect > ")

		choice, _ := r.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			fmt.Fprint(out, "Enter filename: ")
			filename, _ := r.ReadString('\n')
			filename = strings.TrimSpace(filename)

			if filename != "" {
				// Extract code blocks from response
				code := extractCodeBlocks(response)
				if err := os.WriteFile(filename, []byte(code), 0644); err != nil {
					return fmt.Errorf("failed to save file: %w", err)
				}
				fmt.Fprintf(out, "[SUCCESS] Saved to %s\n", filename)
			}

		case "2":
			// Note: clipboard functionality would need additional dependency
			fmt.Fprintln(out, "[INFO] Clipboard functionality not implemented in minimal version")
		}

	case OperatorRefactor:
		if len(files) > 0 {
			fmt.Fprintln(out, "\nOptions:")
			fmt.Fprintln(out, "1) Apply changes")
			fmt.Fprintln(out, "2) Save as new file")
			fmt.Fprintln(out, "3) View diff")
			fmt.Fprintln(out, "4) Skip")
			fmt.Fprint(out, "\nSelect > ")

			choice, _ := r.ReadString('\n')
			choice = strings.TrimSpace(choice)

			switch choice {
			case "1":
				// Apply refactoring to original file
				for i, file := range files {
					bakFile := file + ".bak"
					origContent, _ := os.ReadFile(file)

					// Create backup
					if err := os.WriteFile(bakFile, origContent, 0644); err == nil {
						fmt.Fprintf(out, "[INFO] Backup created: %s\n", bakFile)
					}

					// Apply changes
					code := extractCodeBlocks(response)
					if err := os.WriteFile(file, []byte(code), 0644); err != nil {
						return fmt.Errorf("failed to update %s: %w", file, err)
					}
					fmt.Fprintf(out, "[SUCCESS] Updated %s\n", file)

					if i < len(files)-1 {
						// Parse next file from response if multiple files
						remaining := strings.SplitN(response, "--- File:", 2)
						if len(remaining) > 1 {
							response = remaining[1]
						}
					}
				}

			case "2":
				fmt.Fprint(out, "Enter new filename: ")
				filename, _ := r.ReadString('\n')
				filename = strings.TrimSpace(filename)

				if filename != "" {
					code := extractCodeBlocks(response)
					if err := os.WriteFile(filename, []byte(code), 0644); err != nil {
						return fmt.Errorf("failed to save file: %w", err)
					}
					fmt.Fprintf(out, "[SUCCESS] Saved to %s\n", filename)
				}

			case "3":
				// Show diff (simplified)
				if len(files) > 0 {
					origContent, _ := os.ReadFile(files[0])
					fmt.Fprintln(out, "\n--- Original ---")
					fmt.Fprintln(out, string(origContent))
					fmt.Fprintln(out, "\n--- Refactored ---")
					fmt.Fprintln(out, extractCodeBlocks(response))
				}
			}
		}

	case OperatorTest:
		fmt.Fprintln(out, "\nOptions:")
		fmt.Fprintln(out, "1) Save test file")
		fmt.Fprintln(out, "2) Run tests immediately")
		fmt.Fprintln(out, "3) View only")
		fmt.Fprint(out, "\nSelect > ")

		choice, _ := r.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			// Determine test filename
			testFile := ""
			if len(files) > 0 {
				base := strings.TrimSuffix(files[0], filepath.Ext(files[0]))
				testFile = base + "_test.go"
			}

			fmt.Fprintf(out, "Enter test filename [%s]: ", testFile)
			input, _ := r.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "" {
				testFile = input
			}

			if testFile != "" {
				code := extractCodeBlocks(response)
				if err := os.WriteFile(testFile, []byte(code), 0644); err != nil {
					return fmt.Errorf("failed to save test file: %w", err)
				}
				fmt.Fprintf(out, "[SUCCESS] Saved to %s\n", testFile)
			}

		case "2":
			// Save to temp file and run
			tmpFile := "temp_test.go"
			code := extractCodeBlocks(response)
			if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
				return fmt.Errorf("failed to create temp test file: %w", err)
			}
			defer os.Remove(tmpFile)

			fmt.Fprintln(out, "\n[RUNNING TESTS]")
			// Would run: go test -v ./...
			fmt.Fprintln(out, "[INFO] Test execution not implemented in minimal version")
		}

	case OperatorDocs:
		fmt.Fprintln(out, "\nOptions:")
		fmt.Fprintln(out, "1) Save documentation")
		fmt.Fprintln(out, "2) Append to existing file")
		fmt.Fprintln(out, "3) View only")
		fmt.Fprint(out, "\nSelect > ")

		choice, _ := r.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			fmt.Fprint(out, "Enter filename (e.g., README.md): ")
			filename, _ := r.ReadString('\n')
			filename = strings.TrimSpace(filename)

			if filename != "" {
				// Don't extract code blocks for docs, keep as-is
				if err := os.WriteFile(filename, []byte(response), 0644); err != nil {
					return fmt.Errorf("failed to save documentation: %w", err)
				}
				fmt.Fprintf(out, "[SUCCESS] Saved to %s\n", filename)
			}

		case "2":
			fmt.Fprint(out, "Enter filename to append to: ")
			filename, _ := r.ReadString('\n')
			filename = strings.TrimSpace(filename)

			if filename != "" {
				file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return fmt.Errorf("failed to open file: %w", err)
				}
				defer file.Close()

				if _, err := file.WriteString("\n\n" + response); err != nil {
					return fmt.Errorf("failed to append: %w", err)
				}
				fmt.Fprintf(out, "[SUCCESS] Appended to %s\n", filename)
			}
		}
	}

	fmt.Fprintln(out, "\n[COMPLETE] Coder task finished")
	fmt.Fprintln(out, "Press Enter to continue...")
	r.ReadString('\n')

	return nil
}

// Helper functions

func readFileContent(filepath string, maxLines int) (string, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	if maxLines <= 0 {
		return string(content), nil
	}

	// Limit to maxLines
	lines := strings.Split(string(content), "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		return strings.Join(lines, "\n") + "\n... (truncated)", nil
	}

	return string(content), nil
}

func getProjectStructure(root string, maxDepth int) string {
	var builder strings.Builder

	var walk func(string, string, int)
	walk = func(path, indent string, depth int) {
		if depth > maxDepth {
			return
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}

		for i, entry := range entries {
			// Skip hidden files and common ignore patterns
			name := entry.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				continue
			}

			isLast := i == len(entries)-1
			prefix := "├── "
			if isLast {
				prefix = "└── "
			}

			builder.WriteString(indent + prefix + name)
			if entry.IsDir() {
				builder.WriteString("/")
			}
			builder.WriteString("\n")

			if entry.IsDir() && depth < maxDepth {
				newIndent := indent
				if isLast {
					newIndent += "    "
				} else {
					newIndent += "│   "
				}
				walk(filepath.Join(path, name), newIndent, depth+1)
			}
		}
	}

	builder.WriteString(root + "/\n")
	walk(root, "", 1)

	return builder.String()
}

func extractCodeBlocks(response string) string {
	var codeBlocks []string
	lines := strings.Split(response, "\n")
	inCodeBlock := false
	var currentBlock strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				// End of code block
				codeBlocks = append(codeBlocks, currentBlock.String())
				currentBlock.Reset()
				inCodeBlock = false
			} else {
				// Start of code block
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			currentBlock.WriteString(line + "\n")
		}
	}

	// If no code blocks found, return the entire response
	if len(codeBlocks) == 0 {
		return response
	}

	// Join all code blocks
	return strings.Join(codeBlocks, "\n\n")
}
