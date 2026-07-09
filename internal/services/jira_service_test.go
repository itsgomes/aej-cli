package services

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itsgomes/aej-cli/internal/models"
)

type fakeJiraGateway struct {
	currentUser       *models.User
	count             int
	searchIssues      []models.Issue
	searchJQL         string
	searchLimit       int
	issue             *models.Issue
	issueKey          string
	boards            []models.Board
	boardIssues       []models.Issue
	boardIssuesErr    error
	boardIssuesID     int
	boardIssuesJQL    string
	boardIssuesFields []string
	boardIssuesLimit  int
	issueWorklogs     map[string][]models.Worklog
	getWorklogs       func(context.Context, string) ([]models.Worklog, error)
	worklogIssue      string
	worklogTime       string
	worklogComment    string
	worklogStarted    string
}

var _ JiraGateway = (*fakeJiraGateway)(nil)

func (f *fakeJiraGateway) GetCurrentUser(context.Context) (*models.User, error) {
	return f.currentUser, nil
}

func (f *fakeJiraGateway) CountIssues(context.Context, string) (int, error) {
	return f.count, nil
}

func (f *fakeJiraGateway) SearchIssues(_ context.Context, jql string, _ []string, limit int) ([]models.Issue, error) {
	f.searchJQL = jql
	f.searchLimit = limit
	return f.searchIssues, nil
}

func (f *fakeJiraGateway) GetIssue(_ context.Context, issueKey string) (*models.Issue, error) {
	f.issueKey = issueKey
	return f.issue, nil
}

func (f *fakeJiraGateway) GetBoards(context.Context) ([]models.Board, error) {
	return f.boards, nil
}

func (f *fakeJiraGateway) GetBoardIssues(_ context.Context, boardID int, jql string, fields []string, limit int) ([]models.Issue, error) {
	f.boardIssuesID = boardID
	f.boardIssuesJQL = jql
	f.boardIssuesFields = fields
	f.boardIssuesLimit = limit

	return f.boardIssues, f.boardIssuesErr
}

func (f *fakeJiraGateway) AddWorklog(_ context.Context, issueKey, timeSpent, comment, started string) error {
	f.worklogIssue = issueKey
	f.worklogTime = timeSpent
	f.worklogComment = comment
	f.worklogStarted = started
	return nil
}

func (f *fakeJiraGateway) GetIssueWorklogs(ctx context.Context, issueKey string) ([]models.Worklog, error) {
	if f.getWorklogs != nil {
		return f.getWorklogs(ctx, issueKey)
	}
	return f.issueWorklogs[issueKey], nil
}

func TestJiraServiceGetBoardIssuesAppliesFilterAndLimit(t *testing.T) {
	t.Parallel()

	gateway := &fakeJiraGateway{
		boardIssues: []models.Issue{
			{Key: "AEJ-10"},
			{Key: "AEJ-20"},
		},
	}

	service := New(gateway)

	issues, err := service.GetBoardIssues(context.Background(), 1712)

	if err != nil {
		t.Fatalf("GetBoardIssues() error = %v", err)
	}

	if len(issues) != 2 {
		t.Fatalf("len(issues) = %d, want 2", len(issues))
	}

	if gateway.boardIssuesID != 1712 {
		t.Errorf("board ID = %d, want 1712", gateway.boardIssuesID)
	}

	wantJQL := "statusCategory != Done ORDER BY updated DESC"

	if gateway.boardIssuesJQL != wantJQL {
		t.Errorf("jql = %q, want %q", gateway.boardIssuesJQL, wantJQL)
	}

	wantFields := []string{"summary", "status", "priority", "issuetype"}

	if !slices.Equal(gateway.boardIssuesFields, wantFields) {
		t.Errorf("fields = %v, want %v", gateway.boardIssuesFields, wantFields)
	}

	if gateway.boardIssuesLimit != 50 {
		t.Errorf("limit = %d, want 50", gateway.boardIssuesLimit)
	}
}

