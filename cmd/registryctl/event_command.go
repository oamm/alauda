package main

import (
	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
	"github.com/spf13/cobra"
)

func newEventCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "events",
		Short:   "Manage events",
		Aliases: []string{"event"},
	}
	cmd.AddCommand(newEventListCommand())
	return cmd
}

func newEventListCommand() *cobra.Command {
	var environmentID, serviceID, deploymentID, instanceID, eventType string
	c := &cobra.Command{
		Use:   "list",
		Short: "List events",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewEventServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.ListEvents(ctx, connect.NewRequest(&registryv1.ListEventsRequest{
				EnvironmentId: environmentID,
				ServiceId:     serviceID,
				DeploymentId:  deploymentID,
				InstanceId:    instanceID,
				Type:          eventType,
				Pagination:    &registryv1.PaginationRequest{PageSize: 50},
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&environmentID, "environment-id", "", "Filter by environment ID")
	c.Flags().StringVar(&serviceID, "service-id", "", "Filter by service ID")
	c.Flags().StringVar(&deploymentID, "deployment-id", "", "Filter by deployment ID")
	c.Flags().StringVar(&instanceID, "instance-id", "", "Filter by instance ID")
	c.Flags().StringVar(&eventType, "type", "", "Filter by event type")
	return c
}
