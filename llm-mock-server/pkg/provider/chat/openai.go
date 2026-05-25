package chat

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"llm-mock-server/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type openAiProvider struct{}

func (p *openAiProvider) ShouldHandleRequest(ctx *gin.Context) bool {
	return true
}

func (p *openAiProvider) HandleChatCompletions(ctx *gin.Context) {
	// Bind request body
	var chatRequest chatCompletionRequest
	if err := ctx.ShouldBindJSON(&chatRequest); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate request body
	if err := utils.Validate.Struct(chatRequest); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		for _, fieldError := range validationErrors {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": fieldError.Error()})
			return
		}
	}

	prompt := ""
	if chatRequest.Messages[len(chatRequest.Messages)-1].IsStringContent() {
		prompt = chatRequest.Messages[len(chatRequest.Messages)-1].StringContent()
	}
	response := prompt2Response(prompt)

	if chatRequest.Stream {
		p.handleStreamResponse(ctx, chatRequest, response)
	} else {
		p.handleNonStreamResponse(ctx, chatRequest, response)
	}
}

func (p *openAiProvider) handleStreamResponse(ctx *gin.Context, chatRequest chatCompletionRequest, response string) {
	utils.SetEventStreamHeaders(ctx)
	dataChan := make(chan string)
	stopChan := make(chan bool, 1)
	streamResponse := chatCompletionResponse{
		Id:      completionMockId,
		Object:  objectChatCompletionChunk,
		Created: completionMockCreated,
		Model:   chatRequest.Model,
	}
	streamResponseChoice := chatCompletionChoice{Delta: &chatMessage{}}
	thinkingEnabled := chatRequest.Thinking != nil && chatRequest.Thinking.Type == "enabled"
	go func() {
		responseRunes := []rune(response)
		// When thinking is enabled, send reasoning_content chunks first
		if thinkingEnabled {
			reasoningText := "Thinking process: analyzing the input..."
			for _, s := range []rune(reasoningText) {
				streamResponseChoice.Delta = &chatMessage{ReasoningContent: string(s)}
				streamResponse.Choices = []chatCompletionChoice{streamResponseChoice}
				jsonStr, _ := json.Marshal(streamResponse)
				dataChan <- string(jsonStr)
				time.Sleep(200 * time.Millisecond)
			}
		}

		// Send content chunks
		for i, s := range responseRunes {
			streamResponseChoice.Delta = &chatMessage{Content: string(s)}
			if i == len(responseRunes)-1 {
				streamResponseChoice.FinishReason = ptr(stopReason)
				streamResponse.Usage = &completionMockUsageWithCache
			}
			streamResponse.Choices = []chatCompletionChoice{streamResponseChoice}
			jsonStr, _ := json.Marshal(streamResponse)
			dataChan <- string(jsonStr)

			// Simulate response delay
			time.Sleep(200 * time.Millisecond)
		}
		stopChan <- true
	}()

	ctx.Stream(func(w io.Writer) bool {
		select {
		case data := <-dataChan:
			ctx.Render(-1, streamEvent{Data: "data: " + data})
			return true
		case <-stopChan:
			ctx.Render(-1, streamEvent{Data: "data: [DONE]"})
			return false
		}
	})
}

func (p *openAiProvider) handleNonStreamResponse(ctx *gin.Context, chatRequest chatCompletionRequest, response string) {
	thinkingEnabled := chatRequest.Thinking != nil && chatRequest.Thinking.Type == "enabled"
	completion := createChatCompletionResponse(chatRequest.Model, response, thinkingEnabled)
	ctx.JSON(http.StatusOK, completion)
}

func createChatCompletionResponse(model, response string, thinkingEnabled bool) chatCompletionResponse {
	msg := &chatMessage{
		Role:    roleAssistant,
		Content: response,
	}
	if thinkingEnabled {
		msg.ReasoningContent = "Thinking process: analyzing the input..."
	}
	return chatCompletionResponse{
		Id:      completionMockId,
		Object:  objectChatCompletion,
		Created: completionMockCreated,
		Model:   model,
		Choices: []chatCompletionChoice{
			{
				Index:        0,
				Message:      msg,
				FinishReason: ptr(stopReason),
			},
		},
		Usage: &completionMockUsageWithCache,
	}
}
