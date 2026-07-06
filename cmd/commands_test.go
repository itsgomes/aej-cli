package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/itsgomes/aej-cli/internal/config"
	jiraclient "github.com/itsgomes/aej-cli/internal/jira"
	"github.com/itsgomes/aej-cli/internal/models"
)

type fakeService struct {
	currentUser       *models.User
	openCount         int
	myIssues          []models.Issue
	issue             *models.Issue
	issueErr          error
	search            []models.Issue
	boards            []models.Board
	boardIssues       []models.Issue
	boardIssuesID     int
	boardIssuesCalled bool
	weekly            []models.IssueWorklogSummary
	weeklyTotal       int
}

var _ Service = (*fakeService)(nil)

func (f *fakeService) GetCurrentUserWithStats(context.Context) (*models.User, int, error) {
	return f.currentUser, f.openCount, nil
}

func (f *fakeService) GetMyIssues(context.Context) ([]models.Issue, error) {
	return f.myIssues, nil
}

func (f *fakeService) GetIssue(context.Context, string) (*models.Issue, error) {
	return f.issue, f.issueErr
}

func (f *fakeService) SearchIssues(context.Context, string) ([]models.Issue, error) {
	return f.search, nil
}

func (f *fakeService) GetBoards(context.Context) ([]models.Board, error) {
	return f.boards, nil
}

func (f *fakeService) GetBoardIssues(_ context.Context, boardID int) ([]models.Issue, error) {
	f.boardIssuesCalled = true
	f.boardIssuesID = boardID

	return f.boardIssues, nil
}

func (f *fakeService) AddWorklog(context.Context, string, string, string) error {
	return nil
}

func (f *fakeService) GetWorklogs(context.Context, time.Time, time.Time) ([]models.IssueWorklogSummary, int, error) {
	return f.weekly, f.weeklyTotal, nil
}

type authenticatorFunc func(context.Context) (*models.User, error)

func (f authenticatorFunc) GetCurrentUser(ctx context.Context) (*models.User, error) {
	return f(ctx)
}

func TestMineCommandRendersInjectedIssues(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		myIssues: []models.Issue{
			{
				Key: "AEJ-42",
				Fields: models.IssueFields{
					Summary:  "Desacoplar comandos",
					Status:   models.Status{Name: "Em andamento", StatusCategory: models.StatusCategory{Key: "indeterminate"}},
					Priority: &models.Priority{Name: "High"},
				},
			},
		},
	}

	stdout, _, err := executeForTest(t, testDependencies(service), []string{"mine"}, "")
	if err != nil {
		t.Fatalf("mine error = %v", err)
	}
	if !strings.Contains(stdout, "AEJ-42") || !strings.Contains(stdout, "Desacoplar comandos") {
		t.Errorf("stdout = %q, want injected issue", stdout)
	}
}

func TestIssueCommandClassifiesInjectedNotFoundError(t *testing.T) {
	t.Parallel()

	service := &fakeService{issueErr: jiraclient.ErrNotFound}
	_, _, err := executeForTest(t, testDependencies(service), []string{"issue", "aej-404"}, "")

	if !errors.Is(err, jiraclient.ErrNotFound) {
		t.Fatalf("issue error = %v, want ErrNotFound", err)
	}
	if !strings.Contains(err.Error(), "AEJ-404") {
		t.Errorf("issue error = %q, want normalized issue key", err.Error())
	}
}

func TestLoginCommandUsesInjectedAuthenticatorAndStore(t *testing.T) {
	t.Parallel()

	var saved config.Config
	deps := testDependencies(&fakeService{})
	deps.NewAuthenticator = func(*config.Config) Authenticator {
		return authenticatorFunc(func(context.Context) (*models.User, error) {
			return &models.User{DisplayName: "Ada"}, nil
		})
	}
	deps.SaveConfig = func(cfg config.Config) error {
		saved = cfg
		return nil
	}

	input := "example.atlassian.net\nada@example.com\nsecret-token\n"
	stdout, _, err := executeForTest(t, deps, []string{"login"}, input)
	if err != nil {
		t.Fatalf("login error = %v", err)
	}

	if saved.JiraURL != "https://example.atlassian.net" || saved.Email != "ada@example.com" || saved.APIToken != "secret-token" {
		t.Errorf("saved config = %#v, want normalized input", saved)
	}
	if !strings.Contains(stdout, "Ada") || !strings.Contains(stdout, "Configuração salva") {
		t.Errorf("stdout = %q, want successful greeting", stdout)
	}
}

