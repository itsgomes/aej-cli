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

type JiraGateway interface {
	GetCurrentUser(context.Context) (*models.User, error)
	CountIssues(context.Context, string) (int, error)
	SearchIssues(context.Context, string, []string, int) ([]models.Issue, error)
	GetIssue(context.Context, string) (*models.Issue, error)
	GetBoards(context.Context) ([]models.Board, error)
	GetBoardIssues(context.Context, int, string, []string, int) ([]models.Issue, error)
	GetActiveSprint(context.Context, int) (*models.Sprint, error)
	GetSprintIssues(context.Context, int) ([]models.Issue, error)
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

func (s *JiraService) GetMyIssues(ctx context.Context) ([]models.Issue, error) {
	issues, err := s.client.SearchIssues(ctx, "assignee = currentUser() AND statusCategory != Done ORDER BY updated DESC", []string{"summary", "status", "priority", "issuetype"}, 50)

	if err != nil {
		return nil, err
	}

	return issues, nil
}

func (s *JiraService) GetIssue(ctx context.Context, key string) (*models.Issue, error) {
	normalizedKey, err := normalizeIssueKey(key)
	if err != nil {
		return nil, err
	}

	return s.client.GetIssue(ctx, normalizedKey)
}

func (s *JiraService) SearchIssues(ctx context.Context, query string) ([]models.Issue, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrEmptySearchTerm
	}

	jql := fmt.Sprintf("text ~ %s ORDER BY updated DESC", jqlStringLiteral(query))

	issues, err := s.client.SearchIssues(ctx, jql, []string{"summary", "status", "priority", "issuetype"}, 20)

	if err != nil {
		return nil, err
	}

	return issues, nil
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

func (s *JiraService) GetActiveSprint(ctx context.Context) (*models.SprintStats, error) {
	boards, err := s.client.GetBoards(ctx)

	if err != nil {
		return nil, fmt.Errorf("não foi possível obter os boards: %w", err)
	}

	if len(boards) == 0 {
		return nil, errors.New("nenhum board foi encontrado")
	}

	for _, board := range boards {
		sprint, err := s.client.GetActiveSprint(ctx, board.ID)

		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}

			continue
		}

		if sprint == nil {
			continue
		}

		issues, err := s.client.GetSprintIssues(ctx, sprint.ID)
		if err != nil {
			return nil, fmt.Errorf("obter issues da sprint %q: %w", sprint.Name, err)
		}

		stats := &models.SprintStats{
			Sprint: *sprint,
			Total:  len(issues),
		}

		for _, issue := range issues {
			switch issue.Fields.Status.StatusCategory.Key {
			case "done":
				stats.Done++
			case "indeterminate":
				stats.InProgress++
			default:
				stats.Todo++
			}
		}

		return stats, nil
	}

	return nil, errors.New("nenhuma sprint ativa foi encontrada")
}

func (s *JiraService) AddWorklog(ctx context.Context, issueKey, timeSpent, comment string) error {
	normalizedKey, err := normalizeIssueKey(issueKey)
	if err != nil {
		return err
	}

	started := s.now().Format("2006-01-02T15:04:05.000-0700")
	return s.client.AddWorklog(ctx, normalizedKey, timeSpent, comment, started)
}

func (s *JiraService) GetWeeklyWorklogs(ctx context.Context) ([]models.IssueWorklogSummary, int, error) {
	me, err := s.client.GetCurrentUser(ctx)

	if err != nil {
		return nil, 0, err
	}

	issues, err := s.client.SearchIssues(ctx, "worklogAuthor = currentUser() AND worklogDate >= -7d ORDER BY updated DESC", []string{"summary"}, 0)

	if err != nil {
		return nil, 0, err
	}

	cutoff := s.now().AddDate(0, 0, -7)
	var summaries []models.IssueWorklogSummary
	totalSeconds := 0

	worklogsByIssue, err := s.fetchIssueWorklogs(ctx, issues)
	if err != nil {
		return nil, 0, err
	}

	for index, issue := range issues {
		worklogs := worklogsByIssue[index]

		var entries []models.Worklog
		issueTotal := 0

		for _, wl := range worklogs {

			if wl.Author.AccountID != me.AccountID {
				continue
			}

			started, err := parseJiraDate(wl.Started)

			if err != nil || started.Before(cutoff) {
				continue
			}

			entries = append(entries, wl)
			issueTotal += wl.TimeSpentSeconds
		}

		if issueTotal > 0 {
			summaries = append(summaries, models.IssueWorklogSummary{
				IssueKey: issue.Key,
				Summary:  issue.Fields.Summary,
				Total:    issueTotal,
				Entries:  entries,
			})

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
