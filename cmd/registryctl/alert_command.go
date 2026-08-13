package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
	"github.com/spf13/cobra"
)

func newAlertPolicyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alert-policy",
		Short: "Manage alert policies",
	}
	cmd.AddCommand(newAlertPolicyCreateCommand())
	cmd.AddCommand(newAlertPolicyListCommand())
	return cmd
}

func newAlertPolicyCreateCommand() *cobra.Command {
	var deploymentID, environmentID string
	var enabled, sendRecovery bool
	var cooldownMinutes int32
	var notifyOn, channelIDs []string
	var filters map[string]string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create alert policy",
		RunE: func(cmd *cobra.Command, args []string) error {
			if deploymentID == "" && environmentID == "" {
				return fmt.Errorf("--deployment-id or --environment-id is required")
			}
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewAlertServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.CreateAlertPolicy(ctx, connect.NewRequest(&registryv1.CreateAlertPolicyRequest{
				DeploymentId:             deploymentID,
				EnvironmentId:            environmentID,
				Enabled:                  enabled,
				NotifyOn:                 normalizeList(notifyOn),
				CooldownMinutes:          cooldownMinutes,
				SendRecoveryNotification: sendRecovery,
				Filters:                  filters,
				ChannelIds:               normalizeList(channelIDs),
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&deploymentID, "deployment-id", "", "Deployment ID")
	c.Flags().StringVar(&environmentID, "environment-id", "", "Environment ID")
	c.Flags().BoolVar(&enabled, "enabled", true, "Enable alert policy")
	c.Flags().StringSliceVar(&notifyOn, "notify-on", []string{"unhealthy", "recovered"}, "Notification triggers")
	c.Flags().Int32Var(&cooldownMinutes, "cooldown-minutes", 10, "Cooldown in minutes")
	c.Flags().BoolVar(&sendRecovery, "send-recovery", true, "Send recovery notifications")
	c.Flags().StringToStringVar(&filters, "filter", map[string]string{}, "Policy filters as key=value")
	c.Flags().StringSliceVar(&channelIDs, "channel", []string{}, "Notification channel ID")
	return c
}

func newAlertPolicyListCommand() *cobra.Command {
	var deploymentID, environmentID string
	c := &cobra.Command{
		Use:   "list",
		Short: "List alert policies",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewAlertServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.ListAlertPolicies(ctx, connect.NewRequest(&registryv1.ListAlertPoliciesRequest{
				DeploymentId:  deploymentID,
				EnvironmentId: environmentID,
				Pagination:    &registryv1.PaginationRequest{PageSize: 50},
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&deploymentID, "deployment-id", "", "Filter by deployment ID")
	c.Flags().StringVar(&environmentID, "environment-id", "", "Filter by environment ID")
	return c
}

func newNotificationChannelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notification-channel",
		Short: "Manage notification channels",
	}
	cmd.AddCommand(newNotificationChannelCreateCommand())
	cmd.AddCommand(newNotificationChannelListCommand())
	cmd.AddCommand(newNotificationChannelTestCommand())
	return cmd
}

func newNotificationChannelCreateCommand() *cobra.Command {
	var channelType, name, description string
	var enabled bool
	var configuration, retryPolicy, tags map[string]string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create notification channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			if channelType == "" || name == "" {
				return fmt.Errorf("--type and --name are required")
			}
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewAlertServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.CreateNotificationChannel(ctx, connect.NewRequest(&registryv1.CreateNotificationChannelRequest{
				Type:          channelType,
				Name:          name,
				Enabled:       enabled,
				Description:   description,
				Configuration: configuration,
				RetryPolicy:   retryPolicy,
				Tags:          tags,
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
	c.Flags().StringVar(&channelType, "type", "", "Channel type: webhook|email")
	c.Flags().StringVar(&name, "name", "", "Channel name")
	c.Flags().BoolVar(&enabled, "enabled", true, "Enable notification channel")
	c.Flags().StringVar(&description, "description", "", "Channel description")
	c.Flags().StringToStringVar(&configuration, "config", map[string]string{}, "Channel configuration as key=value")
	c.Flags().StringToStringVar(&retryPolicy, "retry-policy", map[string]string{}, "Retry policy as key=value")
	c.Flags().StringToStringVar(&tags, "tag", map[string]string{}, "Tags as key=value")
	return c
}

func newNotificationChannelListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List notification channels",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newContext()
			defer cancel()

			client := registryv1connect.NewAlertServiceClient(newHTTPClient(), cliConfig.ServerURL)
			resp, err := client.ListNotificationChannels(ctx, connect.NewRequest(&registryv1.ListNotificationChannelsRequest{
				Pagination: &registryv1.PaginationRequest{PageSize: 50},
			}))
			if err != nil {
				return err
			}
			printProto(resp.Msg)
			return nil
		},
	}
}

func newNotificationChannelTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test <channel-id>",
		Short: "Send a test notification",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := http.NewRequest(http.MethodPost, strings.TrimRight(cliConfig.ServerURL, "/")+"/api/v1/alerts/test/"+args[0], nil)
			if err != nil {
				return err
			}
			resp, err := newHTTPClient().Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("test notification failed: %s", strings.TrimSpace(string(body)))
			}
			fmt.Println(strings.TrimSpace(string(body)))
			return nil
		},
	}
}

func normalizeList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
