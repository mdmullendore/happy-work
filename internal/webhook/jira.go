package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ---------------------------------------------------------------------------
// Jira webhook payload types (minimal – only fields we need)
// ---------------------------------------------------------------------------

// JiraEvent is the top-level webhook payload sent by Jira.
type JiraEvent struct {
	WebhookEvent string      `json:"webhookEvent"`
	Transition   *Transition `json:"transition,omitempty"`
	Issue        Issue       `json:"issue"`
	User         User        `json:"user"`
}

type Transition struct {
	ID string `json:"id"`
	To Status `json:"to"`
}

type Status struct {
	Name string `json:"name"`
}

type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
}

type IssueFields struct {
	Summary     string  `json:"summary"`
	Description string  `json:"description"`
	Project     Project `json:"project"`
	IssueType   struct {
		Name string `json:"name"`
	} `json:"issuetype"`
	Assignee *User `json:"assignee"`
}

type Project struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type User struct {
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
}

// ---------------------------------------------------------------------------
// Parser
// ---------------------------------------------------------------------------

// Parse reads and optionally verifies a Jira webhook request, returning the
// parsed event. If secret is non-empty the HMAC-SHA256 signature is checked.
func Parse(r *http.Request, secret string) (*JiraEvent, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20)) // 2 MB limit
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	if secret != "" {
		if err := verifySignature(r, body, secret); err != nil {
			return nil, err
		}
	}

	var event JiraEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("unmarshalling jira event: %w", err)
	}
	return &event, nil
}

// IsTransitionTo returns true when the event is a status-transition to the
// given status name (case-insensitive).
func IsTransitionTo(event *JiraEvent, status string) bool {
	if event.WebhookEvent != "jira:issue_updated" {
		return false
	}
	if event.Transition == nil {
		return false
	}
	return strings.EqualFold(event.Transition.To.Name, status)
}

// verifySignature checks the X-Hub-Signature-256 header produced by Jira.
func verifySignature(r *http.Request, body []byte, secret string) error {
	sig := r.Header.Get("X-Hub-Signature-256")
	if sig == "" {
		return fmt.Errorf("missing X-Hub-Signature-256 header")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}
