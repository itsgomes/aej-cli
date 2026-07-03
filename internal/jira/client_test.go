package jira

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itsgomes/aej-cli/internal/config"
)

func TestClientGetCurrentUser(t *testing.T) {
	t.Parallel()

	var gotMethod string
	var gotPath string
	var gotEmail string
	var gotToken string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotEmail, gotToken, _ = r.BasicAuth()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accountId":"account-1","displayName":"Ada Lovelace","emailAddress":"ada@example.com","active":true}`))
	}))
	t.Cleanup(server.Close)

	client := New(&config.Config{
		JiraURL:  server.URL,
		Email:    "ada@example.com",
		APIToken: "secret-token",
	})

	user, err := client.GetCurrentUser(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentUser() error = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodGet)
	}
	if gotPath != "/rest/api/3/myself" {
		t.Errorf("path = %q, want %q", gotPath, "/rest/api/3/myself")
	}
	if gotEmail != "ada@example.com" || gotToken != "secret-token" {
		t.Errorf("basic auth = (%q, %q), want configured credentials", gotEmail, gotToken)
	}
	if user.AccountID != "account-1" || user.DisplayName != "Ada Lovelace" {
		t.Errorf("user = %#v, want decoded Jira user", user)
	}
}

func TestClientEscapesIssueKeyPathSegment(t *testing.T) {
	t.Parallel()

	var requestURI string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURI = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"key":"AEJ-1","fields":{}}`))
	}))
	t.Cleanup(server.Close)

	client := New(&config.Config{JiraURL: server.URL})
	_, err := client.GetIssue(context.Background(), "AEJ-1/../../myself")
	if err != nil {
		t.Fatalf("GetIssue() error = %v", err)
	}

	if strings.Contains(requestURI, "/../") || !strings.Contains(requestURI, "%2F") {
		t.Errorf("RequestURI = %q, want escaped single path segment", requestURI)
	}
}

func TestClientHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(server.Close)

	client := New(&config.Config{JiraURL: server.URL})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetCurrentUser(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetCurrentUser() error = %v, want context.Canceled", err)
	}
}

func TestClientClassifiesAPIError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		target     error
	}{
		{name: "bad request", statusCode: http.StatusBadRequest, target: ErrBadRequest},
		{name: "unauthorized", statusCode: http.StatusUnauthorized, target: ErrUnauthorized},
		{name: "not found", statusCode: http.StatusNotFound, target: ErrNotFound},
		{name: "rate limited", statusCode: http.StatusTooManyRequests, target: ErrRateLimited},
		{name: "unavailable", statusCode: http.StatusServiceUnavailable, target: ErrUnavailable},
		{name: "unexpected", statusCode: http.StatusTeapot, target: ErrUnexpected},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			t.Cleanup(server.Close)

			client := New(&config.Config{JiraURL: server.URL})
			_, err := client.GetCurrentUser(context.Background())

			if !errors.Is(err, tt.target) {
				t.Fatalf("GetCurrentUser() error = %v, want errors.Is(%v)", err, tt.target)
			}

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("GetCurrentUser() error type = %T, want *APIError", err)
			}
			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.statusCode)
			}
		})
	}
}

func TestClientRequestTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)

	client := New(
		&config.Config{JiraURL: server.URL},
		WithTimeout(20*time.Millisecond),
	)

	_, err := client.GetCurrentUser(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("GetCurrentUser() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestClientDoesNotFollowRedirects(t *testing.T) {
	t.Parallel()

	followed := make(chan struct{}, 1)
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		followed <- struct{}{}
	}))
	t.Cleanup(target.Close)

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	t.Cleanup(redirect.Close)

	client := New(&config.Config{JiraURL: redirect.URL})
	_, err := client.GetCurrentUser(context.Background())
	if err == nil {
		t.Fatal("GetCurrentUser() error = nil, want redirect rejection")
	}

	select {
	case <-followed:
		t.Fatal("client followed redirect and could have forwarded credentials")
	default:
	}
}

func TestReadRetriesTransientServiceFailure(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accountId":"account-1","displayName":"Ada"}`))
	}))
	t.Cleanup(server.Close)

	client := newFastRetryClient(server.URL)
	user, err := client.GetCurrentUser(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentUser() error = %v", err)
	}
	if user.AccountID != "account-1" {
		t.Errorf("user = %#v, want successful retried response", user)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("request count = %d, want 2", got)
	}
}

