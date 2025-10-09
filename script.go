package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ANSI color codes
var (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
)

// HistogramBucket represents a single bucket in a response time histogram.
type HistogramBucket struct {
	Mark  float64 `json:"mark"`
	Count int     `json:"count"`
}

// Metrics holds the collected data from the load test.
type Metrics struct {
	SuccessCount    int64
	FailureCount    int64
	ResponseTimes   []float64
	StatusCodeCount map[int]int
	Histogram       []*HistogramBucket
	ErrorLog        []string
	Lock            sync.Mutex
}

// Summary holds the final calculated results of the load test.
type Summary struct {
	TotalRequestsSent   int64               `json:"totalRequestsSent"`
	SuccessfulRequests  int64               `json:"successfulRequests"`
	FailedRequests      int64               `json:"failedRequests"`
	SuccessRate         float64             `json:"successRate"`
	FailureRate         float64             `json:"failureRate"`
	TotalTimeTaken      float64             `json:"totalTimeTaken"`
	RequestsPerSecond   float64             `json:"requestsPerSecond"`
	AvgResponseTime     float64             `json:"avgResponseTime"`
	MinResponseTime     float64             `json:"minResponseTime"`
	MaxResponseTime     float64             `json:"maxResponseTime"`
	Percentile90        float64             `json:"percentile90"`
	Percentile99        float64             `json:"percentile99"`
	StatusCodeDist      map[int]int         `json:"statusCodeDistribution"`
	Histogram           []*HistogramBucket  `json:"histogram"`
	ErrorSummary        []string            `json:"errorSummary"`
}

// customHeaders is a custom flag type for handling multiple header flags.
type customHeaders []string

func (h *customHeaders) String() string {
	return strings.Join(*h, ", ")
}

func (h *customHeaders) Set(value string) error {
	*h = append(*h, value)
	return nil
}

var (
	metrics          *Metrics
	histogramBuckets = []float64{0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0}
)

func initializeMetrics() {
	metrics = &Metrics{
		StatusCodeCount: make(map[int]int),
		ResponseTimes:   make([]float64, 0),
		ErrorLog:        make([]string, 0),
		Histogram:       make([]*HistogramBucket, len(histogramBuckets)+1),
	}
	for i, mark := range histogramBuckets {
		metrics.Histogram[i] = &HistogramBucket{Mark: mark}
	}
	metrics.Histogram[len(histogramBuckets)] = &HistogramBucket{Mark: math.Inf(1)}

	// Disable colors on Windows
	if runtime.GOOS == "windows" {
		ColorReset = ""
		ColorRed = ""
		ColorGreen = ""
		ColorYellow = ""
		ColorCyan = ""
	}
}

