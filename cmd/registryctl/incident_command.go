package main

import (
	"fmt"
	"strings"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
	"github.com/spf13/cobra"
)

func newIncidentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "incident",
		Aliases: []string{"incidents"},
		Short:   "Manage incidents",
	}
	cmd.AddCommand(newIncidentListCommand())
	cmd.AddCommand(newIncidentGetCommand())
	cmd.AddCommand(newIncidentVerifyRecoveryCommand())
	cmd.AddCommand(newIncidentResolveManuallyCommand())
	return cmd
}

func newIncidentListCommand() *cobra.Command {
	var environmentID, serviceID, deploymentID, instanceID, stateName string
	c := &cobra.Command{
		Use:   "list",
		Short: "List incidents",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := parseIncidentStateFlag(stateName)
			if err != nil {
				return err
			}
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewIncidentServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.ListIncidents(ctx, connect.NewRequest(&registryv1.ListIncidentsRequest{
				EnvironmentId: environmentID,
				ServiceId:     serviceID,
				DeploymentId:  deploymentID,
				InstanceId:    instanceID,
				State:         state,
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
	c.Flags().StringVar(&stateName, "state", "", "Filter by state: open|resolved")
	return c
}

func newIncidentGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <incident-id>",
		Short: "Get an incident",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewIncidentServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.GetIncident(ctx, connect.NewRequest(&registryv1.GetIncidentRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
}

func newIncidentVerifyRecoveryCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "verify-recovery <incident-id>",
		Short: "Verify that an incident condition has recovered",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewIncidentServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.VerifyIncidentRecovery(ctx, connect.NewRequest(&registryv1.VerifyIncidentRecoveryRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	return c
}

func newIncidentResolveManuallyCommand() *cobra.Command {
	var note string
	c := &cobra.Command{
		Use:   "resolve-manually <incident-id>",
		Short: "Resolve an incident without verified recovery",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(note) == "" {
				return fmt.Errorf("--note is required")
			}
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewIncidentServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.ResolveIncidentManually(ctx, connect.NewRequest(&registryv1.ResolveIncidentManuallyRequest{Id: args[0], Note: note}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&note, "note", "", "Required administrative resolution note")
	return c
}

func parseIncidentStateFlag(value string) (registryv1.IncidentState, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return registryv1.IncidentState_INCIDENT_STATE_UNSPECIFIED, nil
	case "open":
		return registryv1.IncidentState_INCIDENT_STATE_OPEN, nil
	case "resolved":
		return registryv1.IncidentState_INCIDENT_STATE_RESOLVED, nil
	default:
		return registryv1.IncidentState_INCIDENT_STATE_UNSPECIFIED, fmt.Errorf("unsupported --state %q; use open or resolved", value)
	}
}
