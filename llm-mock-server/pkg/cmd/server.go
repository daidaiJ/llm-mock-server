package cmd

import (
	"flag"
	"fmt"
	"os"

	"llm-mock-server/pkg/cmd/options"
	"llm-mock-server/pkg/config"
	"llm-mock-server/pkg/log"
	"llm-mock-server/pkg/middleware"
	"llm-mock-server/pkg/provider/chat"
	"llm-mock-server/pkg/provider/embeddings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

func NewServerCommand() *cobra.Command {
	option := options.NewOption()
	cmd := &cobra.Command{
		Use:  "llm-mock-server",
		Long: `llm mock server for higress e2e test`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Infof("run with option: %+v", option)
			return Run(option)
		},
	}
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	option.AddFlags(cmd.Flags())
	return cmd
}

func Run(option *options.Option) error {
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Load config
	cfg, err := loadConfig(option)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	enabledProviders := cfg.EnabledProviders()
	log.Infof("enabled providers: %v", enabledProviders)

	server := gin.New()
	server.Use(middleware.CORS())
	server.Use(middleware.Auth(option.AuthKey))
	middleware.StartLogger(server, option)

	// Set up chat completion routes
	chat.SetupRoutes(server, option.ProviderType, enabledProviders)

	// embeddings
	server.POST("/v1/embeddings", embeddings.HandleEmbeddings)

	log.Infof("Starting server on port %d", option.ServerPort)
	return server.Run(fmt.Sprintf(":%d", option.ServerPort))
}

func loadConfig(option *options.Option) (*config.Config, error) {
	// If --config is specified, load from that file
	if option.ConfigFile != "" {
		return config.LoadConfig(option.ConfigFile)
	}

	// If --provider-type is specified, treat all providers as enabled (legacy mode)
	// Return a config with no providers so SetupRoutes falls back to legacy mode
	if option.ProviderType != "" {
		return &config.Config{}, nil
	}

	// Default: try to load from ./config.yaml, fall back to built-in defaults
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
