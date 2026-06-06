package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"llm-mock-server/pkg/log"
	"llm-mock-server/pkg/provider"

	"github.com/gin-gonic/gin"
)

type requestHandler interface {
	provider.CommonRequestHandler

	HandleChatCompletions(context *gin.Context)
}

// Singleton handler instances
var (
	openAiHandler     = &openAiProvider{}
	qwenHandler       = &qwenProvider{}
	geminiHandler     = &geminiProvider{}
	minimaxHandler    = &minimaxProvider{}
	difyHandler       = &difyProvider{}
	responseHandler   = &responseProvider{}
	anthropicHandler  = &anthropicProvider{}
)

var (
	chatCompletionsHandlers = map[string]requestHandler{
		"minimax": minimaxHandler,
		"dify":    difyHandler,
		"qwen":    qwenHandler,
		"gemini":  geminiHandler,
		"openai":  openAiHandler, // As the last fallback
	}

	chatCompletionsRoutes = []string{
		// baidu
		"/v2/chat/completions",
		// doubao
		"/api/v3/chat/completions",
		// github
		"/chat/completions",
		// groq
		"/openai/v1/chat/completions",
		// minimax
		"/v1/text/chatcompletion_v2",
		"/v1/text/chatcompletion_pro",
		// openai
		"/v1/chat/completions",
		// qwen
		"/compatible-mode/v1/chat/completions",
		"/api/v1/services/aigc/text-generation/generation",
		// zhipu
		"/api/paas/v4/chat/completions",
		// dify
		"/v1/completion-messages",
		"/v1/chat-messages",
		// gemini
		"/v1beta/models/:modelAndAction",
		// cloudflare
		"/client/v4/accounts/:accountId/ai/v1/chat/completions",
	}
)

// providerRoute describes a single route for a provider.
type providerRoute struct {
	Path    string
	Handler gin.HandlerFunc
}

// providerRouteMap maps each logical provider to its routes.
var providerRouteMap = map[string][]providerRoute{
	"openai": {
		{"/v1/chat/completions", openAiHandler.HandleChatCompletions},
		{"/v1/responses", responseHandler.HandleResponses},
	},
	"deepseek": {
		{"/v1/chat/completions", openAiHandler.HandleChatCompletions},
	},
	"anthropic": {
		{"/v1/messages", anthropicHandler.HandleMessages},
	},
	"qwen": {
		{"/compatible-mode/v1/chat/completions", openAiHandler.HandleChatCompletions},
		{"/api/v1/services/aigc/text-generation/generation", qwenHandler.HandleChatCompletions},
	},
	"minimax": {
		{"/v1/text/chatcompletion_v2", openAiHandler.HandleChatCompletions},
		{"/v1/text/chatcompletion_pro", minimaxHandler.HandleChatCompletions},
	},
	"dify": {
		{"/v1/completion-messages", difyHandler.HandleChatCompletions},
		{"/v1/chat-messages", difyHandler.HandleChatCompletions},
	},
	"gemini": {
		{"/v1beta/models/:modelAndAction", geminiHandler.HandleChatCompletions},
	},
	"baidu": {
		{"/v2/chat/completions", openAiHandler.HandleChatCompletions},
	},
	"doubao": {
		{"/api/v3/chat/completions", openAiHandler.HandleChatCompletions},
	},
	"zhipu": {
		{"/api/paas/v4/chat/completions", openAiHandler.HandleChatCompletions},
	},
	"github": {
		{"/chat/completions", openAiHandler.HandleChatCompletions},
	},
	"groq": {
		{"/openai/v1/chat/completions", openAiHandler.HandleChatCompletions},
	},
	"ai360": {
		{"/v1/chat/completions", openAiHandler.HandleChatCompletions},
	},
	"together": {
		{"/v1/chat/completions", openAiHandler.HandleChatCompletions},
	},
	"baichuan": {
		{"/v1/chat/completions", openAiHandler.HandleChatCompletions},
	},
	"yi": {
		{"/v1/chat/completions", openAiHandler.HandleChatCompletions},
	},
	"stepfun": {
		{"/v1/chat/completions", openAiHandler.HandleChatCompletions},
	},
	"cloudflare": {
		{"/client/v4/accounts/:accountId/ai/v1/chat/completions", openAiHandler.HandleChatCompletions},
	},
}

// HandleChatCompletions is the exported handler for /v1/chat/completions.
func HandleChatCompletions(ctx *gin.Context) {
	openAiHandler.HandleChatCompletions(ctx)
}

// HandleAnthropicMessages is the exported handler for /v1/messages.
func HandleAnthropicMessages(ctx *gin.Context) {
	anthropicHandler.HandleMessages(ctx)
}

// HandleResponses is the exported handler for /v1/responses.
func HandleResponses(ctx *gin.Context) {
	responseHandler.HandleResponses(ctx)
}

