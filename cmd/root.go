package cmd

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"tasnim.dev/aws-tui/internal/app"
	internalaws "tasnim.dev/aws-tui/internal/aws"
	"tasnim.dev/aws-tui/internal/cache"
	"tasnim.dev/aws-tui/internal/config"
	"tasnim.dev/aws-tui/internal/log"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/services"
)

var (
	region  string
	profile string
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "awstui",
		Short: "AWS TUI - Terminal UI for AWS",
		RunE:  runApp,
	}

	cmd.Flags().StringVarP(&region, "region", "r", "", "AWS region")
	cmd.Flags().StringVarP(&profile, "profile", "p", "", "AWS profile")

	return cmd
}

func runApp(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Ensure the cache directory exists for log and DB files.
	if err := os.MkdirAll(cacheDir(), 0o755); err != nil {
		return err
	}

	logger, err := log.New(cacheDir() + "/debug.log")
	if err != nil {
		return err
	}
	defer logger.Close()

	// Redirect stderr to the log file so AWS SDK warnings don't corrupt the TUI.
	stderrFile, err := os.OpenFile(cacheDir()+"/stderr.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err == nil {
		origStderr := os.Stderr
		os.Stderr = stderrFile
		defer func() {
			os.Stderr = origStderr
			stderrFile.Close()
		}()
	}

	cacheDB, err := cache.New(cacheDir() + "/cache.db")
	if err != nil {
		return err
	}
	defer cacheDB.Close()

	cfg, err := config.Load(configDir() + "/config.yaml")
	if err != nil {
		return err
	}

	reg := plugin.NewRegistry()

	// Resolve region: CLI flag > last saved > app config > AWS SDK config > fallback
	r := resolveRegion(ctx, cfg, region)
	p := resolveProfile(cfg, profile)

	sess, err := internalaws.NewSession(ctx, r, p)
	if err != nil {
		logger.Error("failed to create AWS session", "err", err)
	} else {
		services.Register(reg, sess.Config, r, p)
	}

	application := app.New(app.AppConfig{
		Registry: reg,
		Cache:    cacheDB,
		Logger:   logger,
		Config:   &cfg,
		Session:  sess,
		Region:   r,
		Profile:  p,
	})

	prog := tea.NewProgram(application)
	// Cancel context when the program exits to clean up in-flight goroutines.
	go func() {
		<-ctx.Done()
		prog.Kill()
	}()
	_, err = prog.Run()
	cancel()
	return err
}

// resolveRegion picks the region with priority: CLI flag > last saved > app config > AWS SDK > fallback.
func resolveRegion(ctx context.Context, cfg config.Config, flag string) string {
	if flag != "" {
		return flag
	}
	if cfg.LastRegion != "" {
		return cfg.LastRegion
	}
	if cfg.DefaultRegion != "" {
		return cfg.DefaultRegion
	}
	// Try AWS SDK default config (~/.aws/config, env vars).
	if sdkCfg, err := awsconfig.LoadDefaultConfig(ctx); err == nil && sdkCfg.Region != "" {
		return sdkCfg.Region
	}
	return "us-east-1"
}

// resolveProfile picks the profile with priority: CLI flag > last saved > app config > fallback.
func resolveProfile(cfg config.Config, flag string) string {
	if flag != "" {
		return flag
	}
	if cfg.LastProfile != "" {
		return cfg.LastProfile
	}
	if cfg.DefaultProfile != "" {
		return cfg.DefaultProfile
	}
	return "default"
}

func cacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".cache", "aws-tui")
}

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "aws-tui")
}

func Execute() error {
	return newRootCmd().Execute()
}
