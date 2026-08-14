package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/juarezr/terraform-provider-bqutils/internal/sqlparse"
)

type objectReferenceModel struct {
	DatasetID    types.String `tfsdk:"dataset_id"`
	ObjectID     types.String `tfsdk:"object_id"`
	ObjectType   types.String `tfsdk:"object_type"`
	ResourceType types.String `tfsdk:"resource_type"`
}

func objectReferenceAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"dataset_id":    types.StringType,
		"object_id":     types.StringType,
		"object_type":   types.StringType,
		"resource_type": types.StringType,
	}
}

func objectReferencesSchema(markdown string) schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		MarkdownDescription: markdown,
		Computed:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"dataset_id": schema.StringAttribute{
					MarkdownDescription: "Dataset ID of the referenced object (no backticks).",
					Computed:            true,
				},
				"object_id": schema.StringAttribute{
					MarkdownDescription: "Object ID (routine, view, or table name; no backticks).",
					Computed:            true,
				},
				"object_type": schema.StringAttribute{
					MarkdownDescription: "SCALAR_FUNCTION, TABLE_VALUED_FUNCTION, PROCEDURE, VIEW, or TABLE (INFORMATION_SCHEMA only). Call-site limits apply: VIEW may include tables/materialized views; SCALAR_FUNCTION may include aggregates.",
					Computed:            true,
				},
				"resource_type": schema.StringAttribute{
					MarkdownDescription: "ROUTINE or VIEW, derived from object_type for Terraform resource selection.",
					Computed:            true,
				},
			},
		},
	}
}

func mapObjectReferences(ctx context.Context, refs []sqlparse.ObjectReference) (types.List, diag.Diagnostics) {
	elemType := types.ObjectType{AttrTypes: objectReferenceAttrTypes()}
	if refs == nil {
		refs = []sqlparse.ObjectReference{}
	}
	models := make([]objectReferenceModel, 0, len(refs))
	for _, r := range refs {
		models = append(models, objectReferenceModel{
			DatasetID:    types.StringValue(r.DatasetID),
			ObjectID:     types.StringValue(r.ObjectID),
			ObjectType:   types.StringValue(r.ObjectType),
			ResourceType: types.StringValue(r.ResourceType),
		})
	}
	return types.ListValueFrom(ctx, elemType, models)
}
