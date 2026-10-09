package main

import (
	"encoding/json"
	"fmt"
	"github.com/company/service-registry/internal/address"
	"github.com/company/service-registry/internal/contract"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type publicCLIRegistration = contract.Registration

func newPublicServicesCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "services", Short: "Manage services using the public registry model"}
	cmd.AddCommand(newPublicBootstrapCommand("services"), newPublicRegisterCommand(), newPublicDeregisterCommand(), newPublicListCommand(), newPublicInstancesCommand(), newPublicEndpointsCommand(), newPublicResolveCommand())
	return cmd
}

func publicEnvironment(cmd *cobra.Command, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if value := os.Getenv("ALAUDA_ENVIRONMENT"); value != "" {
		return value
	}
	return cliConfig.Environment
}

func parsePublicPort(value string) (int32, error) {
	p, err := strconv.ParseInt(value, 10, 64)
	if err != nil || p < 1 || p > 65535 {
		return 0, fmt.Errorf("port must be between 1 and 65535")
	}
	return int32(p), nil
}

func newPublicRegisterCommand() *cobra.Command {
	var name, environment, instance, host, kind, path, file, description, primary string
	var port int64
	var endpoints []string
	var replace, enabled, endpointEnabled bool
	var tags, metadata map[string]string
	c := &cobra.Command{Use: "register", Short: "Register or update a service instance", Args: cobra.NoArgs,
		Long:    "Service and Environment must already exist. Registration is an UPSERT by Service + Environment + Instance name. Omitted fields/endpoints are preserved; --replace-endpoints explicitly removes omitted endpoints. The shorthand uses the address as Instance name, endpoint default, kind http and an empty path. A new singleton is Primary only when Primary is omitted. File fields use the same schema as REST; explicit flags override file values, then ALAUDA_ENVIRONMENT provides a missing Environment.",
		Example: "alauda services register --name Authentication.Grpc --environment stg --address lynx-authentication.lynx --port 81",
		RunE: func(cmd *cobra.Command, args []string) error {
			var input contract.Registration
			if file != "" {
				f, err := os.Open(file)
				if err != nil {
					return err
				}
				defer f.Close()
				dec := yaml.NewDecoder(f)
				dec.KnownFields(true)
				if err := dec.Decode(&input); err != nil {
					return fmt.Errorf("registration file: %w", err)
				}
				var extra any
				if err := dec.Decode(&extra); err != io.EOF {
					return fmt.Errorf("registration file must contain exactly one document")
				}
			}
			if cmd.Flags().Changed("name") {
				input.Service = name
			}
			if cmd.Flags().Changed("environment") {
				input.Environment = environment
			}
			input.Environment = publicEnvironment(cmd, input.Environment)
			if cmd.Flags().Changed("instance") {
				input.Instance.Name = instance
			}
			if cmd.Flags().Changed("address") {
				input.Instance.Address = contract.Pointer(host)
			}
			if cmd.Flags().Changed("description") {
				input.Instance.Description = contract.Pointer(description)
			}
			if cmd.Flags().Changed("enabled") {
				input.Instance.Enabled = contract.Pointer(enabled)
			}
			if cmd.Flags().Changed("tags") {
				input.Instance.Tags = tags
			}
			if cmd.Flags().Changed("metadata") {
				input.Instance.Metadata = metadata
			}
			if cmd.Flags().Changed("replace-endpoints") {
				input.Replace = replace
			}
			if len(endpoints) > 0 {
				input.Endpoints = nil
				for _, raw := range endpoints {
					parts := strings.Split(raw, ",")
					if len(parts) == 1 {
						if !cmd.Flags().Changed("port") {
							return fmt.Errorf("--endpoint name requires --port")
						}
						input.Endpoints = append(input.Endpoints, contract.EndpointPatch{Name: parts[0]})
						continue
					}
					if len(parts) < 3 || len(parts) > 4 {
						return fmt.Errorf("--endpoint must be name or name,kind,port[,path]")
					}
					p, err := parsePublicPort(parts[2])
					if err != nil {
						return err
					}
					epPath := address.PublicPath(parts[1], "/")
					if len(parts) == 4 {
						epPath = parts[3]
					}
					input.Endpoints = append(input.Endpoints, contract.EndpointPatch{Name: parts[0], Kind: contract.Pointer(parts[1]), Port: contract.Pointer(p), Path: contract.Pointer(epPath)})
				}
			}
			if len(input.Endpoints) == 0 && cmd.Flags().Changed("port") {
				input.Endpoints = []contract.EndpointPatch{{Name: "default", Kind: contract.Pointer(kind), Path: contract.Pointer(address.PublicPath(kind, path))}}
			}
			if len(input.Endpoints) > 1 && (cmd.Flags().Changed("port") || cmd.Flags().Changed("kind") || cmd.Flags().Changed("path") || cmd.Flags().Changed("endpoint-enabled")) {
				return fmt.Errorf("endpoint override flags require exactly one endpoint")
			}
			if len(input.Endpoints) == 1 {
				ep := &input.Endpoints[0]
				if cmd.Flags().Changed("port") {
					p, err := parsePublicPort(strconv.FormatInt(port, 10))
					if err != nil {
						return err
					}
					ep.Port = contract.Pointer(p)
				}
				if cmd.Flags().Changed("kind") || ep.Kind == nil && cmd.Flags().Changed("port") && (file == "" || len(endpoints) > 0) {
					ep.Kind = contract.Pointer(kind)
				}
				if cmd.Flags().Changed("path") || ep.Path == nil && cmd.Flags().Changed("port") && (file == "" || len(endpoints) > 0) {
					value := path
					endpointKind := ep.Kind
					if !cmd.Flags().Changed("path") && endpointKind != nil {
						value = address.PublicPath(*endpointKind, value)
					}
					ep.Path = contract.Pointer(value)
				}
				if cmd.Flags().Changed("endpoint-enabled") {
					ep.Enabled = contract.Pointer(endpointEnabled)
				}
			}
			if primary != "" {
				found := false
				for i := range input.Endpoints {
					if input.Endpoints[i].Name == primary {
						input.Endpoints[i].Primary = contract.Pointer(true)
						found = true
					}
				}
				if !found {
					return fmt.Errorf("--primary-endpoint must name a submitted endpoint")
				}
			}
			for _, ep := range input.Endpoints {
				endpointKind := ep.Kind
				if endpointKind != nil {
					endpointPath := ""
					if ep.Path != nil {
						endpointPath = *ep.Path
					}
					if err := address.ValidateEndpointPath(*endpointKind, endpointPath); err != nil {
						return fmt.Errorf("endpoint %q: %w", ep.Name, err)
					}
				}
				if ep.Port != nil {
					if _, err := parsePublicPort(strconv.FormatInt(int64(*ep.Port), 10)); err != nil {
						return err
					}
				}
			}
			if input.Service == "" {
				return fmt.Errorf("--name or file service is required")
			}
			if input.Environment == "" {
				return fmt.Errorf("--environment, file environment or ALAUDA_ENVIRONMENT is required")
			}
			if input.Instance.Name == "" && input.Instance.Address != nil {
				input.Instance.Name = *input.Instance.Address
			}
			if input.Instance.Name == "" || strings.ContainsAny(input.Instance.Name, ":/[]") {
				return fmt.Errorf("--instance is required when the address is not a valid resource name, including IPv6")
			}
			var out any
			if err := doJSON(http.MethodPost, "/api/v1/services/"+url.PathEscape(input.Service)+"/instances", strings.NewReader(mustJSON(input)), &out); err != nil {
				return err
			}
			if !cliConfig.Quiet {
				printPublicOutput(out)
			}
			return nil
		}}
	c.Flags().StringVar(&name, "name", "", "Service key")
	c.Flags().StringVar(&environment, "environment", "", "Environment key (otherwise file or ALAUDA_ENVIRONMENT)")
	c.Flags().StringVar(&instance, "instance", "", "Instance name (defaults to bare address)")
	c.Flags().StringVar(&host, "address", "", "Bare DNS hostname or IP")
	c.Flags().StringArrayVar(&endpoints, "endpoint", nil, "Repeat: name,kind,port[,path], or name with --port")
	c.Flags().StringArrayVar(&endpoints, "endpoints", nil, "Legacy alias for --endpoint")
	_ = c.Flags().MarkDeprecated("endpoints", "use repeated --endpoint")
	c.Flags().StringVar(&kind, "kind", "http", "Shorthand endpoint kind: http|https|grpc|postgres|redis|tcp|udp|custom")
	c.Flags().StringVar(&path, "path", "", "HTTP/HTTPS endpoint path; omitted for non-HTTP kinds")
	c.Flags().Int64Var(&port, "port", 0, "Endpoint port 1..65535")
	c.Flags().StringVar(&description, "description", "", "Instance description; empty explicitly clears")
	c.Flags().StringVar(&file, "file", "", "Strict YAML or JSON registration file")
	c.Flags().BoolVar(&replace, "replace-endpoints", false, "Retire endpoints omitted from registration")
	c.Flags().BoolVar(&enabled, "enabled", true, "Explicit Instance Enabled update")
	c.Flags().BoolVar(&endpointEnabled, "endpoint-enabled", true, "Explicit singleton Endpoint Enabled update")
	c.Flags().StringVar(&primary, "primary-endpoint", "", "Promote this submitted endpoint atomically")
	c.Flags().StringToStringVar(&tags, "tags", nil, "Instance tags, key=value pairs")
	c.Flags().StringToStringVar(&metadata, "metadata", nil, "Instance metadata, key=value pairs")
	return c
}

