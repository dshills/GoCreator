package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// getBinaryPath returns the path to the gocreator binary
func getBinaryPath(t *testing.T) string {
	t.Helper()
	binPath := filepath.Join("..", "..", "bin", "gocreator")
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Skip("Binary not built, run 'make build' first")
	}
	return binPath
}

// runCommand executes the gocreator command and returns stdout, stderr, and error
func runCommand(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	binPath := getBinaryPath(t)

	cmd := exec.Command(binPath, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	return stdout, stderr, err
}

// TestCLIVersion tests the version command
func TestCLIVersion(t *testing.T) {
	stdout, _, err := runCommand(t, "version")
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	if !strings.Contains(stdout, "GoCreator v") {
		t.Errorf("version output missing 'GoCreator v': %s", stdout)
	}
	if !strings.Contains(stdout, "Go version:") {
		t.Errorf("version output missing 'Go version:': %s", stdout)
	}
}

// TestCLIHelp tests the help command
func TestCLIHelp(t *testing.T) {
	stdout, _, err := runCommand(t, "--help")
	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	requiredCommands := []string{
		"clarify",
		"generate",
		"validate",
		"full",
		"dump-fcs",
		"version",
	}

	for _, cmd := range requiredCommands {
		if !strings.Contains(stdout, cmd) {
			t.Errorf("help output missing command '%s': %s", cmd, stdout)
		}
	}
}

// TestCLIClarifyHelp tests the clarify command help
func TestCLIClarifyHelp(t *testing.T) {
	stdout, _, err := runCommand(t, "clarify", "--help")
	if err != nil {
		t.Fatalf("clarify help command failed: %v", err)
	}

	requiredFlags := []string{
		"--output",
		"--interactive",
		"--batch",
	}

	for _, flag := range requiredFlags {
		if !strings.Contains(stdout, flag) {
			t.Errorf("clarify help missing flag '%s': %s", flag, stdout)
		}
	}
}

// TestCLIGenerateHelp tests the generate command help
func TestCLIGenerateHelp(t *testing.T) {
	stdout, _, err := runCommand(t, "generate", "--help")
	if err != nil {
		t.Fatalf("generate help command failed: %v", err)
	}

	requiredFlags := []string{
		"--output",
		"--resume",
		"--batch",
		"--dry-run",
	}

	for _, flag := range requiredFlags {
		if !strings.Contains(stdout, flag) {
			t.Errorf("generate help missing flag '%s': %s", flag, stdout)
		}
	}
}

// TestCLIValidateHelp tests the validate command help
func TestCLIValidateHelp(t *testing.T) {
	stdout, _, err := runCommand(t, "validate", "--help")
	if err != nil {
		t.Fatalf("validate help command failed: %v", err)
	}

	requiredFlags := []string{
		"--skip-build",
		"--skip-lint",
		"--skip-tests",
		"--report",
	}

	for _, flag := range requiredFlags {
		if !strings.Contains(stdout, flag) {
			t.Errorf("validate help missing flag '%s': %s", flag, stdout)
		}
	}
}

// TestCLIFullHelp tests the full command help
func TestCLIFullHelp(t *testing.T) {
	stdout, _, err := runCommand(t, "full", "--help")
	if err != nil {
		t.Fatalf("full help command failed: %v", err)
	}

	requiredFlags := []string{
		"--output",
		"--batch",
		"--resume",
		"--report",
	}

	for _, flag := range requiredFlags {
		if !strings.Contains(stdout, flag) {
			t.Errorf("full help missing flag '%s': %s", flag, stdout)
		}
	}
}

// TestCLIDumpFCSHelp tests the dump-fcs command help
func TestCLIDumpFCSHelp(t *testing.T) {
	stdout, _, err := runCommand(t, "dump-fcs", "--help")
	if err != nil {
		t.Fatalf("dump-fcs help command failed: %v", err)
	}

	requiredFlags := []string{
		"--output",
		"--batch",
		"--pretty",
	}

	for _, flag := range requiredFlags {
		if !strings.Contains(stdout, flag) {
			t.Errorf("dump-fcs help missing flag '%s': %s", flag, stdout)
		}
	}
}

// TestCLIClarifyMissingArgument tests clarify command without spec file
func TestCLIClarifyMissingArgument(t *testing.T) {
	_, _, err := runCommand(t, "clarify")
	if err == nil {
		t.Fatal("expected error for missing argument, got nil")
	}
	// Error is expected, command properly validates arguments
}

// TestCLIGenerateMissingArgument tests generate command without spec file
func TestCLIGenerateMissingArgument(t *testing.T) {
	_, _, err := runCommand(t, "generate")
	if err == nil {
		t.Fatal("expected error for missing argument, got nil")
	}
	// Error is expected, command properly validates arguments
}

// TestCLIValidateMissingArgument tests validate command without project root
func TestCLIValidateMissingArgument(t *testing.T) {
	_, _, err := runCommand(t, "validate")
	if err == nil {
		t.Fatal("expected error for missing argument, got nil")
	}
	// Error is expected, command properly validates arguments
}
