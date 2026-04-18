package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const geminiAPIURL = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"

// ---------------------------------------------------------------------------
// Request / Response types (Gemini generateContent API)
// ---------------------------------------------------------------------------

type part struct {
	Text string `json:"text"`
}

type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

type generationConfig struct {
	MaxOutputTokens int `json:"maxOutputTokens"`
}

type apiRequest struct {
	SystemInstruction *content         `json:"system_instruction,omitempty"`
	Contents          []content        `json:"contents"`
	GenerationConfig  generationConfig `json:"generationConfig"`
}

type candidate struct {
	Content content `json:"content"`
}

type apiResponse struct {
	Candidates []candidate `json:"candidates"`
	Error      *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ---------------------------------------------------------------------------
// FileChange represents a single file that Gemini wants to create/update.
// ---------------------------------------------------------------------------

type FileChange struct {
	Path    string
	Content string
	Message string // commit message for this change
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client calls the Google Gemini API to generate code changes for a Jira issue.
type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// New creates a new Gemini Client.
func New(apiKey, model string) *Client {
	return &Client{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{},
	}
}

// GenerateChanges asks Gemini to implement the Jira issue and returns the
// list of file changes to apply to the repository.
func (c *Client) GenerateChanges(issue IssueContext) ([]FileChange, error) {
	prompt := buildPrompt(issue)

	reqBody := apiRequest{
		SystemInstruction: &content{
			Parts: []part{{Text: systemPrompt()}},
		},
		Contents: []content{
			{Role: "user", Parts: []part{{Text: prompt}}},
		},
		GenerationConfig: generationConfig{
			MaxOutputTokens: 4096,
		},
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	url := fmt.Sprintf(geminiAPIURL, c.model, c.apiKey)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Gemini API: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var apiResp apiResponse
	if err := json.Unmarshal(raw, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing Gemini response: %w", err)
	}
	if apiResp.Error != nil {
		return nil, fmt.Errorf("Gemini API error: %s", apiResp.Error.Message)
	}
	if len(apiResp.Candidates) == 0 {
		return nil, fmt.Errorf("Gemini returned no candidates")
	}

	// Collect all text parts from the first candidate
	var fullText strings.Builder
	for _, p := range apiResp.Candidates[0].Content.Parts {
		fullText.WriteString(p.Text)
	}

	return parseFileChanges(fullText.String())
}

// ---------------------------------------------------------------------------
// IssueContext carries the Jira issue details passed to Gemini.
// ---------------------------------------------------------------------------

type IssueContext struct {
	Key         string
	Summary     string
	Description string
	IssueType   string
	ProjectKey  string
	RepoSlug    string // Bitbucket repo to modify
}

// ---------------------------------------------------------------------------
// Prompt helpers
// ---------------------------------------------------------------------------

func systemPrompt() string {
	return `You are an expert software engineer. When given a Jira issue you will implement the required changes.

Respond ONLY with file changes in the following format – no other text:

<<<FILE: path/to/file.ext>>>
<full file content here>
<<<END>>>

You may include multiple FILE blocks. Each block must contain the complete new content of the file (not a diff).`
}

func buildPrompt(issue IssueContext) string {
	return fmt.Sprintf(`Implement the following Jira issue in the repository %q.

Issue Key:   %s
Issue Type:  %s
Summary:     %s

Description:
%s

Produce the minimal set of file changes required to complete this task.`,
		issue.RepoSlug,
		issue.Key,
		issue.IssueType,
		issue.Summary,
		issue.Description,
	)
}

// ---------------------------------------------------------------------------
// Response parser
// ---------------------------------------------------------------------------

// parseFileChanges parses Gemini's structured output into FileChange records.
func parseFileChanges(text string) ([]FileChange, error) {
	var changes []FileChange

	parts := strings.Split(text, "<<<FILE:")
	for _, part := range parts[1:] { // skip everything before first marker
		endMarker := strings.Index(part, ">>>")
		if endMarker < 0 {
			continue
		}
		path := strings.TrimSpace(part[:endMarker])
		rest := part[endMarker+3:]

		closeMarker := strings.Index(rest, "<<<END>>>")
		if closeMarker < 0 {
			continue
		}
		content := strings.TrimSpace(rest[:closeMarker])

		changes = append(changes, FileChange{
			Path:    path,
			Content: content,
			Message: fmt.Sprintf("chore: implement changes for %s", path),
		})
	}

	if len(changes) == 0 {
		return nil, fmt.Errorf("Gemini returned no parseable file changes")
	}
	return changes, nil
}
