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

	reporter := newRequestTimingReporter(root.ErrOrStderr)

	root.PersistentFlags().BoolVar(
		&reporter.enabled,
		"timing",
		false,
		"Exibir o tempo de resposta de cada requisição ao Jira",
	)

	deps = withRequestObserver(deps, reporter.Observe)

	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(
		newLoginCommand(deps),
		newMeCommand(deps),
		newMineCommand(deps),
		newSearchCommand(deps),
		newIssueCommand(deps),
		newSprintCommand(deps),
		newLogCommand(deps),
		newLogsCommand(deps),
	)

	return root
}

func Execute(ctx context.Context) error {
	return NewRootCommand(defaultDependencies()).ExecuteContext(ctx)
}
