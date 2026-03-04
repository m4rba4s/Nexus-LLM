package operator

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
)

// elevatedSession tracks if current session has elevated privileges
var elevatedSession = false

// RunCommandFlow executes shell commands with safety checks
func RunCommandFlow(ctx context.Context, r *bufio.Reader, out io.Writer) error {
	info(out, "\n=== RUN COMMAND ===")
	fmt.Fprintln(out, "Enter command to execute (or 'exit' to cancel)")
	fmt.Fprint(out, "cmd> ")

	cmdLine, err := r.ReadString('\n')
	if err != nil {
		return err
	}

	cmdLine = strings.TrimSpace(cmdLine)
	if cmdLine == "" || cmdLine == "exit" {
		return nil
	}

	// Safety check
	if isDangerousCommand(cmdLine) {
		warn(out, "\nWARNING: This command may be dangerous!")
		fmt.Fprintf(out, "Command: %s\n", cmdLine)
		fmt.Fprint(out, "Type 'I understand' to continue: ")

		confirm, _ := r.ReadString('\n')
		if strings.TrimSpace(confirm) != "I understand" {
			info(out, "Command cancelled.")
			return nil
		}
	}

	// Show what will be executed
	info(out, fmt.Sprintf("\n[EXEC] %s", cmdLine))
	fmt.Fprintln(out, strings.Repeat("-", 50))

	// Execute command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", cmdLine)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", cmdLine)
	}

	// Set up output
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Stdin = nil // Don't allow interactive input

	// Run with timeout
	done := make(chan error)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			errorLine(out, fmt.Sprintf("\n[ERROR] Command failed: %v", err))
			_ = writeAudit(AuditEntry{Event: "command", Detail: cmdLine, Success: false, Error: err.Error()})
			return err
		}
		success(out, "\n[SUCCESS] Command completed")
		_ = writeAudit(AuditEntry{Event: "command", Detail: cmdLine, Success: true})

	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		errorLine(out, "\n[ERROR] Command timed out after 30 seconds")
		_ = writeAudit(AuditEntry{Event: "command", Detail: cmdLine, Success: false, Error: "timeout"})
	}

	return nil
}

