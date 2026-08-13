package health

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultHTTPMethod = http.MethodGet
)

// Target contains the network location used to execute a health check.
type Target struct {
	Check   *registryv1.HealthCheck
	Address string
	Port    int32
	Path    string
}

// Executor runs health checks and returns normalized health results.
type Executor struct {
	httpClient *http.Client
}

// NewExecutor creates a check executor. A nil client uses http.DefaultClient.
func NewExecutor(httpClient *http.Client) *Executor {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Executor{httpClient: httpClient}
}

// Execute runs a single health check. HTTP and HTTPS checks are currently supported.
func (e *Executor) Execute(ctx context.Context, target Target) *registryv1.HealthResult {
	start := time.Now()
	result := baseResult(target.Check, start)

	if target.Check == nil {
		result.ErrorType = "invalid_target"
		result.ErrorMessage = "health check is required"
		return result
	}

	timeout := time.Duration(target.Check.GetTimeoutSeconds()) * time.Second
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	switch target.Check.GetType() {
	case registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP, registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTPS:
		e.executeHTTP(ctx, target, result, start)
	case registryv1.HealthCheckType_HEALTH_CHECK_TYPE_TCP, registryv1.HealthCheckType_HEALTH_CHECK_TYPE_GRPC:
		executeTCP(ctx, target, result, start)
	default:
		result.ErrorType = "unsupported_type"
		result.ErrorMessage = fmt.Sprintf("unsupported health check type %s", target.Check.GetType().String())
	}

	return result
}

func executeTCP(ctx context.Context, target Target, result *registryv1.HealthResult, start time.Time) {
	if target.Address == "" {
		result.ErrorType = "invalid_target"
		result.ErrorMessage = "address is required"
		return
	}
	if target.Port <= 0 {
		result.ErrorType = "invalid_target"
		result.ErrorMessage = "port must be greater than 0"
		return
	}

	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(target.Address, strconv.Itoa(int(target.Port))))
	result.LatencyMs = int32(time.Since(start).Milliseconds())
	if err != nil {
		result.ErrorType = categorizeError(err)
		result.ErrorMessage = err.Error()
		return
	}
	defer conn.Close()

	result.Success = true
}

func (e *Executor) executeHTTP(ctx context.Context, target Target, result *registryv1.HealthResult, start time.Time) {
	checkURL, err := buildHTTPURL(target)
	if err != nil {
		result.ErrorType = "invalid_target"
		result.ErrorMessage = err.Error()
		return
	}

	method := target.Check.GetMetadata()["httpMethod"]
	if method == "" {
		method = defaultHTTPMethod
	}

	req, err := http.NewRequestWithContext(ctx, method, checkURL, nil)
	if err != nil {
		result.ErrorType = "invalid_request"
		result.ErrorMessage = err.Error()
		return
	}

	resp, err := e.httpClient.Do(req)
	result.LatencyMs = int32(time.Since(start).Milliseconds())
	if err != nil {
		result.ErrorType = categorizeError(err)
		result.ErrorMessage = err.Error()
		return
	}
	defer resp.Body.Close()

	result.StatusCode = int32(resp.StatusCode)
	result.Success = expectedStatus(target.Check, resp.StatusCode)
	if !result.Success {
		result.ErrorType = "unexpected_status"
		result.ErrorMessage = fmt.Sprintf("received HTTP status %d", resp.StatusCode)
	}
}

func baseResult(check *registryv1.HealthCheck, timestamp time.Time) *registryv1.HealthResult {
	result := &registryv1.HealthResult{
		Id:        uuid.NewString(),
		Timestamp: timestamppb.New(timestamp.UTC()),
		Metadata:  map[string]string{},
	}
	if check != nil {
		result.HealthCheckId = check.GetId()
		result.InstanceId = check.GetInstanceId()
	}
	return result
}

func buildHTTPURL(target Target) (string, error) {
	if target.Address == "" {
		return "", errors.New("address is required")
	}
	if target.Port <= 0 {
		return "", errors.New("port must be greater than 0")
	}

	scheme := "http"
	if target.Check.GetType() == registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTPS {
		scheme = "https"
	}

	path := target.Path
	if path == "" {
		path = target.Check.GetMetadata()["path"]
	}
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return (&url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(target.Address, strconv.Itoa(int(target.Port))),
		Path:   path,
	}).String(), nil
}

func expectedStatus(check *registryv1.HealthCheck, statusCode int) bool {
	raw := check.GetMetadata()["expectedStatus"]
	if raw == "" {
		return statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices
	}

	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			min, minErr := strconv.Atoi(strings.TrimSpace(bounds[0]))
			max, maxErr := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if minErr == nil && maxErr == nil && statusCode >= min && statusCode <= max {
				return true
			}
			continue
		}
		if code, err := strconv.Atoi(part); err == nil && statusCode == code {
			return true
		}
	}
	return false
}

func categorizeError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns_error"
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return "connection_error"
	}

	return "request_error"
}
