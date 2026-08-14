locals {
  view_files = toset([
    "mydataset1.myview1.sql",
    "mydataset1.myview2.sql",
  ])
}

data "google_bigquery_dataset" "mydataset1" {
  dataset_id = "mydataset1"
}

data "bqutils_view_parser" "all" {
  for_each = { for f in local.view_files : trimsuffix(f, ".sql") => f }
  sql      = file("${path.module}/${each.value}")
}

data "bqutils_dependency_layering" "views" {
  source_references = [
    for id, p in data.bqutils_view_parser.all : {
      dataset_id  = p.dataset_id
      object_id   = p.table_id
      object_type = "VIEW"
      references  = p.references
    }
  ]
}

locals {
  layered = data.bqutils_dependency_layering.views.layered_references

  views_l1 = {
    for o in local.layered :
    "${o.dataset_id}.${o.object_id}" => data.bqutils_view_parser.all["${o.dataset_id}.${o.object_id}"]
    if o.layer == 1
  }
  views_l2 = {
    for o in local.layered :
    "${o.dataset_id}.${o.object_id}" => data.bqutils_view_parser.all["${o.dataset_id}.${o.object_id}"]
    if o.layer == 2
  }
}

resource "google_bigquery_table" "l1" {
  for_each   = local.views_l1
  project    = data.google_bigquery_dataset.mydataset1.project
  dataset_id = data.google_bigquery_dataset.mydataset1.dataset_id
  table_id   = each.value.table_id

  view {
    query          = each.value.query
    use_legacy_sql = false
  }
}

resource "google_bigquery_table" "l2" {
  for_each   = local.views_l2
  project    = data.google_bigquery_dataset.mydataset1.project
  dataset_id = data.google_bigquery_dataset.mydataset1.dataset_id
  table_id   = each.value.table_id

  view {
    query          = each.value.query
    use_legacy_sql = false
  }

  depends_on = [google_bigquery_table.l1]
}
