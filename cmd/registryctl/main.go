package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cliConfig = defaultCLIConfig()

var rootCmd = &cobra.Command{
	Use:   "registryctl",
	Short: "Service Registry CLI",
	Long:  `Command-line interface for managing the Service Registry`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cliConfig.ServerURL, "server", cliConfig.ServerURL, "Registry server base URL")
	rootCmd.PersistentFlags().StringVar(&cliConfig.Output, "output", cliConfig.Output, "Output format: pretty|json|yaml")
	rootCmd.PersistentFlags().StringVar(&cliConfig.Token, "token", cliConfig.Token, "Bearer token for authenticated registries")

	rootCmd.AddCommand(
		newEnvironmentCommand(),
		newServiceCommand(),
		newDeploymentCommand(),
		newInstanceCommand(),
		newEndpointCommand(),
		newHealthCheckCommand(),
		newIncidentCommand(),
		newEventCommand(),
		newAlertPolicyCommand(),
		newNotificationChannelCommand(),
		newAuthCommand(),
		newAuditCommand(),
	)
}
