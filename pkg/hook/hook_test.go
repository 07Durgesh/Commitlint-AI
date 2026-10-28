package hook

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInstallAndUninstallHook(t *testing.T) {
	// Create a temporary git repo to test hook installation
	tmpDir, err := os.MkdirTemp("", "git-hook-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git in temp directory
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skip("git not available for hook test, skipping")
	}

	// Change working directory to temp git repo
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}

	// 1. Install Hook
	if err := InstallHook(); err != nil {
		t.Fatalf("InstallHook failed: %v", err)
	}

	hookFile := filepath.Join(tmpDir, ".git", "hooks", "prepare-commit-msg")
	if _, err := os.Stat(hookFile); os.IsNotExist(err) {
		t.Fatalf("hook file was not created at %s", hookFile)
	}

	// 2. Uninstall Hook
	if err := UninstallHook(); err != nil {
		t.Fatalf("UninstallHook failed: %v", err)
	}

	if _, err := os.Stat(hookFile); !os.IsNotExist(err) {
		t.Fatalf("hook file should have been deleted after UninstallHook")
	}
}
