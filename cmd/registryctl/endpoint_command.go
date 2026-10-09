package main

import (
	"fmt"
	"github.com/company/service-registry/internal/address"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
	"github.com/spf13/cobra"
)

func newEndpointCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "endpoint",
		Short: "Manage endpoints",
	}
	cmd.AddCommand(newEndpointListCommand())
	cmd.AddCommand(newEndpointGetCommand())
	cmd.AddCommand(newEndpointCreateCommand())
	cmd.AddCommand(newEndpointUpdateCommand())
	cmd.AddCommand(newEndpointRemoveCommand())
	return cmd
}

func newEndpointListCommand() *cobra.Command {
	var instanceID string
	c := &cobra.Command{
		Use:   "list",
		Short: "List endpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewEndpointServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.ListEndpoints(ctx, connect.NewRequest(&registryv1.ListEndpointsRequest{InstanceId: instanceID}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&instanceID, "instance-id", "", "Filter by instance ID")
	return c
}

func newEndpointGetCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "get <endpoint-id>",
		Short: "Get endpoint by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewEndpointServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.GetEndpoint(ctx, connect.NewRequest(&registryv1.GetEndpointRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	return c
}

func newEndpointCreateCommand() *cobra.Command {
	var instanceID, name, path string
	var protocol, port int32
	var enabled bool
	c := &cobra.Command{
		Use:   "create",
		Short: "Create endpoint",
		RunE: func(cmd *cobra.Command, args []string) error {
			if instanceID == "" || name == "" {
				return fmt.Errorf("--instance-id and --name are required")
			}
			if err := address.ValidateEndpoint(registryv1.Protocol(protocol).String(), port, path); err != nil {
				return err
			}
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewEndpointServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.CreateEndpoint(ctx, connect.NewRequest(&registryv1.CreateEndpointRequest{
				InstanceId: instanceID,
				Name:       name,
				Protocol:   registryv1.Protocol(protocol),
				Port:       port,
				Path:       path,
				Enabled:    enabled,
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&instanceID, "instance-id", "", "Instance ID")
	c.Flags().StringVar(&name, "name", "", "Endpoint name")
	c.Flags().Int32Var(&protocol, "protocol", 0, "Protocol (1=HTTP, 2=HTTPS, 3=GRPC, 4=TCP, 5=UDP)")
	c.Flags().Int32Var(&port, "port", 0, "Port number")
	c.Flags().StringVar(&path, "path", "", "Path (for HTTP/HTTPS)")
	c.Flags().BoolVar(&enabled, "enabled", true, "Enable endpoint")
	return c
}

func newEndpointUpdateCommand() *cobra.Command {
	var name, path string
	var protocol, port int32
	var enabled bool
	c := &cobra.Command{
		Use:   "update <endpoint-id>",
		Short: "Update endpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if protocol != 0 {
				if err := address.ValidateEndpointPath(registryv1.Protocol(protocol).String(), path); err != nil {
					return err
				}
			}
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewEndpointServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.UpdateEndpoint(ctx, connect.NewRequest(&registryv1.UpdateEndpointRequest{
				Id:       args[0],
				Name:     name,
				Protocol: registryv1.Protocol(protocol),
				Port:     port,
				Path:     path,
				Enabled:  enabled,
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "Endpoint name")
	c.Flags().Int32Var(&protocol, "protocol", 0, "Protocol (0=unchanged, 1=HTTP, 2=HTTPS, 3=GRPC, 4=TCP, 5=UDP)")
	c.Flags().Int32Var(&port, "port", 0, "Port number")
	c.Flags().StringVar(&path, "path", "", "Path (for HTTP/HTTPS)")
	c.Flags().BoolVar(&enabled, "enabled", true, "Enable endpoint")
	return c
}

func newEndpointRemoveCommand() *cobra.Command {
	c := &cobra.Command{
		Use:     "remove <endpoint-id>",
		Aliases: []string{"delete"},
		Short:   "Remove endpoint",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewEndpointServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.DeleteEndpoint(ctx, connect.NewRequest(&registryv1.DeleteEndpointRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	return c
}
