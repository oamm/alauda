package main

import (
	"fmt"
	"strings"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
	"github.com/spf13/cobra"
)

func newHealthCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:        "health-check",
		Deprecated: "use health checks commands with public resource names",
		Short:      "Manage health checks",
	}
	cmd.AddCommand(newHealthCheckCreateCommand())
	cmd.AddCommand(newHealthCheckListCommand())
	cmd.AddCommand(newHealthCheckRunCommand())
	return cmd
}

func newHealthCheckCreateCommand() *cobra.Command {
	var instanceID, endpointID, name, typeName, description string
	var enabled bool
	var intervalSeconds, timeoutSeconds, failuresBeforeUnhealthy, successesBeforeHealthy int32

	c := &cobra.Command{
		Use:   "create",
		Short: "Create health check",
		RunE: func(cmd *cobra.Command, args []string) error {
			checkType, err := parseHealthCheckTypeFlag(typeName)
			if err != nil {
				return err
			}
			if instanceID == "" || name == "" {
				return fmt.Errorf("--instance-id and --name are required")
			}
			if checkType == registryv1.HealthCheckType_HEALTH_CHECK_TYPE_UNSPECIFIED {
				return fmt.Errorf("--type is required")
			}
			if intervalSeconds <= 0 {
				return fmt.Errorf("--interval-seconds must be greater than 0")
			}
			if timeoutSeconds <= 0 {
				return fmt.Errorf("--timeout-seconds must be greater than 0")
			}
			if failuresBeforeUnhealthy <= 0 {
				return fmt.Errorf("--failures-before-unhealthy must be greater than 0")
			}
			if successesBeforeHealthy <= 0 {
				return fmt.Errorf("--successes-before-healthy must be greater than 0")
			}

			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewHealthServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.CreateHealthCheck(ctx, connect.NewRequest(&registryv1.CreateHealthCheckRequest{
				InstanceId:              instanceID,
				EndpointId:              endpointID,
				Name:                    name,
				Type:                    checkType,
				Enabled:                 enabled,
				IntervalSeconds:         intervalSeconds,
				TimeoutSeconds:          timeoutSeconds,
				FailuresBeforeUnhealthy: failuresBeforeUnhealthy,
				SuccessesBeforeHealthy:  successesBeforeHealthy,
				Description:             description,
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}

	c.Flags().StringVar(&instanceID, "instance-id", "", "Instance ID")
	c.Flags().StringVar(&endpointID, "endpoint-id", "", "Optional endpoint ID")
	c.Flags().StringVar(&name, "name", "", "Health check name")
	c.Flags().StringVar(&typeName, "type", "", "Health check type: http|https|grpc|tcp|udp|heartbeat")
	c.Flags().BoolVar(&enabled, "enabled", true, "Enable health check")
	c.Flags().Int32Var(&intervalSeconds, "interval-seconds", 10, "Interval in seconds")
	c.Flags().Int32Var(&timeoutSeconds, "timeout-seconds", 3, "Timeout in seconds")
	c.Flags().Int32Var(&failuresBeforeUnhealthy, "failures-before-unhealthy", 3, "Deprecated compatibility field; current health uses the latest result")
	c.Flags().Int32Var(&successesBeforeHealthy, "successes-before-healthy", 2, "Deprecated compatibility field; current health uses the latest result")
	c.Flags().StringVar(&description, "description", "", "Health check description")
	return c
}

func newHealthCheckListCommand() *cobra.Command {
	var instanceID string
	var includeDisabled bool
	c := &cobra.Command{
		Use:   "list",
		Short: "List health checks",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewHealthServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.ListHealthChecks(ctx, connect.NewRequest(&registryv1.ListHealthChecksRequest{
				InstanceId:      instanceID,
				IncludeDisabled: includeDisabled,
				Pagination:      &registryv1.PaginationRequest{PageSize: 50},
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&instanceID, "instance-id", "", "Filter by instance ID")
	c.Flags().BoolVar(&includeDisabled, "include-disabled", false, "Include disabled health checks")
	return c
}

func newHealthCheckRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run <health-check-id>",
		Short: "Run a health check immediately",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewHealthServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.RunHealthCheck(ctx, connect.NewRequest(&registryv1.RunHealthCheckRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
}

func parseHealthCheckTypeFlag(value string) (registryv1.HealthCheckType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return registryv1.HealthCheckType_HEALTH_CHECK_TYPE_UNSPECIFIED, nil
	case "http":
		return registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP, nil
	case "https":
		return registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTPS, nil
	case "grpc":
		return registryv1.HealthCheckType_HEALTH_CHECK_TYPE_GRPC, nil
	case "tcp":
		return registryv1.HealthCheckType_HEALTH_CHECK_TYPE_TCP, nil
	case "udp":
		return registryv1.HealthCheckType_HEALTH_CHECK_TYPE_UDP, nil
	case "heartbeat":
		return registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HEARTBEAT, nil
	default:
		return registryv1.HealthCheckType_HEALTH_CHECK_TYPE_UNSPECIFIED, fmt.Errorf("unsupported --type %q; use http, https, grpc, tcp, udp, or heartbeat", value)
	}
}
