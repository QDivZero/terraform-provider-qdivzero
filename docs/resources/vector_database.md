# qdivzero_vector_database

Manages a flexible vector database. `name` and `embedding_serving_name` are
required. `slug` and `description` are optional; `id`, `status`, and
`created_at` are computed. The API validates compatibility when an embedding
model is changed.

```hcl
resource "qdivzero_vector_database" "documents" {
  name                   = "documents"
  slug                   = "documents"
  embedding_serving_name = var.embedding_serving_name
}
```

Import uses the database ID.