// EditFileFlow handles file editing
func EditFileFlow(ctx context.Context, r *bufio.Reader, out io.Writer) error {
	info(out, "\n=== EDIT FILE ===")
	fmt.Fprint(out, "Enter file path: ")

	filePath, err := r.ReadString('\n')
	if err != nil {
		return err
	}

	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return nil
	}

	// Expand home directory
	if strings.HasPrefix(filePath, "~/") {
		home, _ := os.UserHomeDir()
		filePath = filepath.Join(home, filePath[2:])
	}

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprint(out, "File doesn't exist. Create it? (yes/no): ")
			confirm, _ := r.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(confirm)) != "yes" {
				return nil
			}
			// Create directory if needed
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			// Create empty file
			if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			info(out, "[INFO] File created")
			_ = writeAudit(AuditEntry{Event: "file_create", Detail: filePath, Success: true})
		} else {
			return fmt.Errorf("failed to access file: %w", err)
		}
	} else {
		// Show file info
		info(out, fmt.Sprintf("[INFO] File: %s", filePath))
		fmt.Fprintf(out, "[INFO] Size: %d bytes\n", fileInfo.Size())
		fmt.Fprintf(out, "[INFO] Mode: %s\n", fileInfo.Mode())
		fmt.Fprintf(out, "[INFO] Modified: %s\n", fileInfo.ModTime().Format("2006-01-02 15:04:05"))
	}

	// Create backup
	backupPath := filePath + ".backup"
	content, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read file: %w", err)
	}
	if len(content) > 0 {
		if err := os.WriteFile(backupPath, content, 0644); err != nil {
			warn(out, fmt.Sprintf("[WARN] Failed to create backup: %v", err))
		} else {
			info(out, fmt.Sprintf("[INFO] Backup created: %s", backupPath))
		}
	}

	// Edit options
	fmt.Fprintln(out, "\nEdit options:")
	fmt.Fprintln(out, "1) Open in system editor")
	fmt.Fprintln(out, "2) Append text")
	fmt.Fprintln(out, "3) Replace content")
	fmt.Fprintln(out, "4) View content")
	fmt.Fprintln(out, "0) Cancel")
	fmt.Fprint(out, "\nSelect > ")

	choice, _ := r.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		// Open in editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			if runtime.GOOS == "windows" {
				editor = "notepad"
			} else {
				editor = "nano"
			}
		}
		cmd := exec.CommandContext(ctx, editor, filePath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()

	case "2":
		// Append text
		fmt.Fprintln(out, "Enter text to append (empty line to finish):")
		var lines []string
		for {
			line, _ := r.ReadString('\n')
			if line == "\n" {
				break
			}
			lines = append(lines, line)
		}
		text := strings.Join(lines, "")
		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = file.WriteString(text)
		if err != nil {
			return err
		}
		success(out, "[SUCCESS] Text appended")

	case "3":
		// Replace content
		fmt.Fprintln(out, "Enter new content (empty line to finish):")
		var lines []string
		for {
			line, _ := r.ReadString('\n')
			if line == "\n" {
				break
			}
			lines = append(lines, line)
		}
		text := strings.Join(lines, "")
		err := os.WriteFile(filePath, []byte(text), 0644)
		if err != nil {
			return err
		}
		success(out, "[SUCCESS] Content replaced")

	case "4":
		// View content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		info(out, "\n--- File Content ---")
		fmt.Fprintln(out, string(content))
		info(out, "--- End of File ---")
	}

	return nil
}

