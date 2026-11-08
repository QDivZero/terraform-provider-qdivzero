# qdivzero_serving_endpoints

Lists existing serving endpoints visible to the configured account. The
`endpoints` list contains each endpoint's ID, name, display name, state, and
enabled flag when returned by the API.

```hcl
data "qdivzero_serving_endpoints" "available" {}
```
