package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/itsgomes/aej-cli/internal/models"
	"github.com/spf13/cobra"
)

type logsPeriod struct {
	from  time.Time
	to    time.Time
	label string
}

func newLogsCommand(deps Dependencies) *cobra.Command {
	var full bool
	var days int
	var date string

	command := &cobra.Command{
		Use:   "logs",
		Short: "Exibir tempo trabalhado em um período",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			daysWasProvided := cmd.Flags().Changed("days")

			if daysWasProvided && date != "" {
				return errors.New("--days e --date não podem ser usados juntos")
			}

			return runLogs(deps, cmd, args, full, days, date)
		},
	}

	command.Flags().BoolVarP(
		&full,
		"full",
		"f",
		false,
		"Exibir cada issue em bloco, sem truncar os campos",
	)

	command.Flags().IntVar(
		&days,
		"days",
		7,
		"Período em dias: 7, 15 ou 30",
	)

	command.Flags().StringVar(
		&date,
		"date",
		"",
		"Data específica no formato DD-MM-YYYY",
	)

	return command
}

func resolveLogsPeriod(days int, date string, now time.Time) (logsPeriod, error) {
	location := now.Location()

	if date != "" {
		selectedDate, err := time.ParseInLocation("02-01-2006", date, location)

		if err != nil {
			return logsPeriod{}, errors.New("data inválida: use o formato DD-MM-YYYY")
		}

		return logsPeriod{
			from:  selectedDate,
			to:    selectedDate.AddDate(0, 0, 1),
			label: "Dia " + selectedDate.Format("02/01/2006"),
		}, nil
	}

	switch days {
	case 7, 15, 30:
	default:
		return logsPeriod{}, errors.New("--days aceita somente 7, 15 ou 30")
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	from := today.AddDate(0, 0, -(days - 1))
	to := today.AddDate(0, 0, 1)

	return logsPeriod{
		from:  from,
		to:    to,
		label: fmt.Sprintf("Últimos %d dias", days),
	}, nil
}

func runLogs(deps Dependencies, cmd *cobra.Command, _ []string, full bool, days int, date string) error {
	period, err := resolveLogsPeriod(days, date, time.Now())

	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	printer := cli.NewPrinter(out, cmd.ErrOrStderr())

	cfg, err := deps.LoadConfig()

	if err != nil {
		return err
	}

	svc := deps.NewService(cfg)

	summaries, totalSeconds, err := svc.GetWorklogs(cmd.Context(), period.from, period.to)

	if err != nil {
		return fmt.Errorf("obter registros de tempo: %w", err)
	}
	if wantsJSON(cmd) {
		return writeJSON(out, struct {
			From         time.Time                    `json:"from"`
			To           time.Time                    `json:"to"`
			TotalSeconds int                          `json:"totalSeconds"`
			Issues       []models.IssueWorklogSummary `json:"issues"`
		}{period.from, period.to, totalSeconds, summaries})
	}

	printer.Header("⏱ Tempo Trabalhado — " + period.label)

	if len(summaries) == 0 {
		printer.Info("Nenhum registro de tempo encontrado — " + period.label + ".")

		fmt.Fprintln(out)
		return nil
	}

	printWorklogs(printer, summaries, full)

	printer.Printf(
		"  %s %s\n\n",
		cli.Colorize(cli.Gray, "Total:"),
		cli.Colorize(cli.Bold+cli.Green, cli.FormatSeconds(totalSeconds)),
	)

	return nil
}
