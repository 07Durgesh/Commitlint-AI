package hook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/07Durgesh/commitlint-AI/pkg/service"
)

const hookScript = `#!/usr/bin/env sh
# CommitLint.ai Git Hook — Automatically generates Conventional Commits
# Installed by 'commitlint install-hook'

COMMIT_MSG_FILE="$1"
COMMIT_SOURCE="$2"

# If user provided a message via -m or -F, don't override it
if [ -n "$COMMIT_SOURCE" ] && [ "$COMMIT_SOURCE" != "template" ]; then
    exit 0
fi

# Run commitlint hook command to analyze staged changes
commitlint hook "$COMMIT_MSG_FILE" "$COMMIT_SOURCE"
`

// FindGitRoot locates the root of the current git repository.
func FindGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository (or git is not installed)")
	}
	return strings.TrimSpace(string(out)), nil
}

// InstallHook installs the prepare-commit-msg hook into .git/hooks/
func InstallHook() error {
	gitRoot, err := FindGitRoot()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(gitRoot, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	hookPath := filepath.Join(hooksDir, "prepare-commit-msg")
	if err := os.WriteFile(hookPath, []byte(hookScript), 0755); err != nil {
		return fmt.Errorf("failed to write hook script: %w", err)
	}

	return nil
}

// UninstallHook removes the prepare-commit-msg hook.
func UninstallHook() error {
	gitRoot, err := FindGitRoot()
	if err != nil {
		return err
	}

	hookPath := filepath.Join(gitRoot, ".git", "hooks", "prepare-commit-msg")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		return fmt.Errorf("no CommitLint hook found in %s", hookPath)
	}

	return os.Remove(hookPath)
}

// RunHook is called by git during 'git commit' to analyze staged changes.
func RunHook(commitMsgFile, source string, svc *service.GeminiService) error {
	// If a message was already provided (e.g. git commit -m "..."), do nothing
	if source != "" && source != "template" {
		return nil
	}

	// 1. Get staged git diff (git diff --cached)
	cmd := exec.Command("git", "diff", "--cached")
	diffBytes, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get staged git diff: %w", err)
	}

	diffText := strings.TrimSpace(string(diffBytes))
	if diffText == "" {
		// Nothing staged, let git handle the empty commit error
		return nil
	}

	fmt.Fprintln(os.Stderr, "⚡ CommitLint.ai: Analyzing staged changes with Gemini AI...")

	// 2. Generate commit message from AI
	result, err := svc.Analyse(diffText, "automatically generated via git hook", "conventional")
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ CommitLint hook: AI analysis failed (%v). Continuing with blank commit.\n", err)
		return nil // Don't block the user's commit if AI fails
	}

	// 3. Format the commit message file
	var sb strings.Builder
	sb.WriteString(result.CommitMessage.Full)
	sb.WriteString("\n\n")

	// Add observations as git comments (lines starting with # are ignored by git)
	sb.WriteString("# --- CommitLint.ai Review Observations ---\n")
	if len(result.Observations) > 0 {
		for _, obs := range result.Observations {
			sb.WriteString(fmt.Sprintf("# [%s] %s\n", strings.ToUpper(obs.Type), obs.Message))
		}
	} else {
		sb.WriteString("# No code quality issues detected.\n")
	}
	sb.WriteString("# Edit above to customize your commit message before saving.\n")

	// 4. Overwrite COMMIT_EDITMSG
	if err := os.WriteFile(commitMsgFile, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write commit message file: %w", err)
	}

	fmt.Fprintln(os.Stderr, "✨ Commit message pre-filled! Opening your editor...")
	return nil
}
