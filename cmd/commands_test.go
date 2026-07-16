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
	myIssuesStatus    string
	searchQuery       string
	searchTag         string
	searchVersion     string
	transitions       []models.Transition
	transitionIssue   string
	transitionID      string
	assignIssue       string
	assignTarget      string
	assignUser        *models.User
	commentIssue      string
	commentText       string
}

var _ Service = (*fakeService)(nil)

func (f *fakeService) GetCurrentUserWithStats(context.Context) (*models.User, int, error) {
	return f.currentUser, f.openCount, nil
}

func (f *fakeService) GetMyIssues(_ context.Context, status string) ([]models.Issue, error) {
	f.myIssuesStatus = status

	return f.myIssues, nil
}

func (f *fakeService) GetIssue(context.Context, string) (*models.Issue, error) {
	return f.issue, f.issueErr
}

func (f *fakeService) GetIssueTransitions(_ context.Context, issueKey string) ([]models.Transition, error) {
	f.transitionIssue = issueKey
	return f.transitions, nil
}

func (f *fakeService) TransitionIssue(_ context.Context, issueKey, transitionID string) error {
	f.transitionIssue = issueKey
	f.transitionID = transitionID
	return nil
}

func (f *fakeService) AssignIssue(_ context.Context, issueKey, target string) (*models.User, error) {
	f.assignIssue = issueKey
	f.assignTarget = target
	return f.assignUser, nil
}

func (f *fakeService) AddComment(_ context.Context, issueKey, comment string) error {
	f.commentIssue = issueKey
	f.commentText = comment
	return nil
}

