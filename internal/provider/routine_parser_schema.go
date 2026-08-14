package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func routineParserSchema() schema.Schema {
	return schema.Schema{
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
				MarkdownDescription: "Remove the common first-level leading whitespace from each line of definition_body (deeper indentation is kept). Useful for SQL embedded in indented Terraform heredocs. Defaults to true.",
				Optional:            true,
			},
			"target_project": schema.StringAttribute{
				MarkdownDescription: "When set, two-part dataset.entity references in definition_body that point at datasets other than the routine's own dataset are rewritten as `project`.`dataset`.`entity` (backtick-quoted) for the BigQuery Routines API. Also replaces the `${project}` placeholder. If unset, the project from a three-part CREATE name is used when present.",
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
				MarkdownDescription: "The body of the routine. For functions, this is the expression in the AS clause. If language=SQL, it is the substring inside (but excluding) the parentheses. When target_project (or an inferred CREATE project) is available and the body references other datasets, those two-part refs are project-qualified.",
				Computed:            true,
			},
			"dataset_references": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Distinct dataset IDs referenced in definition_body that differ from the routine's own dataset. Empty when the body only uses the home dataset. Useful with google_bigquery_dataset_access for authorized routines.",
				Computed:            true,
			},
			"references": objectReferencesSchema("Objects referenced in definition_body (routines, views, tables), unique and sorted by dataset_id then object_id. Excludes self-references. Use with bqutils_dependency_layering for creation-order waves."),
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
			"arguments":                routineArgumentsSchema(),
			"remote_function_options":  remoteFunctionOptionsSchema(),
			"spark_options":            sparkOptionsSchema(),
			"python_options":           pythonOptionsSchema(),
			"external_runtime_options": externalRuntimeOptionsSchema(),
		},
	}
}

func routineArgumentsSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
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
	}
}

func remoteFunctionOptionsSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Remote function options when present (maps to google_bigquery_routine.remote_function_options).",
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
			"max_batching_rows": schema.StringAttribute{
				MarkdownDescription: "Max rows per batch sent to the remote service.",
				Computed:            true,
			},
			"user_defined_context": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "User-defined context key/value pairs.",
				Computed:            true,
			},
		},
	}
}

func sparkOptionsSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Spark stored procedure options when present (maps to google_bigquery_routine.spark_options).",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"connection": schema.StringAttribute{
				MarkdownDescription: "Spark connection resource name.",
				Computed:            true,
			},
			"runtime_version": schema.StringAttribute{
				MarkdownDescription: "Spark runtime version.",
				Computed:            true,
			},
			"container_image": schema.StringAttribute{
				MarkdownDescription: "Custom container image.",
				Computed:            true,
			},
			"main_file_uri": schema.StringAttribute{
				MarkdownDescription: "Main file/jar URI.",
				Computed:            true,
			},
			"main_class": schema.StringAttribute{
				MarkdownDescription: "Main class for Java/Scala.",
				Computed:            true,
			},
			"py_file_uris": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Python files on PYTHONPATH.",
				Computed:            true,
			},
			"jar_uris": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "JAR URIs.",
				Computed:            true,
			},
			"file_uris": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "File URIs for executors.",
				Computed:            true,
			},
			"archive_uris": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Archive URIs for executors.",
				Computed:            true,
			},
			"properties": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Spark configuration properties.",
				Computed:            true,
			},
		},
	}
}

func pythonOptionsSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Python UDF options when present (maps to google_bigquery_routine.python_options).",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"entry_point": schema.StringAttribute{
				MarkdownDescription: "Python entry point function name.",
				Computed:            true,
			},
			"packages": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Python packages to install.",
				Computed:            true,
			},
		},
	}
}

func externalRuntimeOptionsSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "External runtime options for Python UDFs (maps to google_bigquery_routine.external_runtime_options).",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"container_memory": schema.StringAttribute{
				MarkdownDescription: "Container memory (e.g. 512Mi).",
				Computed:            true,
			},
			"container_cpu": schema.StringAttribute{
				MarkdownDescription: "Container CPU amount.",
				Computed:            true,
			},
			"runtime_connection": schema.StringAttribute{
				MarkdownDescription: "Connection used to run container code.",
				Computed:            true,
			},
			"max_batching_rows": schema.StringAttribute{
				MarkdownDescription: "Max rows per batch to the external runtime.",
				Computed:            true,
			},
			"runtime_version": schema.StringAttribute{
				MarkdownDescription: "Language runtime version (e.g. python-3.11).",
				Computed:            true,
			},
			"container_request_concurrency": schema.StringAttribute{
				MarkdownDescription: "Max concurrent requests per container.",
				Computed:            true,
			},
		},
	}
}
