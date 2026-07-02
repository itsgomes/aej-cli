package services

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrInvalidIssueKey = errors.New("chave de issue inválida")
	ErrEmptySearchTerm = errors.New("termo de busca vazio")
)

var issueKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*-[1-9][0-9]*$`)

func normalizeIssueKey(raw string) (string, error) {
	key := strings.ToUpper(strings.TrimSpace(raw))
	if !issueKeyPattern.MatchString(key) {
		return "", fmt.Errorf("%w: %q", ErrInvalidIssueKey, raw)
	}

	return key, nil
}

func jqlStringLiteral(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + replacer.Replace(value) + `"`
}