func (f *fakeService) SearchIssues(_ context.Context, query string, tag string, version string) ([]models.Issue, error) {
	f.searchQuery = query
	f.searchTag = tag
	f.searchVersion = version
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

func TestMineCommandPassesStatusFilter(t *testing.T) {
	t.Parallel()

	service := &fakeService{}

	_, _, err := executeForTest(t, testDependencies(service), []string{"mine", "--status", "Em andamento"}, "")
	if err != nil {
		t.Fatalf("mine error = %v", err)
	}
	if service.myIssuesStatus != "Em andamento" {
		t.Errorf("status = %q, want Em andamento", service.myIssuesStatus)
	}
}

func TestMineCommandRendersDefaultEmptyMessage(t *testing.T) {
	t.Parallel()

	stdout, _, err := executeForTest(t, testDependencies(&fakeService{}), []string{"mine"}, "")
	if err != nil {
		t.Fatalf("mine error = %v", err)
	}
	if !strings.Contains(stdout, "Nenhuma issue aberta atribuída a você.") {
		t.Errorf("stdout = %q, want default empty message", stdout)
	}
}

func TestMineCommandRendersStatusEmptyMessage(t *testing.T) {
	t.Parallel()

	stdout, _, err := executeForTest(t, testDependencies(&fakeService{}), []string{"mine", "--status", "Em andamento"}, "")
	if err != nil {
		t.Fatalf("mine error = %v", err)
	}
	if !strings.Contains(stdout, "Nenhuma issue atribuída a você encontrada para o status informado.") {
		t.Errorf("stdout = %q, want status empty message", stdout)
	}
}

func TestSearchCommandPassesTagFilter(t *testing.T) {
	t.Parallel()

	service := &fakeService{}

	_, _, err := executeForTest(t, testDependencies(service), []string{"search", "deploy", "--tag", "backend"}, "")
	if err != nil {
		t.Fatalf("search error = %v", err)
	}

	if service.searchQuery != "deploy" {
		t.Errorf("query = %q, want deploy", service.searchQuery)
	}
	if service.searchTag != "backend" {
		t.Errorf("tag = %q, want backend", service.searchTag)
	}
}

func TestSearchCommandAllowsTagOnly(t *testing.T) {
	t.Parallel()

	service := &fakeService{}

	stdout, _, err := executeForTest(t, testDependencies(service), []string{"search", "--tag", "bug"}, "")
	if err != nil {
		t.Fatalf("search error = %v", err)
	}

	if service.searchQuery != "" {
		t.Errorf("query = %q, want empty", service.searchQuery)
	}
	if service.searchTag != "bug" {
		t.Errorf("tag = %q, want bug", service.searchTag)
	}
	if !strings.Contains(stdout, `Resultados para tag "bug"`) {
		t.Errorf("stdout = %q, want tag title", stdout)
	}
}

func TestSearchCommandPassesVersionFilter(t *testing.T) {
	t.Parallel()

	service := &fakeService{}

	stdout, _, err := executeForTest(t, testDependencies(service), []string{"search", "--version", "2.1"}, "")
	if err != nil {
		t.Fatalf("search error = %v", err)
	}

	if service.searchQuery != "" {
		t.Errorf("query = %q, want empty", service.searchQuery)
	}
	if service.searchVersion != "2.1" {
		t.Errorf("version = %q, want 2.1", service.searchVersion)
	}
	if !strings.Contains(stdout, `Resultados para versão "2.1"`) {
		t.Errorf("stdout = %q, want version title", stdout)
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

func TestTransitionCommandListsOptionsAndExecutesSelection(t *testing.T) {
	t.Parallel()

	service := &fakeService{transitions: []models.Transition{
		{ID: "21", Name: "Iniciar trabalho", To: models.Status{Name: "Em andamento"}},
		{ID: "31", Name: "Concluir", To: models.Status{Name: "Concluído"}},
	}}

	stdout, _, err := executeForTest(t, testDependencies(service), []string{"transition", "aej-42"}, "2\n")
	if err != nil {
		t.Fatalf("transition error = %v", err)
	}
	if service.transitionIssue != "AEJ-42" || service.transitionID != "31" {
		t.Errorf("transition = (%q, %q), want (AEJ-42, 31)", service.transitionIssue, service.transitionID)
	}
	if !strings.Contains(stdout, "Em andamento") || !strings.Contains(stdout, "Concluído") || !strings.Contains(stdout, "alterada para") {
		t.Errorf("stdout = %q, want options and success message", stdout)
	}
}

func TestTransitionCommandRejectsInvalidSelection(t *testing.T) {
	t.Parallel()

	service := &fakeService{transitions: []models.Transition{
		{ID: "21", Name: "Iniciar trabalho", To: models.Status{Name: "Em andamento"}},
	}}

	_, _, err := executeForTest(t, testDependencies(service), []string{"transition", "AEJ-42"}, "2\n")
	if err == nil || !strings.Contains(err.Error(), "opção inválida") {
		t.Fatalf("transition error = %v, want invalid option", err)
	}
	if service.transitionID != "" {
		t.Errorf("transition ID = %q, want no transition", service.transitionID)
	}
}

func TestTransitionCommandHandlesNoAvailableTransitions(t *testing.T) {
	t.Parallel()

	service := &fakeService{}
	stdout, _, err := executeForTest(t, testDependencies(service), []string{"transition", "AEJ-42"}, "")
	if err != nil {
		t.Fatalf("transition error = %v", err)
	}
	if !strings.Contains(stdout, "Nenhuma transição disponível") {
		t.Errorf("stdout = %q, want empty transitions message", stdout)
	}
}

func TestAssignCommandAssignsIssueToCurrentUser(t *testing.T) {
	t.Parallel()

	service := &fakeService{assignUser: &models.User{DisplayName: "Ada Lovelace"}}

	stdout, _, err := executeForTest(t, testDependencies(service), []string{"assign", "aej-42"}, "")
	if err != nil {
		t.Fatalf("assign error = %v", err)
	}
	if service.assignIssue != "AEJ-42" {
		t.Errorf("issue key = %q, want AEJ-42", service.assignIssue)
	}
	if service.assignTarget != "me" {
		t.Errorf("target = %q, want me", service.assignTarget)
	}
	if !strings.Contains(stdout, "AEJ-42") || !strings.Contains(stdout, "Ada Lovelace") || !strings.Contains(stdout, "atribuída") {
		t.Errorf("stdout = %q, want issue, user and success message", stdout)
	}
}

func TestAssignCommandSupportsExplicitUserAndUnassign(t *testing.T) {
	t.Parallel()
	userService := &fakeService{assignUser: &models.User{DisplayName: "Grace Hopper"}}
	_, _, err := executeForTest(t, testDependencies(userService), []string{"assign", "AEJ-42", "--to", "grace@example.com"}, "")
	if err != nil || userService.assignTarget != "grace@example.com" {
		t.Fatalf("assign --to error/target = (%v, %q)", err, userService.assignTarget)
	}
	unassignService := &fakeService{}
	stdout, _, err := executeForTest(t, testDependencies(unassignService), []string{"assign", "AEJ-42", "--unassign"}, "")
	if err != nil || unassignService.assignTarget != "unassigned" || !strings.Contains(stdout, "removido") {
		t.Fatalf("assign --unassign = (%v, %q, %q)", err, unassignService.assignTarget, stdout)
	}
}

func TestCommentCommandJoinsTextAndNormalizesKey(t *testing.T) {
	t.Parallel()
	service := &fakeService{}
	stdout, _, err := executeForTest(t, testDependencies(service), []string{"comment", "aej-42", "Pronto", "para", "validar"}, "")
	if err != nil {
		t.Fatalf("comment error = %v", err)
	}
	if service.commentIssue != "AEJ-42" || service.commentText != "Pronto para validar" {
		t.Errorf("comment = (%q, %q)", service.commentIssue, service.commentText)
	}
	if !strings.Contains(stdout, "Comentário adicionado") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestOpenCommandBuildsJiraBrowseURL(t *testing.T) {
	t.Parallel()
	deps := testDependencies(&fakeService{})
	var opened string
	deps.OpenURL = func(target string) error { opened = target; return nil }
	_, _, err := executeForTest(t, deps, []string{"open", "aej-42"}, "")
	if err != nil || opened != "https://example.atlassian.net/browse/AEJ-42" {
		t.Fatalf("open = (%v, %q)", err, opened)
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
		OpenURL: func(string) error { return nil },
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
