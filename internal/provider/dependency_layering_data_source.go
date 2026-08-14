package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	deps "github.com/juarezr/terraform-provider-bqutils/internal/deptrack"
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
	ReferenceID  types.String `tfsdk:"reference_id"`
}

func (d *DependencyLayeringDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dependency_layering"
}

func (d *DependencyLayeringDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Computes creation-order layers (stages) used to track the dependencies between BigQuery routines and views. It allows the resource to be created in the correct order and avoid errors when executing the terraform apply command. Designed to be used coupled with the `for_each` and `depends_on` attributes in dynamic resource blocks of type `google_bigquery_routine` or `google_bigquery_table` from the Google Provider separated by layers.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic id for this data source instance.",
				Computed:            true,
			},
			"source_references": schema.ListNestedAttribute{
				MarkdownDescription: "Receives the list of objects and dependencies to compute the creation order and layering. Typically this list is built from bqutils_routine_parser / bqutils_view_parser instances. Its values may be assigned directly from the `references` attributes from a computed list of parsers.",
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
							MarkdownDescription: "Assigned typically from the `routine_type` attribute for `bqutils_routine_parser` instances, or `VIEW` for `bqutils_view_parser` instances (`MATERIALIZED_VIEW` also allowed).",
							Required:            true,
						},
						"references": schema.ListNestedAttribute{
							MarkdownDescription: "Dependencies from the parsed SQL body (usually obtained from the parser `references` attribute).",
							Required:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"dataset_id": schema.StringAttribute{
										MarkdownDescription: "Dataset of the referenced object extracted from the parsed SQL body.",
										Required:            true,
									},
									"object_id": schema.StringAttribute{
										MarkdownDescription: "Id of the referenced object extracted from the parsed SQL body.",
										Required:            true,
									},
									"object_type": schema.StringAttribute{
										MarkdownDescription: "Call-site object type detected in the parsed SQL body.",
										Required:            true,
									},
									"resource_type": schema.StringAttribute{
										MarkdownDescription: "Values are `ROUTINE` or `VIEW`. It is computed from the `object_type` attribute (can be used by the layering terraform code to filter the objects by resource type and assign them to the correct resource block).",
										Optional:            true,
									},
								},
							},
						},
					},
				},
			},
			"layered_references": schema.ListNestedAttribute{
				MarkdownDescription: "Return the objects received in the `source_references` attribute ordered by the order that they should be created. These objects are sorted by layer then dataset_id then object_id. To create the resources in the correct order, you have to supply these objects to be created in dynamic resource blocks of type `google_bigquery_routine` or `google_bigquery_table` separated by layer. Each layer will be dependent on the previous layer, so you have to set the `depends_on` attributes between the layers/blocks.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"layer": schema.Int64Attribute{
							MarkdownDescription: "Indicates in which layer the object should be created. Is 0 if empty, otherwise starts at 1. Objects in the same layer does not require dependency enforcement on each other. Different layers require the `depends_on` attribute to be set between them.",
							Computed:            true,
						},
						"dataset_id": schema.StringAttribute{
							MarkdownDescription: "Dataset where the object is being created.",
							Computed:            true,
						},
						"object_id": schema.StringAttribute{
							MarkdownDescription: "Id of the object being created (usually is the routine_id or table_id).",
							Computed:            true,
						},
						"object_type": schema.StringAttribute{
							MarkdownDescription: "Pass-through from source_references (e.g. SCALAR_FUNCTION, PROCEDURE, VIEW).",
							Computed:            true,
						},
						"resource_type": schema.StringAttribute{
							MarkdownDescription: "Values are `ROUTINE` or `VIEW`. It is computed from the `object_type` attribute (can be used by the layering terraform code to filter the objects by resource type and assign them to the correct resource block).",
							Computed:            true,
						},
						"reference_id": schema.StringAttribute{
							MarkdownDescription: "Join key `<dataset_id>.<object_id>` matching parser `reference_id` and filename / for_each conventions. Use as the map key when looking up the object from the parser instances in each layer/stage.",
							Computed:            true,
						},
					},
				},
			},
			"max_layers": schema.Int64Attribute{
				MarkdownDescription: "Maximum layer number detected in the `layered_references` attribute. 0 when empty. Use it to know the maximum number of layers that should be created in the terraform code.",
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
		"reference_id":  types.StringType,
	}}
	layeredModels := make([]layeredReferenceModel, 0, len(result.Layered))
	for _, n := range result.Layered {
		layeredModels = append(layeredModels, layeredReferenceModel{
			Layer:        types.Int64Value(int64(n.Layer)),
			DatasetID:    types.StringValue(n.DatasetID),
			ObjectID:     types.StringValue(n.ObjectID),
			ObjectType:   types.StringValue(n.ObjectType),
			ResourceType: types.StringValue(n.ResourceType),
			ReferenceID:  referenceID(n.DatasetID, n.ObjectID),
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
