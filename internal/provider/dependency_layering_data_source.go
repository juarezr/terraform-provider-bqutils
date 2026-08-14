package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/juarezr/terraform-provider-bqutils/internal/deptrack"
)

var _ datasource.DataSource = &DependencyLayeringDataSource{}

func NewDependencyLayeringDataSource() datasource.DataSource {
	return &DependencyLayeringDataSource{}
}

type DependencyLayeringDataSource struct{}

type dependencyLayeringModel struct {
	SourceReferences  types.List   `tfsdk:"source_references"`
	LayeredReferences types.List   `tfsdk:"layered_references"`
	MaxLayers         types.Int64  `tfsdk:"max_layers"`
	ID                types.String `tfsdk:"id"`
}

type sourceReferenceModel struct {
	DatasetID  types.String `tfsdk:"dataset_id"`
	ObjectID   types.String `tfsdk:"object_id"`
	ObjectType types.String `tfsdk:"object_type"`
	References types.List   `tfsdk:"references"`
}

type layeredReferenceModel struct {
	Layer        types.Int64  `tfsdk:"layer"`
	DatasetID    types.String `tfsdk:"dataset_id"`
	ObjectID     types.String `tfsdk:"object_id"`
	ObjectType   types.String `tfsdk:"object_type"`
	ResourceType types.String `tfsdk:"resource_type"`
}

func (d *DependencyLayeringDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dependency_layering"
}

func (d *DependencyLayeringDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Computes creation-order layers (waves) for BigQuery routines and views from parser `references`, for Pattern B layered `for_each` with static inter-layer `depends_on`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic id for this data source instance.",
				Computed:            true,
			},
			"source_references": schema.ListNestedAttribute{
				MarkdownDescription: "Managed objects to layer, typically built from bqutils_routine_parser / bqutils_view_parser instances. Nested `references` may be assigned directly from parser `references` (extra fields like resource_type are accepted).",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"dataset_id": schema.StringAttribute{
							MarkdownDescription: "Dataset where the object will be created.",
							Required:            true,
						},
						"object_id": schema.StringAttribute{
							MarkdownDescription: "Routine id or table/view id being created.",
							Required:            true,
						},
						"object_type": schema.StringAttribute{
							MarkdownDescription: "From routine_type for routines, or VIEW for views (MATERIALIZED_VIEW allowed).",
							Required:            true,
						},
						"references": schema.ListNestedAttribute{
							MarkdownDescription: "Dependencies from the object's SQL body (parser references).",
							Required:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"dataset_id": schema.StringAttribute{
										MarkdownDescription: "Dataset of the referenced object.",
										Required:            true,
									},
									"object_id": schema.StringAttribute{
										MarkdownDescription: "Id of the referenced object.",
										Required:            true,
									},
									"object_type": schema.StringAttribute{
										MarkdownDescription: "Call-site object type from the parser.",
										Required:            true,
									},
									"resource_type": schema.StringAttribute{
										MarkdownDescription: "ROUTINE or VIEW from the parser (accepted for type compatibility; unused by the layering algorithm).",
										Optional:            true,
									},
								},
							},
						},
					},
				},
			},
			"layered_references": schema.ListNestedAttribute{
				MarkdownDescription: "Managed objects sorted by layer then dataset_id then object_id. Filter by layer for wave resources; use resource_type / object_type for alternate-2 / alternate-3 splits.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"layer": schema.Int64Attribute{
							MarkdownDescription: "Creation wave starting at 1. Objects in the same layer have no managed dependency on each other.",
							Computed:            true,
						},
						"dataset_id": schema.StringAttribute{
							MarkdownDescription: "Dataset of the object to create.",
							Computed:            true,
						},
						"object_id": schema.StringAttribute{
							MarkdownDescription: "Object id to create.",
							Computed:            true,
						},
						"object_type": schema.StringAttribute{
							MarkdownDescription: "Pass-through from source_references (e.g. SCALAR_FUNCTION, PROCEDURE, VIEW).",
							Computed:            true,
						},
						"resource_type": schema.StringAttribute{
							MarkdownDescription: "Derived: VIEW (or MATERIALIZED_VIEW) → VIEW; routine kinds → ROUTINE.",
							Computed:            true,
						},
					},
				},
			},
			"max_layers": schema.Int64Attribute{
				MarkdownDescription: "Maximum layer number in layered_references, or 0 when empty. Pre-declare wave resources l1..lN for at least this depth.",
				Computed:            true,
			},
		},
	}
}

func (d *DependencyLayeringDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dependencyLayeringModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sources []sourceReferenceModel
	resp.Diagnostics.Append(data.SourceReferences.ElementsAs(ctx, &sources, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nodes := make([]deps.SourceNode, 0, len(sources))
	for _, s := range sources {
		var edges []objectReferenceModel
		resp.Diagnostics.Append(s.References.ElementsAs(ctx, &edges, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		n := deps.SourceNode{
			DatasetID:  s.DatasetID.ValueString(),
			ObjectID:   s.ObjectID.ValueString(),
			ObjectType: s.ObjectType.ValueString(),
			References: make([]deps.EdgeRef, 0, len(edges)),
		}
		for _, e := range edges {
			n.References = append(n.References, deps.EdgeRef{
				DatasetID:  e.DatasetID.ValueString(),
				ObjectID:   e.ObjectID.ValueString(),
				ObjectType: e.ObjectType.ValueString(),
			})
		}
		nodes = append(nodes, n)
	}

	result, err := deps.ComputeLayers(nodes)
	if err != nil {
		resp.Diagnostics.AddError("Dependency layering error", err.Error())
		return
	}

	layeredType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"layer":         types.Int64Type,
		"dataset_id":    types.StringType,
		"object_id":     types.StringType,
		"object_type":   types.StringType,
		"resource_type": types.StringType,
	}}
	layeredModels := make([]layeredReferenceModel, 0, len(result.Layered))
	for _, n := range result.Layered {
		layeredModels = append(layeredModels, layeredReferenceModel{
			Layer:        types.Int64Value(int64(n.Layer)),
			DatasetID:    types.StringValue(n.DatasetID),
			ObjectID:     types.StringValue(n.ObjectID),
			ObjectType:   types.StringValue(n.ObjectType),
			ResourceType: types.StringValue(n.ResourceType),
		})
	}
	layered, diags := types.ListValueFrom(ctx, layeredType, layeredModels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.LayeredReferences = layered
	data.MaxLayers = types.Int64Value(int64(result.MaxLayers))
	data.ID = types.StringValue("dependency_layering")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
