package cmd

import (
	"fmt"
	"strings"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newMineCommand(deps Dependencies) *cobra.Command {
	var full bool
	var status string

	command := &cobra.Command{
		Use:   "mine",
		Short: "Listar as últimas 50 issues atribuídas a mim",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMine(deps, cmd, args, full, status)
		},
	}

	command.Flags().BoolVarP(
		&full,
		"full",
		"f",
		false,
		"Exibir cada issue em bloco, sem truncar os campos",
	)
	command.Flags().StringVarP(
		&status,
		"status",
		"s",
		"",
		"Filtrar issues por parte do nome do status no Jira",
	)

	return command
}

func runMine(deps Dependencies, cmd *cobra.Command, _ []string, full bool, status string) error {
	out := cmd.OutOrStdout()
	printer := cli.NewPrinter(out, cmd.ErrOrStderr())

	cfg, err := deps.LoadConfig()

	if err != nil {
		return err
	}

	svc := deps.NewService(cfg)
	issues, err := svc.GetMyIssues(cmd.Context(), status)

	if err != nil {
		return fmt.Errorf("listar issues atribuídas: %w", err)
	}
	if wantsJSON(cmd) {
		return writeJSON(out, issues)
	}

	printer.Header("📋 Minhas Issues")

	if len(issues) == 0 {
		if strings.TrimSpace(status) == "" {
			printer.Info("Nenhuma issue aberta atribuída a você.")
		} else {
			printer.Info("Nenhuma issue atribuída a você encontrada para o status informado.")
		}
		fmt.Fprintln(out)
		return nil
	}

	printIssues(printer, issues, full)

	return nil
}
