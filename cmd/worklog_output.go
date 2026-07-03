package cmd

import (
	"fmt"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/itsgomes/aej-cli/internal/models"
)

func printWorklogs(printer *cli.Printer, summaries []models.IssueWorklogSummary, full bool) {
	if full {
		printWorklogBlocks(printer, summaries)
		return
	}

	printWorklogTable(printer, summaries)
}

func printWorklogTable(printer *cli.Printer, summaries []models.IssueWorklogSummary) {
	rows := make([][]string, 0)

	for _, summary := range summaries {
		for _, entry := range summary.Entries {
			comment := cli.ExtractDescription(entry.Comment)
			comment = cli.SanitizeInline(comment)
			comment = cli.Truncate(comment, 40)

			rows = append(rows, []string{
				cli.Colorize(cli.Cyan, cli.SanitizeInline(summary.IssueKey)),
				cli.Truncate(cli.SanitizeInline(summary.Summary), 40),
				cli.FormatDate(entry.Started),
				cli.Colorize(cli.Green, cli.SanitizeInline(entry.TimeSpent)),
				comment,
			})
		}
	}

	printer.Table(
		[]string{"Issue", "Resumo", "Data", "Tempo", "comentário"},
		rows,
	)
}

func printWorklogBlocks(printer *cli.Printer, summaries []models.IssueWorklogSummary) {
	for _, summary := range summaries {
		printer.Field(
			"Issue",
			cli.Colorize(cli.Cyan, cli.SanitizeInline(summary.IssueKey)),
		)

		printer.Field(
			"Resumo",
			cli.SanitizeInline(summary.Summary),
		)

		for index, entry := range summary.Entries {
			title := fmt.Sprintf("Registro %d", index+1)

			printer.Printf(
				"\n %s\n",
				cli.Colorize(cli.Bold+cli.Cyan, title),
			)

			printer.Field(
				"Data",
				cli.FormatDate(entry.Started),
			)

			printer.Field(
				"Tempo",
				cli.Colorize(
					cli.Green,
					cli.SanitizeInline(entry.TimeSpent),
				),
			)

			printer.Field(
				"Comentário",
				cli.SanitizeInline(cli.ExtractDescription((entry.Comment))),
			)
		}
	}

	printer.Printf("\n")
}
