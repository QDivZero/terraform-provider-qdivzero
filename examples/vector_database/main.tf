terraform {
  required_providers {
    qdivzero = {
      source = "qdivzero/qdivzero"
    }
  }
}

provider "qdivzero" {}

variable "embedding_serving_name" {
  type = string
}

resource "qdivzero_vector_database" "documents" {
  name                   = "documents"
  slug                   = "documents"
  description            = "Document embeddings"
  embedding_serving_name = var.embedding_serving_name
}

output "database_id" {
  value = qdivzero_vector_database.documents.id
}
