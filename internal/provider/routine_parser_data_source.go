package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/juarezr/terraform-provider-bqutils/internal/sqlparse"
)

var _ datasource.DataSource = &RoutineParserDataSource{}

func NewRoutineParserDataSource() datasource.DataSource {
	return &RoutineParserDataSource{}
}

type RoutineParserDataSource struct{}

type routineParserModel struct {
	SQL                   types.String `tfsdk:"sql"`
	TrimBody              types.Bool   `tfsdk:"trim_body"`
	TrimComments          types.Bool   `tfsdk:"trim_comments"`
	ID                    types.String `tfsdk:"id"`
	Project               types.String `tfsdk:"project"`
	DatasetID             types.String `tfsdk:"dataset_id"`
	RoutineID             types.String `tfsdk:"routine_id"`
	RoutineType           types.String `tfsdk:"routine_type"`
	DefinitionBody        types.String `tfsdk:"definition_body"`
	Language              types.String `tfsdk:"language"`
	ReturnType            types.String `tfsdk:"return_type"`
	ReturnTableType       types.String `tfsdk:"return_table_type"`
	Description           types.String `tfsdk:"description"`
	ImportedLibraries     types.List   `tfsdk:"imported_libraries"`
	DeterminismLevel      types.String `tfsdk:"determinism_level"`
	DataGovernanceType    types.String `tfsdk:"data_governance_type"`
	Arguments             types.List   `tfsdk:"arguments"`
	RemoteFunctionOptions types.Object `tfsdk:"remote_function_options"`
	SparkOptions          types.Object `tfsdk:"spark_options"`
}

func (d *RoutineParserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_routine_parser"
}

func (d *RoutineParserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Parses a BigQuery CREATE FUNCTION / TABLE FUNCTION / PROCEDURE / AGGREGATE FUNCTION statement and exposes attributes for google_bigquery_routine.",
		Attributes: map[string]schema.Attribute{
			"sql": schema.StringAttribute{
				MarkdownDescription: "Full CREATE statement SQL text.",
				Required:            true,
			},
			"trim_body": schema.BoolAttribute{
				MarkdownDescription: "Trim leading/trailing whitespace and empty lines from definition_body. Defaults to true.",
				Optional:            true,
			},
			"trim_comments": schema.BoolAttribute{
				MarkdownDescription: "Remove SQL comments from definition_body. Defaults to false.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic id matching google_bigquery_routine: projects/<project>/datasets/<dataset_id>/routines/<routine_id>. Missing project or dataset segments use the placeholder \"any\" (not exposed on project/dataset_id).",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "Project from a three-part routine name, if present.",
				Computed:            true,
			},
			"dataset_id": schema.StringAttribute{
				MarkdownDescription: "Dataset from a qualified routine name, if present.",
				Computed:            true,
			},
			"routine_id": schema.StringAttribute{
				Computed: true,
			},
			"routine_type": schema.StringAttribute{
				MarkdownDescription: "SCALAR_FUNCTION, TABLE_VALUED_FUNCTION, or PROCEDURE.",
				Computed:            true,
			},
			"definition_body": schema.StringAttribute{
				Computed: true,
			},
			"language": schema.StringAttribute{
				Computed: true,
			},
			"return_type": schema.StringAttribute{
				MarkdownDescription: "StandardSqlDataType JSON string.",
				Computed:            true,
			},
			"return_table_type": schema.StringAttribute{
				MarkdownDescription: "JSON for RETURNS TABLE<...> when present.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				Computed: true,
			},
			"imported_libraries": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
			"determinism_level": schema.StringAttribute{
				Computed: true,
			},
			"data_governance_type": schema.StringAttribute{
				Computed: true,
			},
			"arguments": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed: true,
						},
						"data_type": schema.StringAttribute{
							MarkdownDescription: "StandardSqlDataType JSON string.",
							Computed:            true,
						},
						"argument_kind": schema.StringAttribute{
							Computed: true,
						},
						"mode": schema.StringAttribute{
							Computed: true,
						},
						"is_aggregate": schema.BoolAttribute{
							MarkdownDescription: "For CREATE AGGREGATE FUNCTION parameters: false when the SQL has NOT AGGREGATE, true for aggregate parameters. Null for non-UDAF routines. google_bigquery_routine does not expose this field yet.",
							Computed:            true,
						},
					},
				},
			},
			"remote_function_options": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"connection": schema.StringAttribute{Computed: true},
					"endpoint":   schema.StringAttribute{Computed: true},
				},
			},
			"spark_options": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"raw": schema.StringAttribute{
						MarkdownDescription: "Raw spark options JSON when present.",
						Computed:            true,
					},
				},
			},
		},
	}
}

