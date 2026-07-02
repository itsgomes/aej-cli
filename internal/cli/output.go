package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/itsgomes/aej-cli/internal/models"
	"golang.org/x/term"
)

type Printer struct {
	out      io.Writer
	err      io.Writer
	colorOut bool
	colorErr bool
}

func NewPrinter(out, errOut io.Writer) *Printer {
	return &Printer{
		out:      out,
		err:      errOut,
		colorOut: supportsColor(out),
		colorErr: supportsColor(errOut),
	}
}

const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Gray    = "\033[90m"
)

func Colorize(color, text string) string {
	return color + text + Reset
}

func BoldText(text string) string {
	return Bold + text + Reset
}

func SanitizeText(text string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
}

func SanitizeInline(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, text)
}

func (p *Printer) Success(msg string) {
	p.writeOut(fmt.Sprintf("%s %s\n", Colorize(Green, "✔"), msg))
}

func (p *Printer) Error(msg string) {
	p.writeErr(fmt.Sprintf("%s %s\n", Colorize(Red, "✗"), msg))
}

func (p *Printer) Info(msg string) {
	p.writeOut(fmt.Sprintf("%s %s\n", Colorize(Blue, "ℹ"), msg))
}

func (p *Printer) Header(title string) {
	p.writeOut(fmt.Sprintf("\n%s\n%s\n", Colorize(Bold+Cyan, title), Colorize(Gray, strings.Repeat("─", 50))))
}

func (p *Printer) Field(label, value string) {
	p.writeOut(fmt.Sprintf("  %s %s\n", Colorize(Gray, label+":"), value))
}

func (p *Printer) Printf(format string, args ...any) {
	p.writeOut(fmt.Sprintf(format, args...))
}

func StatusColor(categoryKey string) string {
	switch strings.ToLower(categoryKey) {
	case "done":
		return Green
	case "indeterminate":
		return Yellow
	default:
		return Gray
	}
}

func PriorityColor(priority string) string {
	switch strings.ToLower(priority) {
	case "highest", "critical", "blocker":
		return Red
	case "high":
		return Magenta
	case "medium":
		return Yellow
	case "low":
		return Blue
	default:
		return Gray
	}
}

func Truncate(s string, maxLen int) string {
	runes := []rune(s)

	if len(runes) <= maxLen {
		return s
	}

	return string(runes[:maxLen-1]) + "…"
}

func (p *Printer) Table(headers []string, rows [][]string) {
	widths := make([]int, len(headers))

	for i, h := range headers {
		widths[i] = utf8.RuneCountInString(h)
	}

	for _, row := range rows {
		for i := range headers {
			if i < len(row) {
				l := utf8.RuneCountInString(stripANSI(row[i]))

				if l > widths[i] {
					widths[i] = l
				}
			}
		}
	}

	p.writeOut("  ")

	for i, h := range headers {
		p.writeOut(fmt.Sprintf("%s%s  ", Colorize(Bold+Cyan, h), strings.Repeat(" ", widths[i]-utf8.RuneCountInString(h))))
	}

	p.writeOut("\n")

	p.writeOut("  ")

	for _, w := range widths {
		p.writeOut(fmt.Sprintf("%s  ", Colorize(Gray, strings.Repeat("─", w))))
	}

	p.writeOut("\n")

	for _, row := range rows {
		p.writeOut("  ")

		for i := range headers {
			cell := ""

			if i < len(row) {
				cell = row[i]
			}

			padding := widths[i] - utf8.RuneCountInString(stripANSI(cell))
			p.writeOut(fmt.Sprintf("%s%s  ", cell, strings.Repeat(" ", padding)))
		}
		p.writeOut("\n")
	}
	p.writeOut("\n")
}

func (p *Printer) writeOut(text string) {
	if !p.colorOut {
		text = stripANSI(text)
	}
	_, _ = io.WriteString(p.out, text)
}

func (p *Printer) writeErr(text string) {
	if !p.colorErr {
		text = stripANSI(text)
	}
	_, _ = io.WriteString(p.err, text)
}

func supportsColor(writer io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	file, ok := writer.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func FormatDate(dateStr string) string {
	if dateStr == "" {
		return "—"
	}

	formats := []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05.000-0700", "2006-01-02T15:04:05.999-0700"}

	for _, f := range formats {
		if t, err := time.Parse(f, dateStr); err == nil {
			return t.Format("02/01/2006 15:04")
		}
	}

	return dateStr
}

func ExtractDescription(document *models.ADFDocument) string {
	if document == nil {
		return "—"
	}

	parts := make([]string, 0, len(document.Content))
	for _, node := range document.Content {
		if text := extractADFText(node); text != "" {
			parts = append(parts, text)
		}
	}

	text := strings.TrimSpace(strings.Join(parts, ""))

	if text == "" {
		return "—"
	}

	return text
}

func extractADFText(node models.ADFNode) string {
	if node.Type == "text" {
		return node.Text
	}

	if len(node.Content) == 0 {
		return ""
	}

	parts := make([]string, 0, len(node.Content))
	for _, child := range node.Content {
		if text := extractADFText(child); text != "" {
			parts = append(parts, text)
		}
	}

	switch node.Type {
	case "paragraph", "heading", "bulletList", "orderedList", "listItem", "blockquote", "codeBlock":
		return strings.Join(parts, "") + "\n"
	default:
		return strings.Join(parts, "")
	}
}

func FormatSeconds(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60

	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

func stripANSI(s string) string {
	var b strings.Builder

	inEsc := false

	for _, r := range s {
		switch {
		case r == '\033':
			inEsc = true
		case inEsc && r == 'm':
			inEsc = false
		case !inEsc:
			b.WriteRune(r)
		}
	}

	return b.String()
}
