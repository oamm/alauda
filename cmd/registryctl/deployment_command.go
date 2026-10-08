package main

import (
	"fmt"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
	"github.com/spf13/cobra"
)

func newDeploymentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployment",
		Short: "Manage deployments",
	}
	cmd.AddCommand(newDeploymentListCommand())
	cmd.AddCommand(newDeploymentGetCommand())
	cmd.AddCommand(newDeploymentCreateCommand())
	cmd.AddCommand(newDeploymentUpdateCommand())
	cmd.AddCommand(newDeploymentDeleteCommand())
	return cmd
}

func newDeploymentListCommand() *cobra.Command {
	var serviceID, environmentID string
	c := &cobra.Command{
		Use:   "list",
		Short: "List deployments",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewDeploymentServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.ListDeployments(ctx, connect.NewRequest(&registryv1.ListDeploymentsRequest{
				ServiceId:     serviceID,
				EnvironmentId: environmentID,
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&serviceID, "service-id", "", "Filter by service ID")
	c.Flags().StringVar(&environmentID, "environment-id", "", "Filter by environment ID")
	return c
}

func newDeploymentGetCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "get <deployment-id>",
		Short: "Get deployment by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewDeploymentServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.GetDeployment(ctx, connect.NewRequest(&registryv1.GetDeploymentRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	return c
}

func newDeploymentCreateCommand() *cobra.Command {
	var serviceID, environmentID string
	var healthEnabled, alertsEnabled bool
	var alertCooldownMinutes int32
	c := &cobra.Command{
		Use:   "create",
		Short: "Create deployment",
		RunE: func(cmd *cobra.Command, args []string) error {
			if serviceID == "" || environmentID == "" {
				return fmt.Errorf("--service-id and --environment-id are required")
			}
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewDeploymentServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.CreateDeployment(ctx, connect.NewRequest(&registryv1.CreateDeploymentRequest{
				ServiceId:            serviceID,
				EnvironmentId:        environmentID,
				HealthEnabled:        healthEnabled,
				AlertsEnabled:        alertsEnabled,
				AlertCooldownMinutes: alertCooldownMinutes,
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&serviceID, "service-id", "", "Service ID")
	c.Flags().StringVar(&environmentID, "environment-id", "", "Environment ID")
	c.Flags().BoolVar(&healthEnabled, "health-enabled", true, "Enable health checks")
	c.Flags().BoolVar(&alertsEnabled, "alerts-enabled", true, "Enable alerts")
	c.Flags().Int32Var(&alertCooldownMinutes, "alert-cooldown-minutes", 10, "Alert cooldown in minutes")
	return c
}

func newDeploymentUpdateCommand() *cobra.Command {
	var healthEnabled, alertsEnabled bool
	var alertCooldownMinutes int32
	c := &cobra.Command{
		Use:   "update <deployment-id>",
		Short: "Update deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewDeploymentServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.UpdateDeployment(ctx, connect.NewRequest(&registryv1.UpdateDeploymentRequest{
				Id:                   args[0],
				HealthEnabled:        healthEnabled,
				AlertsEnabled:        alertsEnabled,
				AlertCooldownMinutes: alertCooldownMinutes,
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().BoolVar(&healthEnabled, "health-enabled", true, "Enable health checks")
	c.Flags().BoolVar(&alertsEnabled, "alerts-enabled", true, "Enable alerts")
	c.Flags().Int32Var(&alertCooldownMinutes, "alert-cooldown-minutes", 10, "Alert cooldown in minutes")
	return c
}

func newDeploymentDeleteCommand() *cobra.Command {
	c := &cobra.Command{
		Use:     "delete <deployment-id>",
		Aliases: []string{"remove"},
		Short:   "Delete deployment",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewDeploymentServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.DeleteDeployment(ctx, connect.NewRequest(&registryv1.DeleteDeploymentRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	return c
}
