package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"llm-mock-server/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const (
	anthropicMockMsgId = "msg_mock_" + completionMockId
)

type anthropicProvider struct{}

func (p *anthropicProvider) HandleMessages(ctx *gin.Context) {
	// Validate API key
	apiKey := ctx.GetHeader("x-api-key")
	if apiKey == "" {
		apiKey = ctx.GetHeader("Authorization")
	}
	if apiKey == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"type":    "error",
			"error": gin.H{"type": "authentication_error", "message": "Missing API key"},
		})
		return
	}

	var req anthropicRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"type":  "error",
			"error": gin.H{"type": "invalid_request_error", "message": err.Error()},
		})
		return
	}

	if err := utils.Validate.Struct(req); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		for _, fieldError := range validationErrors {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"type":  "error",
				"error": gin.H{"type": "invalid_request_error", "message": fieldError.Error()},
			})
			return
		}
	}

	// Extract text from the last user message
	prompt := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			prompt = req.Messages[i].StringContent()
			break
		}
	}
	response := prompt2Response(prompt)

	if req.Stream {
		p.handleStreamResponse(ctx, req, response)
	} else {
		p.handleNonStreamResponse(ctx, req, response)
	}
}

func (p *anthropicProvider) handleNonStreamResponse(ctx *gin.Context, req anthropicRequest, response string) {
	result := createAnthropicResponse(req.Model, response)
	ctx.JSON(http.StatusOK, result)
}

func (p *anthropicProvider) handleStreamResponse(ctx *gin.Context, req anthropicRequest, response string) {
	utils.SetEventStreamHeaders(ctx)
	dataChan := make(chan string)
	stopChan := make(chan bool, 1)

	go func() {
		// 1. message_start
		startResp := createAnthropicResponse(req.Model, "")
		startResp.Content = nil
		startResp.StopReason = ""
		startResp.Usage = anthropicUsage{
			InputTokens:              9,
			OutputTokens:             0,
			CacheCreationInputTokens: 5,
			CacheReadInputTokens:     4,
		}
		p.sendSSEEvent(dataChan, "message_start", map[string]any{
			"type":    "message_start",
			"message": startResp,
		})

		// 2. content_block_start
		p.sendSSEEvent(dataChan, "content_block_start", map[string]any{
			"type":         "content_block_start",
			"index":        0,
			"content_block": anthropicBlock{Type: "text", Text: ""},
		})

		// 3. ping
		p.sendSSEEvent(dataChan, "ping", map[string]any{
			"type": "ping",
		})

		// 4. content_block_delta (逐字符)
		responseRunes := []rune(response)
		for _, s := range responseRunes {
			p.sendSSEEvent(dataChan, "content_block_delta", map[string]any{
				"type":  "content_block_delta",
				"index": 0,
				"delta": map[string]string{
					"type": "text_delta",
					"text": string(s),
				},
			})
		}

		// 5. content_block_stop
		p.sendSSEEvent(dataChan, "content_block_stop", map[string]any{
			"type":  "content_block_stop",
			"index": 0,
		})

		// 6. message_delta (with stop_reason and final usage)
		p.sendSSEEvent(dataChan, "message_delta", map[string]any{
			"type": "message_delta",
			"delta": map[string]string{
				"stop_reason": "end_turn",
			},
			"usage": anthropicUsage{
				OutputTokens:             len(responseRunes),
				CacheCreationInputTokens: 5,
				CacheReadInputTokens:     4,
			},
		})

		// 7. message_stop
		p.sendSSEEvent(dataChan, "message_stop", map[string]any{
			"type": "message_stop",
		})

		stopChan <- true
	}()

	ctx.Stream(func(w io.Writer) bool {
		select {
		case data := <-dataChan:
			ctx.Render(-1, streamEvent{Data: data})
			return true
		case <-stopChan:
			return false
		}
	})
}

func (p *anthropicProvider) sendSSEEvent(ch chan<- string, eventName string, payload map[string]any) {
	jsonStr, _ := json.Marshal(payload)
	ch <- fmt.Sprintf("event: %s\ndata: %s", eventName, string(jsonStr))
	time.Sleep(200 * time.Millisecond)
}

func createAnthropicResponse(model, response string) anthropicResponse {
	return anthropicResponse{
		Id:         anthropicMockMsgId,
		Type:       "message",
		Role:       roleAssistant,
		Content:    []anthropicBlock{{Type: "text", Text: response}},
		Model:      model,
		StopReason: "end_turn",
		Usage: anthropicUsage{
			InputTokens:              9,
			OutputTokens:             1,
			CacheCreationInputTokens: 5,
			CacheReadInputTokens:     4,
		},
	}
}
