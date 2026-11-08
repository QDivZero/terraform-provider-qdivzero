// Package qdivzero contains the HTTP client used by the Terraform provider.
package qdivzero

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.qdiv0.com"

type Config struct {
	BaseURL          string
	APIKey           string
	AccountID        string
	PrivateBetaToken string
}

type Client struct {
	baseURL          string
	apiKey           string
	accountID        string
	privateBetaToken string
	http             *http.Client
}

type InstanceInput struct {
	Name                              string           `json:"name"`
	HuggingfaceRepoID                 string           `json:"huggingface_repo_id,omitempty"`
	Description                       *string          `json:"description,omitempty"`
	ServingName                       *string          `json:"serving_name,omitempty"`
	RuntimePreset                     *string          `json:"runtime_preset,omitempty"`
	SchedulingMode                    *string          `json:"scheduling_mode,omitempty"`
	ServerlessEnabled                 *bool            `json:"serverless_enabled,omitempty"`
	IdleTimeoutSeconds                *int64           `json:"idle_timeout_seconds,omitempty"`
	GPUPreferences                    *[]GPUPreference `json:"gpu_preferences,omitempty"`
	SmartRegion                       *string          `json:"smart_region,omitempty"`
	SmartRegions                      *[]string        `json:"smart_regions,omitempty"`
	SmartMaxPricePerHourEUR           *float64         `json:"smart_max_price_per_hour_eur,omitempty"`
	SmartMinGPUClass                  *string          `json:"smart_min_gpu_class,omitempty"`
	SmartMinTotalTFLOPS               *float64         `json:"smart_min_total_tflops,omitempty"`
	SmartAllowSpot                    *bool            `json:"smart_allow_spot,omitempty"`
	SmartAllowCommunity               *bool            `json:"smart_allow_community,omitempty"`
	SmartProviderPreference           *string          `json:"smart_provider_preference,omitempty"`
	SmartProviderFilterMode           *string          `json:"smart_provider_filter_mode,omitempty"`
	SmartSelectionLabel               *string          `json:"smart_selection_label,omitempty"`
	CostOptimizationEnabled           *bool            `json:"cost_optimization_enabled,omitempty"`
	CostOptimizationMinSavingsPercent *float64         `json:"cost_optimization_min_savings_percent,omitempty"`
	WorkloadKind                      *string          `json:"workload_kind,omitempty"`
	SupportedProtocols                *[]string        `json:"supported_protocols,omitempty"`
	ReplicaCount                      *int64           `json:"replica_count,omitempty"`
}

type GPUPreference struct {
	PublicGPUID string `json:"public_gpu_id"`
	Count       int64  `json:"count"`
}

type Instance struct {
	ID                                string           `json:"id"`
	Name                              *string          `json:"name"`
	HuggingfaceRepoID                 *string          `json:"huggingface_repo_id"`
	Description                       *string          `json:"description"`
	ServingName                       *string          `json:"serving_name"`
	RuntimePreset                     *string          `json:"runtime_preset"`
	SchedulingMode                    *string          `json:"scheduling_mode"`
	ServerlessEnabled                 *bool            `json:"serverless_enabled"`
	IdleTimeoutSeconds                *int64           `json:"idle_timeout_seconds"`
	State                             *string          `json:"state"`
	CreatedAt                         *string          `json:"created_at"`
	GPUPreferences                    *[]GPUPreference `json:"gpu_preferences"`
	SmartRegion                       *string          `json:"smart_region"`
	SmartRegions                      *[]string        `json:"smart_regions"`
	SmartMaxPricePerHourEUR           *float64         `json:"smart_max_price_per_hour_eur"`
	SmartMinGPUClass                  *string          `json:"smart_min_gpu_class"`
	SmartMinTotalTFLOPS               *float64         `json:"smart_min_total_tflops"`
	SmartAllowSpot                    *bool            `json:"smart_allow_spot"`
	SmartAllowCommunity               *bool            `json:"smart_allow_community"`
	SmartProviderPreference           *string          `json:"smart_provider_preference"`
	SmartProviderFilterMode           *string          `json:"smart_provider_filter_mode"`
	SmartSelectionLabel               *string          `json:"smart_selection_label"`
	CostOptimizationEnabled           *bool            `json:"cost_optimization_enabled"`
	CostOptimizationMinSavingsPercent *float64         `json:"cost_optimization_min_savings_percent"`
	WorkloadKind                      *string          `json:"workload_kind"`
	SupportedProtocols                *[]string        `json:"supported_protocols"`
	ReplicaSetID                      *string          `json:"replica_set_id"`
}

type ServingEndpoint struct {
	ID          string  `json:"id"`
	Name        *string `json:"name"`
	DisplayName *string `json:"display_name"`
	State       *string `json:"state"`
	Enabled     *bool   `json:"enabled"`
}

