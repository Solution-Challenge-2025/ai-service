package analytics

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	chunkSize      = 8000 // characters per chunk for Gemini API
	geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent"
)

type AnalyticsService struct {
	apiKey string
}

type LogEntry struct {
	Timestamp string            `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Path      string            `json:"path"`
	Method    string            `json:"method"`
	Duration  int64             `json:"duration"`
	Status    int               `json:"status"`
	Metadata  map[string]string `json:"metadata"`
}

type AnalysisResult struct {
	PopularPages    []string          `json:"popular_pages"`
	SlowPages       []PerformanceData `json:"slow_pages"`
	PotentialIssues []Issue           `json:"potential_issues"`
	Insights        []string          `json:"insights"`
}

type PerformanceData struct {
	Path         string  `json:"path"`
	AvgDuration  int64   `json:"avg_duration"`
	RequestCount int     `json:"request_count"`
	ErrorRate    float64 `json:"error_rate"`
}

type Issue struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Severity    string      `json:"severity"`
	Path        interface{} `json:"path"` // Can be either string or []string
}

type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func NewAnalyticsService(apiKey string) *AnalyticsService {
	return &AnalyticsService{
		apiKey: apiKey,
	}
}

func cleanJSONResponse(response string) string {
	// Remove any backticks or markdown formatting
	response = strings.ReplaceAll(response, "`", "")
	response = strings.ReplaceAll(response, "```json", "")
	response = strings.ReplaceAll(response, "```", "")

	// Find the first { and last }
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")

	if start >= 0 && end > start {
		return response[start : end+1]
	}

	return response
}

func (s *AnalyticsService) AnalyzeLogs(ctx context.Context, logs []LogEntry) (*AnalysisResult, error) {
	// Create a summary of the logs instead of sending raw data
	var summary strings.Builder
	summary.WriteString("Log Summary:\n\n")

	// Group logs by path for better analysis
	pathStats := make(map[string]struct {
		count     int
		totalTime int64
		errors    int
	})

	for _, log := range logs {
		stats := pathStats[log.Path]
		stats.count++
		stats.totalTime += log.Duration
		if log.Status >= 400 {
			stats.errors++
		}
		pathStats[log.Path] = stats

		// Add important events (errors, warnings, slow requests)
		if log.Status >= 400 || log.Level == "error" || log.Level == "warning" || log.Duration > 1000 {
			summary.WriteString(fmt.Sprintf("- %s [%s] %s (Duration: %dms, Status: %d)\n",
				log.Timestamp, log.Level, log.Path, log.Duration, log.Status))
		}
	}

	// Add path statistics
	summary.WriteString("\nPath Statistics:\n")
	for path, stats := range pathStats {
		avgTime := stats.totalTime / int64(stats.count)
		errorRate := float64(stats.errors) / float64(stats.count) * 100
		summary.WriteString(fmt.Sprintf("- %s: %d requests, avg time %dms, error rate %.1f%%\n",
			path, stats.count, avgTime, errorRate))
	}

	prompt := fmt.Sprintf(`Analyze this log summary and provide insights. Return ONLY a JSON object with this exact structure (no markdown, no backticks):
{
    "popular_pages": ["page1", "page2"],
    "slow_pages": [{"path": "/example", "avg_duration": 1000, "request_count": 10, "error_rate": 5.0}],
    "potential_issues": [{"type": "security", "description": "desc", "severity": "high", "path": "/example"}],
    "insights": ["insight1", "insight2"]
}

Log Summary:
%s`, summary.String())

	response, err := s.callGeminiAPI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("error generating analysis: %v", err)
	}

	// Clean and parse the response
	cleanedResponse := cleanJSONResponse(response)
	var result AnalysisResult
	if err := json.Unmarshal([]byte(cleanedResponse), &result); err != nil {
		return nil, fmt.Errorf("error parsing analysis result: %v, response: %s", err, cleanedResponse)
	}

	return &result, nil
}

