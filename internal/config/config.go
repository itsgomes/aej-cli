package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	keyJiraURL  = "jira_url"
	keyEmail    = "email"
	keyAPIToken = "api_token"
)

var (
	ErrNotConfigured = errors.New("configuração não encontrada; execute 'aej login' para configurar")
	ErrIncomplete    = errors.New("configuração incompleta; execute 'aej login' para reconfigurar")
	ErrInvalidURL    = errors.New("endereço do Jira inválido")
)

type Config struct {
	JiraURL  string
	Email    string
	APIToken string
}

func New(jiraURL, email, apiToken string) (*Config, error) {
	normalizedURL, err := NormalizeJiraURL(jiraURL)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		JiraURL:  normalizedURL,
		Email:    strings.TrimSpace(email),
		APIToken: strings.TrimSpace(apiToken),
	}

	if cfg.Email == "" || cfg.APIToken == "" {
		return nil, ErrIncomplete
	}

	return cfg, nil
}

func NormalizeJiraURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("%w: valor vazio", ErrInvalidURL)
	}

	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	if !strings.EqualFold(parsed.Scheme, "https") {
		return "", fmt.Errorf("%w: use HTTPS", ErrInvalidURL)
	}
	if parsed.Hostname() == "" {
		return "", fmt.Errorf("%w: host ausente", ErrInvalidURL)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("%w: credenciais não são permitidas na URL", ErrInvalidURL)
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("%w: informe somente a URL-base, sem caminho", ErrInvalidURL)
	}
	if parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return "", fmt.Errorf("%w: query e fragmento não são permitidos", ErrInvalidURL)
	}

	parsed.Scheme = "https"
	parsed.Path = ""
	return parsed.String(), nil
}

func Load() (*Config, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}

	v := newViper(dir)

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("ler configuração: %w", err)
		}
	}

	jiraURL := v.GetString(keyJiraURL)
	email := v.GetString(keyEmail)
	apiToken := v.GetString(keyAPIToken)

	if jiraURL == "" && email == "" && apiToken == "" {
		return nil, ErrNotConfigured
	}

	return New(jiraURL, email, apiToken)
}

func Save(cfg Config) error {
	validated, err := New(cfg.JiraURL, cfg.Email, cfg.APIToken)
	if err != nil {
		return err
	}

	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("criar diretório de configuração: %w", err)
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return fmt.Errorf("proteger diretório de configuração: %w", err)
	}

	v := newViper(dir)
	v.Set(keyJiraURL, validated.JiraURL)
	v.Set(keyEmail, validated.Email)
	v.Set(keyAPIToken, validated.APIToken)

	configFile := filepath.Join(dir, "config.yaml")

	if err := v.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf("salvar configuração: %w", err)
	}

	return os.Chmod(configFile, 0600)
}

func newViper(dir string) *viper.Viper {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(dir)
	v.SetEnvPrefix("AEJ")
	v.AutomaticEnv()

	return v
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("obter diretório do usuário: %w", err)
	}

	if strings.TrimSpace(home) == "" {
		return "", errors.New("diretório do usuário está vazio")
	}

	return filepath.Join(home, ".aej-cli"), nil
}
