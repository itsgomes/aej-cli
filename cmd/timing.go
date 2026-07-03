package cmd

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/itsgomes/aej-cli/internal/cli"
	"github.com/spf13/cobra"
)

type commandTimingReporter struct {
	enabled bool
	output  func() io.Writer
}

func newCommandTimingReporter(output func() io.Writer) *commandTimingReporter {
	return &commandTimingReporter{output: output}
}

func (reporter *commandTimingReporter) Wrap(command *cobra.Command) {
	originalRunE := command.RunE

	if originalRunE == nil {
		return
	}

	command.RunE = func(cmd *cobra.Command, args []string) error {
		if !reporter.enabled {
			return originalRunE(cmd, args)
		}

		startedAt := time.Now()

		defer func() {
			elapsed := time.Since(startedAt)

			printer := cli.NewPrinter(reporter.output(), reporter.output())

			printer.Printf(
				"%s %s %s\n",
				cli.Colorize(cli.Cyan, "⏱"),
				cli.Colorize(cli.Gray, "Tempo de execução:"),
				cli.Colorize(cli.Bold+cli.Cyan, formatCommandDuration(elapsed)),
			)
		}()

		return originalRunE(cmd, args)
	}
}

func formatCommandDuration(duration time.Duration) string {
	if duration < time.Second {
		milliseconds := duration.Round(time.Millisecond)

		if milliseconds == 0 {
			return "<1 ms"
		}

		return fmt.Sprintf("%d ms", milliseconds/time.Millisecond)
	}

	seconds := fmt.Sprintf("%.1f", duration.Seconds())
	seconds = strings.Replace(seconds, ".", ",", 1)

	return seconds + " s"
}
