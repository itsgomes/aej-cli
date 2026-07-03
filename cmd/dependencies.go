package cmd

import (
	"context"

	"github.com/itsgomes/aej-cli/internal/config"
	jiraclient "github.com/itsgomes/aej-cli/internal/jira"
	"github.com/itsgomes/aej-cli/internal/models"
	"github.com/itsgomes/aej-cli/internal/services"
)

type Service interface {
	GetCurrentUserWithStats(context.Context) (*models.User, int, error)
	GetMyIssues(context.Context) ([]models.Issue, error)
	GetIssue(context.Context, string) (*models.Issue, error)
	SearchIssues(context.Context, string) ([]models.Issue, error)
	GetBoards(context.Context) ([]models.Board, error)
	GetBoardIssues(context.Context, int) ([]models.Issue, error)
	AddWorklog(context.Context, string, string, string) error
	GetWeeklyWorklogs(context.Context) ([]models.IssueWorklogSummary, int, error)
}

type Authenticator interface {
	GetCurrentUser(context.Context) (*models.User, error)
}

type Dependencies struct {
	LoadConfig       func() (*config.Config, error)
	SaveConfig       func(config.Config) error
	NewService       func(*config.Config) Service
	NewAuthenticator func(*config.Config) Authenticator
}

func defaultDependencies() Dependencies {
	return Dependencies{
		LoadConfig: config.Load,
		SaveConfig: config.Save,
		NewService: func(cfg *config.Config) Service {
			return services.New(jiraclient.New(cfg))
		},
		NewAuthenticator: func(cfg *config.Config) Authenticator {
			return jiraclient.New(cfg)
		},
	}
}
