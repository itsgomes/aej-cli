package cmd

import (
	"fmt"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/itsgomes/aej-cli/internal/models"
	"github.com/spf13/cobra"
)

func newMeCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Exibir informações do usuário atual",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMe(deps, cmd, args)
		},
	}
}

func runMe(deps Dependencies, cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	printer := cli.NewPrinter(out, cmd.ErrOrStderr())

	cfg, err := deps.LoadConfig()

	if err != nil {
		return err
	}

	svc := deps.NewService(cfg)
	user, openCount, err := svc.GetCurrentUserWithStats(cmd.Context())

	if err != nil {
		return fmt.Errorf("obter perfil: %w", err)
	}
	if wantsJSON(cmd) {
		return writeJSON(out, struct {
			User       *models.User `json:"user"`
			OpenIssues int          `json:"openIssues"`
		}{User: user, OpenIssues: openCount})
	}

	printer.Header("👤 Meu Perfil")
	printer.Field("Nome", cli.BoldText(cli.SanitizeInline(user.DisplayName)))
	printer.Field("E-mail", cli.SanitizeInline(user.EmailAddress))
	printer.Field("Issues abertas (aprox.)", fmt.Sprintf("%s %d", cli.Colorize(cli.Yellow, "●"), openCount))
	fmt.Fprintln(out)
	return nil
}
