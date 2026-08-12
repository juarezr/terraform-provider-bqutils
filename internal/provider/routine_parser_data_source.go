package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/juarezr/terraform-provider-bqutils/internal/sqlparse"
)

var _ datasource.DataSource = &RoutineParserDataSource{}

func NewRoutineParserDataSource() datasource.DataSource {
	return &RoutineParserDataSource{}
}

type RoutineParserDataSource struct{}

type routineParserModel struct {
	SQL                    types.String `tfsdk:"sql"`
	TrimBody               types.Bool   `tfsdk:"trim_body"`
	TrimComments           types.Bool   `tfsdk:"trim_comments"`
	TrimIndentation        types.Bool   `tfsdk:"trim_indentation"`
	ID                     types.String `tfsdk:"id"`
	Project                types.String `tfsdk:"project"`
	DatasetID              types.String `tfsdk:"dataset_id"`
	RoutineID              types.String `tfsdk:"routine_id"`
	RoutineType            types.String `tfsdk:"routine_type"`
	DefinitionBody         types.String `tfsdk:"definition_body"`
	Language               types.String `tfsdk:"language"`
	ReturnType             types.String `tfsdk:"return_type"`
	ReturnTableType        types.String `tfsdk:"return_table_type"`
	Description            types.String `tfsdk:"description"`
	ImportedLibraries      types.List   `tfsdk:"imported_libraries"`
	DeterminismLevel       types.String `tfsdk:"determinism_level"`
	DataGovernanceType     types.String `tfsdk:"data_governance_type"`
	Arguments              types.List   `tfsdk:"arguments"`
	RemoteFunctionOptions  types.Object `tfsdk:"remote_function_options"`
	SparkOptions           types.Object `tfsdk:"spark_options"`
	PythonOptions          types.Object `tfsdk:"python_options"`
	ExternalRuntimeOptions types.Object `tfsdk:"external_runtime_options"`
}

func (d *RoutineParserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_routine_parser"
}

func (d *RoutineParserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = routineParserSchema()
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
	trimIndentation := true
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

	argsList, diags := mapRoutineArguments(ctx, result.Arguments)
	resp.Diagnostics.Append(diags...)
	data.Arguments = argsList

	remote, diags := mapRemoteFunctionOptions(ctx, result.RemoteFunctionOptions)
	resp.Diagnostics.Append(diags...)
	data.RemoteFunctionOptions = remote

	spark, diags := mapSparkOptions(ctx, result.SparkOptions)
	resp.Diagnostics.Append(diags...)
	data.SparkOptions = spark

	python, diags := mapPythonOptions(ctx, result.PythonOptions)
	resp.Diagnostics.Append(diags...)
	data.PythonOptions = python

	ext, diags := mapExternalRuntimeOptions(result.ExternalRuntimeOptions)
	resp.Diagnostics.Append(diags...)
	data.ExternalRuntimeOptions = ext

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
