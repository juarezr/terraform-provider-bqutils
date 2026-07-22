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
	TrimIndentation       types.Bool   `tfsdk:"trim_indentation"`
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
		MarkdownDescription: "Parses a BigQuery CREATE SQL statement from a string and supplies its parts as attributes for google_bigquery_routine. Main use case: create and update BigQuery routines from SQL files with Terraform.",
		Attributes: map[string]schema.Attribute{
			"sql": schema.StringAttribute{
				MarkdownDescription: "SQL text containing the CREATE FUNCTION or CREATE PROCEDURE statement to be parsed.",
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
			"trim_indentation": schema.BoolAttribute{
				MarkdownDescription: "Remove the common first-level leading whitespace from each line of definition_body (deeper indentation is kept). Useful for SQL embedded in indented Terraform heredocs. Defaults to false.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic id matching google_bigquery_routine: projects/<project>/datasets/<dataset_id>/routines/<routine_id>. Missing project or dataset segments use the placeholder \"any\" (not exposed on project/dataset_id).",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "Project parsed from a three-part name, if present.",
				Computed:            true,
			},
			"dataset_id": schema.StringAttribute{
				MarkdownDescription: "Routine dataset parsed from the SQL statement, if present.",
				Computed:            true,
			},
			"routine_id": schema.StringAttribute{
				MarkdownDescription: "Name of the routine parsed from the SQL statement.",
				Computed:            true,
			},
			"routine_type": schema.StringAttribute{
				MarkdownDescription: "SCALAR_FUNCTION, TABLE_VALUED_FUNCTION, PROCEDURE, or AGGREGATE_FUNCTION.",
				Computed:            true,
			},
			"definition_body": schema.StringAttribute{
				MarkdownDescription: "The body of the routine. For functions, this is the expression in the AS clause. If language=SQL, it is the substring inside (but excluding) the parentheses.",
				Computed:            true,
			},
			"language": schema.StringAttribute{
				MarkdownDescription: "The language of the routine. Possible values: SQL, JAVASCRIPT, PYTHON, JAVA, SCALA.",
				Computed:            true,
			},
			"return_type": schema.StringAttribute{
				MarkdownDescription: "StandardSqlDataType as JSON schema for the function return type when present.",
				Computed:            true,
			},
			"return_table_type": schema.StringAttribute{
				MarkdownDescription: "JSON for RETURNS TABLE<...> when present (table-valued functions).",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description parsed from the SQL OPTIONS clause, if present.",
				Computed:            true,
			},
			"imported_libraries": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "If language is JAVASCRIPT, paths of imported JavaScript libraries.",
				Computed:            true,
			},
			"determinism_level": schema.StringAttribute{
				MarkdownDescription: "Determinism level of a JavaScript UDF if defined. Possible values: DETERMINISM_LEVEL_UNSPECIFIED, DETERMINISTIC, NOT_DETERMINISTIC.",
				Computed:            true,
			},
			"data_governance_type": schema.StringAttribute{
				MarkdownDescription: "If set to DATA_MASKING, the function is validated and made available as a masking function.",
				Computed:            true,
			},
			"arguments": schema.ListNestedAttribute{
				MarkdownDescription: "Routine arguments parsed from the SQL CREATE FUNCTION or CREATE PROCEDURE statement.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the routine argument.",
							Computed:            true,
						},
						"data_type": schema.StringAttribute{
							MarkdownDescription: "Standard SqlDataType as JSON schema of the argument data type.",
							Computed:            true,
						},
						"argument_kind": schema.StringAttribute{
							MarkdownDescription: "Default FIXED_TYPE. Possible values: FIXED_TYPE, ANY_TYPE.",
							Computed:            true,
						},
						"mode": schema.StringAttribute{
							MarkdownDescription: "Argument mode for procedures when present (IN, OUT, INOUT).",
							Computed:            true,
						},
						"is_aggregate": schema.BoolAttribute{
							MarkdownDescription: "Gives `True` when the SQL includes NOT AGGREGATE in CREATE AGGREGATE FUNCTION routines, `False` otherwise and `Null` for non-UDAF routines. google_bigquery_routine does not expose this field yet.",
							Computed:            true,
						},
					},
				},
			},
			"remote_function_options": schema.SingleNestedAttribute{
				MarkdownDescription: "Remote function options when present.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"connection": schema.StringAttribute{
						MarkdownDescription: "Connection resource name for the remote function.",
						Computed:            true,
					},
					"endpoint": schema.StringAttribute{
						MarkdownDescription: "Remote function endpoint URL.",
						Computed:            true,
					},
				},
			},
			"spark_options": schema.SingleNestedAttribute{
				MarkdownDescription: "If language is PYTHON, JAVA, or SCALA, then it returns the Spark options of the routine.",
				Computed:            true,
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
	trimIndentation := false
	if !data.TrimIndentation.IsNull() && !data.TrimIndentation.IsUnknown() {
		trimIndentation = data.TrimIndentation.ValueBool()
	}

	result, err := sqlparse.ParseRoutine(data.SQL.ValueString(), sqlparse.Options{
		TrimBody:        trimBody,
		TrimComments:    trimComments,
		TrimIndentation: trimIndentation,
	})
	if err != nil {
		resp.Diagnostics.AddError("SQL parse error", err.Error())
		return
	}

	data.ID = types.StringValue(resourceID("routines", result.Project, result.DatasetID, result.ObjectID))
	data.TrimBody = types.BoolValue(trimBody)
	data.TrimComments = types.BoolValue(trimComments)
	data.TrimIndentation = types.BoolValue(trimIndentation)
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
