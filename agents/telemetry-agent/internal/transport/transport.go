package transport

// transport.go is responsible for taking an event from the buffer later and sending it to the Gateway over HTTP.
// Convert a telemetry event into an HTTP request, authenticate it, send it to the Gateway, and report whether the Gateway accepted it.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Aryan-Desai12/sentinel-ai/agents/telemetry-agent/internal/model"
)

type Transport struct {
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

func New(gatewayURL, apiKey string) *Transport {
	return &Transport{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		gatewayURL: gatewayURL,
		apiKey:     apiKey,
	}
}

func (t *Transport) Send(ctx context.Context, event model.Telemetry) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal telemetry: %w", err)
	}

	url := t.gatewayURL + "/metrics/node"

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(data),
	)
	if err != nil {
		return fmt.Errorf("create telemetry request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", t.apiKey)

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("send telemetry request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusAccepted:
		return nil

	case http.StatusBadRequest:
		return &Error{
			StatusCode: resp.StatusCode,
			Message:    "gateway rejected telemetry: bad request",
		}

	case http.StatusUnauthorized:
		return &Error{
			StatusCode: resp.StatusCode,
			Message:    "gateway rejected telemetry: unauthorized",
		}

	case http.StatusServiceUnavailable:
		return &Error{
			StatusCode: resp.StatusCode,
			Message:    "gateway unavailable",
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

//             TELEMETRY AGENT
//                    │
//                    ▼
//              Collect metrics
//                    │
//                    ▼
//             Create Telemetry
//                    │
//                    ▼
//                 Validate
//                    │
//                    ▼
//                 Buffer
//                    │
//                    ▼
//                 Peek()
//                    │
//                    ▼
//             transport.Send()
//                    │
//           ┌────────┴────────┐
//           │                 │
//           ▼                 ▼
//       HTTP POST          Network error
//           │                 │
//           ▼                 │
//        Gateway              │
//           │                 │
//    ┌──────┼──────┐          │
//    │      │      │          │
//   202    400    401/503     │
//    │      │      │          │
//    ▼      ▼      ▼          ▼
// success  error  error      retry
//    │
//    ▼
// Remove()

        //       Validated Event
        //             ↓
        //           Buffer
        //             ↓
        //           Peek
        //             ↓
        //      transport.Send()
        //             ↓
        //      ┌──────┴──────┐
        //      │             │
        //   Success        Failure
        //     202              │
        //      │         ┌─────┴─────┐
        //      │         │           │
        //      │      Temporary   Permanent
        //      │       failure     failure
        //      │         │           │
        //      │       Retry        Remove
        //      │         │           │
        //      └─────→ Success       ↓
        //                │          Next event
        //                ↓
        //             Remove

// permanent" here means permanent for the current delivery attempt/payload, not necessarily that the entire system is permanently broken. For example, a 401 might be fixed later by updating the API key, but we don't want the current event blocking the entire queue while waiting for that configuration fix.
// So our design prioritizes:
// Don't lose events unnecessarily, but also don't let one undeliverable event block all newer events.
