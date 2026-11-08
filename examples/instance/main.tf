terraform {
  required_providers {
    qdivzero = {
      source = "qdivzero/qdivzero"
    }
  }
}

provider "qdivzero" {}

variable "huggingface_repo_id" {
  type = string
}

variable "desired_state" {
  type    = string
  default = null
}

data "qdivzero_gpus" "available" {}

resource "qdivzero_instance" "model" {
  name                         = "example-model"
  huggingface_repo_id          = var.huggingface_repo_id
  scheduling_mode              = "smart"
  smart_regions                = ["Europe"]
  smart_max_price_per_hour_eur = 0.75
  smart_allow_community        = true
  smart_allow_spot             = true
  serverless_enabled           = true
  idle_timeout_seconds         = 900
  desired_state                = var.desired_state
}

output "instance_id" {
  value = qdivzero_instance.model.id
}

output "available_gpus" {
  value = data.qdivzero_gpus.available.gpus
}
