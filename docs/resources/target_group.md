# qdivzero_target_group

Manages a named group of inference instances. `name` and `members` are required.
Each member supplies `instance_id`, `weight`, `priority`, and `enabled`.
Optional attributes are `display_name`, `description`, `enabled`,
`workload_kind`, and `selection_policy` (`priority`, `weighted`, or
`round_robin`). Changes to `name` or `workload_kind` require replacement.

```hcl
resource "qdivzero_target_group" "workers" {
  name             = "workers"
  workload_kind    = "chat"
  selection_policy = "priority"
  members = [{
    instance_id = var.instance_id
    weight      = 1
    priority    = 1
    enabled     = true
  }]
}
```

Import uses the target-group ID.
