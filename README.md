# LogCollector for ZincSearch

## Overview
LogCollector is a lightweight Go library designed to efficiently collect logs from a file and send them to [ZincSearch](https://zincsearch.com). It supports:
- **Log file monitoring** (including log rotation handling)
- **Buffering logs** before sending them in bulk
- **Custom log processing functions** (e.g., parsing, formatting)
- **JSON and plain-text log support**

## Installation
```sh
go get github.com/lzcdr/logcollector
```

## Usage
### Basic Example
```go
package main

import (
	"context"
	"fmt"
	"logcollector"
	"time"
)

func main() {
	ctx := context.Background()
	logFile := "/var/log/app.log"
	zincURL := "http://localhost:4080"
	zincUser := "admin"
	zincPass := "admin123"
	bufferSize := 10
	bufferTimeout := 10 * time.Second
	zincIndex := "logs"

	collector, err := logcollector.NewLogCollector(ctx, zincIndex, logFile, zincURL, zincUser, zincPass, bufferSize, bufferTimeout, nil)
	if err != nil {
		fmt.Println("Error initializing log collector:", err)
		return
	}

	defer collector.Stop()

	select {} // Keep running
}
```

### Using a Custom Log Processor
You can define a function to transform logs before sending:
```go
func processLog(line string) string {
	return fmt.Sprintf("Processed: %s", line)
}

collector, err := logcollector.NewLogCollector(ctx, zincIndex, logFile, zincURL, zincUser, zincPass, bufferSize, bufferTimeout, processLog)
```

## Features
- **Handles log rotation**: Uses `fsnotify` to detect file changes
- **Custom processing**: Allows defining a `LogLineProcessingFunc`
- **Bulk sending**: Sends logs efficiently in batches
- **Timeout-based flushing**: Ensures logs are sent even if the buffer is not full
- **Handle plain-text and json logs**

## Configuration
| Parameter       | Type                      | Description |
|---------------|-------------------------|-------------|
| `ctx`         | `context.Context`        | Context for managing lifecycle |
| `zincIndex`   | `string`                 | ZincSearch index name |
| `filePath`    | `string`                 | Path to the log file |
| `zincURL`     | `string`                 | ZincSearch server URL |
| `zincUser`    | `string`                 | ZincSearch username |
| `zincPass`    | `string`                 | ZincSearch password |
| `bufferSize`  | `int`                     | Number of logs to buffer before sending |
| `bufferTimeout` | `time.Duration`         | Time before forcing a flush |
| `logProcFunc` | `LogLineProcessingFunc` | Optional function for processing logs |

## License
MIT License. See [LICENSE](LICENSE) for details.

