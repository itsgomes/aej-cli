package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newTransitionCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:     "transition <CHAVE>",
		Short:   "Alterar o status de uma issue",
		Example: "  aej transition DEV-123",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTransition(deps, cmd, args)
		},
	}
}

func runTransition(deps Dependencies, cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	printer := cli.NewPrinter(out, cmd.ErrOrStderr())
	issueKey := strings.ToUpper(strings.TrimSpace(args[0]))

	cfg, err := deps.LoadConfig()
	if err != nil {
		return err
	}

	svc := deps.NewService(cfg)
	transitions, err := svc.GetIssueTransitions(cmd.Context(), issueKey)
	if err != nil {
		return fmt.Errorf("listar transições de %s: %w", issueKey, err)
	}

	if len(transitions) == 0 {
		printer.Info(fmt.Sprintf("Nenhuma transição disponível para %s.", cli.Colorize(cli.Cyan, issueKey)))
		return nil
	}

	printer.Header(fmt.Sprintf("Alterar status de %s", cli.Colorize(cli.Cyan, issueKey)))
	for index, transition := range transitions {
		fmt.Fprintf(out, "  %d. %s (%s)\n",
			index+1,
			cli.SanitizeInline(transition.To.Name),
			cli.SanitizeInline(transition.Name),
		)
	}

	fmt.Fprintln(out)
	fmt.Fprint(out, "  Selecione uma opção: ")
	choice, err := readTransitionChoice(bufio.NewReader(cmd.InOrStdin()), len(transitions))
	if err != nil {
		return err
	}

	selected := transitions[choice-1]
	if err := svc.TransitionIssue(cmd.Context(), issueKey, selected.ID); err != nil {
		return fmt.Errorf("alterar status de %s: %w", issueKey, err)
	}

	printer.Success(fmt.Sprintf("%s alterada para %s.",
		cli.Colorize(cli.Cyan, issueKey),
		cli.BoldText(cli.SanitizeInline(selected.To.Name)),
	))
	return nil
}

func readTransitionChoice(reader *bufio.Reader, optionCount int) (int, error) {
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, fmt.Errorf("ler opção: %w", err)
	}

	choice, conversionErr := strconv.Atoi(strings.TrimSpace(line))
	if conversionErr != nil || choice < 1 || choice > optionCount {
		return 0, fmt.Errorf("opção inválida: informe um número entre 1 e %d", optionCount)
	}

	return choice, nil
}
