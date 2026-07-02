package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/itsgomes/aej-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newLoginCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Configurar credenciais do Jira",
		Long:  "Solicita URL, e-mail e API Token, valida a conexão e salva as credenciais localmente.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(deps, cmd, args)
		},
	}
}

func runLogin(deps Dependencies, cmd *cobra.Command, _ []string) error {
	in := cmd.InOrStdin()
	out := cmd.OutOrStdout()
	printer := cli.NewPrinter(out, cmd.ErrOrStderr())

	printer.Header("🔑 Configurar Jira")
	fmt.Fprintln(out, "  Configure suas credenciais para acessar o Jira.")
	fmt.Fprintln(out)

	reader := bufio.NewReader(in)

	fmt.Fprint(out, "  URL do Jira (ex: https://empresa.atlassian.net): ")
	jiraURL, err := readLine(reader)
	if err != nil {
		return fmt.Errorf("ler URL do Jira: %w", err)
	}

	fmt.Fprint(out, "  E-mail: ")
	email, err := readLine(reader)
	if err != nil {
		return fmt.Errorf("ler e-mail: %w", err)
	}

	apiToken, err := readSecret(in, reader, out, "  API Token: ")

	if err != nil {
		return fmt.Errorf("ler API Token: %w", err)
	}

	if jiraURL == "" || email == "" || apiToken == "" {
		return errors.New("URL, e-mail e API Token são obrigatórios")
	}

	cfg, err := config.New(jiraURL, email, apiToken)
	if err != nil {
		return fmt.Errorf("validar configuração: %w", err)
	}

	fmt.Fprintln(out)
	fmt.Fprint(out, "  Validando credenciais...")

	user, err := deps.NewAuthenticator(cfg).GetCurrentUser(cmd.Context())

	if err != nil {
		fmt.Fprintln(out)
		return fmt.Errorf("validar credenciais: %w", err)
	}

	printer.Printf(" %s\n\n", cli.Colorize(cli.Green, "OK"))

	if err := deps.SaveConfig(*cfg); err != nil {
		return fmt.Errorf("salvar configuração: %w", err)
	}

	printer.Success(fmt.Sprintf("Olá, %s! Configuração salva com sucesso.", cli.BoldText(cli.SanitizeInline(user.DisplayName))))
	fmt.Fprintln(out)
	return nil
}

func readSecret(in io.Reader, reader *bufio.Reader, out io.Writer, prompt string) (string, error) {
	fmt.Fprint(out, prompt)

	if file, ok := in.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		b, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(out)

		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(b)), nil
	}

	return readLine(reader)
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	return strings.TrimSpace(line), nil
}
