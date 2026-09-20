package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"errors"
	

	"sentinel-ai/agents/synthetic-monitor/internal/model"
)

type Client struct {
	client     *http.Client
	gatewayURL string
	apiKey     string
}

type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	return e.Message
}

func NewClient(
	httpClient *http.Client,
	gatewayURL string,
	apiKey string,
) *Client {
	return &Client{
		client:     httpClient,
		gatewayURL: gatewayURL,
		apiKey:     apiKey,
	}
}

func (c *Client) Send(
	ctx context.Context,
	event model.SyntheticCheckEvent,
) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf(
			"marshal synthetic check: %w",
			err,
		)
	}

	url := c.gatewayURL + "/checks/synthetic"

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(data),
	)
	if err != nil {
		return fmt.Errorf(
			"create synthetic check request: %w",
			err,
		)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf(
			"send synthetic check request: %w",
			err,
		)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusAccepted:
		return nil

	case http.StatusBadRequest:
		return &Error{
			StatusCode: resp.StatusCode,
			Message: "gateway rejected synthetic check: bad request",
		}

	case http.StatusUnauthorized:
		return &Error{
			StatusCode: resp.StatusCode,
			Message: "gateway rejected synthetic check: unauthorized",
		}

	case http.StatusServiceUnavailable:
		return &Error{
			StatusCode: resp.StatusCode,
			Message: "gateway unavailable",
		}

	default:
		return &Error{
			StatusCode: resp.StatusCode,
			Message: fmt.Sprintf(
				"unexpected gateway response: %s",
				resp.Status,
			),
		}
	}
}

func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	var clientErr *Error

	if errors.As(err, &clientErr) {
		switch {
		case clientErr.StatusCode == http.StatusServiceUnavailable:
			return true

		case clientErr.StatusCode >= 500:
			return true

		case clientErr.StatusCode >= 400 &&
			clientErr.StatusCode < 500:
			return false

		default:
			return true
		}
	}

	// Network, DNS, TLS, timeout, etc.
	// are retryable delivery failures.
	return true
}