package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"

	"github.com/oh-my-pi/omp/pkg/ai"
	"github.com/oh-my-pi/omp/pkg/config"
	"github.com/oh-my-pi/omp/pkg/plugin"
	"github.com/oh-my-pi/omp/pkg/stats"
	"github.com/oh-my-pi/omp/pkg/tui"
	"github.com/oh-my-pi/omp/pkg/utils"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	model         string
	ollamaURL     string
	pprofAddr     string
	provider      string
	openaiKey     string
	anthropicKey  string
	openrouterKey string
	logLevel      string
	logFile       string
	verbose       bool
)

var rootCmd = &cobra.Command{
	Use:   "omp",
	Short: "oh-my-pi — AI coding agent",
	Long:  `omp is an AI coding agent with TUI, LSP integration, and multi-provider LLM support.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if pprofAddr != "" {
			go func() {
				log.Info().Str("addr", pprofAddr).Msg("pprof server starting")
				if err := http.ListenAndServe(pprofAddr, nil); err != nil {
					log.Error().Err(err).Msg("pprof server error")
				}
			}()
		}
		return nil
	},
	RunE:  runInteractive,
	Version: "0.0.1",
}

var statsPort int

func init() {
	cfgPath, _ := config.Find()
	cfg, cfgErr := config.Load(cfgPath)
	if cfgErr == nil {
		config.ApplyEnv(cfg)
	}

	cobra.OnInitialize(initLogger)
	rootCmd.PersistentFlags().StringVar(&pprofAddr, "pprof", "", "pprof HTTP server address (e.g. :6060)")

	if cfgErr != nil {
		cfg = config.Defaults()
	}
	rootCmd.Flags().StringVarP(&model, "model", "m", cfg.Model, "LLM model to use")
	rootCmd.Flags().StringVar(&ollamaURL, "ollama-url", cfg.OllamaURL, "Ollama server URL")
	rootCmd.Flags().StringVar(&provider, "provider", cfg.Provider, "LLM provider (ollama, openai, anthropic, openrouter)")
	rootCmd.Flags().StringVar(&openaiKey, "openai-key", cfg.OpenAIKey, "OpenAI API key (or $OPENAI_API_KEY)")
	rootCmd.Flags().StringVar(&anthropicKey, "anthropic-key", cfg.AnthropicKey, "Anthropic API key (or $ANTHROPIC_API_KEY)")
	rootCmd.Flags().StringVar(&openrouterKey, "openrouter-key", cfg.OpenRouterKey, "OpenRouter API key (or $OPENROUTER_API_KEY)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", cfg.LogLevel, "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFile, "log-file", cfg.LogFile, "Log file path (default: ~/.omp/omp.log)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "Enable debug logging")

	pluginCmd := &cobra.Command{Use: "plugin", Short: "Manage plugins"}
	pluginInstallCmd := &cobra.Command{
		Use:   "install [package]",
		Short: "Install a plugin",
		Args:  cobra.ExactArgs(1),
		RunE:  runPluginInstall,
	}
	pluginListCmd := &cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		RunE:  runPluginList,
	}
	pluginUninstallCmd := &cobra.Command{
		Use:   "uninstall [name]",
		Short: "Uninstall a plugin",
		Args:  cobra.ExactArgs(1),
		RunE:  runPluginUninstall,
	}
	pluginCmd.AddCommand(pluginInstallCmd, pluginListCmd, pluginUninstallCmd)
	rootCmd.AddCommand(pluginCmd)

	statsCmd := &cobra.Command{Use: "stats", Short: "AI usage statistics dashboard"}
	statsServeCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the stats dashboard HTTP server",
		RunE:  runStatsServe,
	}
	statsServeCmd.Flags().IntVarP(&statsPort, "port", "p", 3847, "HTTP server port")
	statsCmd.AddCommand(statsServeCmd)
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Show current configuration",
		RunE:  runConfig,
	}
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(statsCmd)
}

func newPluginManager() (*plugin.Manager, error) {
	cacheDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return plugin.NewManager(plugin.ManagerOptions{
		RootDir: filepath.Join(cacheDir, "omp"),
	})
}

func runPluginInstall(cmd *cobra.Command, args []string) error {
	m, err := newPluginManager()
	if err != nil {
		return err
	}
	defer m.Close()

	p, err := m.Install(args[0], plugin.InstallOptions{})
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}
	fmt.Printf("Installed %s@%s\n", p.Name, p.Version)
	return nil
}

func runPluginList(cmd *cobra.Command, args []string) error {
	m, err := newPluginManager()
	if err != nil {
		return err
	}
	defer m.Close()

	plugins, err := m.List()
	if err != nil {
		return err
	}
	if len(plugins) == 0 {
		fmt.Println("No plugins installed")
		return nil
	}
	for _, p := range plugins {
		status := "enabled"
		if !p.Enabled {
			status = "disabled"
		}
		fmt.Printf("%s@%s (%s)\n", p.Name, p.Version, status)
	}
	return nil
}

func runPluginUninstall(cmd *cobra.Command, args []string) error {
	m, err := newPluginManager()
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Uninstall(args[0]); err != nil {
		return fmt.Errorf("uninstall: %w", err)
	}
	fmt.Printf("Uninstalled %s\n", args[0])
	return nil
}

func runStatsServe(cmd *cobra.Command, args []string) error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("cache dir: %w", err)
	}
	dbPath := filepath.Join(cacheDir, "omp", "stats.db")
	addr := fmt.Sprintf(":%d", statsPort)
	return stats.StartStatsServer(dbPath, addr)
}

func initLogger() {
	// Load .env files first
	utils.LoadEnv(".env")
	if home, err := os.UserHomeDir(); err == nil {
		utils.LoadEnv(filepath.Join(home, ".omp", ".env"))
	}

	lvl := logLevel
	if verbose {
		lvl = "debug"
	}
	lf := logFile
	if lf == "" {
		if home, err := os.UserHomeDir(); err == nil {
			lf = filepath.Join(home, ".omp", "omp.log")
		}
	}

	utils.InitLogger(utils.LogConfig{
		Level:  lvl,
		File:   lf,
		Pretty: true,
	})
}

func runInteractive(cmd *cobra.Command, args []string) error {
	var p ai.Provider
	switch provider {
	case "openai":
		p = ai.NewOpenAIProvider(model, "", openaiKey)
	case "anthropic":
		p = ai.NewAnthropicProvider(model, "", anthropicKey)
	case "openrouter":
		if model == "llama3.2" {
			model = "openrouter/free"
		}
		p = ai.NewOpenRouterProvider(model, openrouterKey)
	default:
		p = ai.NewOllamaProvider(model, ollamaURL)
	}

	prog, err := tui.NewProgram(p, ai.Request{Model: model})
	if err != nil {
		return fmt.Errorf("failed to start TUI: %w", err)
	}

	if _, err := prog.Run(); err != nil {
		return fmt.Errorf("TUI exited: %w", err)
	}
	return nil
}

func runConfig(cmd *cobra.Command, args []string) error {
	path, err := config.Find()
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = config.Defaults()
		} else {
			return err
		}
	}
	config.ApplyEnv(cfg)
	fmt.Printf("Config path: %s\n\n%s", path, cfg)
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
