package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/itsgomes/aej-cli/internal/models"
)

const maxConcurrentWorklogRequests = 4
const myIssuesLimit = 50
const searchIssuesLimit = 20

type JiraGateway interface {
	GetCurrentUser(context.Context) (*models.User, error)
	CountIssues(context.Context, string) (int, error)
	SearchIssues(context.Context, string, []string, int) ([]models.Issue, error)
	GetIssue(context.Context, string) (*models.Issue, error)
	GetIssueTransitions(context.Context, string) ([]models.Transition, error)
	TransitionIssue(context.Context, string, string) error
	AssignIssue(context.Context, string, string) error
	FindAssignableUsers(context.Context, string, string) ([]models.User, error)
	AddComment(context.Context, string, string) error
	GetBoards(context.Context) ([]models.Board, error)
	GetBoardIssues(context.Context, int, string, []string, int) ([]models.Issue, error)
	AddWorklog(context.Context, string, string, string, string) error
	GetIssueWorklogs(context.Context, string) ([]models.Worklog, error)
}

type JiraService struct {
	client JiraGateway
	now    func() time.Time
}

type Option func(*JiraService)

func WithClock(now func() time.Time) Option {
	return func(service *JiraService) {
		service.now = now
	}
}

func New(client JiraGateway, options ...Option) *JiraService {
	service := &JiraService{
		client: client,
		now:    time.Now,
	}

	for _, option := range options {
		option(service)
	}

	return service
}

func (s *JiraService) GetCurrentUserWithStats(ctx context.Context) (*models.User, int, error) {
	user, err := s.client.GetCurrentUser(ctx)
	if err != nil {
		return nil, 0, err
	}

	openCount, err := s.client.CountIssues(ctx, "assignee = currentUser() AND statusCategory != Done")

	if err != nil {
		return nil, 0, fmt.Errorf("contar issues abertas: %w", err)
	}

	return user, openCount, nil
}

func (s *JiraService) GetMyIssues(ctx context.Context, status string) ([]models.Issue, error) {
	status = strings.TrimSpace(status)

	if status == "" {
		issues, err := s.client.SearchIssues(ctx, "assignee = currentUser() AND statusCategory != Done ORDER BY updated DESC", []string{"summary", "status", "priority", "issuetype"}, myIssuesLimit)

		if err != nil {
			return nil, err
		}

		return issues, nil
	}

	issues, err := s.client.SearchIssues(ctx, "assignee = currentUser() ORDER BY updated DESC", []string{"summary", "status", "priority", "issuetype"}, 0)

	if err != nil {
		return nil, err
	}

	filtered := make([]models.Issue, 0, myIssuesLimit)
	for _, issue := range issues {
		if !statusContains(issue.Fields.Status.Name, status) {
			continue
		}

		filtered = append(filtered, issue)
		if len(filtered) == myIssuesLimit {
			break
		}
	}

	return filtered, nil
}

func statusContains(statusName string, query string) bool {
	return strings.Contains(strings.ToLower(statusName), strings.ToLower(query))
}

func (s *JiraService) GetIssue(ctx context.Context, key string) (*models.Issue, error) {
	normalizedKey, err := normalizeIssueKey(key)
	if err != nil {
		return nil, err
	}

	return s.client.GetIssue(ctx, normalizedKey)
}

func (s *JiraService) GetIssueTransitions(ctx context.Context, key string) ([]models.Transition, error) {
	normalizedKey, err := normalizeIssueKey(key)
	if err != nil {
		return nil, err
	}

	return s.client.GetIssueTransitions(ctx, normalizedKey)
}

func (s *JiraService) TransitionIssue(ctx context.Context, key, transitionID string) error {
	normalizedKey, err := normalizeIssueKey(key)
	if err != nil {
		return err
	}

	transitionID = strings.TrimSpace(transitionID)
	if transitionID == "" {
		return errors.New("o ID da transição é obrigatório")
	}

	return s.client.TransitionIssue(ctx, normalizedKey, transitionID)
}