// ServiceFlow manages system services
func ServiceFlow(ctx context.Context, r *bufio.Reader, out io.Writer) error {
	info(out, "\n=== MANAGE SERVICE ===")

	// Check OS
	var serviceCmd string
	switch runtime.GOOS {
	case "linux":
		serviceCmd = "systemctl"
		if _, err := exec.LookPath(serviceCmd); err != nil {
			serviceCmd = "service"
		}
	case "darwin":
		serviceCmd = "launchctl"
	case "windows":
		serviceCmd = "sc"
	default:
		return fmt.Errorf("service management not supported on %s", runtime.GOOS)
	}

	info(out, fmt.Sprintf("Using service manager: %s", serviceCmd))
	fmt.Fprint(out, "Enter service name: ")

	serviceName, _ := r.ReadString('\n')
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return nil
	}

	fmt.Fprintln(out, "\nActions:")
	fmt.Fprintln(out, "1) Start service")
	fmt.Fprintln(out, "2) Stop service")
	fmt.Fprintln(out, "3) Restart service")
	fmt.Fprintln(out, "4) Status")
	fmt.Fprintln(out, "0) Cancel")
	fmt.Fprint(out, "\nSelect > ")

	action, _ := r.ReadString('\n')
	action = strings.TrimSpace(action)

	var cmdArgs []string
	needsElevation := false

	switch runtime.GOOS {
	case "linux":
		needsElevation = true
		switch action {
		case "1":
			cmdArgs = []string{"start", serviceName}
		case "2":
			cmdArgs = []string{"stop", serviceName}
		case "3":
			cmdArgs = []string{"restart", serviceName}
		case "4":
			cmdArgs = []string{"status", serviceName}
			needsElevation = false
		default:
			return nil
		}

	case "darwin":
		switch action {
		case "1":
			cmdArgs = []string{"load", serviceName}
		case "2":
			cmdArgs = []string{"unload", serviceName}
		case "3":
			cmdArgs = []string{"unload", serviceName}
			// Will need to load after unload
		case "4":
			cmdArgs = []string{"list", serviceName}
		default:
			return nil
		}

	case "windows":
		switch action {
		case "1":
			cmdArgs = []string{"start", serviceName}
		case "2":
			cmdArgs = []string{"stop", serviceName}
		case "3":
			cmdArgs = []string{"stop", serviceName}
			// Will need to start after stop
		case "4":
			cmdArgs = []string{"query", serviceName}
		default:
			return nil
		}
	}

	// Check if elevation needed
	if needsElevation && !elevatedSession {
		fmt.Fprintln(out, "\n[INFO] This operation requires administrator privileges")
		ok, err := RequestElevation(ctx, fmt.Sprintf("manage service: %s %s", serviceCmd, strings.Join(cmdArgs, " ")))
		if err != nil || !ok {
			return fmt.Errorf("elevation required but not granted")
		}
	}

	// Execute command
	fmt.Fprintf(out, "\n[EXEC] %s %s\n", serviceCmd, strings.Join(cmdArgs, " "))

	var cmd *exec.Cmd
	if needsElevation && runtime.GOOS != "windows" {
		cmd = exec.CommandContext(ctx, "sudo", append([]string{serviceCmd}, cmdArgs...)...)
	} else {
		cmd = exec.CommandContext(ctx, serviceCmd, cmdArgs...)
	}

	output, err := cmd.CombinedOutput()
	fmt.Fprintln(out, string(output))

	if err != nil {
		return fmt.Errorf("service command failed: %w", err)
	}

	// Handle restart on macOS
	if runtime.GOOS == "darwin" && action == "3" {
		time.Sleep(1 * time.Second)
		cmd = exec.CommandContext(ctx, serviceCmd, "load", serviceName)
		output, err = cmd.CombinedOutput()
		fmt.Fprintln(out, string(output))
		if err != nil {
			return fmt.Errorf("service restart (load) failed: %w", err)
		}
	}

	// Handle restart on Windows
	if runtime.GOOS == "windows" && action == "3" {
		time.Sleep(1 * time.Second)
		cmd = exec.CommandContext(ctx, serviceCmd, "start", serviceName)
		output, err = cmd.CombinedOutput()
		fmt.Fprintln(out, string(output))
		if err != nil {
			return fmt.Errorf("service restart (start) failed: %w", err)
		}
	}

	fmt.Fprintln(out, "[SUCCESS] Service operation completed")
	return nil
}

