# terraform-provider-bqutils

Terraform provider utilities that parse BigQuery `CREATE` SQL and expose attributes for wiring into [`hashicorp/google`](https://registry.terraform.io/providers/hashicorp/google/latest) resources (`google_bigquery_routine`, `google_bigquery_table`).

No Google Cloud API calls are made. The provider only parses strings.

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

No configuration arguments are required.

## Data sources

### `bqutils_routine_parser`

Parses `CREATE FUNCTION` / `TABLE FUNCTION` / `PROCEDURE` / `AGGREGATE FUNCTION` and fills attributes for `google_bigquery_routine`.

TEMPORARY routines are rejected with an error (they cannot be managed as persistent BigQuery objects).

### `bqutils_view_parser`

Parses `CREATE VIEW` / `MATERIALIZED VIEW` and fills attributes for `google_bigquery_table` (including `view` / `materialized_view` blocks, partitioning, clustering, labels, etc.).

Unmappable `OPTIONS` (for example `retain_partitions`) are ignored.

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

See [`examples/`](examples/) and [`samples/`](samples/) for more patterns (including views and materialized views).

## Optional inputs

| Argument | Default | Meaning |
|----------|---------|---------|
| `trim_body` | `true` | Trim leading/trailing whitespace and empty lines from the body/`query` |
| `trim_comments` | `false` | Strip `--` / `/* */` comments from the body/`query` |

Computed helpers include `project` / `dataset_id` when present in a qualified object name, and `is_materialized` on the view parser.

`return_type` / argument `data_type` values are StandardSqlDataType **JSON** strings as required by `google_bigquery_routine`.

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
