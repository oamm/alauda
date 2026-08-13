package main

import (
	"fmt"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
	"github.com/spf13/cobra"
)

func newServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage services",
	}
	cmd.AddCommand(newServiceListCommand())
	cmd.AddCommand(newServiceGetCommand())
	cmd.AddCommand(newServiceCreateCommand())
	cmd.AddCommand(newServiceUpdateCommand())
	cmd.AddCommand(newServiceDeleteCommand())
	return cmd
}

func newServiceListCommand() *cobra.Command {
	var environmentID string
	c := &cobra.Command{
		Use:   "list",
		Short: "List services",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewCatalogServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.ListServices(ctx, connect.NewRequest(&registryv1.ListServicesRequest{EnvironmentId: environmentID}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&environmentID, "environment-id", "", "Filter by environment ID")
	return c
}

func newServiceGetCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "get <service-id>",
		Short: "Get a service by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewCatalogServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.GetService(ctx, connect.NewRequest(&registryv1.GetServiceRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	return c
}

func newServiceCreateCommand() *cobra.Command {
	var name, displayName, description string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || displayName == "" {
				return fmt.Errorf("--name and --display-name are required")
			}
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewCatalogServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.CreateService(ctx, connect.NewRequest(&registryv1.CreateServiceRequest{
				Name:        name,
				DisplayName: displayName,
				Description: description,
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "Service name")
	c.Flags().StringVar(&displayName, "display-name", "", "Service display name")
	c.Flags().StringVar(&description, "description", "", "Service description")
	return c
}

func newServiceUpdateCommand() *cobra.Command {
	var displayName, description string
	c := &cobra.Command{
		Use:   "update <service-id>",
		Short: "Update service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if displayName == "" {
				return fmt.Errorf("--display-name is required")
			}
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewCatalogServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.UpdateService(ctx, connect.NewRequest(&registryv1.UpdateServiceRequest{
				Id:          args[0],
				DisplayName: displayName,
				Description: description,
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&displayName, "display-name", "", "Service display name")
	c.Flags().StringVar(&description, "description", "", "Service description")
	return c
}

func newServiceDeleteCommand() *cobra.Command {
	c := &cobra.Command{
		Use:     "delete <service-id>",
		Aliases: []string{"remove"},
		Short:   "Delete service",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewCatalogServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.DeleteService(ctx, connect.NewRequest(&registryv1.DeleteServiceRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	return c
}
