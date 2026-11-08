# qdivzero_instance_schedule

Manages one instance schedule rule. Set `instance_id`, `action` (`start`,
`stop`, or `scale`), and `cron`. Optional attributes are `timezone`, `enabled`,
`desired_replicas`, and `replica_set_id`. Computed attributes include `id`,
`next_run_at`, and `last_run_at`.

```hcl
resource "qdivzero_instance_schedule" "start" {
  instance_id = var.instance_id
  action      = "start"
  cron        = "0 9 * * 1-5"
  timezone    = "Europe/Madrid"
  enabled     = true
}
```

Import identifiers use `instance_id/rule_id`. A fixed instance `desired_state`
can conflict with a schedule that changes the instance state.
