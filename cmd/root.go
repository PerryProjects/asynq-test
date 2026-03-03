package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/asynq-test/internal/config"
)

var rootCmd = &cobra.Command{
	Use:   "asynqtest",
	Short: "Asynq Multi-Pod Prototype",
	Long:  "Demonstrates the full feature set of hibiken/asynq in a multi-pod environment.",
}

// Execute is called from main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	if err := config.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	// Sync Viper with any cobra flags that have been bound.
	_ = viper.ReadInConfig()
}
