# qdivzero_smart_balancer

Manages a Smart Balancer. `name` and `routes` are required. Each route requires
`name`, `priority`, `destination_type`, and `destination_ref`; `intent` defaults
to an empty string, `enabled` to `true`, and `is_default` to `false`. Valid
destination types include `instance`, `target_group`, and `public_model`.

Optional balancer settings include `display_name`, `enabled`, `workload_kind`,
`routing_mode`, `router_model_type`, `router_model_ref`,
`serverless_start_requests`, and `serverless_start_window_seconds`. Configure
both router-model attributes for classifier routing. The API may normalize
route priorities; use consecutive positive priorities for predictable state.

```hcl
resource "qdivzero_smart_balancer" "router" {
  name          = "router"
  workload_kind = "chat"
  routing_mode  = "priority"
  routes = [{
    name             = "primary"
    priority         = 1
    destination_type = "target_group"
    destination_ref  = qdivzero_target_group.workers.id
  }]
}
```

The computed `serving_endpoint_id` identifies the associated serving endpoint.
Import uses the balancer ID.
