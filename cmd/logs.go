package cmd

import (
	"fmt"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newLogsCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Exibir tempo trabalhado nos últimos 7 dias",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogs(deps, cmd, args)
		},
	}
}

func runLogs(deps Dependencies, cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	printer := cli.NewPrinter(out, cmd.ErrOrStderr())

	cfg, err := deps.LoadConfig()

	if err != nil {
		return err
	}

	svc := deps.NewService(cfg)
	summaries, totalSeconds, err := svc.GetWeeklyWorklogs(cmd.Context())

	if err != nil {
		return fmt.Errorf("obter registros de tempo: %w", err)
	}

	printer.Header("⏱ Tempo Trabalhado — Últimos 7 dias")

	if len(summaries) == 0 {
		printer.Info("Nenhum registro de tempo encontrado nos últimos 7 dias.")
		fmt.Fprintln(out)
		return nil
	}

	rows := make([][]string, 0)

	for _, s := range summaries {
		for _, entry := range s.Entries {
			comment := cli.Truncate(cli.SanitizeInline(cli.ExtractDescription(entry.Comment)), 40)
			rows = append(rows, []string{cli.Colorize(cli.Cyan, cli.SanitizeInline(s.IssueKey)), cli.Truncate(cli.SanitizeInline(s.Summary), 40), cli.FormatDate(entry.Started), cli.Colorize(cli.Green, cli.SanitizeInline(entry.TimeSpent)), comment})
		}
	}

	printer.Table([]string{"Issue", "Resumo", "Data", "Tempo", "Comentário"}, rows)

	printer.Printf("  %s %s\n\n", cli.Colorize(cli.Gray, "Total:"), cli.Colorize(cli.Bold+cli.Green, cli.FormatSeconds(totalSeconds)))
	return nil
}
