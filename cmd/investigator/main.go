package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgPath string
	logger  *slog.Logger
	version = "dev"
)

func main() {
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	rootCmd := &cobra.Command{
		Use:     "diagnos",
		Short:   "Diagnos — machine health investigation engine",
		Version: version,
	}

	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "", "path to config file (auto-discovered when omitted)")
	rootCmd.AddCommand(newDebugCmd())
	rootCmd.AddCommand(newInitCmd(), newConfigCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
