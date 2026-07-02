package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/itsgomes/aej-cli/internal/cli"
	jiraclient "github.com/itsgomes/aej-cli/internal/jira"
	"github.com/spf13/cobra"
)

func newIssueCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:     "issue <CHAVE>",
		Short:   "Exibir detalhes de uma issue",
		Example: "  aej issue DEV-123",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIssue(deps, cmd, args)
		},
	}
}

func runIssue(deps Dependencies, cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	printer := cli.NewPrinter(out, cmd.ErrOrStderr())

	cfg, err := deps.LoadConfig()

	if err != nil {
		return err
	}

	svc := deps.NewService(cfg)
	issue, err := svc.GetIssue(cmd.Context(), args[0])

	if err != nil {
		if errors.Is(err, jiraclient.ErrNotFound) {
			return fmt.Errorf("issue %q não encontrada: %w", strings.ToUpper(args[0]), err)
		}

		return fmt.Errorf("obter issue %q: %w", strings.ToUpper(args[0]), err)
	}

	f := issue.Fields

	priority := "—"
	priorityColor := cli.Gray

	if f.Priority != nil {
		priority = cli.SanitizeInline(f.Priority.Name)
		priorityColor = cli.PriorityColor(priority)
	}

	assignee := "Não atribuído"

	if f.Assignee != nil {
		assignee = cli.SanitizeInline(f.Assignee.DisplayName)
	}

	labels := "—"

	if len(f.Labels) > 0 {
		labels = cli.SanitizeInline(strings.Join(f.Labels, ", "))
	}

	statusColor := cli.StatusColor(f.Status.StatusCategory.Key)

	description := cli.ExtractDescription(f.Description)
	description = cli.SanitizeText(description)
	description = strings.TrimSpace(description)

	if len([]rune(description)) > 400 {
		runes := []rune(description)
		description = string(runes[:400]) + "…"
	}

	printer.Header(fmt.Sprintf("🎫 %s — %s", cli.SanitizeInline(issue.Key), cli.SanitizeInline(f.Summary)))
	printer.Field("Tipo", cli.SanitizeInline(f.IssueType.Name))
	printer.Field("Status", cli.Colorize(statusColor, cli.SanitizeInline(f.Status.Name)))
	printer.Field("Prioridade", cli.Colorize(priorityColor, priority))
	printer.Field("Responsável", assignee)
	printer.Field("Labels", labels)
	printer.Field("Criado em", cli.FormatDate(f.Created))
	printer.Field("Atualizado em", cli.FormatDate(f.Updated))

	fmt.Fprintln(out)
	printer.Printf("  %s\n", cli.Colorize(cli.Gray, "Descrição:"))

	for _, line := range strings.Split(description, "\n") {
		if strings.TrimSpace(line) != "" {
			printer.Printf("    %s\n", line)
		}
	}

	fmt.Fprintln(out)
	return nil
}
