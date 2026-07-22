# Load the VIEW SQL from a file in the same folder as the Terraform code.

data "bqutils_view_parser" "example" {
  sql = file("${path.module}/mydataset.my_simple_view.sql")
}
