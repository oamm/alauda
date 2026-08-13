package main

import (
	"fmt"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
	"github.com/spf13/cobra"
)

func newEnvironmentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "environment",
		Short: "Manage environments",
	}
	cmd.AddCommand(newEnvironmentListCommand())
	cmd.AddCommand(newEnvironmentGetCommand())
	cmd.AddCommand(newEnvironmentCreateCommand())
	cmd.AddCommand(newEnvironmentUpdateCommand())
	cmd.AddCommand(newEnvironmentDeleteCommand())
	return cmd
}

func newEnvironmentListCommand() *cobra.Command {
	var includeDisabled bool
	c := &cobra.Command{
		Use:   "list",
		Short: "List environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewEnvironmentServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.ListEnvironments(ctx, connect.NewRequest(&registryv1.ListEnvironmentsRequest{IncludeDisabled: includeDisabled}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().BoolVar(&includeDisabled, "include-disabled", false, "Include disabled environments")
	return c
}

func newEnvironmentGetCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "get <environment-id>",
		Short: "Get environment by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewEnvironmentServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.GetEnvironment(ctx, connect.NewRequest(&registryv1.GetEnvironmentRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	return c
}

func newEnvironmentCreateCommand() *cobra.Command {
	var key, name, description, tier string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			if key == "" || name == "" {
				return fmt.Errorf("--key and --name are required")
			}
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewEnvironmentServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.CreateEnvironment(ctx, connect.NewRequest(&registryv1.CreateEnvironmentRequest{
				Key:         key,
				Name:        name,
				Description: description,
				Tier:        tier,
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&key, "key", "", "Environment key")
	c.Flags().StringVar(&name, "name", "", "Environment display name")
	c.Flags().StringVar(&description, "description", "", "Environment description")
	c.Flags().StringVar(&tier, "tier", "", "Environment tier")
	return c
}

func newEnvironmentUpdateCommand() *cobra.Command {
	var name, description, tier string
	var enabled bool
	c := &cobra.Command{
		Use:   "update <environment-id>",
		Short: "Update environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewEnvironmentServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.UpdateEnvironment(ctx, connect.NewRequest(&registryv1.UpdateEnvironmentRequest{
				Id:          args[0],
				Name:        name,
				Description: description,
				Enabled:     enabled,
				Tier:        tier,
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "Environment display name")
	c.Flags().StringVar(&description, "description", "", "Environment description")
	c.Flags().StringVar(&tier, "tier", "", "Environment tier")
	c.Flags().BoolVar(&enabled, "enabled", true, "Enable environment")
	return c
}

func newEnvironmentDeleteCommand() *cobra.Command {
	c := &cobra.Command{
		Use:     "delete <environment-id>",
		Aliases: []string{"remove"},
		Short:   "Delete environment",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewEnvironmentServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.DeleteEnvironment(ctx, connect.NewRequest(&registryv1.DeleteEnvironmentRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	return c
}