func (d *RoutineParserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data routineParserModel
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

	result, err := sqlparse.ParseRoutine(data.SQL.ValueString(), sqlparse.Options{
		TrimBody:     trimBody,
		TrimComments: trimComments,
	})
	if err != nil {
		resp.Diagnostics.AddError("SQL parse error", err.Error())
		return
	}

	data.ID = types.StringValue(resourceID("routines", result.Project, result.DatasetID, result.ObjectID))
	data.TrimBody = types.BoolValue(trimBody)
	data.TrimComments = types.BoolValue(trimComments)
	data.Project = stringOrNull(result.Project)
	data.DatasetID = stringOrNull(result.DatasetID)
	data.RoutineID = types.StringValue(result.ObjectID)
	data.RoutineType = types.StringValue(string(result.Kind))
	data.DefinitionBody = types.StringValue(result.DefinitionBody)
	data.Language = stringOrNull(result.Language)
	data.ReturnType = stringOrNull(result.ReturnTypeJSON)
	data.ReturnTableType = stringOrNull(result.ReturnTableTypeJSON)
	data.Description = stringOrNull(result.Description)
	data.DeterminismLevel = stringOrNull(result.DeterminismLevel)
	data.DataGovernanceType = stringOrNull(result.DataGovernanceType)

	libs, diags := types.ListValueFrom(ctx, types.StringType, result.ImportedLibraries)
	resp.Diagnostics.Append(diags...)
	data.ImportedLibraries = libs

	argType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":          types.StringType,
			"data_type":     types.StringType,
			"argument_kind": types.StringType,
			"mode":          types.StringType,
			"is_aggregate":  types.BoolType,
		},
	}
	argVals := make([]attr.Value, 0, len(result.Arguments))
	for _, a := range result.Arguments {
		obj, diags := types.ObjectValue(argType.AttrTypes, map[string]attr.Value{
			"name":          types.StringValue(a.Name),
			"data_type":     stringOrNull(a.DataTypeJSON),
			"argument_kind": stringOrNull(a.ArgumentKind),
			"mode":          stringOrNull(a.Mode),
			"is_aggregate":  boolPtrOrNull(a.IsAggregate),
		})
		resp.Diagnostics.Append(diags...)
		argVals = append(argVals, obj)
	}
	argsList, diags := types.ListValue(argType, argVals)
	resp.Diagnostics.Append(diags...)
	data.Arguments = argsList

	remoteType := map[string]attr.Type{"connection": types.StringType, "endpoint": types.StringType}
	if result.RemoteConnection != "" || result.RemoteEndpoint != "" {
		obj, diags := types.ObjectValue(remoteType, map[string]attr.Value{
			"connection": stringOrNull(result.RemoteConnection),
			"endpoint":   stringOrNull(result.RemoteEndpoint),
		})
		resp.Diagnostics.Append(diags...)
		data.RemoteFunctionOptions = obj
	} else {
		data.RemoteFunctionOptions = types.ObjectNull(remoteType)
	}

	sparkType := map[string]attr.Type{"raw": types.StringType}
	if result.SparkOptionsJSON != "" {
		obj, diags := types.ObjectValue(sparkType, map[string]attr.Value{
			"raw": types.StringValue(result.SparkOptionsJSON),
		})
		resp.Diagnostics.Append(diags...)
		data.SparkOptions = obj
	} else {
		data.SparkOptions = types.ObjectNull(sparkType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func boolPtrOrNull(b *bool) types.Bool {
	if b == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*b)
}
