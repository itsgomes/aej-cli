package cmd

import (
	"fmt"
	"strconv"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newBoardCommand(deps Dependencies) *cobra.Command {
	var full bool

	command := &cobra.Command{
		Use:   "board [ID]",
		Short: "Listar boards ou issues de um board",
		Example: `  aej board
  aej board 1712
  aej board 1712 --full`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBoard(deps, cmd, args, full)
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

func runBoard(deps Dependencies, cmd *cobra.Command, args []string, full bool) error {
	var boardID int

	if len(args) == 1 {
		parsedID, err := strconv.Atoi(args[0])

		if err != nil || parsedID <= 0 {
			return fmt.Errorf("ID do board inválido: %q", args[0])
		}

		boardID = parsedID
	}

	out := cmd.OutOrStdout()
	printer := cli.NewPrinter(out, cmd.ErrOrStderr())

	cfg, err := deps.LoadConfig()
	if err != nil {
		return err
	}

	svc := deps.NewService(cfg)

	if boardID == 0 {
		boards, err := svc.GetBoards(cmd.Context())
		if err != nil {
			return fmt.Errorf("listar boards: %w", err)
		}
		if wantsJSON(cmd) {
			return writeJSON(out, boards)
		}

		printer.Header("📋 Boards disponíveis")

		if len(boards) == 0 {
			printer.Info("Nenhum board disponível.")
			fmt.Fprintln(out)
			return nil
		}

		rows := make([][]string, 0, len(boards))

		for _, board := range boards {
			rows = append(rows, []string{
				strconv.Itoa(board.ID),
				cli.SanitizeInline(board.Name),
				cli.SanitizeInline(board.Type),
			})
		}

		printer.Table(
			[]string{"ID", "Nome", "Tipo"},
			rows,
		)

		return nil
	}

	issues, err := svc.GetBoardIssues(cmd.Context(), boardID)
	if err != nil {
		return fmt.Errorf(
			"listar issues do board %d: %w",
			boardID,
			err,
		)
	}
	if wantsJSON(cmd) {
		return writeJSON(out, issues)
	}

	printer.Header(fmt.Sprintf("📋 Issues do board %d", boardID))

	if len(issues) == 0 {
		printer.Info("Nenhuma issue aberta encontrada neste board.")
		fmt.Fprintln(out)
		return nil
	}

	printIssues(printer, issues, full)

	return nil
}
