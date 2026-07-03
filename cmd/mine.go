package cmd

import (
	"fmt"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newMineCommand(deps Dependencies) *cobra.Command {
	var full bool

	command := &cobra.Command{
		Use:   "mine",
		Short: "Listar as últimas 50 issues atribuídas a mim que não estejam como concluídas",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMine(deps, cmd, args, full)
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

func runMine(deps Dependencies, cmd *cobra.Command, _ []string, full bool) error {
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

	printIssues(printer, issues, full)

	return nil
}
