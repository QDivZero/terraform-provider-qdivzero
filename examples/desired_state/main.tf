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

resource "qdivzero_instance" "always_running" {
  name                         = "always-running"
  huggingface_repo_id          = var.huggingface_repo_id
  smart_region                 = "Europe"
  smart_max_price_per_hour_eur = 1.00
  smart_allow_community        = true
  smart_allow_spot             = false
  serverless_enabled           = false
  desired_state                = "running"
  wait_timeout_seconds         = 1200
}
