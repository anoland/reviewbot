package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"forgejo-test-evaluator/pkg/models"
)

type GeminiClient interface {
	EvaluateChanges(ctx context.Context, systemPrompt string, userPrompt string) (*models.EvaluationResult, error)
}

type DefaultGeminiClient struct {
	APIKey     string
	ModelName  string
	HTTPClient *http.Client
}

func NewGeminiClient(apiKey string) *DefaultGeminiClient {
	return &DefaultGeminiClient{
		APIKey:     apiKey,
		ModelName:  "gemini-2.5-pro",
		HTTPClient: &http.Client{},
	}
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiReq struct {
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  map[string]any `json:"generation_config,omitempty"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

type geminiResp struct {
	Candidates []geminiCandidate `json:"candidates"`
}

func (g *DefaultGeminiClient) EvaluateChanges(ctx context.Context, systemPrompt string, userPrompt string) (*models.EvaluationResult, error) {
	if g.APIKey == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.ModelName, g.APIKey)

	reqPayload := geminiReq{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		},
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: userPrompt}},
			},
		},
		GenerationConfig: map[string]any{
			"response_mime_type": "application/json",
		},
	}

	jsonBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Gemini request payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Gemini API call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gemini API response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, string(body))
	}

	var gResp geminiResp
	if err := json.Unmarshal(body, &gResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Gemini API response: %w", err)
	}

	if len(gResp.Candidates) == 0 || len(gResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("received empty candidates response from Gemini API")
	}

	rawText := gResp.Candidates[0].Content.Parts[0].Text
	rawText = strings.TrimPrefix(rawText, "```json")
	rawText = strings.TrimPrefix(rawText, "```")
	rawText = strings.TrimSuffix(rawText, "```")
	rawText = strings.TrimSpace(rawText)

	return ParseEvaluationResponse(rawText)
}

// MockGeminiClient provides pre-configured response fixture for tests.
type MockGeminiClient struct {
	ResultToReturn *models.EvaluationResult
	ErrToReturn    error
}

func (m *MockGeminiClient) EvaluateChanges(ctx context.Context, systemPrompt string, userPrompt string) (*models.EvaluationResult, error) {
	if m.ErrToReturn != nil {
		return nil, m.ErrToReturn
	}
	if m.ResultToReturn != nil {
		return m.ResultToReturn, nil
	}
	return &models.EvaluationResult{
		OverallRating:  3,
		RatingCategory: "Level 3: Robust Functional",
		Summary:        "Mock evaluation complete.",
		SecurityCoverage: models.SecurityCoverage{
			SensitiveAreasIdentified: []string{"Input Validation"},
			GapsFound:                []string{},
		},
		ActionItems: []models.ActionItem{},
	}, nil
}

// ParseEvaluationResponse parses raw JSON string into EvaluationResult.
func ParseEvaluationResponse(rawJSON string) (*models.EvaluationResult, error) {
	var result models.EvaluationResult
	if err := json.Unmarshal([]byte(rawJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse evaluation response: %w", err)
	}
	return &result, nil
}
