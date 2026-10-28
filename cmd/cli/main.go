package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

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

func main() {
	loadEnv(".env")
	loadEnv("../../.env")
	loadEnv("../.env")

	// CLI Flags
	diffFlag := flag.String("diff", "", "Git diff text directly (alternative to piping or -file)")
	styleFlag := flag.String("style", "conventional", "Commit style: conventional, angular, simple")
	contextFlag := flag.String("context", "", "Optional extra context about the change")
	fileFlag := flag.String("file", "", "Path to diff or patch file")
	onlyCommitFlag := flag.Bool("commit-only", false, "Print only the generated commit subject")
	versionFlag := flag.Bool("version", false, "Print CLI version and build metadata")
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

	// 1. Check if -diff was provided directly
	if *diffFlag != "" {
		diffText = *diffFlag
	} else if *fileFlag != "" {
		// 2. Check if a file was provided via -file
		content, err := os.ReadFile(*fileFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", *fileFlag, err)
			os.Exit(1)
		}
		diffText = string(content)
	} else {
		// 3. Check if data is being piped via stdin
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
		fmt.Fprintln(os.Stderr, "Error: No diff provided. Use -diff, -file, or pipe via stdin.")
		fmt.Fprintln(os.Stderr, "Example: commitlint-ai -diff \"$(git diff)\"")
		flag.Usage()
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
