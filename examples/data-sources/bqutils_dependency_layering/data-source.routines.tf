locals {
  routine_files = toset([
    "mydataset1.myfunction1.sql",
    "mydataset1.myfunction2.sql",
    "mydataset1.myfunction3.sql",
  ])
}

data "google_bigquery_dataset" "mydataset1" {
  dataset_id = "mydataset1"
}

data "bqutils_routine_parser" "all" {
  for_each       = { for f in local.routine_files : trimsuffix(f, ".sql") => f }
  sql            = file("${path.module}/${each.value}")
  target_project = data.google_bigquery_dataset.mydataset1.project
}

data "bqutils_dependency_layering" "routines" {
  source_references = [
    for id, p in data.bqutils_routine_parser.all : {
      dataset_id  = p.dataset_id
      object_id   = p.routine_id
      object_type = p.routine_type
      references  = p.references
    }
  ]
}

locals {
  layered = data.bqutils_dependency_layering.routines.layered_references

  routines_l1 = {
    for o in local.layered :
    o.reference_id => data.bqutils_routine_parser.all[o.reference_id]
    if o.layer == 1
  }
  routines_l2 = {
    for o in local.layered :
    o.reference_id => data.bqutils_routine_parser.all[o.reference_id]
    if o.layer == 2
  }
  routines_l3 = {
    for o in local.layered :
    o.reference_id => data.bqutils_routine_parser.all[o.reference_id]
    if o.layer == 3
  }
}

resource "google_bigquery_routine" "l1" {
  for_each        = local.routines_l1
  project         = data.google_bigquery_dataset.mydataset1.project
  dataset_id      = data.google_bigquery_dataset.mydataset1.dataset_id
  routine_id      = each.value.routine_id
  routine_type    = each.value.routine_type
  language        = each.value.language
  definition_body = each.value.definition_body
}

resource "google_bigquery_routine" "l2" {
  for_each        = local.routines_l2
  project         = data.google_bigquery_dataset.mydataset1.project
  dataset_id      = data.google_bigquery_dataset.mydataset1.dataset_id
  routine_id      = each.value.routine_id
  routine_type    = each.value.routine_type
  language        = each.value.language
  definition_body = each.value.definition_body
  depends_on      = [google_bigquery_routine.l1]
}

resource "google_bigquery_routine" "l3" {
  for_each        = local.routines_l3
  project         = data.google_bigquery_dataset.mydataset1.project
  dataset_id      = data.google_bigquery_dataset.mydataset1.dataset_id
  routine_id      = each.value.routine_id
  routine_type    = each.value.routine_type
  language        = each.value.language
  definition_body = each.value.definition_body
  depends_on      = [google_bigquery_routine.l2]
}
