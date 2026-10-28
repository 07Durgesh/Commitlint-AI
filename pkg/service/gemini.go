package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/07Durgesh/commitlint-AI/pkg/model"
)

type GeminiService struct {
	apiKey     string
	httpClient *http.Client
	model      string
}

func NewGeminiService(apiKey string) *GeminiService {
	modelName := os.Getenv("GEMINI_MODEL")
	if modelName == "" {
		modelName = "gemini-3.5-flash-lite"
	}

	return &GeminiService{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		model:      modelName,
	}
}

func BuildPrompt(diff, context, style string) string {
	prompt := fmt.Sprintf(`You are CommitLint.ai, an expert at analyzing code diffs and generating 
high-quality git commit messages and pull request descriptions.

TASK:
Analyze the given git diff and generate a clear, professional commit message following the Conventional Commits specification.

GIT DIFF:
%s

ADDITIONAL CONTEXT (from the developer):
%s

COMMIT STYLE PREFERENCE:
%s

RULES:
1. Subject line MUST follow the format: <type>(<scope>): <description>
2. Valid types: feat, fix, refactor, docs, style, test, chore, perf, ci, build, revert
3. Subject line must be <= 72 characters, lowercase, no period at the end
4. Body should explain WHAT changed and WHY (not HOW — the diff shows how)
5. If there are multiple logical changes, note them as observations
6. Be specific — avoid vague words like "update", "fix stuff", "changes"

RESPONSE FORMAT:
You MUST respond with valid JSON only. No markdown, no code fences, no extra text.

{
  "commit_message": {
    "subject": "<type>(<scope>): <concise description>",
    "body": "<detailed explanation of what and why>",
    "full": "<subject>\n\n<body>"
  },
  "pr_description": {
    "what": "<what this PR/change does in 1-2 sentences>",
    "why": "<why this change is needed>",
    "how": "<brief technical approach>",
    "full_markdown": "<complete PR description in markdown format>"
  },
  "observations": [
    {
      "type": "<suggestion|warning|issue>",
      "message": "<actionable observation about the code changes>"
    }
  ]
}

OBSERVATION TYPES:
- "suggestion": Best practices, improvements, or alternative approaches
- "warning": Potential bugs, security concerns, or missing error handling
- "issue": Breaking changes, removed functionality, or critical problems

IMPORTANT:
- Return ONLY the JSON object. No explanation before or after.
- If context or style is empty, use your best judgment based on the diff alone.
- Keep observations relevant and actionable. Max 5 observations.
- The PR description should be useful for code reviewers.`, diff, context, style)

	return prompt
}

type geminiRequest struct {
	Contents         []geminiContent `json:"contents"`
	GenerationConfig geminiGenConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenConfig struct {
	ResponseMimeType string `json:"responseMimeType"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// Analyse executes the analysis using context.Background().
func (s *GeminiService) Analyse(diff, userContext, style string) (*model.AnalyzeResponse, error) {
	return s.AnalyseWithContext(context.Background(), diff, userContext, style)
}

// Analyze is an alias for Analyse to support alternate spelling conventions.
func (s *GeminiService) Analyze(diff, userContext, style string) (*model.AnalyzeResponse, error) {
	return s.Analyse(diff, userContext, style)
}

// AnalyseWithContext executes the Gemini API request while respecting context cancellation
// and retrying transient network/5xx server errors with backoff.
func (s *GeminiService) AnalyseWithContext(ctx context.Context, diff, userContext, style string) (*model.AnalyzeResponse, error) {
	const maxRetries = 2
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*500) * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		result, shouldRetry, err := s.doRequest(ctx, diff, userContext, style)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !shouldRetry {
			return nil, err
		}
	}

	return nil, fmt.Errorf("gemini request failed after %d retries: %w", maxRetries, lastErr)
}

func (s *GeminiService) doRequest(ctx context.Context, diff, userContext, style string) (*model.AnalyzeResponse, bool, error) {
	prompt := BuildPrompt(diff, userContext, style)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: geminiGenConfig{
			ResponseMimeType: "application/json",
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, false, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		s.model, s.apiKey,
	)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, false, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		// Network-level error, candidate for retry
		return nil, true, fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read response: %w", err)
	}

	// 5xx errors or 429 rate limit errors from Gemini are candidates for retry
	if resp.StatusCode != http.StatusOK {
		shouldRetry := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		return nil, shouldRetry, fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(body))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, false, fmt.Errorf("failed to decode Gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, false, fmt.Errorf("empty response from Gemini")
	}

	resultText := geminiResp.Candidates[0].Content.Parts[0].Text

	var result model.AnalyzeResponse
	if err := json.Unmarshal([]byte(resultText), &result); err != nil {
		return nil, false, fmt.Errorf("failed to parse AI response as JSON: %w", err)
	}

	return &result, false, nil
}
