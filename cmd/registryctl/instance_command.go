package main

import (
	"fmt"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
	"github.com/spf13/cobra"
)

func newInstanceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instance",
		Short: "Manage service instances",
	}
	cmd.AddCommand(newInstanceListCommand())
	cmd.AddCommand(newInstanceGetCommand())
	cmd.AddCommand(newInstanceRegisterCommand())
	cmd.AddCommand(newInstanceUpdateCommand())
	cmd.AddCommand(newInstanceRemoveCommand())
	return cmd
}

func newInstanceListCommand() *cobra.Command {
	var deploymentID string
	c := &cobra.Command{
		Use:   "list",
		Short: "List instances",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewInstanceServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.ListInstances(ctx, connect.NewRequest(&registryv1.ListInstancesRequest{DeploymentId: deploymentID}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&deploymentID, "deployment-id", "", "Filter by deployment ID")
	return c
}

func newInstanceGetCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "get <instance-id>",
		Short: "Get instance by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewInstanceServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.GetInstance(ctx, connect.NewRequest(&registryv1.GetInstanceRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	return c
}

func newInstanceRegisterCommand() *cobra.Command {
	var deploymentID, name, address, description string
	var port int32
	c := &cobra.Command{
		Use:   "register",
		Short: "Register instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			if deploymentID == "" || name == "" || address == "" {
				return fmt.Errorf("--deployment-id, --name and --address are required")
			}
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewInstanceServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.CreateInstance(ctx, connect.NewRequest(&registryv1.CreateInstanceRequest{
				DeploymentId: deploymentID,
				Name:         name,
				Address:      address,
				Port:         port,
				Description:  description,
				Enabled:      true,
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&deploymentID, "deployment-id", "", "Deployment ID")
	c.Flags().StringVar(&name, "name", "", "Instance name")
	c.Flags().StringVar(&address, "address", "", "Instance address")
	c.Flags().Int32Var(&port, "port", 0, "Instance port")
	c.Flags().StringVar(&description, "description", "", "Instance description")
	return c
}

func newInstanceUpdateCommand() *cobra.Command {
	var address, description string
	var port int32
	var enabled bool
	c := &cobra.Command{
		Use:   "update <instance-id>",
		Short: "Update instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if address == "" {
				return fmt.Errorf("--address is required")
			}
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewInstanceServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.UpdateInstance(ctx, connect.NewRequest(&registryv1.UpdateInstanceRequest{
				Id:          args[0],
				Address:     address,
				Port:        port,
				Description: description,
				Enabled:     enabled,
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&address, "address", "", "Instance address")
	c.Flags().Int32Var(&port, "port", 0, "Instance port")
	c.Flags().StringVar(&description, "description", "", "Instance description")
	c.Flags().BoolVar(&enabled, "enabled", true, "Enable instance")
	return c
}

func newInstanceRemoveCommand() *cobra.Command {
	c := &cobra.Command{
		Use:     "remove <instance-id>",
		Aliases: []string{"delete"},
		Short:   "Remove instance",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewInstanceServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.DeleteInstance(ctx, connect.NewRequest(&registryv1.DeleteInstanceRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	return c
}