func TestJiraServiceGetMyIssuesUsesOpenIssuesByDefault(t *testing.T) {
	t.Parallel()

	gateway := &fakeJiraGateway{searchIssues: []models.Issue{{Key: "AEJ-10"}}}

	service := New(gateway)

	issues, err := service.GetMyIssues(context.Background(), "")

	if err != nil {
		t.Fatalf("GetMyIssues() error = %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("len(issues) = %d, want 1", len(issues))
	}

	wantJQL := "assignee = currentUser() AND statusCategory != Done ORDER BY updated DESC"

	if gateway.searchJQL != wantJQL {
		t.Errorf("jql = %q, want %q", gateway.searchJQL, wantJQL)
	}

	if gateway.searchLimit != 50 {
		t.Errorf("limit = %d, want 50", gateway.searchLimit)
	}
}

func TestJiraServiceGetMyIssuesFiltersByStatus(t *testing.T) {
	t.Parallel()

	gateway := &fakeJiraGateway{
		searchIssues: []models.Issue{
			{
				Key: "FD-4421",
				Fields: models.IssueFields{
					Status: models.Status{Name: "Dev. Aguardando Deploy"},
				},
			},
			{
				Key: "FD-4422",
				Fields: models.IssueFields{
					Status: models.Status{Name: "Dev. Em andamento"},
				},
			},
		},
	}
	service := New(gateway)

	issues, err := service.GetMyIssues(context.Background(), "deploy")

	if err != nil {
		t.Fatalf("GetMyIssues() error = %v", err)
	}

	wantJQL := "assignee = currentUser() ORDER BY updated DESC"

	if gateway.searchJQL != wantJQL {
		t.Errorf("jql = %q, want %q", gateway.searchJQL, wantJQL)
	}

	if gateway.searchLimit != 0 {
		t.Errorf("limit = %d, want 0 to allow client-side status filtering", gateway.searchLimit)
	}

	if len(issues) != 1 || issues[0].Key != "FD-4421" {
		t.Errorf("issues = %#v, want only FD-4421", issues)
	}
}

func TestJiraServiceGetBoardIssuesRejectsInvalidID(t *testing.T) {
	t.Parallel()

	service := New(&fakeJiraGateway{})

	_, err := service.GetBoardIssues(context.Background(), 0)

	if err == nil {
		t.Fatal("GetBoardIssues() error = nil, want invalid board ID error")
	}
}

func TestJiraServiceAddWorklogUsesInjectedClock(t *testing.T) {
	t.Parallel()

	gateway := &fakeJiraGateway{}
	fixedNow := time.Date(2026, time.July, 2, 14, 30, 45, 0, time.FixedZone("BRT", -3*60*60))
	service := New(gateway, WithClock(func() time.Time { return fixedNow }))

	err := service.AddWorklog(context.Background(), "aej-123", "1h 30m", "Implementação")
	if err != nil {
		t.Fatalf("AddWorklog() error = %v", err)
	}

	if gateway.worklogIssue != "AEJ-123" {
		t.Errorf("issue key = %q, want AEJ-123", gateway.worklogIssue)
	}
	if gateway.worklogTime != "1h 30m" || gateway.worklogComment != "Implementação" {
		t.Errorf("worklog = (%q, %q), want provided time and comment", gateway.worklogTime, gateway.worklogComment)
	}
	if gateway.worklogStarted != "2026-07-02T14:30:45.000-0300" {
		t.Errorf("started = %q, want fixed Jira timestamp", gateway.worklogStarted)
	}
}

func TestJiraServiceGetWeeklyWorklogsFiltersAuthorAndDate(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, time.July, 2, 12, 0, 0, 0, time.UTC)
	gateway := &fakeJiraGateway{
		currentUser: &models.User{AccountID: "me"},
		searchIssues: []models.Issue{
			{Key: "AEJ-1", Fields: models.IssueFields{Summary: "Paginação"}},
		},
		issueWorklogs: map[string][]models.Worklog{
			"AEJ-1": {
				{ID: "recent", Author: models.User{AccountID: "me"}, Started: "2026-06-30T12:00:00Z", TimeSpent: "1h", TimeSpentSeconds: 3600},
				{ID: "other-author", Author: models.User{AccountID: "other"}, Started: "2026-07-01T12:00:00Z", TimeSpentSeconds: 7200},
				{ID: "old", Author: models.User{AccountID: "me"}, Started: "2026-06-20T12:00:00Z", TimeSpentSeconds: 1800},
			},
		},
	}

	from := time.Date(2026, time.June, 26, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.July, 3, 0, 0, 0, 0, time.UTC)

	service := New(gateway, WithClock(func() time.Time { return fixedNow }))
	summaries, totalSeconds, err := service.GetWorklogs(context.Background(), from, to)
	if err != nil {
		t.Fatalf("GetWorklogs() error = %v", err)
	}

	if gateway.searchLimit != 0 {
		t.Errorf("search limit = %d, want 0 for all matching issues", gateway.searchLimit)
	}
	if len(summaries) != 1 || len(summaries[0].Entries) != 1 {
		t.Fatalf("summaries = %#v, want one issue with one entry", summaries)
	}
	if summaries[0].Entries[0].ID != "recent" || totalSeconds != 3600 {
		t.Errorf("entry/total = (%q, %d), want recent/3600", summaries[0].Entries[0].ID, totalSeconds)
	}
}

