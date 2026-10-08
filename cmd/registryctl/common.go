package main

import (
	"connectrpc.com/connect"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

type config struct {
	ServerURL   string
	Output      string
	Timeout     time.Duration
	Token       string
	Quiet       bool
	ConfigFile  string
	Environment string
}

type publicCLIError struct {
	Status int                 `json:"status"`
	Code   string              `json:"code"`
	Detail string              `json:"detail"`
	Errors map[string][]string `json:"errors"`
}

func (e publicCLIError) Error() string {
	message := e.Code
	if e.Detail != "" {
		message += ": " + e.Detail
	}
	fields := make([]string, 0, len(e.Errors))
	for field := range e.Errors {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	for _, field := range fields {
		messages := e.Errors[field]
		if len(messages) > 0 {
			message += fmt.Sprintf(" (%s: %s)", field, strings.Join(messages, ", "))
		}
	}
	return message
}

func defaultCLIConfig() *config {
	server := os.Getenv("ALAUDA_URL")
	if server == "" {
		server = "http://localhost:9700"
	}
	return &config{
		ServerURL:  server,
		Output:     "pretty",
		Timeout:    10 * time.Second,
		ConfigFile: defaultConfigPath(),
	}
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "alauda", "config.yaml")
}

func loadNonsecretConfig() (string, string, string, error) {
	if cliConfig.ConfigFile == "" {
		return "", "", "", nil
	}
	f, err := os.Open(cliConfig.ConfigFile)
	if os.IsNotExist(err) && cliConfig.ConfigFile == defaultConfigPath() {
		return "", "", "", nil
	}
	if err != nil {
		return "", "", "", err
	}
	defer f.Close()
	var cfg struct {
		Server      string `yaml:"server"`
		Environment string `yaml:"environment"`
		Output      string `yaml:"output"`
	}
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err = dec.Decode(&cfg); err != nil {
		return "", "", "", fmt.Errorf("nonsecret config: %w", err)
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return "", "", "", fmt.Errorf("config must contain one document")
	}
	return cfg.Server, cfg.Environment, cfg.Output, nil
}

func newContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cliConfig.Timeout)
}

func printProto(msg proto.Message) {
	if cliConfig.Output == "json" {
		b, _ := protojson.Marshal(msg)
		fmt.Println(string(b))
		return
	}
	if cliConfig.Output == "yaml" {
		jsonBytes, _ := protojson.Marshal(msg)
		var value any
		_ = json.Unmarshal(jsonBytes, &value)
		yamlBytes, _ := yaml.Marshal(value)
		fmt.Print(string(yamlBytes))
		return
	}
	b, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(msg)
	fmt.Println(string(b))
}

func doJSON(method, path string, body io.Reader, out any) error {
	req, err := http.NewRequest(method, strings.TrimRight(cliConfig.ServerURL, "/")+path, body)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := newHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return publicCLIError{Status: 502, Code: "invalid_server_response", Detail: "Unable to read server response."}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiError publicCLIError
		if json.Unmarshal(responseBody, &apiError) == nil && apiError.Code != "" {
			apiError.Status = resp.StatusCode
			return apiError
		}
		return publicCLIError{Status: resp.StatusCode, Code: "http_error", Detail: http.StatusText(resp.StatusCode)}
	}
	if out != nil && len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, out); err != nil {
			return publicCLIError{Status: 502, Code: "invalid_server_response", Detail: "Server response is not valid JSON."}
		}
	}
	return nil
}

func printAny(value any) {
	switch cliConfig.Output {
	case "json":
		encoded, _ := json.Marshal(value)
		fmt.Println(string(encoded))
	case "yaml":
		encoded, _ := yaml.Marshal(value)
		fmt.Print(string(encoded))
	default:
		encoded, _ := json.MarshalIndent(value, "", "  ")
		fmt.Println(string(encoded))
	}
}

func newHTTPClient() *http.Client {
	token := cliConfig.Token
	if token == "" {
		token = os.Getenv("ALAUDA_TOKEN")
		if token == "" {
			token = os.Getenv("REGISTRY_TOKEN")
		}
	}
	return &http.Client{
		Timeout:   cliConfig.Timeout,
		Transport: authTransport{token: token, base: http.DefaultTransport},
	}
}

type authTransport struct {
	token string
	base  http.RoundTripper
}

func (t authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token != "" {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var api publicCLIError
	if errors.As(err, &api) {
		if api.Code == "no_healthy_instance" {
			return 7
		}
		switch api.Status {
		case 400, 413, 415, 422:
			return 2
		case 401:
			return 3
		case 403:
			return 4
		case 404:
			return 5
		case 409:
			return 6
		default:
			return 8
		}
	}
	var netErr interface{ Timeout() bool }
	if errors.As(err, &netErr) {
		return 8
	}
	var rpc *connect.Error
	if errors.As(err, &rpc) {
		switch rpc.Code() {
		case connect.CodeInvalidArgument:
			return 2
		case connect.CodeUnauthenticated:
			return 3
		case connect.CodePermissionDenied:
			return 4
		case connect.CodeNotFound:
			return 5
		case connect.CodeAlreadyExists, connect.CodeFailedPrecondition:
			return 6
		default:
			return 8
		}
	}
	return 2
}

func safeError(err error) string {
	message := err.Error()
	for _, secret := range []string{cliConfig.Token, os.Getenv("ALAUDA_TOKEN"), os.Getenv("REGISTRY_TOKEN")} {
		if secret != "" {
			message = strings.ReplaceAll(message, secret, "[REDACTED]")
		}
	}
	return message
}
