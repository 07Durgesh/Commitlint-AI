package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/07Durgesh/commitlint-AI/pkg/hook"
	"github.com/07Durgesh/commitlint-AI/pkg/service"
)

// Build metadata injected at compile time via -ldflags
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// loadEnv reads key-value pairs from a .env file if it exists
func loadEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
}

func printUsage() {
	fmt.Println("⚡ CommitLint.ai — AI-Powered Git Commit & PR Generator")
	fmt.Println("\nUSAGE:")
	fmt.Println("  commitlint [command] [options]")
	fmt.Println("\nCOMMANDS:")
	fmt.Println("  install-hook      Install automatic 'prepare-commit-msg' git hook into current repo")
	fmt.Println("  uninstall-hook    Remove the CommitLint git hook from current repo")
	fmt.Println("  hook <file>       Internal command invoked automatically by git during 'git commit'")
	fmt.Println("\nFLAGS:")
	fmt.Println("  -diff string      Git diff text to analyze directly")
	fmt.Println("  -context string   Optional extra context about the changes")
	fmt.Println("  -style string     Commit style: conventional (default), angular, simple")
	fmt.Println("  -file string      Path to a diff or patch file")
	fmt.Println("  -commit-only      Print only the generated commit subject")
	fmt.Println("  -version          Print CLI version and build metadata")
	fmt.Println("\nEXAMPLES:")
	fmt.Println("  git diff | commitlint")
	fmt.Println("  commitlint -diff \"$(git diff)\"")
	fmt.Println("  commitlint install-hook")
}

func main() {
	loadEnv(".env")
	loadEnv("../../.env")
	loadEnv("../.env")

	// 1. Handle Subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install-hook":
			if err := hook.InstallHook(); err != nil {
				fmt.Fprintf(os.Stderr, "Error installing git hook: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ CommitLint git hook installed into .git/hooks/prepare-commit-msg")
			fmt.Println("⚡ From now on, simply run 'git commit' and your commit message will be pre-filled automatically!")
			return

		case "uninstall-hook":
			if err := hook.UninstallHook(); err != nil {
				fmt.Fprintf(os.Stderr, "Error uninstalling git hook: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ CommitLint git hook uninstalled successfully.")
			return

		case "hook":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "Error: 'hook' command requires commit message file path.")
				os.Exit(1)
			}
			commitMsgFile := os.Args[2]
			source := ""
			if len(os.Args) > 3 {
				source = os.Args[3]
			}

			apiKey := os.Getenv("GEMINI_API_KEY")
			if apiKey == "" {
				fmt.Fprintln(os.Stderr, "⚠️ CommitLint: GEMINI_API_KEY not set. Skipping automatic commit generation.")
				return
			}
			svc := service.NewGeminiService(apiKey)
			if err := hook.RunHook(commitMsgFile, source, svc); err != nil {
				fmt.Fprintf(os.Stderr, "Hook error: %v\n", err)
			}
			return

		case "-h", "--help", "help":
			printUsage()
			return
		}
	}

	// 2. Standard Flag Parsing
	diffFlag := flag.String("diff", "", "Git diff text directly (alternative to piping or -file)")
	styleFlag := flag.String("style", "conventional", "Commit style: conventional, angular, simple")
	contextFlag := flag.String("context", "", "Optional extra context about the change")
	fileFlag := flag.String("file", "", "Path to diff or patch file")
	onlyCommitFlag := flag.Bool("commit-only", false, "Print only the generated commit subject")
	versionFlag := flag.Bool("version", false, "Print CLI version and build metadata")
	flag.Usage = printUsage
	flag.Parse()

	if *versionFlag {
		fmt.Printf("CommitLint CLI %s (commit: %s, built: %s)\n", Version, Commit, BuildDate)
		return
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: GEMINI_API_KEY environment variable is not set.")
		os.Exit(1)
	}

	var diffText string

	// Check if -diff was provided directly
	if *diffFlag != "" {
		diffText = *diffFlag
	} else if *fileFlag != "" {
		content, err := os.ReadFile(*fileFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", *fileFlag, err)
			os.Exit(1)
		}
		diffText = string(content)
	} else {
		// Check stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			bytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
				os.Exit(1)
			}
			diffText = string(bytes)
		}
	}

	diffText = strings.TrimSpace(diffText)
	if diffText == "" {
		printUsage()
		os.Exit(1)
	}

	svc := service.NewGeminiService(apiKey)
	fmt.Fprintln(os.Stderr, "Analyzing diff with Gemini AI...")

	result, err := svc.Analyse(diffText, *contextFlag, *styleFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Analysis failed: %v\n", err)
		os.Exit(1)
	}

	if *onlyCommitFlag {
		fmt.Println(result.CommitMessage.Subject)
		return
	}

	// Pretty-print results to terminal
	fmt.Println("\n==================================================")
	fmt.Println("📝 RECOMMENDED COMMIT MESSAGE")
	fmt.Println("==================================================")
	fmt.Println(result.CommitMessage.Full)

	fmt.Println("\n==================================================")
	fmt.Println("📄 PULL REQUEST DESCRIPTION")
	fmt.Println("==================================================")
	if result.PRDescription.FullMarkdown != "" {
		fmt.Println(result.PRDescription.FullMarkdown)
	} else {
		fmt.Printf("### What\n%s\n\n### Why\n%s\n\n### How\n%s\n",
			result.PRDescription.What, result.PRDescription.Why, result.PRDescription.How)
	}

	if len(result.Observations) > 0 {
		fmt.Println("\n==================================================")
		fmt.Println("💡 CODE OBSERVATIONS")
		fmt.Println("==================================================")
		for _, obs := range result.Observations {
			icon := "💡"
			if obs.Type == "warning" {
				icon = "⚠️"
			} else if obs.Type == "issue" {
				icon = "🛑"
			}
			fmt.Printf("%s [%s] %s\n", icon, strings.ToUpper(obs.Type), obs.Message)
		}
	}
	fmt.Println()
}
