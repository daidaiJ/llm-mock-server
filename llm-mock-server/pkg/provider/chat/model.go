package chat

import (
	"encoding/json"
	"fmt"
)

const (
	completionMockId = "chatcmpl-llm-mock"

	objectChatCompletion      = "chat.completion"
	objectChatCompletionChunk = "chat.completion.chunk"

	roleAssistant = "assistant"

	stopReason = "stop"

	contentTypeText     = "text"
	contentTypeImageUrl = "image_url"
)

var (
	completionMockCreated int64 = 10
	completionMockUsage         = usage{
		PromptTokens:     9,
		CompletionTokens: 1,
		TotalTokens:      10,
	}
	completionMockUsageWithCache = usage{
		PromptTokens:     9,
		CompletionTokens: 1,
		TotalTokens:      10,
		PromptTokensDetails: &promptTokensDetails{
			CachedTokens: 5,
		},
		CompletionTokensDetails: &completionTokensDetails{
			ReasoningTokens: 0,
		},
		PromptCacheHitTokens:  5,
		PromptCacheMissTokens: 4,
	}
)

// cacheRatio is the fraction of input tokens marked as cache hits (90%).
const cacheRatio = 90

// buildUsage calculates usage based on input character count.
// prompt_tokens equals the input character count; 90% are cache hits.
// completion_tokens equals 10% of input character count (random substring output).
func buildUsage(inputChars int) usage {
	promptTokens := inputChars
	if promptTokens < 1 {
		promptTokens = 1
	}
	completionTokens := promptTokens / 10
	if completionTokens < 1 {
		completionTokens = 1
	}
	cached := promptTokens * cacheRatio / 100
	return usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		PromptTokensDetails: &promptTokensDetails{
			CachedTokens: cached,
		},
		CompletionTokensDetails: &completionTokensDetails{
			ReasoningTokens: 0,
		},
		PromptCacheHitTokens:  cached,
		PromptCacheMissTokens: promptTokens - cached,
	}
}

// buildAnthropicUsage calculates Anthropic-style usage from input character count.
func buildAnthropicUsage(inputChars int) anthropicUsage {
	promptTokens := inputChars
	if promptTokens < 1 {
		promptTokens = 1
	}
	completionTokens := promptTokens / 10
	if completionTokens < 1 {
		completionTokens = 1
	}
	cached := promptTokens * cacheRatio / 100
	return anthropicUsage{
		InputTokens:              promptTokens,
		OutputTokens:             completionTokens,
		CacheCreationInputTokens: promptTokens - cached,
		CacheReadInputTokens:     cached,
	}
}

