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

type responseProvider struct{}

func (p *responseProvider) HandleResponses(ctx *gin.Context) {
	var req responseRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := utils.Validate.Struct(req); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		for _, fieldError := range validationErrors {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": fieldError.Error()})
			return
		}
	}

	inputText := req.InputText()
	response := prompt2Response(inputText)
	usageData := buildUsage(len([]rune(inputText)))

	if req.Stream {
		p.handleStreamResponse(ctx, req, response, usageData)
	} else {
		p.handleNonStreamResponse(ctx, req, response, usageData)
	}
}

func (p *responseProvider) handleNonStreamResponse(ctx *gin.Context, req responseRequest, response string, usageData usage) {
	result := createResponseResult(req.Model, response, &usageData)
	ctx.JSON(http.StatusOK, result)
}

func (p *responseProvider) handleStreamResponse(ctx *gin.Context, req responseRequest, response string, usageData usage) {
	utils.SetEventStreamHeaders(ctx)
	dataChan := make(chan string)
	stopChan := make(chan bool, 1)

	go func() {
		// 1. response.created
		result := createResponseResult(req.Model, "", &usageData)
		result.Status = "in_progress"
		p.sendEvent(dataChan, "response.created", map[string]any{"type": "response.created", "response": result})

		// 2. response.output_item.added
		p.sendEvent(dataChan, "response.output_item.added", map[string]any{
			"type":        "response.output_item.added",
			"output_index": 0,
			"item": responseOutput{
				Type:   "message",
				Id:     "msg_mock_" + completionMockId,
				Status: "in_progress",
				Role:   roleAssistant,
			},
		})

		// 3. response.content_part.added
		p.sendEvent(dataChan, "response.content_part.added", map[string]any{
			"type":         "response.content_part.added",
			"output_index":  0,
			"content_index": 0,
			"part":          responseContent{Type: "output_text", Text: ""},
		})

		// 4. response.output_text.delta (逐字符)
		responseRunes := []rune(response)
		for i, s := range responseRunes {
			p.sendEvent(dataChan, "response.output_text.delta", map[string]any{
				"type":          "response.output_text.delta",
				"output_index":  0,
				"content_index": 0,
				"delta":         string(s),
			})
			if i < len(responseRunes)-1 {
				time.Sleep(200 * time.Millisecond)
			}
		}

		// 5. response.output_text.done
		p.sendEvent(dataChan, "response.output_text.done", map[string]any{
			"type":          "response.output_text.done",
			"output_index":  0,
			"content_index": 0,
			"text":          response,
		})

		// 6. response.content_part.done
		p.sendEvent(dataChan, "response.content_part.done", map[string]any{
			"type":         "response.content_part.done",
			"output_index":  0,
			"content_index": 0,
			"part":          responseContent{Type: "output_text", Text: response},
		})

		// 7. response.output_item.done
		p.sendEvent(dataChan, "response.output_item.done", map[string]any{
			"type":         "response.output_item.done",
			"output_index": 0,
			"item": responseOutput{
				Type:    "message",
				Id:      "msg_mock_" + completionMockId,
				Status:  "completed",
				Role:    roleAssistant,
				Content: []responseContent{{Type: "output_text", Text: response}},
			},
		})

		// 8. response.completed (with usage + cache tokens)
		completedResult := createResponseResult(req.Model, response, &usageData)
		completedResult.Status = "completed"
		completedResult.Usage = &usageData
		p.sendEvent(dataChan, "response.completed", map[string]any{
			"type":     "response.completed",
			"response": completedResult,
		})

		stopChan <- true
	}()

	ctx.Stream(func(w io.Writer) bool {
		select {
		case data := <-dataChan:
			ctx.Render(-1, streamEvent{Data: data})
			return true
		case <-stopChan:
			ctx.Render(-1, streamEvent{Data: "[DONE]"})
			return false
		}
	})
}

func (p *responseProvider) sendEvent(ch chan<- string, _ string, payload map[string]any) {
	jsonStr, _ := json.Marshal(payload)
	ch <- string(jsonStr)
	time.Sleep(200 * time.Millisecond)
}

func createResponseResult(model, response string, usageData *usage) responseResult {
	output := []responseOutput{}
	if response != "" {
		output = append(output, responseOutput{
			Type:    "message",
			Id:      "msg_mock_" + completionMockId,
			Status:  "completed",
			Role:    roleAssistant,
			Content: []responseContent{{Type: "output_text", Text: response}},
		})
	}
	return responseResult{
		Id:        fmt.Sprintf("resp_%s", completionMockId),
		Object:    "response",
		CreatedAt: completionMockCreated,
		Model:     model,
		Output:    output,
		Usage:     usageData,
		Status:    "completed",
	}
}
