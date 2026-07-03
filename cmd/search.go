package cmd

import (
	"fmt"
	"strings"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newSearchCommand(deps Dependencies) *cobra.Command {
	var full bool

	command := &cobra.Command{
		Use:     "search <TERMO>",
		Short:   "Pesquisar issues por texto",
		Example: "  aej search \"bug de login\"\n  aej search autenticação",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(deps, cmd, args, full)
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

func runSearch(deps Dependencies, cmd *cobra.Command, args []string, full bool) error {
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

	printIssues(printer, issues, full)

	return nil
}
