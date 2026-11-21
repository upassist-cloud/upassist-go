package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	upassist "github.com/upassist-cloud/upassist-go"
)

func getAPIKey() string {
	key := os.Getenv("UPASSIST_API_KEY")
	if key == "" {
		fmt.Fprintln(os.Stderr, "Error: UPASSIST_API_KEY environment variable is required")
		os.Exit(1)
	}
	return key
}

func getBaseURL() string {
	url := os.Getenv("UPASSIST_BASE_URL")
	if url == "" {
		return "https://api.upassist.cloud"
	}
	return url
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "status", "whoami":
		handleStatus()
	case "heartbeats":
		handleHeartbeats(os.Args[2:])
	case "monitors":
		handleMonitors(os.Args[2:])
	case "incidents":
		handleIncidents(os.Args[2:])
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`Upassist CLI (Go Edition)

Usage:
  upassist status
  upassist heartbeats ping <slug> [--error] [--message <msg>]
  upassist heartbeats run <slug> -- <command...>
  upassist monitors list
  upassist incidents list

Environment Variables:
  UPASSIST_API_KEY   Required project API key
  UPASSIST_BASE_URL  Optional management host (default: https://api.upassist.cloud)`)
}

func handleStatus() {
	apiKey := getAPIKey()
	client, err := upassist.NewClient(getBaseURL())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.ProjectGet(context.Background(), func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+apiKey)
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "API error: HTTP %d\n", resp.StatusCode)
		os.Exit(1)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Project: %v (ID: %v)\n", result["name"], result["id"])
	fmt.Printf("Plan:    %v (Paid: %v)\n", result["tariff_name"], result["is_paid"])
	if quotas, ok := result["quotas"].(map[string]interface{}); ok {
		fmt.Println("Quotas:")
		fmt.Printf("  Monitors:   %v / %v\n", quotas["monitors_used"], quotas["monitors_limit"])
		fmt.Printf("  Heartbeats: %v / %v\n", quotas["heartbeats_used"], quotas["heartbeats_limit"])
		fmt.Printf("  Sites:      %v / %v\n", quotas["sites_used"], quotas["sites_limit"])
	}
}

func handleHeartbeats(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: upassist heartbeats <ping|run|list>")
		os.Exit(1)
	}

	sub := args[0]
	apiKey := getAPIKey()
	client, err := upassist.NewClient(getBaseURL())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Client error: %v\n", err)
		os.Exit(1)
	}
	authHook := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+apiKey)
		return nil
	}

	switch sub {
	case "list":
		params := &upassist.HeartbeatListParams{}
		resp, err := client.HeartbeatList(context.Background(), params, authHook)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var res map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&res)
		if items, ok := res["data"].([]interface{}); ok {
			fmt.Printf("Heartbeats (%d items):\n", len(items))
			for _, item := range items {
				if m, ok := item.(map[string]interface{}); ok {
					fmt.Printf("  * %v (slug: %v)\n", m["name"], m["slug"])
				}
			}
		}

	case "ping":
		fs := flag.NewFlagSet("ping", flag.ExitOnError)
		isErr := fs.Bool("error", false, "Report error ping")
		msg := fs.String("message", "", "Status or error message")
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: upassist heartbeats ping <slug> [--error] [--message <msg>]")
			os.Exit(1)
		}
		slug := args[1]
		fs.Parse(args[2:])

		_, err := upassist.HeartbeatPing(context.Background(), apiKey, slug, *isErr, *msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ping failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Ping successfully sent for '%s'\n", slug)

	case "run":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: upassist heartbeats run <slug> -- <command...>")
			os.Exit(1)
		}
		slug := args[1]
		var cmdArgs []string
		for i, a := range args[2:] {
			if a == "--" {
				cmdArgs = args[2+i+1:]
				break
			}
		}
		if len(cmdArgs) == 0 {
			cmdArgs = args[2:]
		}

		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		runErr := cmd.Run()

		if runErr != nil {
			errText := fmt.Sprintf("Command failed: %s (%v)", strings.Join(cmdArgs, " "), runErr)
			_, _ = upassist.HeartbeatPing(context.Background(), apiKey, slug, true, errText)
			os.Exit(1)
		} else {
			_, _ = upassist.HeartbeatPing(context.Background(), apiKey, slug, false, "")
			os.Exit(0)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown heartbeats subcommand: %s\n", sub)
		os.Exit(1)
	}
}

func handleMonitors(args []string) {
	apiKey := getAPIKey()
	client, err := upassist.NewClient(getBaseURL())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Client error: %v\n", err)
		os.Exit(1)
	}
	resp, err := client.MonitorList(context.Background(), &upassist.MonitorListParams{}, func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+apiKey)
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	if items, ok := res["data"].([]interface{}); ok {
		fmt.Printf("Monitors (%d items):\n", len(items))
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				fmt.Printf("  * %v -> %v (id: %v)\n", m["name"], m["url"], m["id"])
			}
		}
	}
}

func handleIncidents(args []string) {
	apiKey := getAPIKey()
	client, err := upassist.NewClient(getBaseURL())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Client error: %v\n", err)
		os.Exit(1)
	}
	resp, err := client.IncidentList(context.Background(), &upassist.IncidentListParams{}, func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+apiKey)
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	if items, ok := res["data"].([]interface{}); ok {
		fmt.Printf("Incidents (%d items):\n", len(items))
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				fmt.Printf("  * %v (id: %v)\n", m["summary"], m["id"])
			}
		}
	}
}