type chatCompletionRequest struct {
	Model            string                 `json:"model" validate:"required"`
	Messages         []chatMessage          `json:"messages" validate:"required,min=1"`
	MaxTokens        int                    `json:"max_tokens,omitempty"`
	FrequencyPenalty float64                `json:"frequency_penalty,omitempty"`
	N                int                    `json:"n,omitempty"`
	PresencePenalty  float64                `json:"presence_penalty,omitempty"`
	Seed             int                    `json:"seed,omitempty"`
	Stream           bool                   `json:"stream,omitempty"`
	StreamOptions    *streamOptions         `json:"stream_options,omitempty"`
	Temperature      float64                `json:"temperature,omitempty"`
	TopP             float64                `json:"top_p,omitempty"`
	Tools            []tool                 `json:"tools,omitempty"`
	ToolChoice       *toolChoice            `json:"tool_choice,omitempty"`
	User             string                 `json:"user,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	ResponseFormat   map[string]interface{} `json:"response_format,omitempty"`
	Thinking         *thinkingConfig        `json:"thinking,omitempty"`
}

type thinkingConfig struct {
	Type           string `json:"type"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type tool struct {
	Type     string   `json:"type"`
	Function function `json:"function"`
}

type function struct {
	Description string                 `json:"description,omitempty"`
	Name        string                 `json:"name"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// toolChoice represents the different types of tool choice options
// It can be a string ("auto", "none", "required") or an object with specific configurations
type toolChoice struct {
	// For string values: "auto", "none", "required"
	StringValue *string `json:"-"`

	// For allowed_tools configuration
	AllowedTools *allowedToolsChoice `json:"-"`

	// For function tool choice
	FunctionChoice *functionToolChoice `json:"-"`

	// For custom tool choice
	CustomChoice *customToolChoice `json:"-"`
}

// allowedToolsChoice represents the allowed_tools configuration
type allowedToolsChoice struct {
	Type         string        `json:"type"`          // Always "allowed_tools"
	AllowedTools []allowedTool `json:"allowed_tools"` // Constrains the tools available to the model
}

// allowedTool represents a tool in the allowed tools list
type allowedTool struct {
	Mode     string   `json:"mode"`     // Tool mode
	Function function `json:"function"` // Function definition
}

// functionToolChoice represents a specific function tool choice
type functionToolChoice struct {
	Type     string   `json:"type"`     // Always "function"
	Function function `json:"function"` // The specific function to call
}

// customToolChoice represents a custom tool choice
type customToolChoice struct {
	Type   string     `json:"type"`   // Always "custom"
	Custom customTool `json:"custom"` // The custom tool configuration
}

// customTool represents a custom tool configuration
type customTool struct {
	Name string `json:"name"` // Custom tool name
}

// UnmarshalJSON implements custom JSON unmarshaling for toolChoice
// to handle string values ("auto", "none", "required") and different object types
func (tc *toolChoice) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		// Validate string value - only allow "auto", "none", "required"
		switch str {
		case "auto", "none", "required":
			tc.StringValue = &str
			return nil
		default:
			return fmt.Errorf("invalid tool_choice string value: %q, must be one of: \"auto\", \"none\", \"required\"", str)
		}
	}

	// If not a string, try to unmarshal as object
	// First, check the type field to determine which object type it is
	var typeCheck struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return err
	}

	switch typeCheck.Type {
	case "allowed_tools":
		var allowedTools allowedToolsChoice
		if err := json.Unmarshal(data, &allowedTools); err != nil {
			return err
		}
		tc.AllowedTools = &allowedTools
	case "function":
		var functionChoice functionToolChoice
		if err := json.Unmarshal(data, &functionChoice); err != nil {
			return err
		}
		tc.FunctionChoice = &functionChoice
	case "custom":
		var customChoice customToolChoice
		if err := json.Unmarshal(data, &customChoice); err != nil {
			return err
		}
		tc.CustomChoice = &customChoice
	default:
		// For backward compatibility, try to unmarshal as the old format
		var functionChoice functionToolChoice
		if err := json.Unmarshal(data, &functionChoice); err != nil {
			return err
		}
		tc.FunctionChoice = &functionChoice
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for toolChoice
func (tc *toolChoice) MarshalJSON() ([]byte, error) {
	if tc.StringValue != nil {
		return json.Marshal(*tc.StringValue)
	}
	if tc.AllowedTools != nil {
		return json.Marshal(tc.AllowedTools)
	}
	if tc.FunctionChoice != nil {
		return json.Marshal(tc.FunctionChoice)
	}
	if tc.CustomChoice != nil {
		return json.Marshal(tc.CustomChoice)
	}
	return json.Marshal(nil)
}

// IsString returns true if the tool choice is a string value
func (tc *toolChoice) IsString() bool {
	return tc.StringValue != nil
}

// GetStringValue returns the string value if it exists
func (tc *toolChoice) GetStringValue() string {
	if tc.StringValue != nil {
		return *tc.StringValue
	}
	return ""
}

// IsAllowedTools returns true if the tool choice is an allowed_tools configuration
func (tc *toolChoice) IsAllowedTools() bool {
	return tc.AllowedTools != nil
}

// IsFunction returns true if the tool choice is a function configuration
func (tc *toolChoice) IsFunction() bool {
	return tc.FunctionChoice != nil
}

// IsCustom returns true if the tool choice is a custom configuration
func (tc *toolChoice) IsCustom() bool {
	return tc.CustomChoice != nil
}

type chatCompletionResponse struct {
	Id                string                 `json:"id,omitempty"`
	Choices           []chatCompletionChoice `json:"choices"`
	Created           int64                  `json:"created,omitempty"`
	Model             string                 `json:"model,omitempty"`
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"`
	Object            string                 `json:"object,omitempty"`
	Usage             *usage                 `json:"usage"`
}

type chatCompletionChoice struct {
	Index        int                    `json:"index"`
	Message      *chatMessage           `json:"message,omitempty"`
	Delta        *chatMessage           `json:"delta,omitempty"`
	FinishReason *string                `json:"finish_reason"`
	Logprobs     map[string]interface{} `json:"logprobs"`
}

type usage struct {
	PromptTokens            int                  `json:"prompt_tokens,omitempty"`
	CompletionTokens        int                  `json:"completion_tokens,omitempty"`
	TotalTokens             int                  `json:"total_tokens,omitempty"`
	PromptTokensDetails     *promptTokensDetails `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *completionTokensDetails `json:"completion_tokens_details,omitempty"`
	// DeepSeek-specific cache fields
	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens,omitempty"`
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens,omitempty"`
}

type promptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type completionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

type chatMessage struct {
	Name              string     `json:"name,omitempty"`
	Role              string     `json:"role,omitempty"`
	Content           any        `json:"content,omitempty"`
	ReasoningContent  string     `json:"reasoning_content,omitempty"`
	ToolCalls         []toolCall `json:"tool_calls,omitempty"`
}

type messageContent struct {
	Type     string    `json:"type,omitempty"`
	Text     string    `json:"text"`
	ImageUrl *imageUrl `json:"image_url,omitempty"`
}

type imageUrl struct {
	Url    string `json:"url,omitempty"`
	Detail string `json:"detail,omitempty"`
}

func (m *chatMessage) IsEmpty() bool {
	if m.IsStringContent() && m.Content != "" {
		return false
	}
	anyList, ok := m.Content.([]any)
	if ok && len(anyList) > 0 {
		return false
	}
	if len(m.ToolCalls) != 0 {
		nonEmpty := false
		for _, toolCall := range m.ToolCalls {
			if !toolCall.Function.IsEmpty() {
				nonEmpty = true
				break
			}
		}
		if nonEmpty {
			return false
		}
	}
	return true
}

func (m *chatMessage) IsStringContent() bool {
	_, ok := m.Content.(string)
	return ok
}

func (m *chatMessage) StringContent() string {
	content, ok := m.Content.(string)
	if ok {
		return content
	}
	contentList, ok := m.Content.([]any)
	if ok {
		var contentStr string
		for _, contentItem := range contentList {
			contentMap, ok := contentItem.(map[string]any)
			if !ok {
				continue
			}
			if contentMap["type"] == contentTypeText {
				if subStr, ok := contentMap[contentTypeText].(string); ok {
					contentStr += subStr + "\n"
				}
			}
		}
		return contentStr
	}
	return ""
}

func (m *chatMessage) ParseContent() []messageContent {
	var contentList []messageContent
	content, ok := m.Content.(string)
	if ok {
		contentList = append(contentList, messageContent{
			Type: contentTypeText,
			Text: content,
		})
		return contentList
	}
	anyList, ok := m.Content.([]any)
	if ok {
		for _, contentItem := range anyList {
			contentMap, ok := contentItem.(map[string]any)
			if !ok {
				continue
			}
			switch contentMap["type"] {
			case contentTypeText:
				if subStr, ok := contentMap[contentTypeText].(string); ok {
					contentList = append(contentList, messageContent{
						Type: contentTypeText,
						Text: subStr,
					})
				}
			case contentTypeImageUrl:
				if subObj, ok := contentMap[contentTypeImageUrl].(map[string]any); ok {
					contentList = append(contentList, messageContent{
						Type: contentTypeImageUrl,
						ImageUrl: &imageUrl{
							Url: subObj["url"].(string),
						},
					})
				}
			}
		}
		return contentList
	}
	return nil
}

type toolCall struct {
	Index    int          `json:"index"`
	Id       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

type functionCall struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func (m *functionCall) IsEmpty() bool {
	return m.Name == "" && m.Arguments == ""
}

// Anthropic request/response models

type anthropicRequest struct {
	Model       string          `json:"model" validate:"required"`
	Messages    []anthropicMsg  `json:"messages" validate:"required,min=1"`
	MaxTokens   int             `json:"max_tokens" validate:"required,min=1"`
	System      any             `json:"system,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	TopK        *int            `json:"top_k,omitempty"`
	StopSeq     []string        `json:"stop_sequences,omitempty"`
}

type anthropicMsg struct {
	Role    string `json:"role" validate:"required"`
	Content any    `json:"content" validate:"required"`
}

func (m *anthropicMsg) StringContent() string {
	if s, ok := m.Content.(string); ok {
		return s
	}
	if arr, ok := m.Content.([]any); ok {
		for _, item := range arr {
			if block, ok := item.(map[string]any); ok {
				if block["type"] == "text" {
					if text, ok := block["text"].(string); ok {
						return text
					}
				}
			}
		}
	}
	return ""
}

type anthropicResponse struct {
	Id           string            `json:"id"`
	Type         string            `json:"type"`
	Role         string            `json:"role"`
	Content      []anthropicBlock  `json:"content"`
	Model        string            `json:"model"`
	StopReason   string            `json:"stop_reason"`
	StopSequence *string           `json:"stop_sequence"`
	Usage        anthropicUsage    `json:"usage"`
}

type anthropicBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type anthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// OpenAI Responses API models

type responseRequest struct {
	Model        string `json:"model" validate:"required"`
	Input        any    `json:"input" validate:"required"`
	Stream       bool   `json:"stream,omitempty"`
	MaxOutputTokens int  `json:"max_output_tokens,omitempty"`
	Temperature  *float64 `json:"temperature,omitempty"`
	Instructions string `json:"instructions,omitempty"`
}

func (r *responseRequest) InputText() string {
	if s, ok := r.Input.(string); ok {
		return s
	}
	if arr, ok := r.Input.([]any); ok {
		for _, item := range arr {
			if msg, ok := item.(map[string]any); ok {
				if content, ok := msg["content"].(string); ok {
					return content
				}
				if contentArr, ok := msg["content"].([]any); ok {
					for _, c := range contentArr {
						if block, ok := c.(map[string]any); ok {
							if block["type"] == "input_text" {
								if text, ok := block["text"].(string); ok {
									return text
								}
							}
						}
					}
				}
			}
		}
	}
	return ""
}

type responseOutput struct {
	Type    string              `json:"type"`
	Id      string              `json:"id,omitempty"`
	Status  string              `json:"status,omitempty"`
	Role    string              `json:"role,omitempty"`
	Content []responseContent   `json:"content,omitempty"`
}

type responseContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type responseResult struct {
	Id         string           `json:"id"`
	Object     string           `json:"object"`
	CreatedAt  int64            `json:"created_at"`
	Model      string           `json:"model"`
	Output     []responseOutput `json:"output"`
	Usage      *usage           `json:"usage,omitempty"`
	Status     string           `json:"status"`
}
