package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

type config struct {
	ServerURL string
	Output    string
	Timeout   time.Duration
	Token     string
}

func defaultCLIConfig() *config {
	return &config{
		ServerURL: "http://localhost:9700",
		Output:    "pretty",
		Timeout:   10 * time.Second,
	}
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
	responseBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s failed: %s", method, path, string(responseBody))
	}
	if out != nil && len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, out); err != nil {
			return err
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
		token = os.Getenv("REGISTRY_TOKEN")
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