func main() {
	initializeMetrics()

	// --- Command-Line Flags ---
	url := flag.String("url", "", "The target URL to test. (Required)")
	requests := flag.Int("requests", 0, "Total number of requests to send. Incompatible with -duration.")
	concurrency := flag.Int("concurrency", 10, "Number of concurrent requests to send.")
	duration := flag.Duration("duration", 0, "Duration of the test (e.g., '60s', '5m'). Incompatible with -requests.")
	method := flag.String("method", "GET", "HTTP method to use (e.g., GET, POST).")
	body := flag.String("body", "", "Request body for POST, PUT, etc. Incompatible with -body-file.")
	bodyFile := flag.String("body-file", "", "Path to a file containing the request body. Incompatible with -body.")
	outputFile := flag.String("output", "", "Path to save the summary report as a JSON file.")
	var headers customHeaders
	flag.Var(&headers, "header", "Custom header(s) to send with requests (can be specified multiple times). Format: 'Key:Value'")

	flag.Parse()

	// --- Input Validation ---
	if *url == "" {
		fmt.Println("Error: -url is required.")
		flag.Usage()
		os.Exit(1)
	}

	// Prepend https:// if no scheme is provided
	if !strings.HasPrefix(*url, "http://") && !strings.HasPrefix(*url, "https://") {
		*url = "https://" + *url
	}

	if *requests > 0 && *duration > 0 {
		fmt.Println("Error: -requests and -duration are mutually exclusive. Please choose one.")
		os.Exit(1)
	}
	if *requests == 0 && *duration == 0 {
		fmt.Println("Error: Either -requests or -duration must be specified.")
		os.Exit(1)
	}

	// --- Setup Context for Graceful Shutdown ---
	ctx, cancel := context.WithCancel(context.Background())
	if *duration > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), *duration)
	}
	defer cancel()

	// Listen for interrupt signals (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Printf("\n%sInterrupt signal received. Shutting down gracefully...%s\n", ColorYellow, ColorReset)
		cancel()
	}()

	// --- Test Execution ---
	var requestBody string
	if *body != "" && *bodyFile != "" {
		fmt.Println("Error: -body and -body-file are mutually exclusive. Please choose one.")
		os.Exit(1)
	} else if *bodyFile != "" {
		bodyBytes, err := ioutil.ReadFile(*bodyFile)
		if err != nil {
			fmt.Printf("Error reading body file: %v\n", err)
			os.Exit(1)
		}
		requestBody = string(bodyBytes)
	} else {
		requestBody = *body
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	startTime := time.Now()
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, *concurrency)

	go printLiveMetrics(ctx, startTime, *requests)

	worker := func() {
		defer wg.Done()
		defer func() { <-semaphore }()
		sendRequest(ctx, client, *method, *url, &headers, requestBody)
	}

	if *requests > 0 { // Fixed number of requests
		for i := 0; i < *requests; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				wg.Add(1)
				semaphore <- struct{}{}
				go worker()
			}
		}
	} else { // Duration-based test
		for {
			select {
			case <-ctx.Done():
				wg.Wait()
				printSummary(startTime, *outputFile)
				return
			default:
				wg.Add(1)
				semaphore <- struct{}{}
				go worker()
			}
		}
	}

	wg.Wait()
	printSummary(startTime, *outputFile)
}

func sendRequest(ctx context.Context, client *http.Client, method, url string, headers *customHeaders, body string) {
	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(body))
	if err != nil {
		metrics.Lock.Lock()
		metrics.FailureCount++
		metrics.ErrorLog = append(metrics.ErrorLog, fmt.Sprintf("error creating request: %v", err))
		metrics.Lock.Unlock()
		return
	}

	req.Header.Set("User-Agent", "httptest-load-tester/1.0")
	for _, h := range *headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	startTime := time.Now()
	resp, err := client.Do(req)
	elapsedTime := time.Since(startTime).Seconds()

	metrics.Lock.Lock()
	defer metrics.Lock.Unlock()

	metrics.ResponseTimes = append(metrics.ResponseTimes, elapsedTime)

	for _, bucket := range metrics.Histogram {
		if elapsedTime <= bucket.Mark {
			bucket.Count++
			break
		}
	}

	if err != nil {
		metrics.FailureCount++
		metrics.StatusCodeCount[0]++ // Representing client-side errors
		if len(metrics.ErrorLog) < 100 {
			metrics.ErrorLog = append(metrics.ErrorLog, err.Error())
		}
	} else {
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			metrics.SuccessCount++
		} else {
			metrics.FailureCount++
		}
		metrics.StatusCodeCount[resp.StatusCode]++
	}
}

