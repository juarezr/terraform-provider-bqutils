# terraform-provider-bqutils

Terraform provider that parse `CREATE` SQL statements to make easier to create BigQuery routines and views with the [`hashicorp/google`](https://registry.terraform.io/providers/hashicorp/google/latest) provider.

## Requirements

- [Terraform](https://www.terraform.io/downloads) >= 1.0
- [Go](https://go.dev/doc/install) >= 1.22 (for building from source)

## Provider setup

```hcl
terraform {
  required_providers {
    bqutils = {
      source  = "juarezr/bqutils"
      version = "~> 0.1"
    }
  }
}

provider "bqutils" {}
```

No provider settings are required.

## Data sources

### `bqutils_routine_parser`

Parses `CREATE FUNCTION` / `TABLE FUNCTION` / `PROCEDURE` / `AGGREGATE FUNCTION` and fills attributes for [google_bigquery_routine](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_routine#nested_arguments).

### `bqutils_view_parser`

Parses `CREATE VIEW` / `MATERIALIZED VIEW` and supplies attributes for creating a view with the `google_bigquery_table` resource (including `view` / `materialized_view` blocks, partitioning, clustering, labels, etc.).

## Example (SQL from a file)

```hcl
data "bqutils_routine_parser" "fn" {
  sql           = file("${path.module}/sql/list_partitions.sql")
  trim_body     = true
  trim_comments = false
}

resource "google_bigquery_routine" "fn" {
  dataset_id      = google_bigquery_dataset.ds.dataset_id
  routine_id      = data.bqutils_routine_parser.fn.routine_id
  routine_type    = data.bqutils_routine_parser.fn.routine_type
  language        = data.bqutils_routine_parser.fn.language
  description     = data.bqutils_routine_parser.fn.description
  definition_body = data.bqutils_routine_parser.fn.definition_body

  dynamic "arguments" {
    for_each = data.bqutils_routine_parser.fn.arguments
    content {
      name      = arguments.value.name
      data_type = arguments.value.data_type
    }
  }
}
```

## Documentation

- See [`examples/`](examples/) folder for more patterns (including views and materialized views).
- Check the [provider documentation](https://registry.terraform.io/providers/juarezr/bqutils/latest/docs) on the Terraform Registry.

## Building and testing

```bash
# Build
make build

# Install into local Terraform plugin directory
make install

# Unit tests (parser)
go test ./internal/sqlparse/ -v

# Acceptance tests (requires terraform on PATH)
TF_ACC=1 go test ./internal/provider/ -v -count=1

# Regenerate goyacc grammar + docs tooling
make tools
make generate
```

## Publishing to the Terraform Registry

This repository is used for development. For Registry publishing, the provider binary is expected under a repository named **`terraform-provider-bqutils`** (provider type `bqutils`).

Typical flow:

1. Move/push the code to `github.com/<namespace>/terraform-provider-bqutils`
2. Create a GPG signing key and upload the public key to the Terraform Registry
3. Tag a release (`v0.1.0`) and use GoReleaser (see `.goreleaser.yml`) via the Release GitHub Action
4. Add the provider on [registry.terraform.io](https://registry.terraform.io/publish/provider)

## License

GPL-2.0 (see [LICENSE](LICENSE)).