func (s *JiraService) AssignIssue(ctx context.Context, key, assignee string) (*models.User, error) {
	normalizedKey, err := normalizeIssueKey(key)
	if err != nil {
		return nil, err
	}

	assignee = strings.TrimSpace(assignee)
	var user *models.User
	var accountID string

	switch assignee {
	case "me":
		user, err = s.client.GetCurrentUser(ctx)
		if err != nil {
			return nil, fmt.Errorf("obter usuário atual: %w", err)
		}
		if user == nil || strings.TrimSpace(user.AccountID) == "" {
			return nil, errors.New("usuário atual sem accountId no Jira")
		}
		accountID = user.AccountID
	case "unassigned":
		accountID = ""
	case "default":
		accountID = "-1"
	default:
		if assignee == "" {
			return nil, errors.New("o responsável é obrigatório")
		}
		users, findErr := s.client.FindAssignableUsers(ctx, normalizedKey, assignee)
		if findErr != nil {
			return nil, findErr
		}
		user, err = selectAssignableUser(users, assignee)
		if err != nil {
			return nil, err
		}
		accountID = user.AccountID
	}

	if err := s.client.AssignIssue(ctx, normalizedKey, accountID); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *JiraService) AssignIssueToMe(ctx context.Context, key string) (*models.User, error) {
	return s.AssignIssue(ctx, key, "me")
}

func selectAssignableUser(users []models.User, query string) (*models.User, error) {
	query = strings.TrimSpace(query)
	var exactMatches []*models.User
	for index := range users {
		user := &users[index]
		if strings.EqualFold(user.AccountID, query) || strings.EqualFold(user.EmailAddress, query) || strings.EqualFold(user.DisplayName, query) {
			exactMatches = append(exactMatches, user)
		}
	}
	if len(exactMatches) == 1 {
		return exactMatches[0], nil
	}
	if len(exactMatches) > 1 {
		return nil, fmt.Errorf("mais de um usuário atribuível encontrado para %q; informe o e-mail ou accountId", query)
	}
	if len(users) == 1 {
		return &users[0], nil
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("nenhum usuário atribuível encontrado para %q", query)
	}
	return nil, fmt.Errorf("mais de um usuário atribuível encontrado para %q; informe o e-mail ou accountId", query)
}

func (s *JiraService) AddComment(ctx context.Context, key, comment string) error {
	normalizedKey, err := normalizeIssueKey(key)
	if err != nil {
		return err
	}
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return errors.New("o comentário é obrigatório")
	}
	return s.client.AddComment(ctx, normalizedKey, comment)
}

func (s *JiraService) SearchIssues(ctx context.Context, query string, tag string, version string) ([]models.Issue, error) {
	query = strings.TrimSpace(query)
	tag = strings.TrimSpace(tag)
	version = strings.TrimSpace(version)
	if query == "" && tag == "" && version == "" {
		return nil, ErrEmptySearchTerm
	}

	filters := make([]string, 0, 2)
	if query != "" {
		filters = append(filters, fmt.Sprintf("text ~ %s", jqlStringLiteral(query)))
	}
	if tag != "" {
		filters = append(filters, fmt.Sprintf("labels = %s", jqlStringLiteral(tag)))
	}
	if version != "" {
		filters = append(filters, "fixVersion IS NOT EMPTY")
	}

	jql := fmt.Sprintf("%s ORDER BY updated DESC", strings.Join(filters, " AND "))
	fields := []string{"summary", "status", "priority", "issuetype"}
	limit := searchIssuesLimit
	if version != "" {
		fields = append(fields, "fixVersions")
		limit = 0
	}

	issues, err := s.client.SearchIssues(ctx, jql, fields, limit)

	if err != nil {
		return nil, err
	}

	if version != "" {
		issues = filterIssuesByFixVersion(issues, version, searchIssuesLimit)
	}

	return issues, nil
}

func filterIssuesByFixVersion(issues []models.Issue, version string, limit int) []models.Issue {
	matches := make([]models.Issue, 0, min(len(issues), limit))
	needle := strings.ToLower(version)

	for _, issue := range issues {
		for _, fixVersion := range issue.Fields.FixVersions {
			if strings.Contains(strings.ToLower(fixVersion.Name), needle) {
				matches = append(matches, issue)
				break
			}
		}

		if len(matches) == limit {
			break
		}
	}

	return matches
}

