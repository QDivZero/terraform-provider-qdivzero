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

variable "public_gpu_id" {
  type = string
}

resource "qdivzero_instance" "selected_gpu" {
  name                = "selected-gpu"
  huggingface_repo_id = var.huggingface_repo_id
  scheduling_mode     = "manual"
  gpu_preferences = [{
    public_gpu_id = var.public_gpu_id
    count         = 1
  }]
}
