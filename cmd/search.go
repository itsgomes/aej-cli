package cmd

import (
	"fmt"
	"strings"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newSearchCommand(deps Dependencies) *cobra.Command {
	var full bool
	var tag string
	var version string

	command := &cobra.Command{
		Use:     "search [TERMO]",
		Short:   "Pesquisar issues por texto, tag ou versão",
		Example: "  aej search \"bug de login\"\n  aej search autenticação\n  aej search \"deploy\" --tag backend\n  aej search --tag bug\n  aej search --version 2.1",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 || strings.TrimSpace(tag) != "" || strings.TrimSpace(version) != "" {
				return nil
			}

			return fmt.Errorf("informe um termo de busca, uma tag ou uma versão")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(deps, cmd, args, full, tag, version)
		},
	}

	command.Flags().BoolVarP(
		&full,
		"full",
		"f",
		false,
		"Exibir cada issue em bloco, sem truncar os campos",
	)
	command.Flags().StringVar(
		&tag,
		"tag",
		"",
		"Pesquisar issues por tag/label",
	)
	command.Flags().StringVar(
		&version,
		"version",
		"",
		"Pesquisar issues por versão usando correspondência parcial",
	)

	return command
}

func runSearch(deps Dependencies, cmd *cobra.Command, args []string, full bool, tag string, version string) error {
	out := cmd.OutOrStdout()
	printer := cli.NewPrinter(out, cmd.ErrOrStderr())

	cfg, err := deps.LoadConfig()

	if err != nil {
		return err
	}

	query := strings.Join(args, " ")
	tag = strings.TrimSpace(tag)
	version = strings.TrimSpace(version)
	svc := deps.NewService(cfg)
	issues, err := svc.SearchIssues(cmd.Context(), query, tag, version)

	if err != nil {
		return fmt.Errorf("pesquisar issues: %w", err)
	}

	title := fmt.Sprintf("🔍 Resultados para \"%s\"", cli.SanitizeInline(query))
	if query == "" {
		title = fmt.Sprintf("🔍 Resultados para tag \"%s\"", cli.SanitizeInline(tag))
	} else if tag != "" {
		title = fmt.Sprintf("🔍 Resultados para \"%s\" com tag \"%s\"", cli.SanitizeInline(query), cli.SanitizeInline(tag))
	}
	if version != "" {
		if query == "" && tag == "" {
			title = fmt.Sprintf("🔍 Resultados para versão \"%s\"", cli.SanitizeInline(version))
		} else {
			title = fmt.Sprintf("%s e versão \"%s\"", title, cli.SanitizeInline(version))
		}
	}

	printer.Header(title)

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
