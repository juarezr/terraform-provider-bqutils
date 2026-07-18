terraform {
  required_providers {
    bqutils = {
      source = "juarezr/bqutils"
    }
  }
}

provider "bqutils" {}

data "bqutils_routine_parser" "list_partitions" {
  sql           = file("${path.module}/list_partitions.sql")
  trim_body     = true
  trim_comments = false
}

output "routine_id" {
  value = data.bqutils_routine_parser.list_partitions.routine_id
}

output "arguments" {
  value = data.bqutils_routine_parser.list_partitions.arguments
}

output "definition_body" {
  value = data.bqutils_routine_parser.list_partitions.definition_body
}