func TestBoardCommandRendersInjectedBoards(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		boards: []models.Board{
			{ID: 1712, Name: "Produto Principal", Type: "scrum"},
			{ID: 1840, Name: "Sustentação", Type: "kanban"},
		},
	}

	stdout, _, err := executeForTest(t, testDependencies(service), []string{"board"}, "")

	if err != nil {
		t.Fatalf("board error = %v", err)
	}

	if !strings.Contains(stdout, "1712") ||
		!strings.Contains(stdout, "Produto Principal") ||
		!strings.Contains(stdout, "scrum") {
		t.Errorf("stdout = %q, want injected boards", stdout)
	}
}

func TestBoardCommandRendersInjectedIssues(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		boardIssues: []models.Issue{
			{
				Key: "AEJ-42",
				Fields: models.IssueFields{
					Summary: "Implementar comando board",
					Status: models.Status{
						Name: "Em andamento",
						StatusCategory: models.StatusCategory{
							Key: "indeterminate",
						},
					},
					Priority: &models.Priority{
						Name: "High",
					},
				},
			},
		},
	}

	stdout, _, err := executeForTest(t, testDependencies(service), []string{"board", "1712"}, "")

	if err != nil {
		t.Fatalf("board error = %v", err)
	}

	if !service.boardIssuesCalled {
		t.Fatal("GetBoardIssues() was not called")
	}

	if service.boardIssuesID != 1712 {
		t.Errorf("board ID = %d, want 1712", service.boardIssuesID)
	}

	if !strings.Contains(stdout, "AEJ-42") ||
		!strings.Contains(stdout, "Implementar comando board") {
		t.Errorf("stdout = %q, want injected board issue", stdout)
	}
}

func TestBoardCommandRejectsInvalidID(t *testing.T) {
	t.Parallel()

	service := &fakeService{}

	_, _, err := executeForTest(t, testDependencies(service), []string{"board", "abc"}, "")

	if err == nil {
		t.Fatal("board error = nil, want invalid ID error")
	}

	if service.boardIssuesCalled {
		t.Error("GetBoardIssues() was called with an invalid ID")
	}
}

func TestCommandPropagatesConfigErrorWithoutBuildingService(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("configuração indisponível")
	serviceBuilt := false
	deps := testDependencies(&fakeService{})
	deps.LoadConfig = func() (*config.Config, error) {
		return nil, wantErr
	}
	deps.NewService = func(*config.Config) Service {
		serviceBuilt = true
		return &fakeService{}
	}

	_, _, err := executeForTest(t, deps, []string{"me"}, "")
	if !errors.Is(err, wantErr) {
		t.Fatalf("me error = %v, want config error", err)
	}
	if serviceBuilt {
		t.Error("service was built after configuration failure")
	}
}

func testDependencies(service Service) Dependencies {
	return Dependencies{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{JiraURL: "https://example.atlassian.net", Email: "dev@example.com", APIToken: "token"}, nil
		},
		SaveConfig: func(config.Config) error { return nil },
		NewService: func(*config.Config) Service {
			return service
		},
		NewAuthenticator: func(*config.Config) Authenticator {
			return authenticatorFunc(func(context.Context) (*models.User, error) {
				return &models.User{}, nil
			})
		},
	}
}

func executeForTest(t *testing.T, deps Dependencies, args []string, input string) (string, string, error) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRootCommand(deps)
	root.SetIn(strings.NewReader(input))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)

	err := root.ExecuteContext(context.Background())
	return stdout.String(), stderr.String(), err
}
