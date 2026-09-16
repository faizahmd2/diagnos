package main

import (
	"fmt"

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

// func newConfigCmd() *cobra.Command {
// 	cmd := &cobra.Command{Use: "config", Short: "inspect configuration"}
// 	cmd.AddCommand(&cobra.Command{Use: "check", Short: "validate and print effective configuration", RunE: func(cmd *cobra.Command, args []string) error {
// 		cfg, err := config.Load(cfgPath)
// 		if err != nil {
// 			return err
// 		}
// 		fmt.Fprintf(cmd.OutOrStdout(), "ssh:\n  user: %s\n  port: %d\n  key_path: %s\n  host_key_policy: %s\n  max_parallel: %d\n  max_output_bytes: %d\nai:\n  model: %s\n  api_key: [redacted]\n", cfg.SSH.User, cfg.SSH.Port, cfg.SSH.KeyPath, cfg.SSH.HostKeyPolicy, cfg.SSH.MaxParallel, cfg.SSH.MaxOutputBytes, cfg.AI.Model)
// 		return nil
// 	}})
// 	_ = os.Stdout
// 	return cmd
// }

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "inspect configuration"}

	cmd.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "validate and print effective configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			reportDir, err := config.ResolveOutputDirectory(cfg.Output.Directory)
			if err != nil {
				return fmt.Errorf("resolve report directory: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "config:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  path: %s\n", cfgPath)
			fmt.Fprintf(cmd.OutOrStdout(), "\nssh:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  user: %s\n", cfg.SSH.User)
			fmt.Fprintf(cmd.OutOrStdout(), "  port: %d\n", cfg.SSH.Port)
			fmt.Fprintf(cmd.OutOrStdout(), "  key_path: %s\n", cfg.SSH.KeyPath)
			fmt.Fprintf(cmd.OutOrStdout(), "  host_key_policy: %s\n", cfg.SSH.HostKeyPolicy)
			fmt.Fprintf(cmd.OutOrStdout(), "  max_parallel: %d\n", cfg.SSH.MaxParallel)
			fmt.Fprintf(cmd.OutOrStdout(), "  max_output_bytes: %d\n", cfg.SSH.MaxOutputBytes)

			fmt.Fprintf(cmd.OutOrStdout(), "\ntargets:\n")
			if len(cfg.Targets) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "  none\n")
			} else {
				for name, target := range cfg.Targets {
					fmt.Fprintf(
						cmd.OutOrStdout(),
						"  %s: %s@%s:%d\n",
						name,
						target.User,
						target.Host,
						target.Port,
					)
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "\nai:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  model: %s\n", cfg.AI.Model)
			fmt.Fprintf(cmd.OutOrStdout(), "  api_key: [redacted]\n")

			fmt.Fprintf(cmd.OutOrStdout(), "\noutput:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  report_type: %s\n", cfg.Output.ReportType)
			fmt.Fprintf(cmd.OutOrStdout(), "  directory: %s\n", reportDir)

			return nil
		},
	})

	return cmd
}
