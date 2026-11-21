package upassist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	HeartbeatsURL = "https://heartbeats.upassist.cloud"
	LogsURL       = "https://logs.upassist.cloud"
	MetricsURL    = "https://metrics.upassist.cloud"
	RumURL        = "https://rum.upassist.cloud"
)

func HeartbeatPing(ctx context.Context, apiKey, slug string, failed bool, message string) (string, error) {
	query := url.Values{}
	if failed {
		query.Set("error", "true")
	}
	if message != "" {
		query.Set("errorMessage", message)
	}
	endpoint := HeartbeatsURL + "/api/event/" + url.PathEscape(slug)
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	return do(request)
}

func CollectLogs(ctx context.Context, token string, entries any) (string, error) {
	return postJSON(ctx, LogsURL+"/collect", token, entries)
}

func CollectMetrics(ctx context.Context, token string, points any) (string, error) {
	return postJSON(ctx, MetricsURL+"/collect", token, points)
}

func TrackSiteEvent(ctx context.Context, event any) (string, error) {
	return postJSON(ctx, RumURL+"/api/sites/events", "", event)
}

func postJSON(ctx context.Context, endpoint, token string, payload any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	return do(request)
}

func do(request *http.Request) (string, error) {
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	if response.StatusCode >= 400 {
		return "", fmt.Errorf("%s", payload)
	}
	return string(payload), nil
}
