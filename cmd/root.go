package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/asynq-test/internal/config"
	"github.com/asynq-test/internal/tasks"
)

// NewRootCmd creates the root cobra command and wires all subcommands explicitly.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "asynqtest",
		Short: "Asynq Multi-Pod Prototype",
		Long:  "Demonstrates the full feature set of hibiken/asynq in a multi-pod environment.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Load(); err != nil {
				return err
			}
			return tasks.SetPayloadFormat(config.C.Serialization.Format)
		},
	}

	rootCmd.PersistentFlags().String("payload-format", "", "Task payload format in Redis: json or proto")
	_ = viper.BindPFlag("serialization.format", rootCmd.PersistentFlags().Lookup("payload-format"))

	rootCmd.AddCommand(
		NewWorkerCmd(),
		NewWebCmd(),
		NewEnqueueCmd(),
	)

	return rootCmd
}

// Execute runs the CLI.
func Execute() error {
	return NewRootCmd().Execute()
}
