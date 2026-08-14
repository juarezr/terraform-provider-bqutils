package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestProviderConfigure(t *testing.T) {
	p := New("test")()
	var resp provider.ConfigureResponse
	p.Configure(context.Background(), provider.ConfigureRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Configure diagnostics: %v", resp.Diagnostics)
	}
}

func TestProviderMetadata(t *testing.T) {
	p := New("0.1.0")()
	var resp provider.MetadataResponse
	p.Metadata(context.Background(), provider.MetadataRequest{}, &resp)
	if resp.TypeName != providerName {
		t.Fatalf("TypeName=%q", resp.TypeName)
	}
	if resp.Version != "0.1.0" {
		t.Fatalf("Version=%q", resp.Version)
	}
}
