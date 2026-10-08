package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cliConfig = defaultCLIConfig()

var rootCmd = newRootCommand()

func newRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "alauda",
		Aliases:       []string{"registryctl"},
		Short:         "Service Registry CLI",
		Long:          `Command-line interface for managing the Service Registry`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			server, environment, output, err := loadNonsecretConfig()
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("server") && os.Getenv("ALAUDA_URL") == "" && server != "" {
				cliConfig.ServerURL = server
			}
			if !cmd.Flags().Changed("output") && output != "" {
				cliConfig.Output = output
			}
			cliConfig.Environment = environment
			switch cliConfig.Output {
			case "table", "pretty", "json", "yaml":
				return nil
			case "value":
				if cmd.Name() == "resolve" {
					return nil
				}
				return fmt.Errorf("--output value is supported only by services resolve")
			default:
				return fmt.Errorf("unknown output format %q", cliConfig.Output)
			}
		},
	}
	rootCmd.PersistentFlags().StringVar(&cliConfig.ServerURL, "server", cliConfig.ServerURL, "Registry server base URL")
	rootCmd.PersistentFlags().StringVar(&cliConfig.Output, "output", cliConfig.Output, "Output format: table|pretty|json|yaml|value")
	rootCmd.PersistentFlags().StringVar(&cliConfig.Token, "token", "", "Bearer credential (otherwise ALAUDA_TOKEN or REGISTRY_TOKEN)")
	rootCmd.PersistentFlags().StringVar(&cliConfig.ConfigFile, "config", cliConfig.ConfigFile, "Nonsecret YAML config: server, environment, output")
	rootCmd.PersistentFlags().BoolVar(&cliConfig.Quiet, "quiet", false, "Suppress ancillary messages, not query data")

	rootCmd.AddCommand(
		newPublicServicesCommand(),
		newPublicEnvironmentsCommand(),
		newPublicHealthCommand(),
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
	for _, name := range []string{"environment", "service", "deployment", "instance", "endpoint"} {
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == name {
				cmd.Deprecated = "legacy UUID-based workflow; use services or environments public commands"
			}
		}
	}
	return rootCmd
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", safeError(err))
		os.Exit(exitCode(err))
	}
}
