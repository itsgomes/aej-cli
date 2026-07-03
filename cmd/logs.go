package cmd

import (
	"fmt"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newLogsCommand(deps Dependencies) *cobra.Command {
	var full bool

	command := &cobra.Command{
		Use:   "logs",
		Short: "Exibir tempo trabalhado nos últimos 7 dias",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogs(deps, cmd, args, full)
		},
	}

	command.Flags().BoolVarP(
		&full,
		"full",
		"f",
		false,
		"Exibir cada issue em bloco, sem truncar os campos",
	)

	return command
}

func runLogs(deps Dependencies, cmd *cobra.Command, _ []string, full bool) error {
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

	printWorklogs(printer, summaries, full)

	printer.Printf("  %s %s\n\n", cli.Colorize(cli.Gray, "Total:"), cli.Colorize(cli.Bold+cli.Green, cli.FormatSeconds(totalSeconds)))
	return nil
}
