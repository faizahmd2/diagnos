package main

import (
	"fmt"
	"os"

	"github.com/faizahmd2/diagnos/internal/config"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var force bool
	var output string
	cmd := &cobra.Command{Use: "init", Short: "write a commented app.yml", RunE: func(cmd *cobra.Command, args []string) error {
		if output == "" {
			output = "app.yml"
		}
		if err := config.WriteReference(output, force); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "wrote", output)
		return nil
	}}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing file")
	cmd.Flags().StringVar(&output, "output", "app.yml", "output path")
	return cmd
}
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "inspect configuration"}
	cmd.AddCommand(&cobra.Command{Use: "check", Short: "validate and print effective configuration", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "ssh:\n  user: %s\n  port: %d\n  key_path: %s\n  host_key_policy: %s\n  max_parallel: %d\n  max_output_bytes: %d\nai:\n  model: %s\n  api_key: [redacted]\n", cfg.SSH.User, cfg.SSH.Port, cfg.SSH.KeyPath, cfg.SSH.HostKeyPolicy, cfg.SSH.MaxParallel, cfg.SSH.MaxOutputBytes, cfg.AI.Model)
		return nil
	}})
	_ = os.Stdout
	return cmd
}
