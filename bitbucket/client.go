package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client wraps the Bitbucket Cloud REST API (v2).
type Client struct {
	baseURL   string
	workspace string
	apiKey    string
	http      *http.Client
}

// New creates a new Bitbucket Client.
func New(baseURL, workspace, apiKey string) *Client {
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		workspace: workspace,
		apiKey:    apiKey,
		http:      &http.Client{},
	}
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// DefaultBranch returns the default branch name of a repository.
func (c *Client) DefaultBranch(repoSlug string) (string, error) {
	var result struct {
		MainBranch struct {
			Name string `json:"name"`
		} `json:"mainbranch"`
	}
	if err := c.get(fmt.Sprintf("/repositories/%s/%s", c.workspace, repoSlug), &result); err != nil {
		return "", err
	}
	if result.MainBranch.Name == "" {
		return "main", nil
	}
	return result.MainBranch.Name, nil
}

// CreateBranch creates a new branch from the tip of baseBranch.
func (c *Client) CreateBranch(repoSlug, newBranch, baseBranch string) error {
	// First get the latest commit on the base branch
	var branchInfo struct {
		Target struct {
			Hash string `json:"hash"`
		} `json:"target"`
	}
	if err := c.get(
		fmt.Sprintf("/repositories/%s/%s/refs/branches/%s", c.workspace, repoSlug, baseBranch),
		&branchInfo,
	); err != nil {
		return fmt.Errorf("getting base branch tip: %w", err)
	}

	payload := map[string]interface{}{
		"name": newBranch,
		"target": map[string]string{
			"hash": branchInfo.Target.Hash,
		},
	}
	return c.post(
		fmt.Sprintf("/repositories/%s/%s/refs/branches", c.workspace, repoSlug),
		payload,
		nil,
	)
}

// CommitFile creates or updates a single file on the given branch using
// the Bitbucket source API (multipart/form-data).
func (c *Client) CommitFile(repoSlug, branch, filePath, content, message string) error {
	url := fmt.Sprintf("%s/repositories/%s/%s/src", c.baseURL, c.workspace, repoSlug)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	// File content field – field name is the file path
	fw, err := mw.CreateFormField(filePath)
	if err != nil {
		return err
	}
	if _, err = fw.Write([]byte(content)); err != nil {
		return err
	}

	// Commit metadata
	_ = writeField(mw, "message", message)
	_ = writeField(mw, "branch", branch)
	mw.Close()

	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bitbucket commit file %s: status %d – %s", filePath, resp.StatusCode, string(raw))
	}
	return nil
}

// OpenPR creates a pull request from sourceBranch into destBranch.
// It returns the PR URL on success.
func (c *Client) OpenPR(repoSlug, title, description, sourceBranch, destBranch string) (string, error) {
	payload := map[string]interface{}{
		"title":       title,
		"description": description,
		"source": map[string]interface{}{
			"branch": map[string]string{"name": sourceBranch},
		},
		"destination": map[string]interface{}{
			"branch": map[string]string{"name": destBranch},
		},
		"close_source_branch": true,
	}

	var result struct {
		Links struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
	}

	if err := c.post(
		fmt.Sprintf("/repositories/%s/%s/pullrequests", c.workspace, repoSlug),
		payload,
		&result,
	); err != nil {
		return "", err
	}
	return result.Links.HTML.Href, nil
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

func (c *Client) get(path string, out interface{}) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bitbucket GET %s: status %d – %s", path, resp.StatusCode, string(raw))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) post(path string, payload, out interface{}) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bitbucket POST %s: status %d – %s", path, resp.StatusCode, string(raw))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func writeField(mw *multipart.Writer, key, value string) error {
	fw, err := mw.CreateFormField(key)
	if err != nil {
		return err
	}
	_, err = fw.Write([]byte(value))
	return err
}
