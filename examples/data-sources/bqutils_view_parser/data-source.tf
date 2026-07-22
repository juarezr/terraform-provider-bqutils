# Load VIEW SQL from a file.

data "bqutils_view_parser" "example" {
  sql = file("${path.module}/view.sql")
}
