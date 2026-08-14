---
page_title: "bqutils_dependency_layering Data Source - bqutils"
description: |-
  Computes creation-order layers (stages) used to track the dependencies between BigQuery routines and views. It allows the resource to be created in the correct order and avoid errors when executing the terraform apply command. Designed to be used coupled with the `for_each` and `depends_on` attributes in dynamic resource blocks of type `google_bigquery_routine` or `google_bigquery_table` from the Google Provider separated by layers.
---

# bqutils_dependency_layering

Computes creation-order **layers** (stages) used to track the dependencies between BigQuery routines and views. It allows the resource to be created in the correct order and avoid errors when executing the terraform apply command. Designed to be used coupled with the `for_each` and `depends_on` attributes in dynamic resource blocks of type `google_bigquery_routine` or `google_bigquery_table` from the Google Provider separated by layers.

BigQuery requires callees (routines) and referenced views to exist before callers. Terraform will not infer that order from SQL alone: stage resources are configured from parsers, not from the previous stage's attributes. Use this data source to assign layers, then wire stages with `depends_on` (or an equivalent whole-resource reference).

## Caveats

- Terraform cannot create a dynamic number of resource blocks that match the number of layers computed by the datasource.
  - The `depends_on` attribute of the resource blocks must be static.
  - So you must create resource blocks for the different layers manually.
- Call-site parsing of SQL `CREATE` statements cannot distinguish between tables vs views or scalar vs aggregate functions.
- Objects in the same layer have no managed dependency on each other; but inter-layer dependencies require a `depends_on` attribute to be set between the resource blocks of the different layers, particularly the next-to-last layer.
- Some detectable conditions are not allowed to occur in the terraform code.
  - Examples: cycles, duplicate `(dataset_id, object_id)`, empty `dataset_id` / `object_id`
  - For these conditions, the datasource will produce plan/apply errors.
- Some references listed in `source_references` (external tables, unmanaged routines, `INFORMATION_SCHEMA` references) are dropped from the computation  of `layered_references` attributes (not errors).

## Example Usage

### Problem: routines that call other routines

In this example, `myfunction2` calls `myfunction1`, and `myfunction3` calls `myfunction2`. Creating them in the wrong order fails in BigQuery.

Suppose the following SQL files:

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

```sql
CREATE OR REPLACE FUNCTION mydataset1.myfunction3()
RETURNS INT64
AS (
  mydataset1.myfunction2() + 1
);
```

To let terraform create the resources in the correct order, we can use the `bqutils_dependency_layering` datasource to keep track of which routines call other routines and compute the layers and then create the resources in the correct order.

To do this, one can follow these steps:

1. Parse the SQL files with the `bqutils_routine_parser` datasource to get the `references` for each routine.
2. Use the `bqutils_dependency_layering` datasource to keep track of which routines call other routines and compute the layers.
3. Create the resources in the correct order using the `for_each` and `depends_on` attributes.

Tips and conventions:

- Name your routines and views following the filename convention: `dataset.object.sql`.
- Load the SQL files into the parsers by using the `for_each` attribute and the `trimsuffix(file, ".sql")` function so the map key matches the parsers `reference_id` attribute value.
- Filter the objects from the parsers by `resource_type` and layer, and then link each one to 1 resource block per layer of the `google_bigquery_routine` and `google_bigquery_table` resources.

```terraform
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
```

### Problem: views that reference other views

Views also have a creation order dependency. In this example, `myview2` references `myview1`, and `myview3` references `myview2`.

Suppose the following SQL files:

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

```sql
CREATE OR REPLACE VIEW mydataset1.myview3 AS
SELECT table_name
FROM mydataset1.myview2;
```

The terraform code to create the resources in the correct order would be:

