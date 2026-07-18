terraform {
  required_providers {
    bqutils = {
      source = "juarezr/bqutils"
    }
  }
}

provider "bqutils" {}

data "bqutils_view_parser" "simple_view" {
  sql = file("${path.module}/simple_view.sql")
}

output "table_id" {
  value = data.bqutils_view_parser.simple_view.table_id
}

output "query" {
  value = data.bqutils_view_parser.simple_view.query
}

output "schema" {
  value = data.bqutils_view_parser.simple_view.schema
}

output "is_materialized" {
  value = data.bqutils_view_parser.simple_view.is_materialized
}