func printLiveMetrics(ctx context.Context, startTime time.Time, totalRequests int) {
	spinner := []string{"|", "/", "-", "\\"}
	spinIdx := 0
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Print("")
			return
		case <-ticker.C:
			metrics.Lock.Lock()
			sent := metrics.SuccessCount + metrics.FailureCount
			elapsedTime := time.Since(startTime).Seconds()

			displayTotal := "/" + fmt.Sprint(totalRequests)
			if totalRequests == 0 {
				displayTotal = ""
			}

			avg := "N/A"
			p99 := "N/A"

			timesCopy := make([]float64, len(metrics.ResponseTimes))
			copy(timesCopy, metrics.ResponseTimes)

			if len(timesCopy) > 0 {
				sort.Float64s(timesCopy)
				avg = fmt.Sprintf("%.4fs", average(timesCopy))
				p99 = fmt.Sprintf("%.4fs", percentile(timesCopy, 99))
			}

			fmt.Printf("\r%s%s Requests Sent: %d%s | %sSuccess: %d%s | %sFailures: %d%s | Avg Resp: %s | 99th Pctl: %s | Elapsed: %.2fs%s ",
				ColorCyan, spinner[spinIdx], sent, displayTotal, ColorGreen, metrics.SuccessCount, ColorReset, ColorRed, metrics.FailureCount, ColorReset, avg, p99, elapsedTime, ColorReset)
			metrics.Lock.Unlock()

			spinIdx = (spinIdx + 1) % len(spinner)
		}
	}
}

func printSummary(startTime time.Time, outputFile string) {
	metrics.Lock.Lock()
	defer metrics.Lock.Unlock()

	elapsedTime := time.Since(startTime).Seconds()
	totalRequests := metrics.SuccessCount + metrics.FailureCount
	if totalRequests == 0 {
		fmt.Println("\nNo requests were sent.")
		return
	}

	finalResponseTimes := make([]float64, len(metrics.ResponseTimes))
	copy(finalResponseTimes, metrics.ResponseTimes)
	sort.Float64s(finalResponseTimes)

	avgResponse := average(finalResponseTimes)
	minResponse := min(finalResponseTimes)
	maxResponse := max(finalResponseTimes)
	p90 := percentile(finalResponseTimes, 90)
	p99 := percentile(finalResponseTimes, 99)

	summary := Summary{
		TotalRequestsSent:   totalRequests,
		SuccessfulRequests:  metrics.SuccessCount,
		FailedRequests:      metrics.FailureCount,
		SuccessRate:         (float64(metrics.SuccessCount) / float64(totalRequests)) * 100,
		FailureRate:         (float64(metrics.FailureCount) / float64(totalRequests)) * 100,
		TotalTimeTaken:      elapsedTime,
				RequestsPerSecond:   0.00,
		AvgResponseTime:     avgResponse,
		MinResponseTime:     minResponse,
		MaxResponseTime:     maxResponse,
		Percentile90:        p90,
		Percentile99:        p99,
		StatusCodeDist:      metrics.StatusCodeCount,
		Histogram:           metrics.Histogram,
		ErrorSummary:        metrics.ErrorLog,
	}
	if elapsedTime > 0 {
		summary.RequestsPerSecond = float64(totalRequests) / elapsedTime
	}

	// --- Console Output ---
	fmt.Printf("\n\n%sLoad Test Summary%s\n%s==================%s\n", ColorYellow, ColorReset, ColorYellow, ColorReset)
	fmt.Printf("Total Requests Sent      : %s%d%s\n", ColorCyan, summary.TotalRequestsSent, ColorReset)
	fmt.Printf("Successful Requests      : %s%d%s\n", ColorGreen, summary.SuccessfulRequests, ColorReset)
	fmt.Printf("Failed Requests          : %s%d%s\n", ColorRed, summary.FailedRequests, ColorReset)
	fmt.Printf("Success Rate             : %s%.2f%%%s\n", ColorGreen, summary.SuccessRate, ColorReset)
	fmt.Printf("Failure Rate             : %s%.2f%%%s\n", ColorRed, summary.FailureRate, ColorReset)
	fmt.Printf("Total Time Taken         : %.2f seconds\n", summary.TotalTimeTaken)
	fmt.Printf("Requests per Second      : %.2f\n", summary.RequestsPerSecond)

	fmt.Printf("\n%sResponse Time Metrics (seconds)%s\n%s--------------------------------%s\n", ColorYellow, ColorReset, ColorYellow, ColorReset)
	fmt.Printf("Average Response Time    : %s%.4f%s\n", ColorCyan, summary.AvgResponseTime, ColorReset)
	fmt.Printf("90th Percentile          : %.4f\n", summary.Percentile90)
	fmt.Printf("99th Percentile          : %.4f\n", summary.Percentile99)
	fmt.Printf("Minimum Response Time    : %.4f\n", summary.MinResponseTime)
	fmt.Printf("Maximum Response Time    : %.4f\n", summary.MaxResponseTime)

	printHistogram(summary.Histogram)

	fmt.Printf("\n%sStatus Code Distribution%s\n%s------------------------%s\n", ColorYellow, ColorReset, ColorYellow, ColorReset)
	for code, count := range summary.StatusCodeDist {
		color := ColorGreen
		if code == 0 || code >= 400 {
			color = ColorRed
		}
		if code == 0 {
			fmt.Printf("Client-Side Errors : %s%d responses%s\n", color, count, ColorReset)
		} else {
			fmt.Printf("Status Code %-7d : %s%d responses%s\n", code, color, count, ColorReset)
		}
	}

	if len(summary.ErrorSummary) > 0 {
		fmt.Printf("\n%sError Summary (first 100)%s\n%s--------------------------%s\n", ColorYellow, ColorReset, ColorYellow, ColorReset)
		limit := 100
		if len(summary.ErrorSummary) < limit {
			limit = len(summary.ErrorSummary)
		}
		for i, err := range summary.ErrorSummary[:limit] {
			fmt.Printf("%s%d. %s%s\n", ColorRed, i+1, err, ColorReset)
		}
	}

	// --- JSON File Output ---
	if outputFile != "" {
		jsonData, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			fmt.Printf("\nError marshalling summary to JSON: %v\n", err)
			return
		}
		err = ioutil.WriteFile(outputFile, jsonData, 0644)
		if err != nil {
			fmt.Printf("\nError writing summary to file '%s': %v\n", outputFile, err)
			return
		}
		fmt.Printf("\nSummary report saved to %s\n", outputFile)
	}
}

