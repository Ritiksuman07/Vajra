package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourorg/vajra-bot/service/internal/config"
	"github.com/yourorg/vajra-bot/service/internal/db"
	"github.com/yourorg/vajra-bot/service/internal/pods"

	log "github.com/sirupsen/logrus"
)

type Router struct {
	cfg      *config.Config
	db       *db.DB
	podMgr   *pods.Manager
	standalone bool
}

func NewRouter(cfg *config.Config, database *db.DB, standalone bool) *gin.Engine {
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"version":   "1.0.0",
			"standalone": standalone,
			"time":      time.Now().Format(time.RFC3339),
		})
	})

	// Session routes
	r.POST("/sessions", func(c *gin.Context) {
		sessionID := uuid.New().String()
		// In a real implementation, create session in DB
		c.JSON(http.StatusCreated, gin.H{
			"id":      sessionID,
			"status":  "created",
			"message": "Session created",
		})
	})

	r.GET("/sessions/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{
			"id":     id,
			"status": "active",
		})
	})

	// Pod routes
	podGroup := r.Group("/pods")
	{
		podGroup.POST("", createPod)
		podGroup.GET("", listPods)
		podGroup.GET("/:id", getPod)
		podGroup.DELETE("/:id", deletePod)
		podGroup.POST("/:id/generate", generatePod)
		podGroup.POST("/:id/stop", stopPod)
	}

	// Model routes
	r.GET("/models", listModels)
	r.POST("/models/pull", pullModel)

	// Config routes
	r.GET("/config", getConfig)
	r.PUT("/config", updateConfig)

	// Provider routes (for /connect)
	r.GET("/providers", listProviders)
	r.POST("/providers", addProvider)
	r.DELETE("/providers/:name", removeProvider)

	// MCP routes
	r.GET("/mcp/servers", listMCPServers)
	r.POST("/mcp/call", callMCPTool)

	// Verification routes
	r.GET("/verification/:trace_id", getVerification)
	r.POST("/verification/human", humanVerification)

	return r
}

// Pod handlers
func createPod(c *gin.Context) {
	var req struct {
		Role      string `json:"role" binding:"required"`
		Image     string `json:"image"`
		CPU       string `json:"cpu"`
		Memory    string `json:"memory"`
		Workspace string `json:"workspace"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	podID := uuid.New().String()
	
	// In real implementation, call pod manager
	// pod, err := podMgr.Create(podID, req.Role, req.Image, req.CPU, req.Memory, req.Workspace)

	c.JSON(http.StatusCreated, gin.H{
		"id":       podID,
		"role":     req.Role,
		"status":   "creating",
		"message":  "Pod creation started",
	})
}

func listPods(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"pods": []gin.H{},
		"total": 0,
	})
}

func getPod(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "running",
		"role":   "agent",
	})
}

func deletePod(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"status":  "deleted",
		"message": "Pod stopped and removed",
	})
}

func generatePod(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Prompt string `json:"prompt" binding:"required"`
		Model  string `json:"model"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In real implementation, delegate to agent pod
	c.JSON(http.StatusOK, gin.H{
		"pod_id":  id,
		"prompt":  req.Prompt,
		"model":   req.Model,
		"result":  "Task completed",
		"verification": gin.H{
			"passed": true,
			"confidence": 0.85,
		},
	})
}

func stopPod(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"status":  "stopped",
		"message": "Pod stopped",
	})
}

// Model handlers
func listModels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"models": []gin.H{
			{
				"name":       "llama3.1:8b",
				"backend":    "ollama",
				"size":       "4.7GB",
				"context":    131072,
				"capabilities": []string{"chat", "tools"},
			},
			{
				"name":       "llama3.1:70b",
				"backend":    "ollama",
				"size":       "40GB",
				"context":    131072,
				"capabilities": []string{"chat", "tools", "reasoning"},
			},
			{
				"name":       "codellama:34b",
				"backend":    "ollama",
				"size":       "19GB",
				"context":    131072,
				"capabilities": []string{"code", "tools"},
			},
		},
	})
}

func pullModel(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		Backend string `json:"backend"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{
		"name":     req.Name,
		"status":   "pulling",
		"message":  "Model pull started",
	})
}

// Config handlers
func getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": gin.H{
			"port": 4096,
			"data_dir": "~/.local/share/vajra-bot",
			"log_level": "info",
		},
		"models": gin.H{
			"default_backend": "ollama",
			"ollama": gin.H{
				"host": "http://localhost:11434",
				"default_model": "llama3.1:8b",
			},
		},
	})
}

func updateConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Config updated"})
}

// Provider handlers
func listProviders(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"providers": []gin.H{
			{"name": "ollama", "type": "local", "status": "available", "auth_methods": []string{"none"}},
			{"name": "lm-studio", "type": "local", "status": "available", "auth_methods": []string{"none"}},
			{"name": "openai", "type": "cloud", "status": "configurable", "auth_methods": []string{"api_key"}},
			{"name": "openrouter", "type": "cloud", "status": "configurable", "auth_methods": []string{"api_key"}},
			{"name": "anthropic", "type": "cloud", "status": "configurable", "auth_methods": []string{"api_key"}},
			{"name": "google", "type": "cloud", "status": "configurable", "auth_methods": []string{"api_key", "oauth"}},
		},
	})
}

func addProvider(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Provider added"})
}

func removeProvider(c *gin.Context) {
	name := c.Param("name")
	c.JSON(http.StatusOK, gin.H{"name": name, "message": "Provider removed"})
}

// MCP handlers
func listMCPServers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"servers": []gin.H{
			{"name": "file-system", "enabled": true, "tools": []string{"read", "write", "list", "delete"}},
			{"name": "shell", "enabled": true, "tools": []string{"exec"}},
			{"name": "git", "enabled": true, "tools": []string{"status", "diff", "commit", "log"}},
			{"name": "code-exec", "enabled": true, "tools": []string{"run"}},
			{"name": "knowledge-graph", "enabled": true, "tools": []string{"query", "add"}},
		},
	})
}

func callMCPTool(c *gin.Context) {
	var req struct {
		Server string `json:"server" binding:"required"`
		Tool   string `json:"tool" binding:"required"`
		Args   map[string]interface{} `json:"args"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In real implementation, call MCP server
	c.JSON(http.StatusOK, gin.H{
		"server": req.Server,
		"tool":   req.Tool,
		"result": fmt.Sprintf("Simulated result for %s.%s", req.Server, req.Tool),
	})
}

// Verification handlers
func getVerification(c *gin.Context) {
	traceID := c.Param("trace_id")
	c.JSON(http.StatusOK, gin.H{
		"trace_id": traceID,
		"layers": []gin.H{
			{"layer": 1, "name": "fast_checks", "passed": true, "latency_ms": 2},
			{"layer": 2, "name": "semantic_similarity", "passed": true, "score": 0.82, "latency_ms": 45},
			{"layer": 3, "name": "llm_judge", "passed": true, "score": 0.78, "latency_ms": 1200},
		},
		"overall": "passed",
	})
}

func humanVerification(c *gin.Context) {
	var req struct {
		TraceID string `json:"trace_id" binding:"required"`
		Label   string `json:"label" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Verification feedback recorded"})
}