// PackageFlow handles package management
func PackageFlow(ctx context.Context, r *bufio.Reader, out io.Writer) error {
	info(out, "\n=== PACKAGE MANAGEMENT ===")

	// Detect package manager
	packageManager := detectPackageManager()
	if packageManager == "" {
		return fmt.Errorf("no supported package manager found")
	}

	info(out, fmt.Sprintf("Using package manager: %s", packageManager))
	fmt.Fprint(out, "Enter package name: ")

	packageName, _ := r.ReadString('\n')
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil
	}

	fmt.Fprintln(out, "\nActions:")
	fmt.Fprintln(out, "1) Install package")
	fmt.Fprintln(out, "2) Remove package")
	fmt.Fprintln(out, "3) Update package")
	fmt.Fprintln(out, "4) Search package")
	fmt.Fprintln(out, "5) Show info")
	fmt.Fprintln(out, "0) Cancel")
	fmt.Fprint(out, "\nSelect > ")

	action, _ := r.ReadString('\n')
	action = strings.TrimSpace(action)

	var cmdArgs []string
	needsElevation := false
	isDryRun := false

	// Build command based on package manager and action
	switch packageManager {
	case "apt", "apt-get":
		switch action {
		case "1":
			cmdArgs = []string{"install", "-y", packageName}
			needsElevation = true
		case "2":
			cmdArgs = []string{"remove", "-y", packageName}
			needsElevation = true
		case "3":
			cmdArgs = []string{"upgrade", "-y", packageName}
			needsElevation = true
		case "4":
			cmdArgs = []string{"search", packageName}
		case "5":
			cmdArgs = []string{"show", packageName}
		default:
			return nil
		}

	case "yum", "dnf":
		switch action {
		case "1":
			cmdArgs = []string{"install", "-y", packageName}
			needsElevation = true
		case "2":
			cmdArgs = []string{"remove", "-y", packageName}
			needsElevation = true
		case "3":
			cmdArgs = []string{"update", "-y", packageName}
			needsElevation = true
		case "4":
			cmdArgs = []string{"search", packageName}
		case "5":
			cmdArgs = []string{"info", packageName}
		default:
			return nil
		}

	case "brew":
		switch action {
		case "1":
			cmdArgs = []string{"install", packageName}
		case "2":
			cmdArgs = []string{"uninstall", packageName}
		case "3":
			cmdArgs = []string{"upgrade", packageName}
		case "4":
			cmdArgs = []string{"search", packageName}
		case "5":
			cmdArgs = []string{"info", packageName}
		default:
			return nil
		}

	case "pacman":
		switch action {
		case "1":
			cmdArgs = []string{"-S", "--noconfirm", packageName}
			needsElevation = true
		case "2":
			cmdArgs = []string{"-R", "--noconfirm", packageName}
			needsElevation = true
		case "3":
			cmdArgs = []string{"-Syu", "--noconfirm", packageName}
			needsElevation = true
		case "4":
			cmdArgs = []string{"-Ss", packageName}
		case "5":
			cmdArgs = []string{"-Si", packageName}
		default:
			return nil
		}

	case "choco":
		switch action {
		case "1":
			cmdArgs = []string{"install", "-y", packageName}
			needsElevation = true
		case "2":
			cmdArgs = []string{"uninstall", "-y", packageName}
			needsElevation = true
		case "3":
			cmdArgs = []string{"upgrade", "-y", packageName}
			needsElevation = true
		case "4":
			cmdArgs = []string{"search", packageName}
		case "5":
			cmdArgs = []string{"info", packageName}
		default:
			return nil
		}
	}

	// Dry run for install/remove/update
	if action == "1" || action == "2" || action == "3" {
		fmt.Fprintln(out, "\n[DRY RUN] Checking what would happen...")
		isDryRun = true

		// Add dry run flags
		dryRunArgs := make([]string, len(cmdArgs))
		copy(dryRunArgs, cmdArgs)

		switch packageManager {
		case "apt", "apt-get":
			dryRunArgs = append([]string{"--dry-run"}, dryRunArgs...)
		case "yum", "dnf":
			dryRunArgs[0] = dryRunArgs[0] + " --assumeno"
		}

		var cmd *exec.Cmd
		if isDryRun && packageManager != "brew" && packageManager != "choco" {
			if needsElevation && runtime.GOOS != "windows" {
				cmd = exec.CommandContext(ctx, "sudo", append([]string{packageManager}, dryRunArgs...)...)
			} else {
				cmd = exec.CommandContext(ctx, packageManager, dryRunArgs...)
			}

			output, _ := cmd.CombinedOutput()
			fmt.Fprintln(out, string(output))

			fmt.Fprint(out, "\nProceed with actual operation? (yes/no): ")
			confirm, _ := r.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(confirm)) != "yes" {
				fmt.Fprintln(out, "Operation cancelled.")
				return nil
			}
		}
	}

	// Check if elevation needed
	if needsElevation && !elevatedSession {
		fmt.Fprintln(out, "\n[INFO] This operation requires administrator privileges")
		ok, err := RequestElevation(ctx, fmt.Sprintf("package management: %s %s", packageManager, strings.Join(cmdArgs, " ")))
		if err != nil || !ok {
			return fmt.Errorf("elevation required but not granted")
		}
	}

	// Execute actual command
	info(out, fmt.Sprintf("\n[EXEC] %s %s", packageManager, strings.Join(cmdArgs, " ")))

	var cmd *exec.Cmd
	if needsElevation && runtime.GOOS != "windows" {
		cmd = exec.CommandContext(ctx, "sudo", append([]string{packageManager}, cmdArgs...)...)
	} else {
		cmd = exec.CommandContext(ctx, packageManager, cmdArgs...)
	}

	cmd.Stdout = out
	cmd.Stderr = out

	err := cmd.Run()
	if err != nil {
		_ = writeAudit(AuditEntry{Event: "package", Detail: packageManager + " " + strings.Join(cmdArgs, " "), Success: false, Error: err.Error()})
		return fmt.Errorf("package command failed: %w", err)
	}

	success(out, "\n[SUCCESS] Package operation completed")
	_ = writeAudit(AuditEntry{Event: "package", Detail: packageManager + " " + strings.Join(cmdArgs, " "), Success: true})
	return nil
}

