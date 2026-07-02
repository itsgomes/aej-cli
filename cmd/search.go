package cmd

import (
	"fmt"
	"strings"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newSearchCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:     "search <TERMO>",
		Short:   "Pesquisar issues por texto",
		Example: "  aej search \"bug de login\"\n  aej search autenticação",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(deps, cmd, args)
		},
	}
}

func runSearch(deps Dependencies, cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	printer := cli.NewPrinter(out, cmd.ErrOrStderr())

	cfg, err := deps.LoadConfig()

	if err != nil {
		return err
	}

	query := strings.Join(args, " ")
	svc := deps.NewService(cfg)
	issues, err := svc.SearchIssues(cmd.Context(), query)

	if err != nil {
		return fmt.Errorf("pesquisar issues: %w", err)
	}

	printer.Header(fmt.Sprintf("🔍 Resultados para \"%s\"", cli.SanitizeInline(query)))

	if len(issues) == 0 {
		printer.Info("Nenhuma issue encontrada.")
		fmt.Fprintln(out)
		return nil
	}

	printer.Printf("  %s Exibindo %d issue(s)\n\n",
		cli.Colorize(cli.Gray, "→"),
		len(issues),
	)

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
