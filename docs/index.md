---
page_title: "bqutils Provider"
description: |-
  Parse BigQuery CREATE SQL into attributes for hashicorp/google BigQuery resources.
---

# bqutils Provider

The **bqutils** provider parses BigQuery `CREATE` SQL statements and exposes Terraform data source attributes that map to `google_bigquery_routine` and `google_bigquery_table`.

It does not call Google Cloud APIs and requires no provider configuration.

## Example Usage

```terraform
terraform {
  required_providers {
    bqutils = {
      source = "juarezr/bqutils"
    }
  }
}

provider "bqutils" {}
```

## Schema

This provider has no configuration schema.
