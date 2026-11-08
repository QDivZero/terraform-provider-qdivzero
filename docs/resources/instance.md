# qdivzero_instance

Manages an inference instance. `scheduling_mode` defaults to `smart`; specify
`manual` to select manual scheduling instead. The name and Hugging Face
repository ID are required. Changing `huggingface_repo_id` or `workload_kind`
requires replacement because the update endpoint does not accept them.

```hcl
resource "qdivzero_instance" "model" {
  name                         = "example-model"
  huggingface_repo_id          = "Qwen/Qwen3.5-9B"
  scheduling_mode              = "smart"
  smart_regions                = ["Europe"]
  smart_max_price_per_hour_eur = 0.75
  smart_allow_spot             = true
  smart_allow_community        = true
  serverless_enabled           = true
  idle_timeout_seconds         = 900
}
```

## Arguments

- `name`, `huggingface_repo_id`: required instance name and model repository.
- `description`, `serving_name`, `runtime_preset`, `workload_kind`, and
  `supported_protocols`: optional workload and runtime settings.
- `replica_count`: optional create-only replica count; changing it requires
  replacement. `replica_set_id` is computed when returned by the API.
- `scheduling_mode`: `smart` by default; may be set to `manual`.
- `gpu_preferences`: list of objects with `public_gpu_id` and `count`.
  Discover public IDs with `data.qdivzero_gpus`.
- `smart_regions` is a list of allowed regions. `smart_region` accepts one
  preferred region.
- `smart_max_price_per_hour_eur`,
  `smart_min_gpu_class`, `smart_min_total_tflops`, `smart_allow_spot`,
  `smart_allow_community`, `smart_provider_preference`,
  `smart_provider_filter_mode`, and `smart_selection_label`: Smart scheduler
  constraints. `smart_allow_spot` controls interruptible capacity.
- `cost_optimization_enabled` and `cost_optimization_min_savings_percent`:
  optional automatic replacement settings.
- `serverless_enabled`, `idle_timeout_seconds`: serverless and idle settings.
- `desired_state`: optional `running` or `stopped`; starts or stops the
  instance and waits for the requested stable state. An unbootstrapped instance
  is considered stopped. An instance already in `failed` state is started once
  when `running` is requested; a subsequent failure is reported.
- `wait_timeout_seconds`: optional state-convergence timeout, default 1200.

Computed attributes include `id`, `state`, and `created_at`. Import uses the
instance ID. If a fixed `desired_state` is configured, serverless idle shutdown
or a schedule may intentionally change the remote state and create drift; omit
`desired_state` when those automatic transitions should control the instance.