func newPublicDeregisterCommand() *cobra.Command {
	var env string
	var yes bool
	c := &cobra.Command{Use: "deregister", Short: "Deregister a service instance", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if !yes {
			return fmt.Errorf("--yes is required for non-interactive deregistration")
		}
		name, _ := cmd.Flags().GetString("name")
		instance, _ := cmd.Flags().GetString("instance")
		q := url.Values{"environment": {publicEnvironment(cmd, env)}}
		if name == "" || instance == "" || q.Get("environment") == "" {
			return fmt.Errorf("--name, --instance and --environment are required")
		}
		if err := doJSON(http.MethodDelete, "/api/v1/services/"+url.PathEscape(name)+"/instances/"+url.PathEscape(instance)+"?"+q.Encode(), nil, nil); err != nil {
			return err
		}
		return nil
	}}
	c.Flags().String("name", "", "Service key")
	c.Flags().StringVar(&env, "environment", "", "Environment key")
	c.Flags().String("instance", "", "Instance name")
	c.Flags().BoolVar(&yes, "yes", false, "Confirm deregistration")
	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("instance")
	return c
}
func newPublicListCommand() *cobra.Command {
	var env, token string
	var size int
	c := &cobra.Command{Use: "list", Short: "List services", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if size < 1 || size > 200 {
			return fmt.Errorf("--page-size must be between 1 and 200")
		}
		q := url.Values{"pageSize": {strconv.Itoa(size)}, "pageToken": {token}}
		if e := publicEnvironment(cmd, env); e != "" {
			q.Set("environment", e)
		}
		var out any
		if err := doJSON(http.MethodGet, "/api/v1/services?"+q.Encode(), nil, &out); err != nil {
			return err
		}
		printPublicOutput(out)
		return nil
	}}
	c.Flags().StringVar(&env, "environment", "", "Filter by Environment key")
	c.Flags().IntVar(&size, "page-size", 50, "Page size 1..200")
	c.Flags().StringVar(&token, "page-token", "", "Continue using nextPageToken")
	return c
}
func newPublicInstancesCommand() *cobra.Command {
	var env, token string
	var size int
	c := &cobra.Command{Use: "instances <service>", Short: "List service instances", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if size < 1 || size > 200 {
			return fmt.Errorf("--page-size must be between 1 and 200")
		}
		q := url.Values{"environment": {publicEnvironment(cmd, env)}, "pageSize": {strconv.Itoa(size)}, "pageToken": {token}}
		var out any
		if err := doJSON(http.MethodGet, "/api/v1/services/"+url.PathEscape(args[0])+"/instances?"+q.Encode(), nil, &out); err != nil {
			return err
		}
		printPublicOutput(out)
		return nil
	}}
	c.Flags().StringVar(&env, "environment", "", "Environment key")
	c.Flags().IntVar(&size, "page-size", 50, "Page size 1..200")
	c.Flags().StringVar(&token, "page-token", "", "Next page token")
	return c
}
func newPublicEndpointsCommand() *cobra.Command {
	var env, instance, token string
	var size int
	c := &cobra.Command{Use: "endpoints <service>", Short: "List registered endpoints, including disabled resources", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if instance == "" {
			return fmt.Errorf("--instance is required")
		}
		if size < 1 || size > 200 {
			return fmt.Errorf("--page-size must be between 1 and 200")
		}
		q := url.Values{"environment": {publicEnvironment(cmd, env)}, "pageSize": {strconv.Itoa(size)}, "pageToken": {token}}
		var out any
		if err := doJSON(http.MethodGet, "/api/v1/services/"+url.PathEscape(args[0])+"/instances/"+url.PathEscape(instance)+"/endpoints?"+q.Encode(), nil, &out); err != nil {
			return err
		}
		printPublicOutput(out)
		return nil
	}}
	c.Flags().StringVar(&env, "environment", "", "Environment key")
	c.Flags().StringVar(&instance, "instance", "", "Instance name")
	c.Flags().IntVar(&size, "page-size", 50, "Page size 1..200")
	c.Flags().StringVar(&token, "page-token", "", "Next page token")
	return c
}
func newPublicResolveCommand() *cobra.Command {
	var env, endpoint, health string
	c := &cobra.Command{Use: "resolve <service>", Short: "Resolve a usable service endpoint", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		q := url.Values{"environment": {publicEnvironment(cmd, env)}, "health": {health}}
		if q.Get("environment") == "" {
			return fmt.Errorf("--environment or ALAUDA_ENVIRONMENT is required")
		}
		if endpoint != "" {
			q.Set("endpoint", endpoint)
		}
		var out map[string]any
		if err := doJSON(http.MethodGet, "/api/v1/discovery/"+url.PathEscape(args[0])+"/resolve?"+q.Encode(), nil, &out); err != nil {
			return err
		}
		if cliConfig.Output == "value" {
			value, err := publicResolvedValue(out)
			if err != nil {
				return err
			}
			fmt.Println(value)
			return nil
		}
		printPublicOutput(out)
		return nil
	}}
	c.Flags().StringVar(&env, "environment", "", "Environment key")
	c.Flags().StringVar(&endpoint, "endpoint", "", "Endpoint name")
	c.Flags().StringVar(&health, "health", "usable", "Health policy: usable|healthy|all")
	return c
}

