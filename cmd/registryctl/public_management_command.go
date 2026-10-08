package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

func newPublicBootstrapCommand(resource string) *cobra.Command {
	var key, name, description string
	c := &cobra.Command{Use: "create", Short: "Explicitly create a " + resource, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if key == "" {
			return fmt.Errorf("--name or --key is required")
		}
		body := map[string]string{"name": key, "description": description}
		if resource == "environments" {
			body["key"] = key
			if name != "" {
				body["name"] = name
			}
		} else if name != "" {
			body["displayName"] = name
		}
		var out any
		if err := doJSON(http.MethodPost, "/api/v1/"+resource, strings.NewReader(mustJSON(body)), &out); err != nil {
			return err
		}
		if !cliConfig.Quiet {
			printPublicOutput(out)
		}
		return nil
	}}
	if resource == "environments" {
		c.Flags().StringVar(&key, "key", "", "Environment key")
	} else {
		c.Flags().StringVar(&key, "name", "", "Service key")
	}
	c.Flags().StringVar(&name, "display-name", "", "Display name (defaults to key)")
	c.Flags().StringVar(&description, "description", "", "Description")
	return c
}

func newPublicEnvironmentsCommand() *cobra.Command {
	c := &cobra.Command{Use: "environments", Short: "Manage Environments by key"}
	var token string
	var size int
	list := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if size < 1 || size > 200 {
			return fmt.Errorf("--page-size must be between 1 and 200")
		}
		q := url.Values{"pageSize": {fmt.Sprint(size)}, "pageToken": {token}}
		var out any
		if err := doJSON(http.MethodGet, "/api/v1/environments?"+q.Encode(), nil, &out); err != nil {
			return err
		}
		printPublicOutput(out)
		return nil
	}}
	list.Flags().IntVar(&size, "page-size", 50, "Page size 1..200")
	list.Flags().StringVar(&token, "page-token", "", "Next page token")
	c.AddCommand(list, newPublicBootstrapCommand("environments"))
	return c
}

func newPublicHealthCommand() *cobra.Command {
	c := &cobra.Command{Use: "health", Short: "Manage Health Checks using public resource names"}
	checks := &cobra.Command{Use: "checks", Short: "Create, list and execute Health Checks"}
	for _, action := range []string{"list", "create", "run"} {
		checks.AddCommand(newPublicHealthAction(action))
	}
	c.AddCommand(checks)
	results := &cobra.Command{Use: "results", Short: "Query Health results"}
	results.AddCommand(newPublicHealthResultsCommand())
	c.AddCommand(results)
	// Preserve the previous health alias grammar during migration.
	for _, legacy := range []*cobra.Command{newHealthCheckCreateCommand(), newHealthCheckListCommand(), newHealthCheckRunCommand()} {
		legacy.Deprecated = "use health checks commands with public resource names"
		c.AddCommand(legacy)
	}
	return c
}

func newPublicHealthResultsCommand() *cobra.Command {
	var service, env, instance, endpoint, check, from, to, token string
	var size int
	c := &cobra.Command{Use: "list", Short: "List Service Health results by public names", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if service == "" || publicEnvironment(cmd, env) == "" {
			return fmt.Errorf("--service and --environment are required")
		}
		q := url.Values{"environment": {publicEnvironment(cmd, env)}, "instance": {instance}, "endpoint": {endpoint}, "check": {check}, "from": {from}, "to": {to}, "pageSize": {fmt.Sprint(size)}, "pageToken": {token}}
		var out any
		if err := doJSON(http.MethodGet, "/api/v1/services/"+url.PathEscape(service)+"/health-results?"+q.Encode(), nil, &out); err != nil {
			return err
		}
		printPublicOutput(out)
		return nil
	}}
	c.Flags().StringVar(&service, "service", "", "Service key")
	c.Flags().StringVar(&env, "environment", "", "Environment key")
	c.Flags().StringVar(&instance, "instance", "", "Instance name filter")
	c.Flags().StringVar(&endpoint, "endpoint", "", "Endpoint name filter")
	c.Flags().StringVar(&check, "check", "", "Health Check name filter")
	c.Flags().StringVar(&from, "from", "", "UTC RFC3339 timestamp")
	c.Flags().StringVar(&to, "to", "", "UTC RFC3339 timestamp")
	c.Flags().IntVar(&size, "page-size", 50, "Page size 1..200")
	c.Flags().StringVar(&token, "page-token", "", "Next page token")
	return c
}

func newPublicHealthAction(action string) *cobra.Command {
	var service, env, instance, endpoint, name, typ, token string
	var enabled bool
	var interval, timeout, size int32
	c := &cobra.Command{Use: action, Short: action + " Health Checks by public names", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		environment := publicEnvironment(cmd, env)
		if service == "" || environment == "" || instance == "" {
			return fmt.Errorf("--service, --environment and --instance are required")
		}
		path := "/api/v1/services/" + url.PathEscape(service) + "/instances/" + url.PathEscape(instance) + "/health-checks"
		q := url.Values{"environment": {environment}, "pageToken": {token}, "pageSize": {fmt.Sprint(size)}}
		method := http.MethodGet
		var body *strings.Reader
		if action == "create" {
			if name == "" || endpoint == "" {
				return fmt.Errorf("--name and --endpoint are required")
			}
			method = http.MethodPost
			body = strings.NewReader(mustJSON(map[string]any{"name": name, "endpoint": endpoint, "type": typ, "enabled": enabled, "intervalSeconds": interval, "timeoutSeconds": timeout}))
		}
		if action == "run" {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			method = http.MethodPost
			path += "/" + url.PathEscape(name) + "/run"
		}
		var out any
		var err error
		if body == nil {
			err = doJSON(method, path+"?"+q.Encode(), nil, &out)
		} else {
			err = doJSON(method, path+"?"+q.Encode(), body, &out)
		}
		if err != nil {
			return err
		}
		printPublicOutput(out)
		return nil
	}}
	c.Flags().StringVar(&service, "service", "", "Service key")
	c.Flags().StringVar(&env, "environment", "", "Environment key")
	c.Flags().StringVar(&instance, "instance", "", "Instance name")
	c.Flags().StringVar(&name, "name", "", "Health Check name")
	if action == "create" {
		c.Flags().StringVar(&endpoint, "endpoint", "", "Endpoint name")
		c.Flags().StringVar(&typ, "type", "http", "http|tcp|dns")
		c.Flags().BoolVar(&enabled, "enabled", true, "Enabled")
		c.Flags().Int32Var(&interval, "interval-seconds", 30, "Check interval")
		c.Flags().Int32Var(&timeout, "timeout-seconds", 5, "Execution timeout")
	}
	c.Flags().StringVar(&token, "page-token", "", "Next page token")
	c.Flags().Int32Var(&size, "page-size", 50, "Page size 1..200")
	return c
}
