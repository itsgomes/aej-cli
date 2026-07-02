package jira

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/itsgomes/aej-cli/internal/config"
	"github.com/itsgomes/aej-cli/internal/models"
)

type Client struct {
	http      *resty.Client
	writeHTTP *resty.Client
}

type silentLogger struct{}

func (silentLogger) Debugf(string, ...any) {}
func (silentLogger) Errorf(string, ...any) {}
func (silentLogger) Warnf(string, ...any)  {}

const defaultTimeout = 15 * time.Second
const defaultPageSize = 100
const defaultRetryCount = 2
const defaultRetryWait = 200 * time.Millisecond
const maxRetryWait = 30 * time.Second

type searchRequest struct {
	JQL           string   `json:"jql"`
	Fields        []string `json:"fields"`
	MaxResults    int      `json:"maxResults"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

type searchPage struct {
	Issues        []models.Issue `json:"issues"`
	NextPageToken string         `json:"nextPageToken"`
	IsLast        bool           `json:"isLast"`
}

type countResult struct {
	Count int `json:"count"`
}

type Option func(*Client)

func WithTimeout(timeout time.Duration) Option {
	return func(client *Client) {
		client.http.SetTimeout(timeout)
		client.writeHTTP.SetTimeout(timeout)
	}
}

func New(cfg *config.Config, options ...Option) *Client {
	client := &Client{
		http:      newHTTPClient(cfg),
		writeHTTP: newHTTPClient(cfg),
	}

	configureReadRetries(client.http)

	for _, option := range options {
		option(client)
	}

	return client
}

func newHTTPClient(cfg *config.Config) *resty.Client {
	return resty.New().
		SetLogger(silentLogger{}).
		SetBaseURL(strings.TrimRight(cfg.JiraURL, "/")).
		SetBasicAuth(cfg.Email, cfg.APIToken).
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetRedirectPolicy(resty.NoRedirectPolicy()).
		SetTimeout(defaultTimeout)
}

func configureReadRetries(client *resty.Client) {
	client.
		SetRetryCount(defaultRetryCount).
		SetRetryWaitTime(defaultRetryWait).
		SetRetryMaxWaitTime(maxRetryWait).
		SetRetryAfter(func(_ *resty.Client, response *resty.Response) (time.Duration, error) {
			if delay, ok := retryAfter(response); ok {
				return delay, nil
			}

			return 0, nil
		}).
		AddRetryCondition(func(response *resty.Response, _ error) bool {
			if response == nil {
				return false
			}

			statusCode := response.StatusCode()
			delay, hasRetryAfter := retryAfter(response)

			switch statusCode {
			case http.StatusTooManyRequests:
				return hasRetryAfter && delay <= maxRetryWait
			case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
				return !hasRetryAfter || delay <= maxRetryWait
			default:
				return false
			}
		})
}

func retryAfter(response *resty.Response) (time.Duration, bool) {
	if response == nil {
		return 0, false
	}

	value := strings.TrimSpace(response.Header().Get("Retry-After"))
	if value == "" {
		return 0, false
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 0 {
		return 0, false
	}

	return time.Duration(seconds) * time.Second, true
}

func (c *Client) GetCurrentUser(ctx context.Context) (*models.User, error) {
	var user models.User

	resp, err := c.http.R().
		SetContext(ctx).
		SetResult(&user).
		Get("/rest/api/3/myself")

	if err != nil {
		return nil, fmt.Errorf("erro de rede: %w", err)
	}

	return &user, handleResponse(resp)
}

func (c *Client) SearchIssues(ctx context.Context, jql string, fields []string, limit int) ([]models.Issue, error) {
	if limit < 0 {
		return nil, errors.New("o limite da busca não pode ser negativo")
	}

	pageSize := defaultPageSize
	if limit > 0 && limit < pageSize {
		pageSize = limit
	}

	issues := make([]models.Issue, 0, pageSize)
	nextPageToken := ""

	for {
		var page searchPage
		requestPageSize := pageSize

		if limit > 0 && limit-len(issues) < requestPageSize {
			requestPageSize = limit - len(issues)
		}

		resp, err := c.http.R().
			SetContext(ctx).
			SetBody(searchRequest{
				JQL:           jql,
				Fields:        fields,
				MaxResults:    requestPageSize,
				NextPageToken: nextPageToken,
			}).
			SetResult(&page).
			Post("/rest/api/3/search/jql")

		if err != nil {
			return nil, fmt.Errorf("erro de rede: %w", err)
		}

		if err := handleResponse(resp); err != nil {
			return nil, err
		}

		issues = append(issues, page.Issues...)

		if limit > 0 && len(issues) >= limit {
			return issues[:limit], nil
		}

		if page.IsLast {
			return issues, nil
		}

		if page.NextPageToken == "" {
			return nil, errors.New("resposta de busca inválida: próxima página sem token")
		}

		if page.NextPageToken == nextPageToken {
			return nil, errors.New("resposta de busca inválida: token de paginação repetido")
		}

		nextPageToken = page.NextPageToken
	}
}

func (c *Client) CountIssues(ctx context.Context, jql string) (int, error) {
	var result countResult

	resp, err := c.http.R().
		SetContext(ctx).
		SetBody(map[string]string{"jql": jql}).
		SetResult(&result).
		Post("/rest/api/3/search/approximate-count")

	if err != nil {
		return 0, fmt.Errorf("erro de rede: %w", err)
	}

	if err := handleResponse(resp); err != nil {
		return 0, err
	}

	return result.Count, nil
}

func (c *Client) GetIssue(ctx context.Context, key string) (*models.Issue, error) {
	var issue models.Issue

	resp, err := c.http.R().
		SetContext(ctx).
		SetResult(&issue).
		Get("/rest/api/3/issue/" + url.PathEscape(key))

	if err != nil {
		return nil, fmt.Errorf("erro de rede: %w", err)
	}

	return &issue, handleResponse(resp)
}

func (c *Client) GetBoards(ctx context.Context) ([]models.Board, error) {
	boards := make([]models.Board, 0, defaultPageSize)
	startAt := 0

	for {
		var page models.BoardResult

		resp, err := c.http.R().
			SetContext(ctx).
			SetQueryParams(map[string]string{
				"startAt":    fmt.Sprintf("%d", startAt),
				"maxResults": fmt.Sprintf("%d", defaultPageSize),
			}).
			SetResult(&page).
			Get("/rest/agile/1.0/board")

		if err != nil {
			return nil, fmt.Errorf("erro de rede: %w", err)
		}

		if err := handleResponse(resp); err != nil {
			return nil, err
		}

		boards = append(boards, page.Values...)

		if page.IsLast || len(boards) >= page.Total {
			return boards, nil
		}

		if len(page.Values) == 0 {
			return nil, errors.New("resposta de boards inválida: página vazia antes do total")
		}

		nextStartAt := page.StartAt + len(page.Values)
		if nextStartAt <= startAt {
			return nil, errors.New("resposta de boards inválida: paginação não avançou")
		}

		startAt = nextStartAt
	}
}

func (c *Client) GetActiveSprint(ctx context.Context, boardID int) (*models.Sprint, error) {
	var result models.SprintResult

	resp, err := c.http.R().
		SetContext(ctx).
		SetQueryParam("state", "active").
		SetResult(&result).
		Get(fmt.Sprintf("/rest/agile/1.0/board/%d/sprint", boardID))

	if err != nil {
		return nil, fmt.Errorf("erro de rede: %w", err)
	}

	if err := handleResponse(resp); err != nil {
		return nil, err
	}

	if len(result.Values) == 0 {
		return nil, nil
	}

	return &result.Values[0], nil
}

func (c *Client) AddWorklog(ctx context.Context, issueKey, timeSpent, comment, started string) error {
	body := map[string]any{
		"timeSpent": timeSpent,
		"started":   started,
	}

	if comment != "" {
		body["comment"] = buildADFComment(comment)
	}

	resp, err := c.writeHTTP.R().
		SetContext(ctx).
		SetBody(body).
		Post("/rest/api/3/issue/" + url.PathEscape(issueKey) + "/worklog")

	if err != nil {
		return fmt.Errorf("erro de rede: %w", err)
	}

	return handleResponse(resp)
}

func (c *Client) GetIssueWorklogs(ctx context.Context, issueKey string) ([]models.Worklog, error) {
	worklogs := make([]models.Worklog, 0, defaultPageSize)
	startAt := 0

	for {
		var page models.WorklogResult

		resp, err := c.http.R().
			SetContext(ctx).
			SetQueryParams(map[string]string{
				"startAt":    fmt.Sprintf("%d", startAt),
				"maxResults": fmt.Sprintf("%d", defaultPageSize),
			}).
			SetResult(&page).
			Get("/rest/api/3/issue/" + url.PathEscape(issueKey) + "/worklog")

		if err != nil {
			return nil, fmt.Errorf("erro de rede: %w", err)
		}

		if err := handleResponse(resp); err != nil {
			return nil, err
		}

		worklogs = append(worklogs, page.Worklogs...)

		if len(worklogs) >= page.Total {
			return worklogs, nil
		}

		if len(page.Worklogs) == 0 {
			return nil, errors.New("resposta de worklogs inválida: página vazia antes do total")
		}

		nextStartAt := page.StartAt + len(page.Worklogs)
		if nextStartAt <= startAt {
			return nil, errors.New("resposta de worklogs inválida: paginação não avançou")
		}

		startAt = nextStartAt
	}
}

func buildADFComment(text string) *models.ADFDocument {
	return &models.ADFDocument{
		Type:    "doc",
		Version: 1,
		Content: []models.ADFNode{
			{
				Type: "paragraph",
				Content: []models.ADFNode{
					{Type: "text", Text: text},
				},
			},
		},
	}
}

func (c *Client) GetSprintIssues(ctx context.Context, sprintID int) ([]models.Issue, error) {
	issues := make([]models.Issue, 0, defaultPageSize)
	nextPageToken := ""

	for {
		var page models.SprintIssuePage

		request := c.http.R().
			SetContext(ctx).
			SetQueryParams(map[string]string{
				"fields":     "summary,status",
				"maxResults": fmt.Sprintf("%d", defaultPageSize),
			}).
			SetResult(&page)

		if nextPageToken != "" {
			request.SetQueryParam("nextPageToken", nextPageToken)
		}

		resp, err := request.Get(fmt.Sprintf("/rest/agile/1.0/sprint/%d/issue", sprintID))
		if err != nil {
			return nil, fmt.Errorf("erro de rede: %w", err)
		}

		if err := handleResponse(resp); err != nil {
			return nil, err
		}

		issues = append(issues, page.Issues...)

		if page.IsLast {
			return issues, nil
		}

		if page.NextPageToken == "" {
			return nil, errors.New("resposta de issues da sprint inválida: próxima página sem token")
		}

		if page.NextPageToken == nextPageToken {
			return nil, errors.New("resposta de issues da sprint inválida: token de paginação repetido")
		}

		nextPageToken = page.NextPageToken
	}
}

func handleResponse(resp *resty.Response) error {
	var err error

	switch resp.StatusCode() {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return nil
	case http.StatusBadRequest:
		err = ErrBadRequest
	case http.StatusUnauthorized:
		err = ErrUnauthorized
	case http.StatusForbidden:
		err = ErrForbidden
	case http.StatusNotFound:
		err = ErrNotFound
	case http.StatusGone:
		err = ErrGone
	case http.StatusTooManyRequests:
		err = ErrRateLimited
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		err = ErrUnavailable
	default:
		err = ErrUnexpected
	}

	return &APIError{StatusCode: resp.StatusCode(), Err: err}
}
