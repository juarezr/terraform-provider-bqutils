package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/juarezr/terraform-provider-bqutils/internal/sqlparse"
)

func TestMapRemoteFunctionOptions(t *testing.T) {
	ctx := context.Background()

	nullObj, diags := mapRemoteFunctionOptions(ctx, nil)
	if diags.HasError() {
		t.Fatalf("nil: %v", diags)
	}
	if !nullObj.IsNull() {
		t.Fatal("expected null object for nil opts")
	}

	obj, diags := mapRemoteFunctionOptions(ctx, &sqlparse.RemoteFunctionOptions{
		Connection:         "proj.us.conn",
		Endpoint:           "https://example.com/fn",
		MaxBatchingRows:    "10",
		UserDefinedContext: map[string]string{"mode": "encrypt"},
	})
	if diags.HasError() {
		t.Fatalf("populated: %v", diags)
	}
	if obj.IsNull() {
		t.Fatal("expected non-null remote options")
	}
	attrs := obj.Attributes()
	if got := attrs["connection"].(types.String).ValueString(); got != "proj.us.conn" {
		t.Fatalf("connection=%q", got)
	}
	if got := attrs["endpoint"].(types.String).ValueString(); got != "https://example.com/fn" {
		t.Fatalf("endpoint=%q", got)
	}
}

func TestMapSparkOptions(t *testing.T) {
	ctx := context.Background()

	nullObj, diags := mapSparkOptions(ctx, nil)
	if diags.HasError() {
		t.Fatalf("nil: %v", diags)
	}
	if !nullObj.IsNull() {
		t.Fatal("expected null object for nil opts")
	}

	obj, diags := mapSparkOptions(ctx, &sqlparse.SparkOptions{
		Connection:     "proj.us.conn",
		RuntimeVersion: "2.2",
		MainFileURI:    "gs://bucket/main.py",
		PyFileURIs:     []string{"gs://bucket/lib.py"},
		Properties:     map[string]string{"spark.executor.memory": "2g"},
	})
	if diags.HasError() {
		t.Fatalf("populated: %v", diags)
	}
	if obj.IsNull() {
		t.Fatal("expected non-null spark options")
	}
	attrs := obj.Attributes()
	if got := attrs["runtime_version"].(types.String).ValueString(); got != "2.2" {
		t.Fatalf("runtime_version=%q", got)
	}
	if got := attrs["main_file_uri"].(types.String).ValueString(); got != "gs://bucket/main.py" {
		t.Fatalf("main_file_uri=%q", got)
	}
}

func TestMapPythonOptions(t *testing.T) {
	ctx := context.Background()

	nullObj, diags := mapPythonOptions(ctx, nil)
	if diags.HasError() {
		t.Fatalf("nil: %v", diags)
	}
	if !nullObj.IsNull() {
		t.Fatal("expected null object for nil opts")
	}

	obj, diags := mapPythonOptions(ctx, &sqlparse.PythonOptions{
		EntryPoint: "handler",
		Packages:   []string{"scipy==1.15.3"},
	})
	if diags.HasError() {
		t.Fatalf("populated: %v", diags)
	}
	if obj.IsNull() {
		t.Fatal("expected non-null python options")
	}
	attrs := obj.Attributes()
	if got := attrs["entry_point"].(types.String).ValueString(); got != "handler" {
		t.Fatalf("entry_point=%q", got)
	}
}

func TestMapExternalRuntimeOptions(t *testing.T) {
	nullObj, diags := mapExternalRuntimeOptions(nil)
	if diags.HasError() {
		t.Fatalf("nil: %v", diags)
	}
	if !nullObj.IsNull() {
		t.Fatal("expected null object for nil opts")
	}

	obj, diags := mapExternalRuntimeOptions(&sqlparse.ExternalRuntimeOptions{
		RuntimeVersion:  "python-3.11",
		ContainerMemory: "2Gi",
		ContainerCPU:    "1",
	})
	if diags.HasError() {
		t.Fatalf("populated: %v", diags)
	}
	if obj.IsNull() {
		t.Fatal("expected non-null external runtime options")
	}
	attrs := obj.Attributes()
	if got := attrs["runtime_version"].(types.String).ValueString(); got != "python-3.11" {
		t.Fatalf("runtime_version=%q", got)
	}
}
