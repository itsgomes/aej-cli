package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

func NewRootCommand(deps Dependencies) *cobra.Command {
	root := &cobra.Command{
		Use:           "aej",
		Short:         "AEJ-CLI — Interface de linha de comando para o Jira",
		SilenceErrors: true,
		SilenceUsage:  true,
		Long: `AEJ-CLI é uma ferramenta CLI para interagir com o Jira diretamente pelo terminal.

	Use 'aej login' para configurar suas credenciais.`,
	}

	reporter := newCommandTimingReporter(root.ErrOrStderr)

	root.PersistentFlags().BoolVar(
		&reporter.enabled,
		"timing",
		false,
		"Exibir o tempo total de execução do comando",
	)

	root.CompletionOptions.DisableDefaultCmd = true

	commands := []*cobra.Command{
		newLoginCommand(deps),
		newBoardCommand(deps),
		newMeCommand(deps),
		newMineCommand(deps),
		newSearchCommand(deps),
		newIssueCommand(deps),
		newTransitionCommand(deps),
		newLogCommand(deps),
		newLogsCommand(deps),
	}

	for _, command := range commands {
		reporter.Wrap(command)
	}

	root.AddCommand(commands...)

	return root
}

func Execute(ctx context.Context) error {
	return NewRootCommand(defaultDependencies()).ExecuteContext(ctx)
}
