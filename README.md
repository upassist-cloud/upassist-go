# Upassist Go SDK & CLI

Official Go client library and command-line interface for the [Upassist](https://upassist.cloud) Public Management & Ingestion API.

## Installation

```bash
go get github.com/upassist/upassist-go
```

## CLI Tool

The Go module includes a high-performance compiled CLI:

```bash
cd cmd/upassist
go build -o upassist .
./upassist --help
```

### CLI Examples

```bash
export UPASSIST_API_KEY="up_live_your_api_key_here"

# Check authentication
upassist whoami

# List monitors
upassist monitors

# Send a cron heartbeat ping
upassist ping my-cron-job --status up --msg "Job finished successfully"
```

## Usage

### 1. Management Client

```go
package main

import (
	"context"
	"fmt"
	"log"

	upassist "github.com/upassist/upassist-go"
)

func main() {
	client, err := upassist.NewClientWithResponses(
		"https://upassist.cloud",
		upassist.WithRequestEditorFn(upassist.WithBearerToken("up_live_your_api_key_here")),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	resp, err := client.MonitorListWithResponse(ctx, nil)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}

	if resp.JSON200 != nil {
		fmt.Printf("Monitors count: %d\n", len(*resp.JSON200))
	}
}
```

### 2. High-Throughput Ingestion (Pings, Logs, Metrics)

```go
// Heartbeat Ping
err := upassist.HeartbeatPing(ctx, "https://upassist.cloud", "up_live_api_key", "backup-job", upassist.HeartbeatPingPayload{
	Status:     "up",
	DurationMs: 340,
	Message:    "Nightly backup done",
})

// Collect Logs
err = upassist.CollectLogs(ctx, "https://upassist.cloud", "up_live_api_key", "billing-worker", []map[string]any{
	{
		"level":   "INFO",
		"message": "Invoice #102 generated",
	},
})

// Collect Metrics
err = upassist.CollectMetrics(ctx, "https://upassist.cloud", "up_live_api_key", []map[string]any{
	{
		"name":  "queue_depth",
		"value": 42,
		"type":  "gauge",
	},
})
```

## License

MIT
