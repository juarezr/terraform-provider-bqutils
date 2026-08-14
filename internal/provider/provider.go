package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Provider constants and definitions.
const (
	providerName    string = "bqutils"
	providerRemarks string = `Reduce the amount of work required to create, test, deploy, grant authorized access to
datasets and manage the source code of BigQuery functions, procedures and views when
using the Google BigQuery Terraform provider by using the same SQL script either in the
BigQuery Console and in Terraform seamlessly.`
)

var providerDescription = strings.ReplaceAll(providerRemarks, "\n", " ")

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
	resp.TypeName = providerName
	resp.Version = p.version
}

func (p *BqutilsProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         providerDescription,
		MarkdownDescription: providerDescription,
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
		NewDependencyLayeringDataSource,
	}
}
