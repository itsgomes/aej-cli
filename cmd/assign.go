package cmd

import (
	"fmt"
	"strings"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newAssignCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:     "assign <CHAVE>",
		Short:   "Atribuir uma issue a mim",
		Example: "  aej assign DEV-123",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAssign(deps, cmd, args)
		},
	}
}

func runAssign(deps Dependencies, cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	printer := cli.NewPrinter(out, cmd.ErrOrStderr())
	issueKey := strings.ToUpper(strings.TrimSpace(args[0]))

	cfg, err := deps.LoadConfig()
	if err != nil {
		return err
	}

	svc := deps.NewService(cfg)
	user, err := svc.AssignIssueToMe(cmd.Context(), issueKey)
	if err != nil {
		return fmt.Errorf("atribuir %s a você: %w", issueKey, err)
	}

	printer.Success(fmt.Sprintf("%s atribuída a %s.",
		cli.Colorize(cli.Cyan, issueKey),
		cli.BoldText(cli.SanitizeInline(user.DisplayName)),
	))

	return nil
}
