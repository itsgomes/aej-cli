package services

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itsgomes/aej-cli/internal/models"
)

type fakeJiraGateway struct {
	currentUser    *models.User
	count          int
	searchIssues   []models.Issue
	searchJQL      string
	searchLimit    int
	issue          *models.Issue
	issueKey       string
	boards         []models.Board
	activeSprints  map[int]*models.Sprint
	sprintIssues   []models.Issue
	issueWorklogs  map[string][]models.Worklog
	getWorklogs    func(context.Context, string) ([]models.Worklog, error)
	worklogIssue   string
	worklogTime    string
	worklogComment string
	worklogStarted string
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

func (f *fakeJiraGateway) GetActiveSprint(_ context.Context, boardID int) (*models.Sprint, error) {
	return f.activeSprints[boardID], nil
}

func (f *fakeJiraGateway) GetSprintIssues(context.Context, int) ([]models.Issue, error) {
	return f.sprintIssues, nil
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

func TestJiraServiceGetActiveSprintCalculatesStats(t *testing.T) {
	t.Parallel()

	gateway := &fakeJiraGateway{
		boards: []models.Board{{ID: 7, Name: "Engineering"}},
		activeSprints: map[int]*models.Sprint{
			7: {ID: 42, Name: "Sprint 42", State: "active"},
		},
		sprintIssues: []models.Issue{
			{Key: "AEJ-1", Fields: issueFieldsWithStatusCategory("done")},
			{Key: "AEJ-2", Fields: issueFieldsWithStatusCategory("indeterminate")},
			{Key: "AEJ-3", Fields: issueFieldsWithStatusCategory("new")},
			{Key: "AEJ-4", Fields: issueFieldsWithStatusCategory("new")},
		},
	}

	service := New(gateway)
	stats, err := service.GetActiveSprint(context.Background())
	if err != nil {
		t.Fatalf("GetActiveSprint() error = %v", err)
	}

	if stats.Total != 4 || stats.Done != 1 || stats.InProgress != 1 || stats.Todo != 2 {
		t.Errorf("stats = %#v, want total=4 done=1 inProgress=1 todo=2", stats)
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

	service := New(gateway, WithClock(func() time.Time { return fixedNow }))
	summaries, totalSeconds, err := service.GetWeeklyWorklogs(context.Background())
	if err != nil {
		t.Fatalf("GetWeeklyWorklogs() error = %v", err)
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

	service := New(gateway, WithClock(func() time.Time {
		return time.Date(2026, time.July, 2, 12, 0, 0, 0, time.UTC)
	}))
	summaries, _, err := service.GetWeeklyWorklogs(context.Background())
	if err != nil {
		t.Fatalf("GetWeeklyWorklogs() error = %v", err)
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

func issueFieldsWithStatusCategory(category string) models.IssueFields {
	return models.IssueFields{
		Status: models.Status{
			StatusCategory: models.StatusCategory{Key: category},
		},
	}
}
