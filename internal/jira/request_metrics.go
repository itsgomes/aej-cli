package jira

import (
	"time"

	"github.com/go-resty/resty/v2"
)

type RequestMetric struct {
	Method     string
	Path       string
	StatusCode int
	Duration   time.Duration
	Attempt    int
}

type RequestObserver func(RequestMetric)

func WithRequestObserver(observer RequestObserver) Option {
	return func(client *Client) {
		if observer == nil {
			return
		}

		registerRequestObserver(client.http, observer)
		registerRequestObserver(client.writeHTTP, observer)
	}
}

func registerRequestObserver(httpClient *resty.Client, observer RequestObserver) {
	httpClient.OnAfterResponse(func(_ *resty.Client, response *resty.Response) error {
		path := response.Request.URL

		if rawRequest := response.Request.RawRequest; rawRequest != nil {
			path = rawRequest.URL.Path
		}

		observer(RequestMetric{
			Method:     response.Request.Method,
			Path:       path,
			StatusCode: response.StatusCode(),
			Duration:   response.Time(),
			Attempt:    response.Request.Attempt,
		})

		return nil
	})
}
