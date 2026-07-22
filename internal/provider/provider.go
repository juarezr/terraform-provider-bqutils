package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Ensure Provider satisfies interfaces.
var _ provider.Provider = &BqutilsProvider{}

// BqutilsProvider is the provider implementation.
type BqutilsProvider struct {
	version string
}

// New returns a new provider factory.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &BqutilsProvider{version: version}
	}
}

func (p *BqutilsProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "bqutils"
	resp.Version = p.version
}

func (p *BqutilsProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The bqutils provider automates the creation and update of BigQuery functions/procedures/views using `CREATE` SQL statements stored in files, aiming to reduce the effort required to manage the source code of these objects.",
		Attributes:          map[string]schema.Attribute{},
	}
}

func (p *BqutilsProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}

func (p *BqutilsProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}

func (p *BqutilsProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewRoutineParserDataSource,
		NewViewParserDataSource,
	}
}