```terraform
locals {
  view_files = toset([
    "mydataset1.myview1.sql",
    "mydataset1.myview2.sql",
    "mydataset1.myview3.sql",
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
    o.reference_id => data.bqutils_view_parser.all[o.reference_id]
    if o.layer == 1
  }
  views_l2 = {
    for o in local.layered :
    o.reference_id => data.bqutils_view_parser.all[o.reference_id]
    if o.layer == 2
  }
  views_l3 = {
    for o in local.layered :
    o.reference_id => data.bqutils_view_parser.all[o.reference_id]
    if o.layer == 3
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

resource "google_bigquery_table" "l3" {
  for_each   = local.views_l3
  project    = data.google_bigquery_dataset.mydataset1.project
  dataset_id = data.google_bigquery_dataset.mydataset1.dataset_id
  table_id   = each.value.table_id

  view {
    query          = each.value.query
    use_legacy_sql = false
  }

  depends_on = [google_bigquery_table.l2]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `source_references` (Attributes List) Receives the list of objects and dependencies to compute the creation order and layering. Typically this list is built from bqutils_routine_parser / bqutils_view_parser instances. Its values may be assigned directly from the `references` attributes from a computed list of parsers. (see [below for nested schema](#nestedatt--source_references))

### Read-Only

- `id` (String) Synthetic id for this data source instance.
- `layered_references` (Attributes List) Return the objects received in the `source_references` attribute ordered by the order that they should be created. These objects are sorted by layer then dataset_id then object_id. To create the resources in the correct order, you have to supply these objects to be created in dynamic resource blocks of type `google_bigquery_routine` or `google_bigquery_table` separated by layer. Each layer will be dependent on the previous layer, so you have to set the `depends_on` attributes between the layers/blocks. (see [below for nested schema](#nestedatt--layered_references))
- `max_layers` (Number) Maximum layer number detected in the `layered_references` attribute. 0 when empty. Use it to know the maximum number of layers that should be created in the terraform code.

<a id="nestedatt--source_references"></a>
### Nested Schema for `source_references`

Required:

- `dataset_id` (String) Dataset where the object will be created.
- `object_id` (String) Routine id or table/view id being created.
- `object_type` (String) Assigned typically from the `routine_type` attribute for `bqutils_routine_parser` instances, or `VIEW` for `bqutils_view_parser` instances (`MATERIALIZED_VIEW` also allowed).
- `references` (Attributes List) Dependencies from the parsed SQL body (usually obtained from the parser `references` attribute). (see [below for nested schema](#nestedatt--source_references--references))

<a id="nestedatt--source_references--references"></a>
### Nested Schema for `source_references.references`

Required:

- `dataset_id` (String) Dataset of the referenced object extracted from the parsed SQL body.
- `object_id` (String) Id of the referenced object extracted from the parsed SQL body.
- `object_type` (String) Call-site object type detected in the parsed SQL body.

Optional:

- `resource_type` (String) Values are `ROUTINE` or `VIEW`. It is computed from the `object_type` attribute (can be used by the layering terraform code to filter the objects by resource type and assign them to the correct resource block).



<a id="nestedatt--layered_references"></a>
### Nested Schema for `layered_references`

Read-Only:

- `dataset_id` (String) Dataset where the object is being created.
- `layer` (Number) Indicates in which layer the object should be created. Is 0 if empty, otherwise starts at 1. Objects in the same layer does not require dependency enforcement on each other. Different layers require the `depends_on` attribute to be set between them.
- `object_id` (String) Id of the object being created (usually is the routine_id or table_id).
- `object_type` (String) Pass-through from source_references (e.g. SCALAR_FUNCTION, PROCEDURE, VIEW).
- `reference_id` (String) Join key `<dataset_id>.<object_id>` matching parser `reference_id` and filename / for_each conventions. Use as the map key when looking up the object from the parser instances in each layer/stage.
- `resource_type` (String) Values are `ROUTINE` or `VIEW`. It is computed from the `object_type` attribute (can be used by the layering terraform code to filter the objects by resource type and assign them to the correct resource block).
