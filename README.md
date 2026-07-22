# terraform-provider-bqutils

Terraform provider automates the creation and update of BigQuery functions/procedures/views by parsing `CREATE` SQL statements stored in SQL Scripts and connecting with the resources the [`hashicorp/google`](https://registry.terraform.io/providers/hashicorp/google/latest) provider for object creation.

## Requirements

- [Terraform](https://www.terraform.io/downloads) >= 1.0
- [Go](https://go.dev/doc/install) >= 1.22 (for building from source)

## Provider setup

```hcl
terraform {
  required_providers {
    bqutils = {
      source  = "juarezr/bqutils"
      version = "~> 0.2"
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

## Usage Example

Using SQL from a file in the Terraform module folder:

```sql
CREATE OR REPLACE TABLE FUNCTION mydataset.list_tables
(
    table_name_filter STRING,
    max_results       INT64
)
OPTIONS (
    description = 'Used to show tables in another dataset.'
) AS (
  SELECT
    t.table_name,
    t.table_type,
    t.creation_time,
    t.is_typed
   FROM `mydataset`.INFORMATION_SCHEMA.TABLES AS t
  WHERE t.table_name LIKE CONCAT('%', COALESCE(table_name_filter,''), '%')
  QUALIFY ROW_NUMBER() OVER(ORDER BY t.table_name) <= COALESCE(max_results, 200)
);
```

```terraform
# Load the Table FUNCTION SQL from a file in the same folder as the Terraform code.
data "bqutils_routine_parser" "list_tables" {
  sql = file("${path.module}/sql/list_tables.sql")

  trim_body = true
}

# Gets the BigQuery dataset where the routine is created.
data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

# Create the routine in BigQuery using the attributes parsed from the SQL file.
resource "google_bigquery_routine" "list_tables" {
  dataset_id      = google_bigquery_dataset.mydataset.dataset_id
  routine_id      = data.bqutils_routine_parser.list_tables.routine_id
  routine_type    = data.bqutils_routine_parser.list_tables.routine_type
  language        = data.bqutils_routine_parser.list_tables.language
  description     = data.bqutils_routine_parser.list_tables.description
  definition_body = data.bqutils_routine_parser.list_tables.definition_body

  dynamic "arguments" {
    for_each = data.bqutils_routine_parser.list_tables.arguments
    content {
      name          = arguments.value.name
      data_type     = arguments.value.data_type
      argument_kind = arguments.value.argument_kind
    }
  }
}
```

## Documentation

- See [`examples/`](examples/) folder for more patterns (including views and materialized views).
- Check the [provider documentation](https://registry.terraform.io/providers/juarezr/bqutils/latest/docs) on the Terraform Registry.

## Building and testing

Check the [guides](guides/index.md) for common tasks.

### Basic build and testing

```bash
# Build
make build

# Unit tests (parser)
go test ./internal/sqlparse/ -v

# Acceptance tests (requires terraform on PATH)
TF_ACC=1 go test ./internal/provider/ -v -count=1

# Install docs tooling and regenerate provider docs
make tools

# Run all tests
make check
```

### Testing terraform with the provider locally installed

```bash
# Install into local Terraform plugin directory
make install

# Setup dev overrides pointing to local Terraform plugin directory
make dev-override

# Test your terraform module
cd ~/src/terraform-module
terraform init && terraform plan
terraform apply -auto-approve && terraform destroy -auto-approve
cd -

# Remove dev overrides and binaries from the local Terraform plugin directory
make uninstall
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
