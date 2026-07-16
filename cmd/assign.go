package cmd

import (
	"fmt"
	"strings"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newAssignCommand(deps Dependencies) *cobra.Command {
	var assignee string
	var unassign bool
	var useDefault bool
	command := &cobra.Command{
		Use:     "assign <CHAVE>",
		Short:   "Alterar o responsável por uma issue",
		Example: "  aej assign DEV-123\n  aej assign DEV-123 --to usuario@empresa.com\n  aej assign DEV-123 --unassign\n  aej assign DEV-123 --default",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAssign(deps, cmd, args, assignee, unassign, useDefault)
		},
	}
	command.Flags().StringVar(&assignee, "to", "", "Atribuir ao usuário informado por e-mail, nome ou accountId")
	command.Flags().BoolVar(&unassign, "unassign", false, "Remover o responsável da issue")
	command.Flags().BoolVar(&useDefault, "default", false, "Atribuir ao responsável padrão do projeto")
	command.MarkFlagsMutuallyExclusive("to", "unassign", "default")
	return command
}

func runAssign(deps Dependencies, cmd *cobra.Command, args []string, assignee string, unassign, useDefault bool) error {
	out := cmd.OutOrStdout()
	printer := cli.NewPrinter(out, cmd.ErrOrStderr())
	issueKey := strings.ToUpper(strings.TrimSpace(args[0]))

	cfg, err := deps.LoadConfig()
	if err != nil {
		return err
	}

	svc := deps.NewService(cfg)
	target := "me"
	if unassign {
		target = "unassigned"
	} else if useDefault {
		target = "default"
	} else if strings.TrimSpace(assignee) != "" {
		target = assignee
	}
	user, err := svc.AssignIssue(cmd.Context(), issueKey, target)
	if err != nil {
		return fmt.Errorf("atribuir %s a você: %w", issueKey, err)
	}

	var message string
	if unassign {
		message = fmt.Sprintf("Responsável removido de %s.", cli.Colorize(cli.Cyan, issueKey))
	} else if useDefault {
		message = fmt.Sprintf("%s atribuída ao responsável padrão do projeto.", cli.Colorize(cli.Cyan, issueKey))
	} else {
		message = fmt.Sprintf("%s atribuída a %s.", cli.Colorize(cli.Cyan, issueKey), cli.BoldText(cli.SanitizeInline(user.DisplayName)))
	}
	printer.Success(message)

	return nil
}
