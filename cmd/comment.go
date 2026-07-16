package cmd

import (
	"fmt"
	"strings"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newCommentCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:     "comment <CHAVE> <COMENTÁRIO>",
		Short:   "Adicionar um comentário a uma issue",
		Example: `  aej comment DEV-123 "Correção disponível para validação"`,
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := strings.ToUpper(strings.TrimSpace(args[0]))
			comment := strings.Join(args[1:], " ")
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}
			if err := deps.NewService(cfg).AddComment(cmd.Context(), issueKey, comment); err != nil {
				return fmt.Errorf("adicionar comentário em %s: %w", issueKey, err)
			}
			cli.NewPrinter(cmd.OutOrStdout(), cmd.ErrOrStderr()).Success(
				fmt.Sprintf("Comentário adicionado em %s.", cli.Colorize(cli.Cyan, issueKey)),
			)
			return nil
		},
	}
}