func TestJiraServiceLimitsConcurrentWorklogRequests(t *testing.T) {
	t.Parallel()

	var active atomic.Int32
	var maximum atomic.Int32
	issues := make([]models.Issue, 8)
	for index := range issues {
		issues[index] = models.Issue{Key: fmt.Sprintf("AEJ-%d", index+1), Fields: models.IssueFields{Summary: "Issue"}}
	}

	gateway := &fakeJiraGateway{
		currentUser:  &models.User{AccountID: "me"},
		searchIssues: issues,
		getWorklogs: func(_ context.Context, issueKey string) ([]models.Worklog, error) {
			current := active.Add(1)
			for {
				observed := maximum.Load()
				if current <= observed || maximum.CompareAndSwap(observed, current) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			active.Add(-1)
			return []models.Worklog{{ID: issueKey, Author: models.User{AccountID: "me"}, Started: "2026-07-01T12:00:00Z", TimeSpentSeconds: 60}}, nil
		},
	}

	from := time.Date(2026, time.June, 26, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.July, 3, 0, 0, 0, 0, time.UTC)

	service := New(gateway, WithClock(func() time.Time {
		return time.Date(2026, time.July, 2, 12, 0, 0, 0, time.UTC)
	}))

	summaries, _, err := service.GetWorklogs(context.Background(), from, to)

	if err != nil {
		t.Fatalf("GetWorklogs() error = %v", err)
	}

	if got := maximum.Load(); got < 2 || got > maxConcurrentWorklogRequests {
		t.Errorf("maximum concurrency = %d, want between 2 and %d", got, maxConcurrentWorklogRequests)
	}

	if len(summaries) != len(issues) || summaries[0].IssueKey != "AEJ-1" || summaries[7].IssueKey != "AEJ-8" {
		t.Errorf("summary order/count = %#v, want original issue order", summaries)
	}
}

func TestJiraServiceSearchIssuesEscapesJQLLiteral(t *testing.T) {
	t.Parallel()

	gateway := &fakeJiraGateway{}
	service := New(gateway)

	_, err := service.SearchIssues(context.Background(), `C:\temp" OR assignee = currentUser()`)
	if err != nil {
		t.Fatalf("SearchIssues() error = %v", err)
	}

	want := `text ~ "C:\\temp\" OR assignee = currentUser()" ORDER BY updated DESC`
	if gateway.searchJQL != want {
		t.Errorf("JQL = %q, want %q", gateway.searchJQL, want)
	}
}

func TestJiraServiceValidatesIssueKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "normalizes case", input: " aej-42 ", want: "AEJ-42"},
		{name: "allows underscore", input: "TEAM_CORE-1", want: "TEAM_CORE-1"},
		{name: "rejects path", input: "AEJ-1/../../myself", wantErr: true},
		{name: "rejects missing number", input: "AEJ", wantErr: true},
		{name: "rejects zero", input: "AEJ-0", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gateway := &fakeJiraGateway{issue: &models.Issue{}}
			service := New(gateway)
			_, err := service.GetIssue(context.Background(), tt.input)

			if tt.wantErr {
				if !errors.Is(err, ErrInvalidIssueKey) {
					t.Fatalf("GetIssue() error = %v, want ErrInvalidIssueKey", err)
				}
				if gateway.issueKey != "" {
					t.Errorf("gateway received invalid key %q", gateway.issueKey)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetIssue() error = %v", err)
			}
			if gateway.issueKey != tt.want {
				t.Errorf("gateway key = %q, want %q", gateway.issueKey, tt.want)
			}
		})
	}
}
