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
)

func main() {
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	rootCmd := &cobra.Command{
		Use:   "diagnos",
		Short: "Diagnos — machine health investigation engine",
	}

	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "configs/app.yaml", "path to config file")
	rootCmd.AddCommand(newDebugCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