func publicResolvedValue(out map[string]any) (string, error) {
	if value, _ := out["resolvedValue"].(string); value != "" {
		return value, nil
	}
	instance, _ := out["instance"].(map[string]any)
	endpoint, _ := out["endpoint"].(map[string]any)
	host, _ := instance["address"].(string)
	if host == "" {
		return "", publicCLIError{Status: 502, Code: "invalid_server_response", Detail: "Resolve response did not contain an address."}
	}
	kind, _ := endpoint["kind"].(string)
	if kind == "" {
		return "", publicCLIError{Status: 502, Code: "invalid_server_response", Detail: "Resolve response did not contain an endpoint kind."}
	}
	portNumber, ok := endpoint["port"].(float64)
	if !ok {
		return "", publicCLIError{Status: 502, Code: "invalid_server_response", Detail: "Resolve response did not contain a port."}
	}
	path, _ := endpoint["path"].(string)
	return address.FormatValue(strings.ToLower(kind), host, int32(portNumber), path)
}
func mustJSON(value any) string { data, _ := json.Marshal(value); return string(data) }
func printPublicOutput(value any) {
	if cliConfig.Output == "table" {
		printPublicTable(value)
		return
	}
	printAny(value)
}

func publicCell(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
func printPublicTable(value any) {
	item, ok := value.(map[string]any)
	if !ok {
		printAny(value)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	defer w.Flush()
	if item["address"] != nil && item["port"] != nil {
		fmt.Fprintln(w, "SERVICE\tENVIRONMENT\tINSTANCE\tENDPOINT\tKIND\tADDRESS\tPORT\tPATH")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", publicCell(item["service"]), publicCell(item["environment"]), publicCell(item["instance"]), publicCell(item["endpoint"]), publicCell(item["kind"]), publicCell(item["address"]), publicCell(item["port"]), publicCell(item["path"]))
		return
	}
	for _, key := range []string{"services", "instances", "endpoints", "environments", "checks", "results"} {
		rows, ok := item[key].([]any)
		if !ok {
			continue
		}
		columns := map[string][]string{"services": {"name", "displayName", "healthStatus", "description"}, "instances": {"name", "address", "enabled", "healthState"}, "endpoints": {"name", "kind", "port", "path", "primary", "enabled", "address"}, "environments": {"key", "name", "enabled"}, "checks": {"name", "endpoint", "type", "enabled", "intervalSeconds", "timeoutSeconds"}, "results": {"check", "instance", "endpoint", "timestamp", "success", "latencyMs", "statusCode"}}[key]
		headings := make([]string, len(columns))
		for i, column := range columns {
			headings[i] = strings.ToUpper(column)
		}
		fmt.Fprintln(w, strings.Join(headings, "\t"))
		for _, raw := range rows {
			row, _ := raw.(map[string]any)
			cells := make([]string, len(columns))
			for i, column := range columns {
				cells[i] = publicCell(row[column])
			}
			fmt.Fprintln(w, strings.Join(cells, "\t"))
		}
		if token := publicCell(item["nextPageToken"]); token != "" && !cliConfig.Quiet {
			fmt.Fprintln(os.Stderr, "nextPageToken:", token)
		}
		return
	}
	printAny(value)
}
