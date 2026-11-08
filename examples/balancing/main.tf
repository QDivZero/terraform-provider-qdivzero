terraform {
  required_providers {
    qdivzero = {
      source = "valendra/qdivzero"
    }
  }
}

provider "qdivzero" {}

variable "instance_id" {
  type = string
}

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

data "qdivzero_serving_endpoints" "available" {}

output "serving_endpoint_id" {
  value = qdivzero_smart_balancer.router.serving_endpoint_id
}