func (s *JiraService) GetBoards(ctx context.Context) ([]models.Board, error) {
	return s.client.GetBoards(ctx)
}

func (s *JiraService) GetBoardIssues(ctx context.Context, boardID int) ([]models.Issue, error) {
	if boardID <= 0 {
		return nil, errors.New("o ID do board deve ser maior que zero")
	}

	return s.client.GetBoardIssues(ctx, boardID, "statusCategory != Done ORDER BY updated DESC", []string{"summary", "status", "priority", "issuetype"}, 50)
}

func (s *JiraService) AddWorklog(ctx context.Context, issueKey, timeSpent, comment string) error {
	normalizedKey, err := normalizeIssueKey(issueKey)
	if err != nil {
		return err
	}

	started := s.now().Format("2006-01-02T15:04:05.000-0700")
	return s.client.AddWorklog(ctx, normalizedKey, timeSpent, comment, started)
}

func (s *JiraService) GetWorklogs(ctx context.Context, from time.Time, to time.Time) ([]models.IssueWorklogSummary, int, error) {
	if !from.Before(to) {
		return nil, 0, errors.New("intervalo de datas inválido")
	}

	me, err := s.client.GetCurrentUser(ctx)

	if err != nil {
		return nil, 0, err
	}

	firstDate := from.Format("2006-01-02")
	lastDate := to.Add(-time.Nanosecond).Format("2006-01-02")

	jql := fmt.Sprintf(
		`worklogAuthor = currentUser() AND worklogDate >= "%s" AND worklogDate <= "%s" ORDER BY updated DESC`,
		firstDate,
		lastDate,
	)

	issues, err := s.client.SearchIssues(ctx, jql, []string{"summary"}, 0)

	if err != nil {
		return nil, 0, err
	}

	worklogsByIssue, err := s.fetchIssueWorklogs(ctx, issues)

	if err != nil {
		return nil, 0, err
	}

	var summaries []models.IssueWorklogSummary
	totalSeconds := 0

	for index, issue := range issues {
		worklogs := worklogsByIssue[index]

		var entries []models.Worklog
		issueTotal := 0

		for _, worklog := range worklogs {
			if worklog.Author.AccountID != me.AccountID {
				continue
			}

			started, err := parseJiraDate(worklog.Started)

			if err != nil {
				continue
			}

			if started.Before(from) {
				continue
			}

			if !started.Before(to) {
				continue
			}

			entries = append(entries, worklog)

			issueTotal += worklog.TimeSpentSeconds
		}

		if issueTotal > 0 {
			summaries = append(
				summaries,
				models.IssueWorklogSummary{
					IssueKey: issue.Key,
					Summary:  issue.Fields.Summary,
					Total:    issueTotal,
					Entries:  entries,
				},
			)

			totalSeconds += issueTotal
		}
	}

	return summaries, totalSeconds, nil
}

func (s *JiraService) fetchIssueWorklogs(ctx context.Context, issues []models.Issue) ([][]models.Worklog, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([][]models.Worklog, len(issues))
	semaphore := make(chan struct{}, maxConcurrentWorklogRequests)
	var waitGroup sync.WaitGroup
	var firstErr error
	var errOnce sync.Once

issueLoop:
	for index, issue := range issues {
		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			break issueLoop
		}

		if ctx.Err() != nil {
			break issueLoop
		}

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()
			defer func() { <-semaphore }()

			worklogs, err := s.client.GetIssueWorklogs(ctx, issue.Key)
			if err != nil {
				errOnce.Do(func() {
					firstErr = fmt.Errorf("obter worklogs da issue %s: %w", issue.Key, err)
					cancel()
				})
				return
			}

			results[index] = worklogs
		}()
	}

	waitGroup.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func parseJiraDate(dateStr string) (time.Time, error) {
	formats := []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05.000-0700", "2006-01-02T15:04:05.999-0700"}

	for _, f := range formats {
		if t, err := time.Parse(f, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("formato de data não reconhecido: %s", dateStr)
}
