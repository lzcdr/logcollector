package logcollector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hpcloud/tail"
)

type LogEntry struct {
	Timestamp time.Time `json:"@timestamp"`
	Message   string    `json:"message"`
	File      string    `json:"file"`
}

type ZincPayload struct {
	Index   string     `json:"index"`
	Records []LogEntry `json:"records"`
}

type LogCollector struct {
	FilePath   string
	ZincURL    string
	ZincUser   string
	ZincPass   string
	ZincIndex  string
	BufferSize int
	buffer     []LogEntry
	mutex      sync.Mutex
	watcher    *fsnotify.Watcher
	ctx        context.Context
	cancel     context.CancelFunc
	timer      *time.Timer
}

func NewLogCollector(
	ctx context.Context,
	zincIndexName string,
	filePath, zincURL, zincUser, zincPass string,
	bufferSize int,
	bufferTimeout time.Duration) (*LogCollector, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	lc := &LogCollector{
		FilePath:   filePath,
		ZincURL:    fmt.Sprintf("%s/api/_bulkv2", zincURL),
		ZincUser:   zincUser,
		ZincPass:   zincPass,
		ZincIndex:  zincIndexName,
		BufferSize: bufferSize,
		buffer:     make([]LogEntry, 0, bufferSize),
		watcher:    watcher,
		ctx:        ctx,
		cancel:     cancel,
	}

	lc.timer = time.AfterFunc(10*time.Second, func() { lc.flushBuffer() })

	go lc.watchFile()
	go lc.tailFile()

	return lc, nil
}

func (lc *LogCollector) watchFile() {
	defer lc.watcher.Close()
	_ = lc.watcher.Add(lc.FilePath)
	for {
		select {
		case <-lc.ctx.Done():
			return
		case event, ok := <-lc.watcher.Events:
			if !ok || event.Op&fsnotify.Write == 0 {
				continue
			}
		}
	}
}

func (lc *LogCollector) tailFile() {
	for {
		t, err := tail.TailFile(lc.FilePath, tail.Config{
			Follow: true,
			ReOpen: true,
			Poll:   true,
		})
		if err != nil {
			fmt.Println("Error tailing file:", err)
			return
		}

		for line := range t.Lines {
			select {
			case <-lc.ctx.Done():
				return
			default:
				lc.processLog(line.Text)
			}
		}
		t.Cleanup()
	}
}

func (lc *LogCollector) processLog(log string) {
	var entry LogEntry
	if json.Valid([]byte(log)) {
		json.Unmarshal([]byte(log), &entry)
	} else {
		entry = LogEntry{Timestamp: time.Now(), Message: log, File: lc.FilePath}
	}
	lc.addToBuffer(entry)
}

func (lc *LogCollector) addToBuffer(entry LogEntry) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	lc.buffer = append(lc.buffer, entry)
	if len(lc.buffer) >= lc.BufferSize {
		lc.flushBuffer()
	} else {
		lc.resetFlushTimer()
	}
}

func (lc *LogCollector) resetFlushTimer() {
	if lc.timer != nil {
		lc.timer.Stop()
	}
	lc.timer = time.AfterFunc(10*time.Second, func() { lc.flushBuffer() })
}

func (lc *LogCollector) flushBuffer() {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	if len(lc.buffer) == 0 {
		return
	}

	data, err := json.Marshal(ZincPayload{
		Index:   lc.ZincIndex,
		Records: lc.buffer,
	})
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}
	fmt.Printf("data:%v\n", string(data))

	req, err := http.NewRequest("POST", lc.ZincURL, bytes.NewBuffer(data))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.SetBasicAuth(lc.ZincUser, lc.ZincPass)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending logs to ZincSearch:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("ZincSearch response error:", string(body))
		return
	}

	lc.buffer = nil
}

func (lc *LogCollector) Stop() {
	lc.cancel()
	lc.flushBuffer()
	lc.watcher.Close()
}
