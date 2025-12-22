//go:build !gcloud

package taskqueue

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"time"
)

type PrimindTasksClient struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
}

func NewPrimindTasksClient(baseURL string, maxRetries int) *PrimindTasksClient {
	if maxRetries <= 0 {
		maxRetries = 3
	}

	return &PrimindTasksClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries: maxRetries,
	}
}

func (c *PrimindTasksClient) CreateTask(ctx context.Context, req CreateTaskRequest) (*TaskResponse, error) {
	encodedBody := base64.StdEncoding.EncodeToString(req.Payload)

	primindReq := PrimindTaskRequest{
		Task: PrimindTask{
			HTTPRequest: PrimindHTTPRequest{
				Body:    encodedBody,
				Headers: req.Headers,
			},
		},
	}

	reqBody, err := json.Marshal(primindReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal primind request: %w", err)
	}

	url := fmt.Sprintf("%s/tasks", c.baseURL)
	if req.QueuePath != "" && req.QueuePath != "default" {
		url = fmt.Sprintf("%s/tasks/%s", c.baseURL, req.QueuePath)
	}

	var lastErr error

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 100 * time.Millisecond
			slog.Debug("retrying task creation",
				slog.Int("attempt", attempt+1),
				slog.Duration("backoff", backoff),
			)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := c.doRequest(ctx, url, reqBody)
		if err == nil {
			return resp, nil
		}

		lastErr = err
	}

	slog.Error("all retries exhausted for task creation",
		slog.Int("max_retries", c.maxRetries),
		slog.String("error", lastErr.Error()),
	)

	return nil, fmt.Errorf("failed to create task after %d retries: %w", c.maxRetries, lastErr)
}

func (c *PrimindTasksClient) doRequest(ctx context.Context, url string, reqBody []byte) (*TaskResponse, error) {
	slog.Debug("creating task in Primind Tasks",
		slog.String("url", url),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Warn("failed to send request to Primind Tasks",
			slog.String("error", err.Error()),
		)

		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("failed to close response body", slog.String("error", err.Error()))
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		slog.Warn("unexpected status code from Primind Tasks",
			slog.Int("status_code", resp.StatusCode),
		)

		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var primindResp PrimindTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&primindResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	createTime, parseErr := time.Parse(time.RFC3339, primindResp.CreateTime)
	if parseErr != nil {
		slog.Warn("failed to parse create time, using zero value",
			slog.String("raw_value", primindResp.CreateTime),
			slog.String("error", parseErr.Error()),
		)
	}

	slog.Debug("task created in Primind Tasks",
		slog.String("task_name", primindResp.Name),
	)

	return &TaskResponse{
		Name:       primindResp.Name,
		CreateTime: createTime,
	}, nil
}

func (c *PrimindTasksClient) Close() error {
	return nil
}
