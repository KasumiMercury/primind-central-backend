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
	queueName  string
	httpClient *http.Client
	maxRetries int
}

func NewPrimindTasksClient(baseURL, queueName string, maxRetries int) *PrimindTasksClient {
	if maxRetries <= 0 {
		maxRetries = 3
	}

	return &PrimindTasksClient{
		baseURL:   baseURL,
		queueName: queueName,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries: maxRetries,
	}
}

func (c *PrimindTasksClient) RegisterRemind(ctx context.Context, req *CreateRemindRequest) (*RemindResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal remind request: %w", err)
	}

	encodedBody := base64.StdEncoding.EncodeToString(payload)

	primindReq := PrimindTaskRequest{
		Task: PrimindTask{
			HTTPRequest: PrimindHTTPRequest{
				Body: encodedBody,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
	}

	reqBody, err := json.Marshal(primindReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal primind request: %w", err)
	}

	url := fmt.Sprintf("%s/tasks", c.baseURL)
	if c.queueName != "" && c.queueName != "default" {
		url = fmt.Sprintf("%s/tasks/%s", c.baseURL, c.queueName)
	}

	var lastErr error

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 100 * time.Millisecond
			slog.Debug("retrying remind registration",
				slog.String("task_id", req.TaskID),
				slog.Int("attempt", attempt+1),
				slog.Duration("backoff", backoff),
			)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := c.doRequest(ctx, url, reqBody, req.TaskID)
		if err == nil {
			return resp, nil
		}

		lastErr = err
	}

	slog.Error("all retries exhausted for remind registration",
		slog.String("task_id", req.TaskID),
		slog.Int("max_retries", c.maxRetries),
		slog.String("error", lastErr.Error()),
	)

	return nil, fmt.Errorf("failed to register remind after %d retries: %w", c.maxRetries, lastErr)
}

func (c *PrimindTasksClient) doRequest(ctx context.Context, url string, reqBody []byte, taskID string) (*RemindResponse, error) {
	slog.Debug("registering remind to Primind Tasks",
		slog.String("url", url),
		slog.String("task_id", taskID),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Warn("failed to send request to Primind Tasks",
			slog.String("task_id", taskID),
			slog.String("error", err.Error()),
		)

		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		slog.Warn("unexpected status code from Primind Tasks",
			slog.String("task_id", taskID),
			slog.Int("status_code", resp.StatusCode),
		)

		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var primindResp PrimindTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&primindResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	createTime, _ := time.Parse(time.RFC3339, primindResp.CreateTime)

	slog.Info("remind task registered to Primind Tasks",
		slog.String("task_name", primindResp.Name),
		slog.String("task_id", taskID),
	)

	return &RemindResponse{
		Name:       primindResp.Name,
		CreateTime: createTime,
	}, nil
}
