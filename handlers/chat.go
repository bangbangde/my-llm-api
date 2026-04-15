package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/my-llm-api/models"
	"github.com/my-llm-api/scheduler"
)

type ChatHandler struct {
	scheduler *scheduler.Scheduler
}

func NewChatHandler(s *scheduler.Scheduler) *ChatHandler {
	return &ChatHandler{
		scheduler: s,
	}
}

func (h *ChatHandler) ChatCompletions(c *gin.Context) {
	var req models.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: &models.ErrorDetail{
				Message: "Invalid request body: " + err.Error(),
				Type:    "invalid_request_error",
				Code:    "invalid_body",
			},
		})
		return
	}

	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: &models.ErrorDetail{
				Message: "messages is required",
				Type:    "invalid_request_error",
				Code:    "missing_parameter",
			},
		})
		return
	}

	if req.Stream {
		h.handleStream(c, &req)
		return
	}

	h.handleNonStream(c, &req)
}

func (h *ChatHandler) handleNonStream(c *gin.Context, req *models.ChatCompletionRequest) {
	resp, err := h.scheduler.ChatCompletion(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ChatHandler) handleStream(c *gin.Context, req *models.ChatCompletionRequest) {
	streamChan, err := h.scheduler.ChatCompletionStream(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	timeout := time.NewTimer(120 * time.Second)
	defer timeout.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case chunk, ok := <-streamChan:
			if !ok {
				fmt.Fprintf(w, "data: [DONE]\n\n")
				return false
			}

			timeout.Reset(120 * time.Second)

			data, err := json.Marshal(chunk)
			if err != nil {
				return true
			}

			fmt.Fprintf(w, "data: %s\n\n", string(data))
			c.Writer.Flush()
			return true

		case <-c.Request.Context().Done():
			return false

		case <-timeout.C:
			return false
		}
	})
}

func (h *ChatHandler) handleError(c *gin.Context, err error) {
	if detail, ok := err.(*models.ErrorDetail); ok {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: detail})
		return
	}

	c.JSON(http.StatusInternalServerError, models.ErrorResponse{
		Error: &models.ErrorDetail{
			Message: err.Error(),
			Type:    "internal_error",
			Code:    "internal_error",
		},
	})
}

func (h *ChatHandler) ListModels(c *gin.Context) {
	modelNames := h.scheduler.ListModels()
	now := time.Now().Unix()

	data := make([]gin.H, 0, len(modelNames))
	for _, name := range modelNames {
		owner := h.scheduler.GetModelOwner(name)
		if owner == "" {
			owner = "system"
		}
		data = append(data, gin.H{
			"id":       name,
			"object":   "model",
			"created":  now,
			"owned_by": owner,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

func (h *ChatHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}
