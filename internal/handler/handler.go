package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Radiqeee/taskapi/api"
	"github.com/Radiqeee/taskapi/internal/auth"
	"github.com/Radiqeee/taskapi/internal/store"
)

const ctxUserID = "userID"

type Handler struct {
	store  store.Store
	tokens *auth.TokenManager
}

func New(s store.Store, tokens *auth.TokenManager) *Handler {
	return &Handler{store: s, tokens: tokens}
}

func (h *Handler) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), requestLogger())

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/openapi.yaml", func(c *gin.Context) { c.Data(http.StatusOK, "application/yaml", api.Spec) })
	r.GET("/docs", func(c *gin.Context) { c.Data(http.StatusOK, "text/html; charset=utf-8", api.DocsPage) })

	v1 := r.Group("/api/v1")
	v1.POST("/auth/register", h.register)
	v1.POST("/auth/login", h.login)

	tasks := v1.Group("/tasks", h.authRequired())
	tasks.GET("", h.listTasks)
	tasks.POST("", h.createTask)
	tasks.GET("/:id", h.getTask)
	tasks.PATCH("/:id", h.updateTask)
	tasks.DELETE("/:id", h.deleteTask)

	return r
}

func (h *Handler) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || token == "" {
			writeError(c, http.StatusUnauthorized, "missing bearer token")
			return
		}
		userID, err := h.tokens.Parse(token)
		if err != nil {
			writeError(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		c.Set(ctxUserID, userID)
		c.Next()
	}
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration", time.Since(start).String(),
		)
	}
}

func writeError(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}

// writeStoreError maps storage errors to HTTP responses without leaking internals.
func writeStoreError(c *gin.Context, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(c, http.StatusNotFound, "not found")
		return
	}
	slog.Error("store error", "err", err, "path", c.Request.URL.Path)
	writeError(c, http.StatusInternalServerError, "internal error")
}
