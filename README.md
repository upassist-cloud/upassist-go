# Upassist Go SDK & CLI

Official Go client library and command-line interface for the [Upassist](https://upassist.cloud) Monitoring, Uptime, Cron Heartbeats, Error Tracking & Observability Platform.

[![Go Reference](https://pkg.go.dev/badge/github.com/upassist-cloud/upassist-go.svg)](https://pkg.go.dev/github.com/upassist-cloud/upassist-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

- **Complete API Coverage**: Monitors, Heartbeats, Incidents, Maintenance Windows, Error Tracking, Status Pages, Dashboards, Members, and API Keys.
- **Type-Safe & idiomatic**: Generated directly from Upassist OpenAPI 3.1 specifications via `oapi-codegen`.
- **High-Throughput Ingestion**: Lightweight HTTP helpers for heartbeat pings, structured log batches, and metric submission.
- **Compiled CLI Tool**: Standalone `upassist` binary for Linux, macOS, and Windows.

---

## Installation

```bash
go get github.com/upassist-cloud/upassist-go
```

---

## CLI Tool

### Installation

```bash
# Install directly via Go:
go install github.com/upassist-cloud/upassist-go/cmd/upassist@latest

# Or build locally:
cd cmd/upassist
go build -o upassist .
```

### CLI Practical Examples

```bash
export UPASSIST_API_KEY="up_live_your_api_key_here"

# 1. Verify credentials & check quota
upassist status

# 2. List monitors
upassist monitors list

# 3. List active incidents
upassist incidents list

# 4. Cron Heartbeat: Send a ping directly
upassist heartbeats ping my-backup-job --message "Nightly database backup completed"

# 5. Cron Heartbeat: Automatically wrap and monitor any command!
# (Records duration, intercepts exit code, and reports down on failure)
upassist heartbeats run my-backup-job -- /usr/local/bin/backup-script.sh --all
```

---

## Practical Go Examples

### 1. Management Client Setup

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	upassist "github.com/upassist-cloud/upassist-go"
)

func main() {
	apiKey := os.Getenv("UPASSIST_API_KEY")

	client, err := upassist.NewClientWithResponses(
		"https://api.upassist.cloud",
		upassist.WithRequestEditorFn(upassist.WithBearerToken(apiKey)),
	)
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}

	ctx := context.Background()

	// List all monitors
	resp, err := client.MonitorListWithResponse(ctx, nil)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}

	if resp.JSON200 != nil {
		for _, m := range resp.JSON200.Items {
			fmt.Printf("[%s] %s -> %s\n", m.Status, m.Name, m.Url)
		}
	}
}
```

---

### 2. Cron Job & Worker Monitoring (Heartbeats)

Report heartbeats from Go worker routines or cron jobs:

```go
package main

import (
	"context"
	"log"
	"time"

	upassist "github.com/upassist-cloud/upassist-go"
)

func runWorkerTask(ctx context.Context, apiKey string) {
	slug := "nightly-cleanup-job"
	start := time.Now()

	// Execute worker logic...
	err := doHeavyWork()

	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		// Report failure
		_ = upassist.HeartbeatPing(ctx, "https://heartbeats.upassist.cloud", apiKey, slug, upassist.HeartbeatPingPayload{
			Status:     "down",
			DurationMs: durationMs,
			Message:    err.Error(),
		})
		log.Fatalf("Task failed: %v", err)
	}

	// Report success
	_ = upassist.HeartbeatPing(ctx, "https://heartbeats.upassist.cloud", apiKey, slug, upassist.HeartbeatPingPayload{
		Status:     "up",
		DurationMs: durationMs,
		Message:    "Processed 5000 items successfully",
	})
}

func doHeavyWork() error {
	time.Sleep(500 * time.Millisecond)
	return nil
}
```

---

### 3. Deploy-Time Maintenance Window (Silence Alerts)

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	upassist "github.com/upassist-cloud/upassist-go"
)

func silenceAlertsDuringDeploy(client *upassist.ClientWithResponses) {
	ctx := context.Background()
	start := time.Now().UTC()
	finish := start.Add(15 * time.Minute)

	body := upassist.MaintenanceCreateSchema{
		Name:        "Kubernetes Cluster Upgrade",
		StartedAt:   start.Format(time.RFC3339),
		FinishedAt:  finish.Format(time.RFC3339),
		Timezone:    "UTC",
	}

	resp, err := client.MaintenanceCreateWithResponse(ctx, body)
	if err != nil || resp.JSON201 == nil {
		log.Fatalf("Failed to create maintenance window: %v", err)
	}

	windowID := resp.JSON201.Id
	fmt.Printf("Maintenance window #%s created until %s\n", windowID, finish)

	// ... Perform rolling deploy ...

	// Clear maintenance early if deploy finished:
	_, _ = client.MaintenanceDeleteWithResponse(ctx, windowID)
	fmt.Println("Alerting resumed.")
}
```

---

### 4. High-Throughput Ingestion (Logs & Metrics)

Stream structured logs and custom metrics directly to Upassist telemetry collectors:

```go
// 1. Send structured logs
err := upassist.CollectLogs(ctx, "https://logs.upassist.cloud", apiKey, "order-service", []map[string]any{
	{
		"level":     "INFO",
		"message":   "Order #1234 settled",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"context": map[string]any{
			"order_id": 1234,
			"amount":   49.99,
		},
	},
})

// 2. Send custom metrics
err = upassist.CollectMetrics(ctx, "https://metrics.upassist.cloud", apiKey, []map[string]any{
	{
		"name":  "active_goroutines",
		"value": 142,
		"type":  "gauge",
		"labels": map[string]string{"env": "prod"},
	},
})
```

---

## License

MIT
