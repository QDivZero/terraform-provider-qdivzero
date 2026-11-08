# QDivZero Terraform Provider

Terraform provider for managing QDivZero inference infrastructure through the
QDivZero API. The provider defaults to Smart scheduling and supports GPU and
region selection, price limits, community and interruptible capacity,
serverless settings, instance schedules, vector databases, target groups, and
Smart Balancers.

## Resources and data sources

| Type | Purpose |
| --- | --- |
| [`qdivzero_instance`](docs/resources/instance.md) | Manage an inference instance and its optional desired state. |
| [`qdivzero_instance_schedule`](docs/resources/instance_schedule.md) | Manage a cron-based instance start, stop, or scale rule. |
| [`qdivzero_vector_database`](docs/resources/vector_database.md) | Manage a flexible vector database. |
| [`qdivzero_target_group`](docs/resources/target_group.md) | Manage a group of instance targets. |
| [`qdivzero_smart_balancer`](docs/resources/smart_balancer.md) | Manage a Smart Balancer and its routes. |
| [`qdivzero_gpus`](docs/data-sources/gpus.md) | Discover available GPU IDs, regions, prices, and market types. |
| [`qdivzero_serving_endpoints`](docs/data-sources/serving_endpoints.md) | Discover existing serving endpoints. |

The current API documents listing and creation of serving endpoints but no
delete operation. They are exposed for discovery rather than as a managed
Terraform resource.

## Configuration

Set `QDIVZERO_API_KEY` to a QDivZero API key. Optional environment variables
are `QDIVZERO_API_URL`, `QDIVZERO_ACCOUNT_ID`, and
`QDIVZERO_PRIVATE_BETA_TOKEN`. Explicit provider attributes take precedence.

```hcl
terraform {
  required_providers {
    qdivzero = {
      source = "qdivzero/qdivzero"
    }
  }
}

provider "qdivzero" {
  # api_key is normally supplied through QDIVZERO_API_KEY.
  # account_id = "your-account-id"
}
```

See [`examples/`](examples/) for instance configuration, schedules, vector
databases, target groups, balancers, and discovery.

## Building and local development

The provider requires Go 1.25 or later and implements Terraform plugin protocol
6. Build the provider binary with:

```sh
mkdir -p bin
go build -o bin/terraform-provider-qdivzero .
```

For local development, configure a Terraform CLI `dev_overrides` entry pointing
to the **absolute path** of that `bin` directory:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/qdivzero/qdivzero" = "/absolute/path/to/qdivzero-terraform/bin"
  }
  direct {}
}
```

With a development override, operational commands use the local binary.
Initialization still attempts to select a published provider version, so it
cannot fetch this unpublished provider from the public registry.

This repository is private and named `qdivzero-terraform`. The public Terraform
Registry requires a public repository named `terraform-provider-qdivzero`; this
repository is not currently published there. Local development overrides or a
separately configured private distribution are therefore required.

## Development checks

```sh
gofmt -l .
go vet ./...
go test -race ./...
go build ./...
```

Tests use mock HTTP servers and the provider protocol directly; they do not
invoke Terraform or create live infrastructure.
