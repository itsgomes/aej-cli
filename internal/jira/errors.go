package jira

import (
	"errors"
	"fmt"
)

var (
	ErrBadRequest   = errors.New("requisição rejeitada pelo Jira")
	ErrUnauthorized = errors.New("autenticação falhou; verifique o e-mail e o API Token")
	ErrForbidden    = errors.New("acesso negado; verifique suas permissões no Jira")
	ErrNotFound     = errors.New("recurso não encontrado no Jira")
	ErrGone         = errors.New("endpoint removido pela API do Jira; atualize o AEJ-CLI")
	ErrRateLimited  = errors.New("limite de requisições do Jira excedido; tente novamente mais tarde")
	ErrUnavailable  = errors.New("servidor Jira indisponível; tente novamente mais tarde")
	ErrUnexpected   = errors.New("resposta inesperada da API do Jira")
)

// APIError records an unsuccessful HTTP response from the Jira API.
type APIError struct {
	StatusCode int
	Err        error
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s (HTTP %d)", e.Err, e.StatusCode)
}

func (e *APIError) Unwrap() error {
	return e.Err
}
