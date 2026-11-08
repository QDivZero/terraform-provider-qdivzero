# qdivzero_gpus

Lists the account-visible GPU catalog in `gpus`, ordered by `public_gpu_id`.
Each element includes the public GPU ID, display name, provider, market type,
location, hourly EUR price, VRAM, GPU count, and availability information when
returned by the API.

```hcl
data "qdivzero_gpus" "available" {}

output "gpu_catalog" {
  value = data.qdivzero_gpus.available.gpus
}
```
