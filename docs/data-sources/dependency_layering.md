---
page_title: "bqutils_dependency_layering Data Source - bqutils"
subcategory: ""
description: |-
  Computes creation-order layers (waves) for BigQuery routines and views from parser `references`, for Pattern B layered `for_each` with static inter-layer `depends_on`.
---

# bqutils_dependency_layering

Computes creation-order **layers** (waves) for BigQuery routines and views from parser `references`, so Terraform can create dependents after their dependencies using this pattern: one Google resource block per wave, `for_each` filtered by `layer`, and static inter-layer `depends_on`.

BigQuery requires callees (routines) and referenced views to exist before callers. Terraform will not infer that order from SQL alone: wave resources are configured from parsers, not from the previous wave's attributes. Use this data source to assign layers, then wire waves with `depends_on` (or an equivalent whole-resource reference).

## Caveats

- Pre-declare one Google resource block per wave (`l1`, `l2`, …). Filter with `if o.layer == N`. Use `max_layers` to know the required depth. Terraform cannot create a dynamic number of resource blocks from `max_layers`.
- Objects in the same layer have no managed dependency on each other; only inter-layer `depends_on` is required.
- Edges to objects not listed in `source_references` (external tables, unmanaged routines) and `INFORMATION_SCHEMA` references are dropped (not errors).
- Detectable problems (cycles, duplicate `(dataset_id, object_id)`, empty `dataset_id` / `object_id`) produce plan/apply errors.
- Call-site parsing cannot distinguish tables vs views or scalar vs aggregate functions.

## Example Usage

### Problem: routines that call other routines

`myfunction2` calls `myfunction1`. Creating them in the wrong order fails in BigQuery. SQL files:

```sql
CREATE OR REPLACE FUNCTION mydataset1.myfunction1()
RETURNS INT64
AS (
  (SELECT COUNT(*) FROM mydataset1.INFORMATION_SCHEMA.TABLES)
);
```

```sql
CREATE OR REPLACE FUNCTION mydataset1.myfunction2()
RETURNS INT64
AS (
  mydataset1.myfunction1() + 1
);
```

Parse with `for_each`, layer, then create wave resources. Key parsers by `trimsuffix(file, ".sql")` so the map key is `dataset.object`:

```terraform
locals {
  routine_files = toset([
    "mydataset1.myfunction1.sql",
    "mydataset1.myfunction2.sql",
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
    "${o.dataset_id}.${o.object_id}" => data.bqutils_routine_parser.all["${o.dataset_id}.${o.object_id}"]
    if o.layer == 1
  }
  routines_l2 = {
    for o in local.layered :
    "${o.dataset_id}.${o.object_id}" => data.bqutils_routine_parser.all["${o.dataset_id}.${o.object_id}"]
    if o.layer == 2
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
```

### Problem: views that reference other views

Same pattern for views → `google_bigquery_table`:

```sql
CREATE OR REPLACE VIEW mydataset1.myview1 AS
SELECT table_name
FROM mydataset1.INFORMATION_SCHEMA.TABLES;
```

```sql
CREATE OR REPLACE VIEW mydataset1.myview2 AS
SELECT table_name
FROM mydataset1.myview1;
```

```terraform
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
```

If `max_layers` is greater than the waves you declared, add more `lN` resources (and `depends_on` chains) before apply.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `source_references` (Attributes List) Managed objects to layer, typically built from bqutils_routine_parser / bqutils_view_parser instances. Nested `references` may be assigned directly from parser `references` (extra fields like resource_type are accepted). (see [below for nested schema](#nestedatt--source_references))

### Read-Only

- `id` (String) Synthetic id for this data source instance.
- `layered_references` (Attributes List) Managed objects sorted by layer then dataset_id then object_id. Filter by layer for wave resources; use resource_type / object_type for alternate-2 / alternate-3 splits. (see [below for nested schema](#nestedatt--layered_references))
- `max_layers` (Number) Maximum layer number in layered_references, or 0 when empty. Pre-declare wave resources l1..lN for at least this depth.

<a id="nestedatt--source_references"></a>
### Nested Schema for `source_references`

Required:

- `dataset_id` (String) Dataset where the object will be created.
- `object_id` (String) Routine id or table/view id being created.
- `object_type` (String) From routine_type for routines, or VIEW for views (MATERIALIZED_VIEW allowed).
- `references` (Attributes List) Dependencies from the object's SQL body (parser references). (see [below for nested schema](#nestedatt--source_references--references))

<a id="nestedatt--source_references--references"></a>
### Nested Schema for `source_references.references`

Required:

- `dataset_id` (String) Dataset of the referenced object.
- `object_id` (String) Id of the referenced object.
- `object_type` (String) Call-site object type from the parser.

Optional:

- `resource_type` (String) ROUTINE or VIEW from the parser (accepted for type compatibility; unused by the layering algorithm).



<a id="nestedatt--layered_references"></a>
### Nested Schema for `layered_references`

Read-Only:

- `dataset_id` (String) Dataset of the object to create.
- `layer` (Number) Creation wave starting at 1. Objects in the same layer have no managed dependency on each other.
- `object_id` (String) Object id to create.
- `object_type` (String) Pass-through from source_references (e.g. SCALAR_FUNCTION, PROCEDURE, VIEW).
- `resource_type` (String) Derived: VIEW (or MATERIALIZED_VIEW) → VIEW; routine kinds → ROUTINE.
