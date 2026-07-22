package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/juarezr/terraform-provider-bqutils/internal/sqlparse"
)

var _ datasource.DataSource = &ViewParserDataSource{}

func NewViewParserDataSource() datasource.DataSource {
	return &ViewParserDataSource{}
}

type ViewParserDataSource struct{}

type viewParserModel struct {
	SQL                           types.String `tfsdk:"sql"`
	TrimBody                      types.Bool   `tfsdk:"trim_body"`
	TrimComments                  types.Bool   `tfsdk:"trim_comments"`
	ID                            types.String `tfsdk:"id"`
	Project                       types.String `tfsdk:"project"`
	DatasetID                     types.String `tfsdk:"dataset_id"`
	TableID                       types.String `tfsdk:"table_id"`
	Query                         types.String `tfsdk:"query"`
	Description                   types.String `tfsdk:"description"`
	FriendlyName                  types.String `tfsdk:"friendly_name"`
	Labels                        types.Map    `tfsdk:"labels"`
	IsMaterialized                types.Bool   `tfsdk:"is_materialized"`
	Schema                        types.String `tfsdk:"schema"`
	EnableRefresh                 types.Bool   `tfsdk:"enable_refresh"`
	AllowNonIncrementalDefinition types.Bool   `tfsdk:"allow_non_incremental_definition"`
	RefreshIntervalMs             types.Int64  `tfsdk:"refresh_interval_ms"`
	MaxStaleness                  types.String `tfsdk:"max_staleness"`
	KmsKeyName                    types.String `tfsdk:"kms_key_name"`
	PartitioningType              types.String `tfsdk:"partitioning_type"`
	PartitioningField             types.String `tfsdk:"partitioning_field"`
	Clustering                    types.List   `tfsdk:"clustering"`
}

func (d *ViewParserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_view_parser"
}

func (d *ViewParserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Parses a BigQuery CREATE VIEW / MATERIALIZED VIEW statement and exposes attributes for google_bigquery_table.",
		Attributes: map[string]schema.Attribute{
			"sql": schema.StringAttribute{
				MarkdownDescription: "Full CREATE VIEW statement SQL text.",
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
			"id":            schema.StringAttribute{Computed: true},
			"project":       schema.StringAttribute{Computed: true},
			"dataset_id":    schema.StringAttribute{Computed: true},
			"table_id":      schema.StringAttribute{Computed: true},
			"query":         schema.StringAttribute{Computed: true},
			"description":   schema.StringAttribute{Computed: true},
			"friendly_name": schema.StringAttribute{Computed: true},
			"labels": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
			"is_materialized": schema.BoolAttribute{
				MarkdownDescription: "True when the statement is CREATE MATERIALIZED VIEW.",
				Computed:            true,
			},
			"schema": schema.StringAttribute{
				MarkdownDescription: "JSON schema derived from the view column list when present.",
				Computed:            true,
			},
			"enable_refresh":                   schema.BoolAttribute{Computed: true},
			"allow_non_incremental_definition": schema.BoolAttribute{Computed: true},
			"refresh_interval_ms":              schema.Int64Attribute{Computed: true},
			"max_staleness": schema.StringAttribute{
				MarkdownDescription: "IntervalValue encoding (Y-M D H:M:S) for google_bigquery_table. SQL INTERVAL options are converted automatically.",
				Computed:            true,
			},
			"kms_key_name":                     schema.StringAttribute{Computed: true},
			"partitioning_type":                schema.StringAttribute{Computed: true},
			"partitioning_field":               schema.StringAttribute{Computed: true},
			"clustering": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

func (d *ViewParserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data viewParserModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	trimBody := true
	if !data.TrimBody.IsNull() && !data.TrimBody.IsUnknown() {
		trimBody = data.TrimBody.ValueBool()
	}
	trimComments := false
	if !data.TrimComments.IsNull() && !data.TrimComments.IsUnknown() {
		trimComments = data.TrimComments.ValueBool()
	}

	result, err := sqlparse.ParseView(data.SQL.ValueString(), sqlparse.Options{
		TrimBody:     trimBody,
		TrimComments: trimComments,
	})
	if err != nil {
		resp.Diagnostics.AddError("SQL parse error", err.Error())
		return
	}

	data.ID = types.StringValue(result.ObjectID)
	data.TrimBody = types.BoolValue(trimBody)
	data.TrimComments = types.BoolValue(trimComments)
	data.Project = stringOrNull(result.Project)
	data.DatasetID = stringOrNull(result.DatasetID)
	data.TableID = types.StringValue(result.ObjectID)
	data.Query = types.StringValue(result.Query)
	data.Description = stringOrNull(result.Description)
	data.FriendlyName = stringOrNull(result.FriendlyName)
	data.IsMaterialized = types.BoolValue(result.IsMaterialized)
	data.Schema = stringOrNull(result.SchemaJSON)
	data.MaxStaleness = stringOrNull(result.MaxStaleness)
	data.KmsKeyName = stringOrNull(result.KmsKeyName)
	data.PartitioningType = stringOrNull(result.PartitioningType)
	data.PartitioningField = stringOrNull(result.PartitioningField)

	if result.EnableRefresh != nil {
		data.EnableRefresh = types.BoolValue(*result.EnableRefresh)
	} else {
		data.EnableRefresh = types.BoolNull()
	}
	if result.AllowNonIncrementalDefinition != nil {
		data.AllowNonIncrementalDefinition = types.BoolValue(*result.AllowNonIncrementalDefinition)
	} else {
		data.AllowNonIncrementalDefinition = types.BoolNull()
	}
	if result.RefreshIntervalMs != nil {
		data.RefreshIntervalMs = types.Int64Value(*result.RefreshIntervalMs)
	} else {
		data.RefreshIntervalMs = types.Int64Null()
	}

	if result.Labels == nil {
		result.Labels = map[string]string{}
	}
	labels, diags := types.MapValueFrom(ctx, types.StringType, result.Labels)
	resp.Diagnostics.Append(diags...)
	data.Labels = labels

	clustering, diags := types.ListValueFrom(ctx, types.StringType, result.Clustering)
	resp.Diagnostics.Append(diags...)
	data.Clustering = clustering

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
