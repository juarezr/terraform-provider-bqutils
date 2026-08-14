package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/juarezr/terraform-provider-bqutils/internal/sqlparse"
)

func mapRoutineArguments(ctx context.Context, args []sqlparse.Argument) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	argType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":          types.StringType,
			"data_type":     types.StringType,
			"argument_kind": types.StringType,
			"mode":          types.StringType,
			"is_aggregate":  types.BoolType,
		},
	}
	argVals := make([]attr.Value, 0, len(args))
	for _, a := range args {
		obj, d := types.ObjectValue(argType.AttrTypes, map[string]attr.Value{
			"name":          types.StringValue(a.Name),
			"data_type":     stringOrNull(a.DataTypeJSON),
			"argument_kind": stringOrNull(a.ArgumentKind),
			"mode":          stringOrNull(a.Mode),
			"is_aggregate":  boolPtrOrNull(a.IsAggregate),
		})
		diags.Append(d...)
		argVals = append(argVals, obj)
	}
	list, d := types.ListValue(argType, argVals)
	diags.Append(d...)
	return list, diags
}

func mapRemoteFunctionOptions(ctx context.Context, opts *sqlparse.RemoteFunctionOptions) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	remoteType := map[string]attr.Type{
		"connection":           types.StringType,
		"endpoint":             types.StringType,
		"max_batching_rows":    types.StringType,
		"user_defined_context": types.MapType{ElemType: types.StringType},
	}
	if opts == nil {
		return types.ObjectNull(remoteType), diags
	}
	ctxMap := opts.UserDefinedContext
	if ctxMap == nil {
		ctxMap = map[string]string{}
	}
	ctxVal, d := types.MapValueFrom(ctx, types.StringType, ctxMap)
	diags.Append(d...)
	obj, d := types.ObjectValue(remoteType, map[string]attr.Value{
		"connection":           stringOrNull(opts.Connection),
		"endpoint":             stringOrNull(opts.Endpoint),
		"max_batching_rows":    stringOrNull(opts.MaxBatchingRows),
		"user_defined_context": ctxVal,
	})
	diags.Append(d...)
	return obj, diags
}

func mapSparkOptions(ctx context.Context, opts *sqlparse.SparkOptions) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	sparkType := map[string]attr.Type{
		"connection":      types.StringType,
		"runtime_version": types.StringType,
		"container_image": types.StringType,
		"main_file_uri":   types.StringType,
		"main_class":      types.StringType,
		"py_file_uris":    types.ListType{ElemType: types.StringType},
		"jar_uris":        types.ListType{ElemType: types.StringType},
		"file_uris":       types.ListType{ElemType: types.StringType},
		"archive_uris":    types.ListType{ElemType: types.StringType},
		"properties":      types.MapType{ElemType: types.StringType},
	}
	if opts == nil {
		return types.ObjectNull(sparkType), diags
	}
	props := opts.Properties
	if props == nil {
		props = map[string]string{}
	}
	propsVal, d := types.MapValueFrom(ctx, types.StringType, props)
	diags.Append(d...)
	py, d := types.ListValueFrom(ctx, types.StringType, opts.PyFileURIs)
	diags.Append(d...)
	jar, d := types.ListValueFrom(ctx, types.StringType, opts.JarURIs)
	diags.Append(d...)
	files, d := types.ListValueFrom(ctx, types.StringType, opts.FileURIs)
	diags.Append(d...)
	arch, d := types.ListValueFrom(ctx, types.StringType, opts.ArchiveURIs)
	diags.Append(d...)
	obj, d := types.ObjectValue(sparkType, map[string]attr.Value{
		"connection":      stringOrNull(opts.Connection),
		"runtime_version": stringOrNull(opts.RuntimeVersion),
		"container_image": stringOrNull(opts.ContainerImage),
		"main_file_uri":   stringOrNull(opts.MainFileURI),
		"main_class":      stringOrNull(opts.MainClass),
		"py_file_uris":    py,
		"jar_uris":        jar,
		"file_uris":       files,
		"archive_uris":    arch,
		"properties":      propsVal,
	})
	diags.Append(d...)
	return obj, diags
}

func mapPythonOptions(ctx context.Context, opts *sqlparse.PythonOptions) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	pythonType := map[string]attr.Type{
		"entry_point": types.StringType,
		"packages":    types.ListType{ElemType: types.StringType},
	}
	if opts == nil {
		return types.ObjectNull(pythonType), diags
	}
	pkgs, d := types.ListValueFrom(ctx, types.StringType, opts.Packages)
	diags.Append(d...)
	obj, d := types.ObjectValue(pythonType, map[string]attr.Value{
		"entry_point": stringOrNull(opts.EntryPoint),
		"packages":    pkgs,
	})
	diags.Append(d...)
	return obj, diags
}

func mapExternalRuntimeOptions(opts *sqlparse.ExternalRuntimeOptions) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	extType := map[string]attr.Type{
		"container_memory":              types.StringType,
		"container_cpu":                 types.StringType,
		"runtime_connection":            types.StringType,
		"max_batching_rows":             types.StringType,
		"runtime_version":               types.StringType,
		"container_request_concurrency": types.StringType,
	}
	if opts == nil {
		return types.ObjectNull(extType), diags
	}
	obj, d := types.ObjectValue(extType, map[string]attr.Value{
		"container_memory":              stringOrNull(opts.ContainerMemory),
		"container_cpu":                 stringOrNull(opts.ContainerCPU),
		"runtime_connection":            stringOrNull(opts.RuntimeConnection),
		"max_batching_rows":             stringOrNull(opts.MaxBatchingRows),
		"runtime_version":               stringOrNull(opts.RuntimeVersion),
		"container_request_concurrency": stringOrNull(opts.ContainerRequestConcurrency),
	})
	diags.Append(d...)
	return obj, diags
}
