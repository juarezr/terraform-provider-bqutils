package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func protoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"bqutils": providerserver.NewProtocol6WithError(New("test")()),
	}
}

func TestAccRoutineParser_tableFunction(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
data "bqutils_routine_parser" "test" {
  sql = <<-EOF
    CREATE OR REPLACE TABLE FUNCTION mydataset.list_partitions
    (
        table_name_filter STRING
    )
    OPTIONS (
      description = 'desc'
    ) AS (
      SELECT 1
    );
  EOF
  trim_comments = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.bqutils_routine_parser.test", "routine_id", "list_partitions"),
					resource.TestCheckResourceAttr("data.bqutils_routine_parser.test", "dataset_id", "mydataset"),
					resource.TestCheckResourceAttr("data.bqutils_routine_parser.test", "routine_type", "TABLE_VALUED_FUNCTION"),
					resource.TestCheckResourceAttr("data.bqutils_routine_parser.test", "language", "SQL"),
					resource.TestCheckResourceAttr("data.bqutils_routine_parser.test", "description", "desc"),
					resource.TestCheckResourceAttr("data.bqutils_routine_parser.test", "arguments.0.name", "table_name_filter"),
				),
			},
		},
	})
}

func TestAccRoutineParser_jsFunction(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
data "bqutils_routine_parser" "test" {
  sql = <<-EOF
    CREATE OR REPLACE FUNCTION parse_json_to_array(json_str STRING)
    RETURNS ARRAY<STRING>
    LANGUAGE js AS r"""
      return [];
    """;
  EOF
  trim_body = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.bqutils_routine_parser.test", "routine_id", "parse_json_to_array"),
					resource.TestCheckResourceAttr("data.bqutils_routine_parser.test", "routine_type", "SCALAR_FUNCTION"),
					resource.TestCheckResourceAttr("data.bqutils_routine_parser.test", "language", "JAVASCRIPT"),
					resource.TestCheckResourceAttrSet("data.bqutils_routine_parser.test", "return_type"),
				),
			},
		},
	})
}

func TestAccRoutineParser_tempError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
data "bqutils_routine_parser" "test" {
  sql = "CREATE TEMP FUNCTION foo(x INT64) AS (x);"
}
`,
				ExpectError: regexp.MustCompile(`TEMP`),
			},
		},
	})
}

func TestAccViewParser_simple(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
data "bqutils_view_parser" "test" {
  sql = <<-EOF
    CREATE OR REPLACE VIEW IF NOT EXISTS ` + "`mydataset.my_simple_view`" + `
    (
      TABLE_SCHEMA OPTIONS(description="schema"),
      TABLE_NAME OPTIONS(description="name")
    ) OPTIONS(
      description="Simple view"
    ) AS
      SELECT TABLE_SCHEMA, TABLE_NAME FROM t;
  EOF
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.bqutils_view_parser.test", "table_id", "my_simple_view"),
					resource.TestCheckResourceAttr("data.bqutils_view_parser.test", "dataset_id", "mydataset"),
					resource.TestCheckResourceAttr("data.bqutils_view_parser.test", "is_materialized", "false"),
					resource.TestCheckResourceAttr("data.bqutils_view_parser.test", "description", "Simple view"),
					resource.TestCheckResourceAttrSet("data.bqutils_view_parser.test", "schema"),
					resource.TestCheckResourceAttrSet("data.bqutils_view_parser.test", "query"),
				),
			},
		},
	})
}

func TestAccViewParser_materialized(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
data "bqutils_view_parser" "test" {
  sql = <<-EOF
    CREATE OR REPLACE MATERIALIZED VIEW IF NOT EXISTS ` + "`mydataset.mv`" + `
    PARTITION BY DATE(CREATED)
    CLUSTER BY A, B
    OPTIONS(
      description="mv",
      enable_refresh=TRUE,
      allow_non_incremental_definition=FALSE,
      refresh_interval_minutes=60,
      kms_key_name="projects/x/key",
      labels=[("org_unit", "development")]
    ) AS
      SELECT 1 AS A, 2 AS B, CURRENT_TIMESTAMP() AS CREATED;
  EOF
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.bqutils_view_parser.test", "is_materialized", "true"),
					resource.TestCheckResourceAttr("data.bqutils_view_parser.test", "partitioning_type", "DAY"),
					resource.TestCheckResourceAttr("data.bqutils_view_parser.test", "partitioning_field", "CREATED"),
					resource.TestCheckResourceAttr("data.bqutils_view_parser.test", "refresh_interval_ms", "3600000"),
					resource.TestCheckResourceAttr("data.bqutils_view_parser.test", "labels.org_unit", "development"),
					resource.TestCheckResourceAttr("data.bqutils_view_parser.test", "clustering.0", "A"),
				),
			},
		},
	})
}
