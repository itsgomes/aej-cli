package cmd

import (
	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/itsgomes/aej-cli/internal/models"
)

func printIssues(printer *cli.Printer, issues []models.Issue, full bool) {
	if full {
		printIssueBlocks(printer, issues)
		return
	}

	printIssueTable(printer, issues)
}

func printIssueTable(printer *cli.Printer, issues []models.Issue) {
	rows := make([][]string, 0, len(issues))

	for _, issue := range issues {
		priority, priorityColor := formatIssuePriority(issue)
		statusColor := cli.StatusColor(issue.Fields.Status.StatusCategory.Key)

		rows = append(rows, []string{
			cli.Colorize(cli.Cyan, cli.SanitizeInline(issue.Key)),
			cli.Truncate(cli.SanitizeInline(issue.Fields.Summary), 60),
			cli.Colorize(statusColor, cli.SanitizeInline(issue.Fields.Status.Name)),
			cli.Colorize(priorityColor, priority),
		})
	}

	printer.Table(
		[]string{"Chave", "Resumo", "Status", "Prioridade"},
		rows,
	)
}

func printIssueBlocks(printer *cli.Printer, issues []models.Issue) {
	for _, issue := range issues {
		priority, priorityColor := formatIssuePriority(issue)
		statusColor := cli.StatusColor(issue.Fields.Status.StatusCategory.Key)

		printer.Field(
			"Chave",
			cli.Colorize(cli.Cyan, cli.SanitizeInline(issue.Key)),
		)

		printer.Field(
			"Resumo",
			cli.SanitizeInline(issue.Fields.Summary),
		)

		printer.Field(
			"Status",
			cli.Colorize(statusColor, cli.SanitizeInline(issue.Fields.Status.Name)),
		)

		printer.Field(
			"Prioridade",
			cli.Colorize(priorityColor, priority),
		)

		printer.Printf("\n")
	}
}

func formatIssuePriority(issue models.Issue) (string, string) {
	priority := "-"
	priorityColor := cli.Gray

	if issue.Fields.Priority != nil {
		priority = cli.SanitizeInline(issue.Fields.Priority.Name)
		priorityColor = cli.PriorityColor(priority)
	}

	return priority, priorityColor
}
