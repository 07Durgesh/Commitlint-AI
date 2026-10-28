package model

type AnalyzeRequest struct {
	Diff    string `json:"diff"`
	Context string `json:"context,omitempty"`
	Style   string `json:"style"`
}

type CommitMessage struct {
	Subject string `json:"subject"`
	Body    string `json:"body,omitempty"`
	Full    string `json:"full"`
}

type PRDescription struct {
	What         string `json:"what"`
	Why          string `json:"why"`
	How          string `json:"how"`
	FullMarkdown string `json:"full_markdown"`
}

type Observation struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type AnalyzeResponse struct {
	CommitMessage CommitMessage `json:"commit_message"`
	PRDescription PRDescription `json:"pr_description"`
	Observations  []Observation `json:"observations"`
}

type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}