// RequestElevation requests elevated privileges
func RequestElevation(ctx context.Context, reason string) (bool, error) {
	if elevatedSession {
		return true, nil
	}

	fmt.Printf("\n[ELEVATION REQUEST]\n")
	fmt.Printf("Reason: %s\n", reason)
	fmt.Printf("This operation requires administrator/root privileges.\n")
	fmt.Print("Grant elevation for this session? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response == "yes" || response == "y" {
		// Check if we can actually elevate
		if runtime.GOOS == "windows" {
			// On Windows, check if running as admin
			cmd := exec.Command("net", "session")
			err := cmd.Run()
			if err != nil {
				return false, fmt.Errorf("not running as administrator")
			}
		} else {
			// On Unix-like systems, check sudo
			cmd := exec.Command("sudo", "-n", "true")
			err := cmd.Run()
			if err != nil {
				// Try to authenticate
				fmt.Println("Please enter your password:")
				cmd = exec.Command("sudo", "-v")
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
				if err != nil {
					return false, fmt.Errorf("sudo authentication failed")
				}
			}
		}

		elevatedSession = true
		fmt.Println("[INFO] Elevation granted for this session")
		return true, nil
	}

	return false, nil
}

// Helper functions

func isDangerousCommand(cmd string) bool {
	dangerous := []string{
		"rm -rf",
		"format",
		"del /f",
		"dd if=",
		"mkfs",
		"> /dev/",
		"sudo rm",
		"chmod 777",
		"kill -9",
	}

	cmdLower := strings.ToLower(cmd)
	for _, d := range dangerous {
		if strings.Contains(cmdLower, d) {
			return true
		}
	}
	return false
}

func detectPackageManager() string {
	managers := []string{
		"apt",
		"apt-get",
		"dnf",
		"yum",
		"pacman",
		"brew",
		"choco",
		"snap",
		"flatpak",
	}

	for _, mgr := range managers {
		if _, err := exec.LookPath(mgr); err == nil {
			return mgr
		}
	}

	return ""
}

// Colored helpers and audit writer
func info(out io.Writer, s string)      { fmt.Fprintln(out, color.New(color.FgCyan).Sprint(s)) }
func warn(out io.Writer, s string)      { fmt.Fprintln(out, color.New(color.FgYellow).Sprint(s)) }
func success(out io.Writer, s string)   { fmt.Fprintln(out, color.New(color.FgGreen).Sprint(s)) }
func errorLine(out io.Writer, s string) { fmt.Fprintln(out, color.New(color.FgRed).Sprint(s)) }

type AuditEntry struct {
	Timestamp string `json:"ts"`
	Event     string `json:"event"`
	Detail    string `json:"detail"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

func writeAudit(e AuditEntry) error {
	e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dir := filepath.Join(home, ".gollm")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil
	}
	path := filepath.Join(dir, "audit.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil
	}
	defer f.Close()
	enc, _ := json.Marshal(e)
	_, _ = f.Write(append(enc, '\n'))
	return nil
}