type GPU struct {
	PublicGPUID       string   `json:"public_gpu_id"`
	DisplayName       *string  `json:"display_name"`
	ProviderCodename  *string  `json:"provider_codename"`
	MarketType        *string  `json:"market_type"`
	Continent         *string  `json:"continent"`
	Country           *string  `json:"country"`
	City              *string  `json:"city"`
	PricePerHourEUR   *float64 `json:"price_per_hour_eur"`
	VRAMGB            *float64 `json:"vram_gb"`
	GPUCount          *int64   `json:"gpu_count"`
	AvailabilityCount *int64   `json:"availability_count"`
	StockStatus       *string  `json:"stock_status"`
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("QDivZero API returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("QDivZero API returned HTTP %d: %s", e.StatusCode, e.Message)
}

func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("QDivZero API key is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("QDivZero API URL must be an HTTP(S) URL without credentials, query, or fragment")
	}
	return &Client{
		baseURL:          strings.TrimRight(parsed.String(), "/"),
		apiKey:           cfg.APIKey,
		accountID:        cfg.AccountID,
		privateBetaToken: cfg.PrivateBetaToken,
		http: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (c *Client) CreateInstance(ctx context.Context, input InstanceInput) (Instance, error) {
	var result Instance
	err := c.request(ctx, http.MethodPost, "/v1/instances", input, &result)
	return result, err
}

func (c *Client) GetInstance(ctx context.Context, id string) (Instance, error) {
	var result Instance
	err := c.request(ctx, http.MethodGet, "/v1/instances/"+url.PathEscape(id), nil, &result)
	return result, err
}

func (c *Client) UpdateInstance(ctx context.Context, id string, input InstanceInput) (Instance, error) {
	input.HuggingfaceRepoID = ""
	input.WorkloadKind = nil
	input.ReplicaCount = nil
	var result Instance
	err := c.request(ctx, http.MethodPatch, "/v1/instances/"+url.PathEscape(id), input, &result)
	return result, err
}

func (c *Client) DeleteInstance(ctx context.Context, id string) error {
	err := c.request(ctx, http.MethodDelete, "/v1/instances/"+url.PathEscape(id), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}

// EnsureInstanceState starts or stops an instance when needed, then polls until
// the requested stable state is observed or the context expires.
func (c *Client) EnsureInstanceState(ctx context.Context, id string, current Instance, desired string, interval time.Duration) (Instance, error) {
	if desired != "running" && desired != "stopped" {
		return current, fmt.Errorf("unsupported desired instance state %q", desired)
	}
	if interval <= 0 {
		return current, errors.New("instance polling interval must be positive")
	}
	actionSent := false
	for {
		state := ""
		if current.State != nil {
			state = strings.ToLower(*current.State)
		}
		if state == desired || (desired == "stopped" && state == "non_bootstrap") {
			return current, nil
		}
		if state == "error" || (state == "failed" && (desired != "running" || actionSent)) {
			return current, fmt.Errorf("instance %s entered failed state %q", id, state)
		}
		if !actionSent && ((desired == "running" && (state == "stopped" || state == "non_bootstrap" || state == "failed")) ||
			(desired == "stopped" && (state == "running" || state == "pending" || state == "starting"))) {
			var result Instance
			action := "start"
			if desired == "stopped" {
				action = "stop"
			}
			if err := c.request(ctx, http.MethodPost, "/v1/instances/"+url.PathEscape(id)+"/"+action, nil, &result); err != nil {
				return current, err
			}
			if result.ID != id {
				return current, errors.New("instance state transition returned a missing or different ID")
			}
			current = result
			actionSent = true
			continue
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return current, fmt.Errorf("waiting for instance %s to become %s: %w", id, desired, ctx.Err())
		case <-timer.C:
		}
		next, err := c.GetInstance(ctx, id)
		if err != nil {
			return current, err
		}
		if next.ID != id {
			return current, errors.New("instance status response returned a missing or different ID")
		}
		current = next
	}
}

func (c *Client) ListServingEndpoints(ctx context.Context) ([]ServingEndpoint, error) {
	var groups map[string][]ServingEndpoint
	if err := c.request(ctx, http.MethodGet, "/v1/serving-endpoints", nil, &groups); err != nil {
		return nil, err
	}
	var result []ServingEndpoint
	for _, endpoints := range groups {
		result = append(result, endpoints...)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (c *Client) ListGPUs(ctx context.Context) ([]GPU, error) {
	var response struct {
		GPUs []GPU `json:"gpus"`
	}
	if err := c.request(ctx, http.MethodGet, "/v1/instance-config/gpus", nil, &response); err != nil {
		return nil, err
	}
	sort.Slice(response.GPUs, func(i, j int) bool { return response.GPUs[i].PublicGPUID < response.GPUs[j].PublicGPUID })
	return response.GPUs, nil
}

func (c *Client) request(ctx context.Context, method, path string, input, output any) error {
	var body io.Reader
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode QDivZero request: %w", err)
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("build QDivZero request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.accountID != "" {
		req.Header.Set("X-Account-Id", c.accountID)
	}
	if c.privateBetaToken != "" {
		req.Header.Set("X-Private-Beta-Token", c.privateBetaToken)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("QDivZero API request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var result struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&result)
		message := strings.ReplaceAll(result.Error, c.apiKey, "[redacted]")
		if c.privateBetaToken != "" {
			message = strings.ReplaceAll(message, c.privateBetaToken, "[redacted]")
		}
		return &APIError{StatusCode: resp.StatusCode, Message: message}
	}
	if output != nil {
		if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
			return fmt.Errorf("decode QDivZero response: %w", err)
		}
	}
	return nil
}
