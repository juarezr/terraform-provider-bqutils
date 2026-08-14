package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func viewParserSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Parses a BigQuery CREATE VIEW or CREATE MATERIALIZED VIEW statement from a string and exposes the required attributes in `google_bigquery_table` resource in order to create the view in BigQuery.",
		Attributes: map[string]schema.Attribute{
			"sql": schema.StringAttribute{
				MarkdownDescription: "SQL text containing the CREATE VIEW or CREATE MATERIALIZED VIEW statement to be parsed.",
				Required:            true,
			},
			"trim_body": schema.BoolAttribute{
				MarkdownDescription: "Trim leading/trailing whitespace and empty lines from query. Defaults to true.",
				Optional:            true,
			},
			"trim_comments": schema.BoolAttribute{
				MarkdownDescription: "Remove SQL comments from query. Defaults to false.",
				Optional:            true,
			},
			"trim_indentation": schema.BoolAttribute{
				MarkdownDescription: "Remove the common first-level leading whitespace from each line of query (deeper indentation is kept). Useful for SQL embedded in indented Terraform heredocs. Defaults to true.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic id matching google_bigquery_table: projects/<project>/datasets/<dataset_id>/tables/<table_id>. Missing project or dataset segments use the placeholder \"any\" (not exposed on project/dataset_id).",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "Project parsed from a three-part view name, if present.",
				Computed:            true,
			},
			"dataset_id": schema.StringAttribute{
				MarkdownDescription: "Dataset parsed from the SQL statement, if present.",
				Computed:            true,
			},
			"table_id": schema.StringAttribute{
				MarkdownDescription: "Table/view id parsed from the SQL statement.",
				Computed:            true,
			},
			"reference_id": schema.StringAttribute{
				MarkdownDescription: "Join key `<dataset_id>.<table_id>` for dependency tracking and layering dependency layering (same as filename / layering convention). Null when dataset_id is absent. Distinct from the GCP-path `id`.",
				Computed:            true,
			},
			"query": schema.StringAttribute{
				MarkdownDescription: "View query body after the AS element in the SQL statement.",
				Computed:            true,
			},
			"dataset_references": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Distinct dataset IDs referenced in the view query that differ from the view's own dataset. Empty when the query only uses the home dataset. Useful with google_bigquery_dataset_access for authorized views.",
				Computed:            true,
			},
			"references": objectReferencesSchema("Detected references to other objects in the routine's definition_body (routines, views, tables). Thery are unique and sorted by dataset_id then object_id. Excludes self-references. Use with bqutils_dependency_layering for determining the creation-order and applying them in layer/stages."),
			"description": schema.StringAttribute{
				MarkdownDescription: "Description from the OPTIONS section of the SQL statement, if present.",
				Computed:            true,
			},
			"friendly_name": schema.StringAttribute{
				MarkdownDescription: "Friendly name from the OPTIONS section of the SQL statement, if present.",
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Labels from the OPTIONS section of the SQL statement, if present.",
				Computed:            true,
			},
			"is_materialized": schema.BoolAttribute{
				MarkdownDescription: "Gives `True` when the SQL statement is CREATE MATERIALIZED VIEW, `False` otherwise.",
				Computed:            true,
			},
			"schema": schema.StringAttribute{
				MarkdownDescription: "JSON schema from the view column list when present (types default to STRING when not specified in SQL).",
				Computed:            true,
			},
			"enable_refresh": schema.BoolAttribute{
				MarkdownDescription: "Materialized view enable_refresh from the OPTIONS section when present.",
				Computed:            true,
			},
			"allow_non_incremental_definition": schema.BoolAttribute{
				MarkdownDescription: "Materialized view allow_non_incremental_definition option when present.",
				Computed:            true,
			},
			"refresh_interval_ms": schema.Int64Attribute{
				MarkdownDescription: "Converted from refresh_interval_minutes from the OPTIONS section when present.",
				Computed:            true,
			},
			"max_staleness": schema.StringAttribute{
				MarkdownDescription: "IntervalValue encoding (Y-M D H:M:S) for google_bigquery_table.max_staleness. SQL INTERVAL options such as INTERVAL 90 MINUTE or INTERVAL \"4:0:0\" HOUR TO SECOND are converted automatically.",
				Computed:            true,
			},
			"kms_key_name": schema.StringAttribute{
				MarkdownDescription: "KMS key name from the OPTIONS section of the SQL statement, if present.",
				Computed:            true,
			},
			"partitioning_type": schema.StringAttribute{
				MarkdownDescription: "Time partitioning type derived from PARTITION BY clause in the SQL statement when present.",
				Computed:            true,
			},
			"partitioning_field": schema.StringAttribute{
				MarkdownDescription: "Partitioning field derived from PARTITION BY clause in the SQL statement when present.",
				Computed:            true,
			},
			"clustering": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Clustering columns from CLUSTER BY clause in the SQL statement when present.",
				Computed:            true,
			},
		},
	}
}