func TestReadOnlyPostRetriesRateLimitWithRetryAfter(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":7}`))
	}))
	t.Cleanup(server.Close)

	client := newFastRetryClient(server.URL)
	count, err := client.CountIssues(context.Background(), "project = AEJ")
	if err != nil {
		t.Fatalf("CountIssues() error = %v", err)
	}
	if count != 7 || calls.Load() != 2 {
		t.Errorf("count/calls = (%d, %d), want 7/2", count, calls.Load())
	}
}

func TestRateLimitWithoutRetryAfterIsNotRetried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)

	client := newFastRetryClient(server.URL)
	_, err := client.GetCurrentUser(context.Background())
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("GetCurrentUser() error = %v, want ErrRateLimited", err)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("request count = %d, want 1", got)
	}
}

func TestAddWorklogDoesNotRetry(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	client := newFastRetryClient(server.URL)
	err := client.AddWorklog(context.Background(), "AEJ-1", "1h", "", "2026-07-02T12:00:00.000+0000")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("AddWorklog() error = %v, want ErrUnavailable", err)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("request count = %d, want 1; write must not be retried", got)
	}
}

func TestBuildADFComment(t *testing.T) {
	t.Parallel()

	document := buildADFComment("Implementação concluída")
	if document.Type != "doc" || document.Version != 1 || len(document.Content) != 1 {
		t.Fatalf("document = %#v, want ADF document", document)
	}
	paragraph := document.Content[0]
	if paragraph.Type != "paragraph" || len(paragraph.Content) != 1 || paragraph.Content[0].Text != "Implementação concluída" {
		t.Errorf("paragraph = %#v, want typed text paragraph", paragraph)
	}
}

func newFastRetryClient(baseURL string) *Client {
	return New(
		&config.Config{JiraURL: baseURL},
		Option(func(client *Client) {
			client.http.
				SetRetryCount(1).
				SetRetryWaitTime(time.Millisecond).
				SetRetryMaxWaitTime(5 * time.Millisecond)
		}),
	)
}

func TestClientSearchIssuesFollowsNextPageToken(t *testing.T) {
	t.Parallel()

	var requests []searchRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request searchRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		requests = append(requests, request)
		w.Header().Set("Content-Type", "application/json")

		switch request.NextPageToken {
		case "":
			_, _ = w.Write([]byte(`{
				"issues":[{"key":"AEJ-1","fields":{}},{"key":"AEJ-2","fields":{}}],
				"nextPageToken":"page-2",
				"isLast":false
			}`))
		case "page-2":
			_, _ = w.Write([]byte(`{
				"issues":[{"key":"AEJ-3","fields":{}}],
				"isLast":true
			}`))
		default:
			http.Error(w, "unexpected page token", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	client := New(&config.Config{JiraURL: server.URL})
	issues, err := client.SearchIssues(
		context.Background(),
		"project = AEJ ORDER BY updated DESC",
		[]string{"summary", "status"},
		3,
	)
	if err != nil {
		t.Fatalf("SearchIssues() error = %v", err)
	}

	if len(issues) != 3 {
		t.Fatalf("len(issues) = %d, want 3", len(issues))
	}
	if issues[0].Key != "AEJ-1" || issues[2].Key != "AEJ-3" {
		t.Errorf("issue keys = %q, %q, want AEJ-1, AEJ-3", issues[0].Key, issues[2].Key)
	}
	if len(requests) != 2 {
		t.Fatalf("request count = %d, want 2", len(requests))
	}
	if requests[0].NextPageToken != "" || requests[1].NextPageToken != "page-2" {
		t.Errorf("page tokens = %q, %q, want empty, page-2", requests[0].NextPageToken, requests[1].NextPageToken)
	}
	if requests[0].MaxResults != 3 || requests[1].MaxResults != 1 {
		t.Errorf("page sizes = %d, %d, want 3, 1", requests[0].MaxResults, requests[1].MaxResults)
	}
}

func TestClientCountIssuesUsesApproximateCountEndpoint(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotJQL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		gotJQL = body["jql"]

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":153}`))
	}))
	t.Cleanup(server.Close)

	client := New(&config.Config{JiraURL: server.URL})
	count, err := client.CountIssues(context.Background(), "project = AEJ")
	if err != nil {
		t.Fatalf("CountIssues() error = %v", err)
	}

	if gotPath != "/rest/api/3/search/approximate-count" {
		t.Errorf("path = %q, want approximate count endpoint", gotPath)
	}
	if gotJQL != "project = AEJ" {
		t.Errorf("jql = %q, want %q", gotJQL, "project = AEJ")
	}
	if count != 153 {
		t.Errorf("count = %d, want 153", count)
	}
}

