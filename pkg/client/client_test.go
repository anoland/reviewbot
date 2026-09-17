package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anoland/reviewbot/pkg/client"
	"github.com/anoland/reviewbot/pkg/models"
)

func TestForgejoClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/repos/owner/repo/pulls/1.diff":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-old\n+new"))
		case "/api/v1/repos/owner/repo/raw/main/main.go":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("package main\n\nfunc main() {}\n"))
		case "/api/v1/repos/owner/repo/issues/1/comments":
			if r.Method == "GET" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[{"id":101,"body":"Existing comment","user":{"username":"testbot"}}]`))
			} else if r.Method == "POST" {
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"id":102,"body":"Posted"}`))
			}
		case "/api/v1/repos/owner/repo/pulls/1/comments":
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":103,"body":"Inline comment"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cli := client.NewForgejoClient(server.URL, "dummy-token")

	diff, err := cli.GetPRDiff("owner", "repo", 1)
	if err != nil {
		t.Fatalf("GetPRDiff failed: %v", err)
	}
	if len(diff) == 0 {
		t.Errorf("expected non-empty diff")
	}

	content, err := cli.GetPRFileContent("owner", "repo", "main", "main.go")
	if err != nil {
		t.Fatalf("GetPRFileContent failed: %v", err)
	}
	if len(content) == 0 {
		t.Errorf("expected non-empty file content")
	}

	err = cli.PostPRComment("owner", "repo", 1, "New PR summary comment")
	if err != nil {
		t.Fatalf("PostPRComment failed: %v", err)
	}

	err = cli.PostPRInlineComment("owner", "repo", 1, "Check line 10", "sha123", "main.go", 10)
	if err != nil {
		t.Fatalf("PostPRInlineComment failed: %v", err)
	}

	comments, err := cli.GetPRComments("owner", "repo", 1)
	if err != nil {
		t.Fatalf("GetPRComments failed: %v", err)
	}
	if len(comments) != 1 || comments[0].ID != 101 {
		t.Errorf("unexpected comments result: %+v", comments)
	}
}

func TestMockGeminiClient(t *testing.T) {
	expectedResult := &models.EvaluationResult{
		OverallRating:  4,
		RatingCategory: "Level 4: Regressive & Security-Critical",
		Summary:        "Excellent security tests",
	}

	gemini := &client.MockGeminiClient{
		ResultToReturn: expectedResult,
	}

	res, err := gemini.EvaluateChanges(context.Background(), "sys prompt", "user prompt")
	if err != nil {
		t.Fatalf("EvaluateChanges failed: %v", err)
	}

	if res.OverallRating != 4 {
		t.Errorf("expected overall rating 4, got %d", res.OverallRating)
	}
}
