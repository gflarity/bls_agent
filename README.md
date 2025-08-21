# BLS Agent - Temporal.io Go Project

This is a boilerplate Temporal.io Go project that demonstrates basic workflow patterns including activities, signals, and queries.

## Project Structure

```
bls_agent/
├── go.mod                 # Go module file with Temporal dependencies
├── go.sum                 # Go dependencies checksums
├── cmd/
│   ├── worker/
│   │   └── main.go       # Initializes and runs the Temporal worker
│   └── starter/
│       └── main.go       # Initializes a client and starts workflows
├── internal/
│   ├── workflows/
│   │   └── bls/
│   │       ├── workflow.go   # Workflow definitions
│   │       └── activities.go # Activity definitions
│   └── config/
│       └── config.go     # Configuration loading from environment
└── README.md             # This file
```

## Prerequisites

-   Go 1.24.1 or later
-   Temporal server running locally or accessible remotely
-   Docker (optional, for running Temporal locally)

## Quick Start

### 1. Install Dependencies

```bash
go mod tidy
```

### 2. Set Environment Variables

Copy the example environment file and modify as needed:

```bash
cp env.example .env
```

Or create a `.env` file in the project root with the following content:

```bash
# Temporal Configuration
TEMPORAL_HOST_PORT=localhost:7233
TEMPORAL_NAMESPACE=default
TEMPORAL_TASK_QUEUE=my-task-queue
```

### 3. Run Temporal Server (Optional)

If you don't have a Temporal server running, you can start one using Docker:

```bash
docker run --rm -p 7233:7233 temporalio/auto-setup:1.22.3
```

### 4. Start the Worker

In one terminal, start the worker:

```bash
go run cmd/worker/main.go
```

### 5. Start Workflows

In another terminal, start workflows:

```bash
go run cmd/starter/main.go
```

## Features

### Workflows

-   **BLSReleaseSummaryWorkflow**: Main workflow that generates BLS release summaries
-   **MyWorkflowWithSignals**: Demonstrates signal handling and timeouts
-   **MyWorkflowWithQuery**: Shows how to query workflow state

### Activities

-   **MyActivity**: Simple activity with logging
-   **MyActivityWithRetry**: Demonstrates retry policies
-   **LongRunningActivity**: Simulates long-running work
-   **DataProcessingActivity**: Processes arrays of data

### Configuration

The application loads configuration from environment variables with sensible defaults:

-   `TEMPORAL_HOST_PORT`: Temporal server address (default: localhost:7233)
-   `TEMPORAL_NAMESPACE`: Temporal namespace (default: default)
-   `TEMPORAL_TASK_QUEUE`: Task queue name (default: my-task-queue)

## Development

### Adding New Workflows

1. Define the workflow function in `internal/workflows/bls/workflow.go`
2. Register it in `cmd/worker/main.go`
3. Use it in `cmd/starter/main.go`

### Adding New Activities

1. Define the activity function in `internal/workflows/bls/activities.go`
2. Register it in `cmd/worker/main.go`
3. Call it from your workflows

### Testing

Run the tests:

```bash
go test ./...
```

## Dependencies

-   `go.temporal.io/sdk`: Temporal Go SDK
-   `go.temporal.io/api`: Temporal API definitions
-   `github.com/joho/godotenv`: Environment variable loading

## License

This project is part of the BLS Agent codebase.
