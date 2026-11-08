package qdivzero

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

type VectorDatabaseInput struct {
	Name                 string  `json:"name"`
	EmbeddingServingName string  `json:"embedding_serving_name"`
	Slug                 *string `json:"slug,omitempty"`
	Description          *string `json:"description,omitempty"`
}

type VectorDatabase struct {
	ID                   string  `json:"id"`
	Name                 *string `json:"name"`
	EmbeddingServingName *string `json:"embedding_serving_name"`
	Slug                 *string `json:"slug"`
	Description          *string `json:"description"`
	Status               *string `json:"status"`
	CreatedAt            *string `json:"created_at"`
}

func (c *Client) CreateVectorDatabase(ctx context.Context, input VectorDatabaseInput) (VectorDatabase, error) {
	var result VectorDatabase
	err := c.request(ctx, http.MethodPost, "/v1/flexible-vector-databases", input, &result)
	return result, err
}

func (c *Client) GetVectorDatabase(ctx context.Context, id string) (VectorDatabase, error) {
	var result VectorDatabase
	err := c.request(ctx, http.MethodGet, "/v1/flexible-vector-databases/"+url.PathEscape(id), nil, &result)
	return result, err
}

func (c *Client) UpdateVectorDatabase(ctx context.Context, id string, input VectorDatabaseInput) (VectorDatabase, error) {
	var result VectorDatabase
	err := c.request(ctx, http.MethodPatch, "/v1/flexible-vector-databases/"+url.PathEscape(id), input, &result)
	return result, err
}

func (c *Client) DeleteVectorDatabase(ctx context.Context, id string) error {
	err := c.request(ctx, http.MethodDelete, "/v1/flexible-vector-databases/"+url.PathEscape(id), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}

type BalancerDestination struct {
	Type string `json:"type"`
	Ref  string `json:"ref"`
}

type SmartBalancerRoute struct {
	Name        *string              `json:"name,omitempty"`
	Intent      *string              `json:"intent,omitempty"`
	Priority    *int64               `json:"priority,omitempty"`
	Enabled     *bool                `json:"enabled,omitempty"`
	IsDefault   *bool                `json:"is_default,omitempty"`
	Destination *BalancerDestination `json:"destination,omitempty"`
}

type SmartBalancerInput struct {
	Name                         string                `json:"name"`
	DisplayName                  *string               `json:"display_name,omitempty"`
	Enabled                      *bool                 `json:"enabled,omitempty"`
	WorkloadKind                 *string               `json:"workload_kind,omitempty"`
	RoutingMode                  *string               `json:"routing_mode,omitempty"`
	RouterModel                  *BalancerDestination  `json:"router_model,omitempty"`
	Routes                       *[]SmartBalancerRoute `json:"routes,omitempty"`
	ServerlessStartRequests      *int64                `json:"serverless_start_requests,omitempty"`
	ServerlessStartWindowSeconds *int64                `json:"serverless_start_window_seconds,omitempty"`
}

type SmartBalancer struct {
	ID                           string                `json:"id"`
	Name                         *string               `json:"name"`
	DisplayName                  *string               `json:"display_name"`
	Enabled                      *bool                 `json:"enabled"`
	WorkloadKind                 *string               `json:"workload_kind"`
	RoutingMode                  *string               `json:"routing_mode"`
	RouterModel                  *BalancerDestination  `json:"router_model"`
	Routes                       *[]SmartBalancerRoute `json:"routes"`
	ServerlessStartRequests      *int64                `json:"serverless_start_requests"`
	ServerlessStartWindowSeconds *int64                `json:"serverless_start_window_seconds"`
	ServingEndpointID            *string               `json:"serving_endpoint_id"`
}

func (c *Client) CreateSmartBalancer(ctx context.Context, input SmartBalancerInput) (SmartBalancer, error) {
	var result SmartBalancer
	err := c.request(ctx, http.MethodPost, "/v1/smart-balancers", input, &result)
	return result, err
}

func (c *Client) GetSmartBalancer(ctx context.Context, id string) (SmartBalancer, error) {
	var groups map[string][]SmartBalancer
	if err := c.request(ctx, http.MethodGet, "/v1/smart-balancers", nil, &groups); err != nil {
		return SmartBalancer{}, err
	}
	if len(groups) == 0 {
		return SmartBalancer{}, errors.New("QDivZero API returned a missing smart balancer list")
	}
	for _, group := range groups {
		if group == nil {
			return SmartBalancer{}, errors.New("QDivZero API returned an invalid smart balancer list")
		}
		for _, balancer := range group {
			if balancer.ID == id {
				return balancer, nil
			}
		}
	}
	return SmartBalancer{}, &APIError{StatusCode: http.StatusNotFound, Message: "smart balancer not found"}
}

func (c *Client) UpdateSmartBalancer(ctx context.Context, id string, input SmartBalancerInput) (SmartBalancer, error) {
	var result SmartBalancer
	err := c.request(ctx, http.MethodPatch, "/v1/smart-balancers/"+url.PathEscape(id), input, &result)
	return result, err
}

func (c *Client) DeleteSmartBalancer(ctx context.Context, id string) error {
	err := c.request(ctx, http.MethodDelete, "/v1/smart-balancers/"+url.PathEscape(id), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}

type TargetGroupMember struct {
	InstanceID string `json:"instance_id"`
	Weight     int64  `json:"weight"`
	Priority   int64  `json:"priority"`
	Enabled    bool   `json:"enabled"`
}

type TargetGroupInput struct {
	Name            string               `json:"name,omitempty"`
	DisplayName     *string              `json:"display_name,omitempty"`
	Description     *string              `json:"description,omitempty"`
	Enabled         *bool                `json:"enabled,omitempty"`
	WorkloadKind    *string              `json:"workload_kind,omitempty"`
	SelectionPolicy *string              `json:"selection_policy,omitempty"`
	Members         *[]TargetGroupMember `json:"members,omitempty"`
}

type TargetGroup struct {
	ID              string               `json:"id"`
	Name            *string              `json:"name"`
	DisplayName     *string              `json:"display_name"`
	Description     *string              `json:"description"`
	Enabled         *bool                `json:"enabled"`
	WorkloadKind    *string              `json:"workload_kind"`
	SelectionPolicy *string              `json:"selection_policy"`
	Members         *[]TargetGroupMember `json:"members"`
}

func (c *Client) CreateTargetGroup(ctx context.Context, input TargetGroupInput) (TargetGroup, error) {
	var result TargetGroup
	err := c.request(ctx, http.MethodPost, "/v1/target-groups", input, &result)
	return result, err
}

func (c *Client) GetTargetGroup(ctx context.Context, id string) (TargetGroup, error) {
	var result TargetGroup
	err := c.request(ctx, http.MethodGet, "/v1/target-groups/"+url.PathEscape(id), nil, &result)
	return result, err
}

func (c *Client) UpdateTargetGroup(ctx context.Context, id string, input TargetGroupInput) (TargetGroup, error) {
	input.Name, input.WorkloadKind = "", nil
	var result TargetGroup
	err := c.request(ctx, http.MethodPatch, "/v1/target-groups/"+url.PathEscape(id), input, &result)
	return result, err
}

func (c *Client) DeleteTargetGroup(ctx context.Context, id string) error {
	err := c.request(ctx, http.MethodDelete, "/v1/target-groups/"+url.PathEscape(id), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}
