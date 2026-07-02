package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/itsgomes/aej-cli/internal/models"
)

func TestPrinterRoutesOutput(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	printer := NewPrinter(&stdout, &stderr)

	printer.Info("informação")
	printer.Error("falha")

	if !strings.Contains(stripANSI(stdout.String()), "informação") {
		t.Errorf("stdout = %q, want info message", stdout.String())
	}
	if strings.Contains(stdout.String(), "falha") {
		t.Errorf("stdout = %q, must not contain error", stdout.String())
	}
	if !strings.Contains(stripANSI(stderr.String()), "falha") {
		t.Errorf("stderr = %q, want error message", stderr.String())
	}
	if strings.Contains(stdout.String(), "\x1b") || strings.Contains(stderr.String(), "\x1b") {
		t.Errorf("buffer output contains ANSI escapes: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestPrinterTablePadsColoredHeaders(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	printer := NewPrinter(&stdout, &bytes.Buffer{})
	printer.Table([]string{"A", "B"}, [][]string{{"long", "x"}})

	firstLine := strings.Split(stripANSI(stdout.String()), "\n")[0]
	if firstLine != "  A     B  " {
		t.Errorf("header = %q, want aligned columns", firstLine)
	}
}

func TestExtractDescriptionFromTypedADF(t *testing.T) {
	t.Parallel()

	document := &models.ADFDocument{
		Type:    "doc",
		Version: 1,
		Content: []models.ADFNode{
			{Type: "paragraph", Content: []models.ADFNode{{Type: "text", Text: "Olá "}, {Type: "text", Text: "mundo"}}},
			{Type: "paragraph", Content: []models.ADFNode{{Type: "text", Text: "Segunda linha"}}},
		},
	}

	if got, want := ExtractDescription(document), "Olá mundo\nSegunda linha"; got != want {
		t.Errorf("ExtractDescription() = %q, want %q", got, want)
	}
	if got := ExtractDescription(nil); got != "—" {
		t.Errorf("ExtractDescription(nil) = %q, want em dash", got)
	}
}

func TestSanitizeUntrustedTerminalText(t *testing.T) {
	t.Parallel()

	input := "issue\x1b]0;owned\x07\nnext"
	if got, want := SanitizeText(input), "issue]0;owned\nnext"; got != want {
		t.Errorf("SanitizeText() = %q, want %q", got, want)
	}
	if got, want := SanitizeInline(input), "issue ]0;owned  next"; got != want {
		t.Errorf("SanitizeInline() = %q, want %q", got, want)
	}
}