// SetupRoutes registers routes based on the given provider names.
// If enabledProviders is empty, it falls back to the legacy single-provider-type mode.
func SetupRoutes(server *gin.Engine, providerType string, enabledProviders []string) {
	if len(enabledProviders) > 0 {
		setupRoutesFromConfig(server, enabledProviders)
		return
	}
	// Legacy mode: use providerType flag
	setupRoutesFromType(server, providerType)
}

// setupRoutesFromConfig registers routes only for the enabled providers.
func setupRoutesFromConfig(server *gin.Engine, enabledProviders []string) {
	seen := map[string]bool{}
	for _, name := range enabledProviders {
		name = strings.ToLower(name)
		routes, ok := providerRouteMap[name]
		if !ok {
			log.Warnf("Unknown provider in config: %s, skipping", name)
			continue
		}
		for _, r := range routes {
			if !seen[r.Path] {
				server.POST(r.Path, r.Handler)
				seen[r.Path] = true
			}
		}
		log.Infof("Enabled provider: %s", name)
	}
}

// setupRoutesFromType is the legacy single-provider mode.
func setupRoutesFromType(server *gin.Engine, providerType string) {
	switch strings.ToLower(providerType) {
	case "minimax":
		server.POST("/v1/text/chatcompletion_v2", chatCompletionsHandlers["openai"].HandleChatCompletions)
		server.POST("/v1/text/chatcompletion_pro", chatCompletionsHandlers["minimax"].HandleChatCompletions)
	case "dify":
		server.POST("/v1/completion-messages", chatCompletionsHandlers["dify"].HandleChatCompletions)
		server.POST("/v1/chat-messages", chatCompletionsHandlers["dify"].HandleChatCompletions)
	case "qwen":
		server.POST("/compatible-mode/v1/chat/completions", chatCompletionsHandlers["openai"].HandleChatCompletions)
		server.POST("/api/v1/services/aigc/text-generation/generation", chatCompletionsHandlers["qwen"].HandleChatCompletions)
	case "gemini":
		server.POST("/v1beta/models/:modelAndAction", chatCompletionsHandlers["gemini"].HandleChatCompletions)
	case "doubao":
		server.POST("/api/v3/chat/completions", chatCompletionsHandlers["openai"].HandleChatCompletions)
	case "baidu":
		server.POST("/v2/chat/completions", chatCompletionsHandlers["openai"].HandleChatCompletions)
	case "zhipu":
		server.POST("/api/paas/v4/chat/completions", chatCompletionsHandlers["openai"].HandleChatCompletions)
	case "github":
		server.POST("/chat/completions", chatCompletionsHandlers["openai"].HandleChatCompletions)
	case "groq":
		server.POST("/openai/v1/chat/completions", chatCompletionsHandlers["openai"].HandleChatCompletions)
	case "anthropic":
		server.POST("/v1/messages", anthropicHandler.HandleMessages)
	case "cloudflare":
		server.POST("/client/v4/accounts/:accountId/ai/v1/chat/completions", openAiHandler.HandleChatCompletions)
	case "openai", "ai360", "deepseek", "together", "baichuan", "yi", "stepfun":
		server.POST("/v1/chat/completions", chatCompletionsHandlers["openai"].HandleChatCompletions)
	default:
		// Unknown provider type, enable all routes
		for _, route := range chatCompletionsRoutes {
			server.POST(route, handleChatCompletions)
		}
		server.POST("/v1/responses", responseHandler.HandleResponses)
		server.POST("/v1/messages", anthropicHandler.HandleMessages)
		if providerType != "" {
			log.Warnf("Unknown provider type: %s, enabled all routes", providerType)
		} else {
			log.Infof("No provider type specified, enabled all routes")
		}
	}
}

func handleChatCompletions(context *gin.Context) {
	if err := buildRequestContext(context); err != nil {
		return
	}
	for _, handler := range chatCompletionsHandlers {
		if handler.ShouldHandleRequest(context) {
			handler.HandleChatCompletions(context)
			return
		}
	}
	context.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
}

type requestContext struct {
	Host  string
	Path  string
	Model string
}

func buildRequestContext(context *gin.Context) error {
	body, err := io.ReadAll(context.Request.Body)
	if err != nil {
		log.Errorf("Error reading request body:", err)
		context.JSON(http.StatusBadRequest, gin.H{"error": "Error reading request body"})
		return err
	}

	// Reset the request body so it can be read again by subsequent handlers
	context.Request.Body = io.NopCloser(strings.NewReader(string(body)))

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Errorf("Error unmarshalling JSON:", err)
		context.JSON(http.StatusBadRequest, gin.H{"error": "Error unmarshalling JSON"})
		return err
	}
	model, _ := data["model"].(string)

	context.Set("requestContext", requestContext{
		Host:  context.Request.Host,
		Path:  context.Request.URL.Path,
		Model: model})

	return nil
}

func getRequestContext(context *gin.Context) (requestContext, error) {
	requestCtx, exists := context.Get("requestContext")
	if !exists {
		return requestContext{}, fmt.Errorf("request context not found")
	}

	ctx, ok := requestCtx.(requestContext)
	if !ok {
		return requestContext{}, fmt.Errorf("invalid request context type")
	}

	return ctx, nil
}
