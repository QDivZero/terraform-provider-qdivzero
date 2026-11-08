terraform {
  required_providers {
    qdivzero = {
      source = "qdivzero/qdivzero"
    }
  }
}

provider "qdivzero" {}

variable "instance_id" {
  type = string
}

resource "qdivzero_instance_schedule" "start_weekdays" {
  instance_id = var.instance_id
  action      = "start"
  cron        = "0 9 * * 1-5"
  timezone    = "Europe/Madrid"
  enabled     = true
}

resource "qdivzero_instance_schedule" "stop_weekdays" {
  instance_id = var.instance_id
  action      = "stop"
  cron        = "0 18 * * 1-5"
  timezone    = "Europe/Madrid"
  enabled     = true
}
