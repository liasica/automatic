// Copyright (C) automatic. 2026-present.
//
// Created at 2026-03-04, by liasica

package feishu

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"resty.dev/v3"
)

// defaultRetryIntervals defines the default retry wait intervals: 1s, 5s, 10s after each failure.
var defaultRetryIntervals = []time.Duration{
	1 * time.Second,
	5 * time.Second,
	10 * time.Second,
}

// HttpClientOption is a functional option for HttpClient.
type HttpClientOption func(*HttpClient)

// WithRetryIntervals sets custom retry wait intervals.
// Each element is the wait duration before the corresponding retry attempt.
func WithRetryIntervals(intervals ...time.Duration) HttpClientOption {
	return func(h *HttpClient) {
		h.retryIntervals = intervals
	}
}

// HttpClient implements larkcore.HttpClient using resty v3,
// with configurable fixed-interval retry logic.
type HttpClient struct {
	client         *resty.Client
	retryIntervals []time.Duration
}

// NewHttpClient creates a new HttpClient with resty v3 and retry support.
func NewHttpClient(opts ...HttpClientOption) *HttpClient {
	h := &HttpClient{
		retryIntervals: defaultRetryIntervals,
	}
	for _, opt := range opts {
		opt(h)
	}

	retryCount := len(h.retryIntervals)
	intervals := h.retryIntervals

	c := resty.New().
		SetRetryCount(retryCount).
		SetAllowNonIdempotentRetry(true).
		EnableRetryDefaultConditions()
	c.SetRetryStrategy(func(resp *resty.Response, _ error) (time.Duration, error) {
		if len(intervals) == 0 {
			return 0, nil
		}
		idx := 0
		if resp != nil && resp.Request != nil && resp.Request.Attempt > 0 {
			// resty attempts start at 1, while slice indexes start at 0.
			idx = resp.Request.Attempt - 1
		}
		if idx >= len(intervals) {
			idx = len(intervals) - 1
		}
		return intervals[idx], nil
	})

	h.client = c
	return h
}

// Do implements larkcore.HttpClient. It executes the given http.Request
// through resty with retry logic applied.
func (h *HttpClient) Do(req *http.Request) (*http.Response, error) {
	// Read and buffer the body so it can be replayed on retries.
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		_ = req.Body.Close()
	}

	r := h.client.R().
		SetContext(req.Context()).
		SetHeaderMultiValues(req.Header).
		SetDoNotParseResponse(true)

	if len(bodyBytes) > 0 {
		r.SetBody(bytes.NewReader(bodyBytes))
	}

	resp, err := r.Execute(req.Method, req.URL.String())
	if err != nil {
		return nil, err
	}

	return resp.RawResponse, nil
}
