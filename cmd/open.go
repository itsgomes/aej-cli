package cmd

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func newOpenCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:     "open <CHAVE>",
		Short:   "Abrir uma issue no navegador",
		Example: "  aej open DEV-123",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := strings.ToUpper(strings.TrimSpace(args[0]))
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}
			issueURL := strings.TrimRight(cfg.JiraURL, "/") + "/browse/" + url.PathEscape(issueKey)
			if err := deps.OpenURL(issueURL); err != nil {
				return fmt.Errorf("abrir %s no navegador: %w", issueKey, err)
			}
			return nil
		},
	}
}

func openURL(target string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		command = exec.Command("open", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	return command.Start()
}
