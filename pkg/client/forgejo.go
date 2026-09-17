package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ForgejoClient interface {
	GetPRDiff(owner, repo string, prNumber int) (string, error)
	GetPRFileContent(owner, repo, ref, filepath string) (string, error)
	PostPRComment(owner, repo string, prNumber int, body string) error
	PostPRInlineComment(owner, repo string, prNumber int, body, commitSHA, filepath string, line int) error
	GetPRComments(owner, repo string, prNumber int) ([]PRComment, error)
}

type PRComment struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	User      User   `json:"user"`
	Path      string `json:"path,omitempty"`
	Line      int    `json:"line,omitempty"`
	ReviewID  int64  `json:"pull_request_review_id,omitempty"`
}

type User struct {
	Username string `json:"username"`
	Login    string `json:"login"`
}

type HTTPForgejoClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewForgejoClient(baseURL, token string) *HTTPForgejoClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &HTTPForgejoClient{
		BaseURL:    baseURL,
		Token:      token,
		HTTPClient: &http.Client{},
	}
}

func (c *HTTPForgejoClient) doRequest(req *http.Request) ([]byte, error) {
	if c.Token != "" {
		req.Header.Set("Authorization", "token "+c.Token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *HTTPForgejoClient) GetPRDiff(owner, repo string, prNumber int) (string, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d.diff", c.BaseURL, owner, repo, prNumber)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	bytesResp, err := c.doRequest(req)
	if err != nil {
		return "", err
	}
	return string(bytesResp), nil
}

func (c *HTTPForgejoClient) GetPRFileContent(owner, repo, ref, filepath string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/raw/%s/%s", c.BaseURL, owner, repo, ref, filepath)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	bytesResp, err := c.doRequest(req)
	if err != nil {
		return "", err
	}
	return string(bytesResp), nil
}

func (c *HTTPForgejoClient) PostPRComment(owner, repo string, prNumber int, body string) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments", c.BaseURL, owner, repo, prNumber)
	payload := map[string]string{"body": body}
	jsonBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	_, err = c.doRequest(req)
	return err
}

type createReviewCommentReq struct {
	Body     string `json:"body"`
	Path     string `json:"path"`
	NewLine  int    `json:"new_position"`
	CommitID string `json:"commit_id"`
}

func (c *HTTPForgejoClient) PostPRInlineComment(owner, repo string, prNumber int, body, commitSHA, filepath string, line int) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d/comments", c.BaseURL, owner, repo, prNumber)
	payload := createReviewCommentReq{
		Body:     body,
		Path:     filepath,
		NewLine:  line,
		CommitID: commitSHA,
	}
	jsonBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	_, err = c.doRequest(req)
	return err
}

func (c *HTTPForgejoClient) GetPRComments(owner, repo string, prNumber int) ([]PRComment, error) {
	// Fetch both issue top-level comments and PR review inline comments
	issueCommentsURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments", c.BaseURL, owner, repo, prNumber)
	prReviewCommentsURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d/comments", c.BaseURL, owner, repo, prNumber)

	var allComments []PRComment

	req, err := http.NewRequest("GET", issueCommentsURL, nil)
	if err == nil {
		if bytesResp, err := c.doRequest(req); err == nil {
			var comments []PRComment
			if json.Unmarshal(bytesResp, &comments) == nil {
				allComments = append(allComments, comments...)
			}
		}
	}

	reqReview, err := http.NewRequest("GET", prReviewCommentsURL, nil)
	if err == nil {
		if bytesResp, err := c.doRequest(reqReview); err == nil {
			var comments []PRComment
			if json.Unmarshal(bytesResp, &comments) == nil {
				allComments = append(allComments, comments...)
			}
		}
	}

	return allComments, nil
}