func printHistogram(histogram []*HistogramBucket) {
	fmt.Printf("\n%sResponse Time Distribution%s\n%s------------------------%s\n", ColorYellow, ColorReset, ColorYellow, ColorReset)
	maxCount := 0
	for _, bucket := range histogram {
		if bucket.Count > maxCount {
			maxCount = bucket.Count
		}
	}

	var lastMark float64
	for _, bucket := range histogram {
		bar := ""
		if maxCount > 0 {
			bar = strings.Repeat("▇", (bucket.Count*40)/maxCount)
		}

		if math.IsInf(bucket.Mark, 1) {
			fmt.Printf("[%s%.2fs+ %s] %s (%d)%s\n", ColorCyan, lastMark, ColorReset, bar, bucket.Count, ColorReset)
		} else {
			fmt.Printf("[%s%.2f-%.2fs%s] %s (%d)%s\n", ColorCyan, lastMark, bucket.Mark, ColorReset, bar, bucket.Count, ColorReset)
		}
		lastMark = bucket.Mark
	}
}

func average(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range data {
		sum += value
	}
	return sum / float64(len(data))
}

func min(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	minVal := data[0]
	for _, value := range data[1:] {
		if value < minVal {
			minVal = value
		}
	}
	return minVal
}

func max(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	maxVal := data[0]
	for _, value := range data[1:] {
		if value > maxVal {
			maxVal = value
		}
	}
	return maxVal
}

func percentile(data []float64, p float64) float64 {
	if len(data) == 0 {
		return 0
	}
	// Assumes data is sorted
	index := int(float64(len(data)) * (p / 100.0))
	if index >= len(data) {
		index = len(data) - 1
	}
	return data[index]
}
