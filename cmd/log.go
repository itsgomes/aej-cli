package cmd

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newLogCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "log <CHAVE> <TEMPO> [COMENTÁRIO]",
		Short: "Registrar tempo trabalhado em uma issue",
		Long: `Registra um worklog na issue informada.

Formatos de tempo aceitos:
2h          → 2 horas
	30m         → 30 minutos
	1h 30m      → 1 hora e 30 minutos (use aspas: "1h 30m")
	1h30m       → convertido automaticamente para "1h 30m"
1d          → 1 dia (conforme configuração do Jira)`,
		Example: "  aej log DEV-123 2h\n  aej log DEV-123 30m \"Revisão de código\"\n  aej log DEV-123 \"1h 30m\" \"Implementando feature\"",
		Args:    cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLog(deps, cmd, args)
		},
	}
}

var ErrInvalidTimeSpent = errors.New("formato de tempo inválido")

var timeTokenPattern = regexp.MustCompile(`(?i)(\d+)([wdhm])`)

func runLog(deps Dependencies, cmd *cobra.Command, args []string) error {
	printer := cli.NewPrinter(cmd.OutOrStdout(), cmd.ErrOrStderr())

	cfg, err := deps.LoadConfig()

	if err != nil {
		return err
	}

	issueKey := strings.ToUpper(args[0])
	timeSpent, err := normalizeTimeSpent(args[1])
	if err != nil {
		return err
	}
	comment := ""

	if len(args) == 3 {
		comment = args[2]
	}

	svc := deps.NewService(cfg)

	if err := svc.AddWorklog(cmd.Context(), issueKey, timeSpent, comment); err != nil {
		return fmt.Errorf("registrar tempo: %w", err)
	}

	printer.Success(fmt.Sprintf("Tempo %s registrado em %s.", cli.BoldText(timeSpent), cli.Colorize(cli.Cyan, issueKey)))
	return nil
}

func normalizeTimeSpent(raw string) (string, error) {
	matches := timeTokenPattern.FindAllStringSubmatchIndex(raw, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("%w: use valores como 2h, 30m ou 1h 30m", ErrInvalidTimeSpent)
	}

	unitOrder := map[string]int{"w": 0, "d": 1, "h": 2, "m": 3}
	lastEnd := 0
	lastOrder := -1
	parts := make([]string, 0, len(matches))

	for _, match := range matches {
		if strings.TrimSpace(raw[lastEnd:match[0]]) != "" {
			return "", fmt.Errorf("%w: %q", ErrInvalidTimeSpent, raw)
		}

		value, err := strconv.Atoi(raw[match[2]:match[3]])
		if err != nil || value <= 0 {
			return "", fmt.Errorf("%w: valores devem ser positivos", ErrInvalidTimeSpent)
		}

		unit := strings.ToLower(raw[match[4]:match[5]])
		order := unitOrder[unit]
		if order <= lastOrder {
			return "", fmt.Errorf("%w: unidades repetidas ou fora de ordem", ErrInvalidTimeSpent)
		}

		parts = append(parts, fmt.Sprintf("%d%s", value, unit))
		lastOrder = order
		lastEnd = match[1]
	}

	if strings.TrimSpace(raw[lastEnd:]) != "" {
		return "", fmt.Errorf("%w: %q", ErrInvalidTimeSpent, raw)
	}

	return strings.Join(parts, " "), nil
}