func TestClientGetBoardsFollowsOffsetPagination(t *testing.T) {
	t.Parallel()

	var startOffsets []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startAt := r.URL.Query().Get("startAt")
		startOffsets = append(startOffsets, startAt)
		w.Header().Set("Content-Type", "application/json")

		switch startAt {
		case "0":
			_, _ = w.Write([]byte(`{
				"values":[{"id":1,"name":"Board 1"},{"id":2,"name":"Board 2"}],
				"startAt":0,"maxResults":2,"total":3,"isLast":false
			}`))
		case "2":
			_, _ = w.Write([]byte(`{
				"values":[{"id":3,"name":"Board 3"}],
				"startAt":2,"maxResults":2,"total":3,"isLast":true
			}`))
		default:
			http.Error(w, "unexpected startAt", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	client := New(&config.Config{JiraURL: server.URL})
	boards, err := client.GetBoards(context.Background())
	if err != nil {
		t.Fatalf("GetBoards() error = %v", err)
	}

	if len(boards) != 3 {
		t.Fatalf("len(boards) = %d, want 3", len(boards))
	}
	if boards[0].ID != 1 || boards[2].ID != 3 {
		t.Errorf("board IDs = %d, %d, want 1, 3", boards[0].ID, boards[2].ID)
	}
	if len(startOffsets) != 2 || startOffsets[0] != "0" || startOffsets[1] != "2" {
		t.Errorf("startAt values = %v, want [0 2]", startOffsets)
	}
}

func TestClientGetBoardIssuesFollowsTokenPagination(t *testing.T) {
	t.Parallel()

	var gotPaths []string
	var gotTokens []string
	var gotMaxResults []string
	var gotJQL string
	var gotFields string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		token := query.Get("nextPageToken")

		gotPaths = append(gotPaths, r.URL.Path)
		gotTokens = append(gotTokens, token)
		gotMaxResults = append(gotMaxResults, query.Get("maxResults"))
		gotJQL = query.Get("jql")
		gotFields = query.Get("fields")

		w.Header().Set("Content-Type", "application/json")

		switch token {
		case "":
			_, _ = w.Write([]byte(`{
				"issues": [
					{"key": "AEJ-1", "fields": {"summary": "Primeira issue"}},
					{"key": "AEJ-2", "fields": {"summary": "Segunda issue"}}
				],
				"nextPageToken": "page-2",
				"isLast": false
			}`))

		case "page-2":
			_, _ = w.Write([]byte(`{
				"issues": [
					{"key": "AEJ-3", "fields": {"summary": "Terceira issue"}}
				],
				"isLast": true
			}`))

		default:
			http.Error(w, "unexpected token", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	client := New(&config.Config{JiraURL: server.URL})

	issues, err := client.GetBoardIssues(
		context.Background(),
		1712,
		"statusCategory != Done ORDER BY updated DESC",
		[]string{"summary", "status", "priority", "issuetype"},
		3,
	)

	if err != nil {
		t.Fatalf("GetBoardIssues() error = %v", err)
	}

	if len(issues) != 3 {
		t.Fatalf("len(issues) = %d, want 3", len(issues))
	}

	if issues[0].Key != "AEJ-1" || issues[2].Key != "AEJ-3" {
		t.Errorf(
			"issue keys = %q, %q, want AEJ-1, AEJ-3",
			issues[0].Key,
			issues[2].Key,
		)
	}

	if len(gotPaths) != 2 ||
		gotPaths[0] != "/rest/software/1.0/board/1712/issue" {
		t.Errorf("paths = %v, want board issues endpoint", gotPaths)
	}

	if len(gotTokens) != 2 ||
		gotTokens[0] != "" ||
		gotTokens[1] != "page-2" {
		t.Errorf("tokens = %v, want [\"\" \"page-2\"]", gotTokens)
	}

	if len(gotMaxResults) != 2 ||
		gotMaxResults[0] != "3" ||
		gotMaxResults[1] != "1" {
		t.Errorf("maxResults = %v, want [3 1]", gotMaxResults)
	}

	if gotJQL != "statusCategory != Done ORDER BY updated DESC" {
		t.Errorf("jql = %q, want open issues ordered by update", gotJQL)
	}

	if gotFields != "summary,status,priority,issuetype" {
		t.Errorf("fields = %q, want requested issue fields", gotFields)
	}
}

func TestClientGetIssueWorklogsFollowsOffsetPagination(t *testing.T) {
	t.Parallel()

	var startOffsets []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startAt := r.URL.Query().Get("startAt")
		startOffsets = append(startOffsets, startAt)
		w.Header().Set("Content-Type", "application/json")

		switch startAt {
		case "0":
			_, _ = w.Write([]byte(`{
				"worklogs":[{"id":"1"},{"id":"2"}],
				"startAt":0,"maxResults":2,"total":3
			}`))
		case "2":
			_, _ = w.Write([]byte(`{
				"worklogs":[{"id":"3"}],
				"startAt":2,"maxResults":2,"total":3
			}`))
		default:
			http.Error(w, "unexpected startAt", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	client := New(&config.Config{JiraURL: server.URL})
	worklogs, err := client.GetIssueWorklogs(context.Background(), "AEJ-1")
	if err != nil {
		t.Fatalf("GetIssueWorklogs() error = %v", err)
	}

	if len(worklogs) != 3 {
		t.Fatalf("len(worklogs) = %d, want 3", len(worklogs))
	}
	if worklogs[0].ID != "1" || worklogs[2].ID != "3" {
		t.Errorf("worklog IDs = %q, %q, want 1, 3", worklogs[0].ID, worklogs[2].ID)
	}
	if len(startOffsets) != 2 || startOffsets[0] != "0" || startOffsets[1] != "2" {
		t.Errorf("startAt values = %v, want [0 2]", startOffsets)
	}
}

func TestClientGetSprintIssuesFollowsNextPageToken(t *testing.T) {
	t.Parallel()

	var pageTokens []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("nextPageToken")
		pageTokens = append(pageTokens, token)
		w.Header().Set("Content-Type", "application/json")

		switch token {
		case "":
			_, _ = w.Write([]byte(`{
				"issues":[{"key":"AEJ-1","fields":{}},{"key":"AEJ-2","fields":{}}],
				"nextPageToken":"page-2","isLast":false
			}`))
		case "page-2":
			_, _ = w.Write([]byte(`{
				"issues":[{"key":"AEJ-3","fields":{}}],
				"isLast":true
			}`))
		default:
			http.Error(w, "unexpected page token", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	client := New(&config.Config{JiraURL: server.URL})
	issues, err := client.GetSprintIssues(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetSprintIssues() error = %v", err)
	}

	if len(issues) != 3 {
		t.Fatalf("len(issues) = %d, want 3", len(issues))
	}
	if issues[0].Key != "AEJ-1" || issues[2].Key != "AEJ-3" {
		t.Errorf("issue keys = %q, %q, want AEJ-1, AEJ-3", issues[0].Key, issues[2].Key)
	}
	if len(pageTokens) != 2 || pageTokens[0] != "" || pageTokens[1] != "page-2" {
		t.Errorf("page tokens = %v, want [empty page-2]", pageTokens)
	}
}
