package cmd

import (
	"fmt"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newSprintCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "sprint",
		Short: "Exibir informações da sprint ativa",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSprint(deps, cmd, args)
		},
	}
}

func runSprint(deps Dependencies, cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	printer := cli.NewPrinter(out, cmd.ErrOrStderr())

	cfg, err := deps.LoadConfig()

	if err != nil {
		return err
	}

	svc := deps.NewService(cfg)
	stats, err := svc.GetActiveSprint(cmd.Context())

	if err != nil {
		return fmt.Errorf("obter sprint ativa (a API requer Jira Software com a API Agile habilitada): %w", err)
	}

	printer.Header(fmt.Sprintf("🏃 Sprint: %s", cli.SanitizeInline(stats.Sprint.Name)))
	printer.Field("Início", cli.FormatDate(stats.Sprint.StartDate))
	printer.Field("Término", cli.FormatDate(stats.Sprint.EndDate))
	fmt.Fprintln(out)

	counted := stats.Done + stats.InProgress + stats.Todo

	if counted == 0 {
		printer.Info("Nenhuma issue na sprint.")
		fmt.Fprintln(out)
		return nil
	}

	donePercent := stats.Done * 100 / counted
	inProgressPercent := stats.InProgress * 100 / counted
	todoPercent := stats.Todo * 100 / counted

	printer.Printf("  %s Issues (%d total)\n\n", cli.Colorize(cli.Bold+cli.Cyan, "◉"), stats.Total)
	printer.Printf("    %s Concluídas:    %s (%d%%)\n", cli.Colorize(cli.Green, "✔"), cli.Colorize(cli.Green, fmt.Sprintf("%3d", stats.Done)), donePercent)
	printer.Printf("    %s Em andamento:  %s (%d%%)\n", cli.Colorize(cli.Yellow, "⚡"), cli.Colorize(cli.Yellow, fmt.Sprintf("%3d", stats.InProgress)), inProgressPercent)
	printer.Printf("    %s Pendentes:     %s (%d%%)\n", cli.Colorize(cli.Gray, "○"), cli.Colorize(cli.Gray, fmt.Sprintf("%3d", stats.Todo)), todoPercent)
	fmt.Fprintln(out)
	return nil
}
