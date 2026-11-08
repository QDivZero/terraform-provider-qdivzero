package qdivzero

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

type ScheduleRuleInput struct {
	Action          string  `json:"action"`
	Cron            string  `json:"cron"`
	Timezone        *string `json:"timezone,omitempty"`
	Enabled         *bool   `json:"enabled,omitempty"`
	DesiredReplicas *int64  `json:"desired_replicas,omitempty"`
	ReplicaSetID    *string `json:"replica_set_id,omitempty"`
}

type ScheduleRule struct {
	ID              string  `json:"id"`
	InstanceID      *string `json:"instance_id"`
	Action          *string `json:"action"`
	Cron            *string `json:"cron"`
	Timezone        *string `json:"timezone"`
	Enabled         *bool   `json:"enabled"`
	DesiredReplicas *int64  `json:"desired_replicas"`
	ReplicaSetID    *string `json:"replica_set_id"`
	NextRunAt       *string `json:"next_run_at"`
	LastRunAt       *string `json:"last_run_at"`
}

func schedulePath(instanceID string) string {
	return "/v1/instances/" + url.PathEscape(instanceID) + "/schedule-rules"
}

func (c *Client) CreateScheduleRule(ctx context.Context, instanceID string, input ScheduleRuleInput) (ScheduleRule, error) {
	var result ScheduleRule
	err := c.request(ctx, http.MethodPost, schedulePath(instanceID), input, &result)
	return result, err
}

func (c *Client) GetScheduleRule(ctx context.Context, instanceID, id string) (ScheduleRule, error) {
	var result struct {
		Rules *[]ScheduleRule `json:"schedule_rules"`
	}
	if err := c.request(ctx, http.MethodGet, schedulePath(instanceID), nil, &result); err != nil {
		return ScheduleRule{}, err
	}
	if result.Rules == nil {
		return ScheduleRule{}, errors.New("QDivZero API returned a missing schedule rules list")
	}
	for _, rule := range *result.Rules {
		if rule.ID == id {
			return rule, nil
		}
	}
	return ScheduleRule{}, &APIError{StatusCode: http.StatusNotFound, Message: "schedule rule not found"}
}

func (c *Client) UpdateScheduleRule(ctx context.Context, instanceID, id string, input ScheduleRuleInput) (ScheduleRule, error) {
	var result ScheduleRule
	err := c.request(ctx, http.MethodPatch, schedulePath(instanceID)+"/"+url.PathEscape(id), input, &result)
	return result, err
}

func (c *Client) DeleteScheduleRule(ctx context.Context, instanceID, id string) error {
	err := c.request(ctx, http.MethodDelete, schedulePath(instanceID)+"/"+url.PathEscape(id), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}
