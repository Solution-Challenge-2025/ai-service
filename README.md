# Analytics AI Service

This service provides AI-powered analytics for website and application logs using Google's Gemini AI. It can analyze log data for insights about user behavior, performance issues, and potential security threats.

## Features

- Log analysis with AI-powered insights
- Performance analysis and monitoring
- CSV data conversion with customizable templates
- Chunked processing for large log files
- Integration with Google Gemini AI

## Prerequisites

- Go 1.21 or later
- Google Cloud Project with Vertex AI API enabled
- Google Cloud CLI installed and configured

## Setup

1. Set your Google Cloud Project ID:

```bash
export GOOGLE_CLOUD_PROJECT=your-project-id
```

2. Install dependencies:

```bash
go mod tidy
```

3. Run the service:

```bash
go run main.go
```

The service will start on port 8080 by default. You can change this by setting the `PORT` environment variable.

## API Endpoints

### 1. Analyze Logs

```http
POST /analyze/logs
Content-Type: application/json

[
  {
    "timestamp": "2024-04-06T10:00:00Z",
    "level": "info",
    "message": "Request completed",
    "path": "/api/users",
    "method": "GET",
    "duration": 150,
    "status": 200,
    "metadata": {
      "user_id": "123"
    }
  }
]
```

### 2. Analyze Performance

```http
POST /analyze/performance
Content-Type: application/json

[
  {
    "timestamp": "2024-04-06T10:00:00Z",
    "level": "info",
    "message": "Request completed",
    "path": "/api/users",
    "method": "GET",
    "duration": 150,
    "status": 200
  }
]
```

### 3. Convert to CSV

```http
POST /convert/to-csv?template=optional-template-string
Content-Type: application/json

[
  {
    "timestamp": "2024-04-06T10:00:00Z",
    "level": "info",
    "message": "Request completed",
    "path": "/api/users",
    "method": "GET",
    "duration": 150,
    "status": 200
  }
]
```

## Example Usage

```bash
# Analyze logs
curl -X POST http://localhost:8080/analyze/logs \
  -H "Content-Type: application/json" \
  -d @logs.json

# Convert to CSV with template
curl -X POST "http://localhost:8080/convert/to-csv?template=timestamp,path,duration" \
  -H "Content-Type: application/json" \
  -d @logs.json \
  --output analytics.csv
```

## Response Format

### Log Analysis Response

```json
{
  "popular_pages": ["/api/users", "/api/products"],
  "slow_pages": [
    {
      "path": "/api/reports",
      "avg_duration": 1500,
      "request_count": 100,
      "error_rate": 0.05
    }
  ],
  "potential_issues": [
    {
      "type": "security",
      "description": "Multiple failed login attempts detected",
      "severity": "high",
      "path": "/api/login"
    }
  ],
  "insights": [
    "Peak traffic occurs between 2-4 PM UTC",
    "90% of requests complete within 200ms"
  ]
}
```