func (s *AnalyticsService) AnalyzePerformance(ctx context.Context, logs []LogEntry) (*PerformanceAnalysis, error) {
	// Create a performance summary
	var summary strings.Builder
	summary.WriteString("Performance Summary:\n\n")

	// Group by path for performance analysis
	pathStats := make(map[string]struct {
		count     int
		totalTime int64
		maxTime   int64
		minTime   int64
		errors    int
	})

	for _, log := range logs {
		stats := pathStats[log.Path]
		if stats.count == 0 {
			stats.minTime = log.Duration
			stats.maxTime = log.Duration
		} else {
			if log.Duration < stats.minTime {
				stats.minTime = log.Duration
			}
			if log.Duration > stats.maxTime {
				stats.maxTime = log.Duration
			}
		}
		stats.count++
		stats.totalTime += log.Duration
		if log.Status >= 400 {
			stats.errors++
		}
		pathStats[log.Path] = stats
	}

	// Add performance statistics
	for path, stats := range pathStats {
		avgTime := stats.totalTime / int64(stats.count)
		errorRate := float64(stats.errors) / float64(stats.count) * 100
		summary.WriteString(fmt.Sprintf("Endpoint: %s\n", path))
		summary.WriteString(fmt.Sprintf("- Requests: %d\n", stats.count))
		summary.WriteString(fmt.Sprintf("- Avg Time: %dms\n", avgTime))
		summary.WriteString(fmt.Sprintf("- Min Time: %dms\n", stats.minTime))
		summary.WriteString(fmt.Sprintf("- Max Time: %dms\n", stats.maxTime))
		summary.WriteString(fmt.Sprintf("- Error Rate: %.1f%%\n\n", errorRate))
	}

	prompt := fmt.Sprintf(`Analyze this performance data and provide insights. Return ONLY a JSON object with this exact structure (no markdown, no backticks):
{
    "slow_endpoints": [{"path": "/example", "avg_duration": 1000, "request_count": 10, "error_rate": 5.0}],
    "performance_patterns": ["pattern1", "pattern2"],
    "resource_issues": [{"type": "memory", "description": "High memory usage", "severity": "high"}],
    "recommendations": ["recommendation1", "recommendation2"]
}

Performance Data:
%s`, summary.String())

	response, err := s.callGeminiAPI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("error generating analysis: %v", err)
	}

	// Clean and parse the response
	cleanedResponse := cleanJSONResponse(response)
	var result PerformanceAnalysis
	if err := json.Unmarshal([]byte(cleanedResponse), &result); err != nil {
		return nil, fmt.Errorf("error parsing analysis result: %v, response: %s", err, cleanedResponse)
	}

	return &result, nil
}

type PerformanceAnalysis struct {
	SlowEndpoints       []PerformanceData `json:"slow_endpoints"`
	PerformancePatterns []string          `json:"performance_patterns"`
	ResourceIssues      []Issue           `json:"resource_issues"`
	Recommendations     []string          `json:"recommendations"`
}

func (s *AnalyticsService) callGeminiAPI(ctx context.Context, prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.3,
			"topP":            0.8,
			"topK":            40,
			"maxOutputTokens": 1024,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", geminiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", s.apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error parsing response: %v", err)
	}

	candidates, ok := result["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "", fmt.Errorf("no candidates in response: %s", string(body))
	}

	content, ok := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format: %s", string(body))
	}

	parts, ok := content["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return "", fmt.Errorf("no parts in response: %s", string(body))
	}

	text, ok := parts[0].(map[string]interface{})["text"].(string)
	if !ok {
		return "", fmt.Errorf("invalid text format in response: %s", string(body))
	}

	return text, nil
}

func (s *AnalyticsService) ConvertToCSV(logs []LogEntry) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	// Write header
	header := []string{"timestamp", "level", "message", "path", "method", "duration", "status"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("error writing CSV header: %v", err)
	}

	// Write log entries
	for _, log := range logs {
		row := []string{
			log.Timestamp,
			log.Level,
			log.Message,
			log.Path,
			log.Method,
			strconv.FormatInt(log.Duration, 10),
			strconv.Itoa(log.Status),
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("error writing CSV row: %v", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("error flushing CSV writer: %v", err)
	}

	return buffer.Bytes(), nil
}

func splitIntoChunks(text string, chunkSize int) []string {
	var chunks []string
	for i := 0; i < len(text); i += chunkSize {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[i:end])
	}
	return chunks
}
