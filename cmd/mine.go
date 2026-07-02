package cmd

import (
	"fmt"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newMineCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "mine",
		Short: "Listar issues atribuídas a mim",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMine(deps, cmd, args)
		},
	}
}

func runMine(deps Dependencies, cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	printer := cli.NewPrinter(out, cmd.ErrOrStderr())

	cfg, err := deps.LoadConfig()

	if err != nil {
		return err
	}

	svc := deps.NewService(cfg)
	issues, err := svc.GetMyIssues(cmd.Context())

	if err != nil {
		return fmt.Errorf("listar issues atribuídas: %w", err)
	}

	printer.Header("📋 Minhas Issues")

	if len(issues) == 0 {
		printer.Info("Nenhuma issue aberta atribuída a você.")
		fmt.Fprintln(out)
		return nil
	}

	rows := make([][]string, 0, len(issues))

	for _, issue := range issues {
		priority := "—"
		priorityColor := cli.Gray

		if issue.Fields.Priority != nil {
			priority = cli.SanitizeInline(issue.Fields.Priority.Name)
			priorityColor = cli.PriorityColor(priority)
		}

		statusColor := cli.StatusColor(issue.Fields.Status.StatusCategory.Key)

		rows = append(rows, []string{cli.Colorize(cli.Cyan, cli.SanitizeInline(issue.Key)), cli.Truncate(cli.SanitizeInline(issue.Fields.Summary), 60), cli.Colorize(statusColor, cli.SanitizeInline(issue.Fields.Status.Name)), cli.Colorize(priorityColor, priority)})
	}

	printer.Table([]string{"Chave", "Resumo", "Status", "Prioridade"}, rows)
	return nil
}
