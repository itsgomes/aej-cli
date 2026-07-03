package cmd

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/itsgomes/aej-cli/internal/config"
	jiraclient "github.com/itsgomes/aej-cli/internal/jira"
)

type requestTimingReporter struct {
	enabled bool
	output  func() io.Writer
	mutex   sync.Mutex
}

func newRequestTimingReporter(output func() io.Writer) *requestTimingReporter {
	return &requestTimingReporter{output: output}
}

func (reporter *requestTimingReporter) Observe(metric jiraclient.RequestMetric) {
	if !reporter.enabled {
		return
	}

	reporter.mutex.Lock()
	defer reporter.mutex.Unlock()

	duration := metric.Duration.Round(time.Millisecond)

	if duration == 0 && metric.Duration > 0 {
		duration = metric.Duration.Round(time.Microsecond)
	}

	fmt.Fprintf(
		reporter.output(),
		"⏱ %s %s — status %d — %s — tentativa %d\n",
		metric.Method,
		metric.Path,
		metric.StatusCode,
		duration,
		metric.Attempt,
	)
}

func withRequestObserver(deps Dependencies, observer jiraclient.RequestObserver) Dependencies {
	originalNewService := deps.NewService
	originalNewAuthenticator := deps.NewAuthenticator

	deps.NewService = func(cfg *config.Config, options ...jiraclient.Option) Service {
		options = append(options, jiraclient.WithRequestObserver(observer))

		return originalNewService(cfg, options...)
	}

	deps.NewAuthenticator = func(cfg *config.Config, options ...jiraclient.Option) Authenticator {
		options = append(options, jiraclient.WithRequestObserver(observer))

		return originalNewAuthenticator(cfg, options...)
	}

	return deps
}
