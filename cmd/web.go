package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/asynq-test/internal/config"
	"github.com/asynq-test/internal/web"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start web UI server",
	Long:  "Runs the Gin HTTP server with htmx dashboard, REST API, and embedded Asynqmon at /monitoring.",
	RunE:  runWeb,
}

func init() {
	rootCmd.AddCommand(webCmd)
	webCmd.Flags().IntP("port", "p", 0, "Web server port (overrides config)")
	_ = viper.BindPFlag("web.port", webCmd.Flags().Lookup("port"))
}

func runWeb(cmd *cobra.Command, args []string) error {
	cfg := config.C
	addr := fmt.Sprintf(":%d", cfg.Web.Port)

	log.Printf("Starting web server on %s", addr)
	log.Printf("  Dashboard:  http://localhost:%d", cfg.Web.Port)
	log.Printf("  Asynqmon:   http://localhost:%d/monitoring", cfg.Web.Port)
	log.Printf("  API:        http://localhost:%d/api", cfg.Web.Port)

	router := web.NewRouter(cfg)
	return router.Run(addr)
}
