package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"analyticsai/ai-service/analytics"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	analyticsService *analytics.AnalyticsService
)

func main() {
	log.Println("Starting Analytics AI service initialization...")

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	// Initialize analytics service
	analyticsService = analytics.NewAnalyticsService(apiKey)
	log.Println("Successfully initialized Analytics service")

	// Initialize router with trusted proxy configuration
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.SetTrustedProxies([]string{"127.0.0.1"})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"message": "Analytics AI service is running",
		})
	})

	// File upload endpoint
	router.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("get form err: %v", err)})
			return
		}

		// Create uploads directory if it doesn't exist
		if err := os.MkdirAll("uploads", 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create uploads directory: %v", err)})
			return
		}

		filename := filepath.Join("uploads", file.Filename)
		if err := c.SaveUploadedFile(file, filename); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("upload file err: %v", err)})
			return
		}

		// Read and parse the uploaded file
		data, err := os.ReadFile(filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("read file err: %v", err)})
			return
		}

		var logs []analytics.LogEntry
		if err := json.Unmarshal(data, &logs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("parse json err: %v", err)})
			return
		}

		// Analyze the logs
		analysis, err := analyticsService.AnalyzeLogs(c.Request.Context(), logs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("analysis err: %v", err)})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "File successfully uploaded and analyzed",
			"analysis": analysis,
		})
	})

	// Log analysis endpoint
	router.POST("/analyze/logs", func(c *gin.Context) {
		var logs []analytics.LogEntry
		if err := c.BindJSON(&logs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request body: %v", err)})
			return
		}

		analysis, err := analyticsService.AnalyzeLogs(c.Request.Context(), logs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("error generating analysis: %v", err)})
			return
		}

		c.JSON(http.StatusOK, gin.H{"analysis": analysis})
	})

	// Performance analysis endpoint
	router.POST("/analyze/performance", func(c *gin.Context) {
		var logs []analytics.LogEntry
		if err := c.BindJSON(&logs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request body: %v", err)})
			return
		}

		analysis, err := analyticsService.AnalyzePerformance(c.Request.Context(), logs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("error generating analysis: %v", err)})
			return
		}

		c.JSON(http.StatusOK, gin.H{"analysis": analysis})
	})

	// CSV conversion endpoint
	router.POST("/convert/to-csv", func(c *gin.Context) {
		var logs []analytics.LogEntry
		if err := c.BindJSON(&logs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request body: %v", err)})
			return
		}

		csvData, err := analyticsService.ConvertToCSV(logs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("error converting to CSV: %v", err)})
			return
		}

		c.Header("Content-Disposition", "attachment; filename=analytics.csv")
		c.Data(http.StatusOK, "text/csv", csvData)
	})

	log.Println("Starting server...")
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

func filterPerformanceInsights(insights []string) []string {
	var performanceInsights []string
	for _, insight := range insights {
		// Simple filtering based on keywords
		if containsAny(insight, []string{"slow", "performance", "latency", "duration", "response time"}) {
			performanceInsights = append(performanceInsights, insight)
		}
	}
	return performanceInsights
}

func containsAny(s string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(strings.ToLower(s), strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}
