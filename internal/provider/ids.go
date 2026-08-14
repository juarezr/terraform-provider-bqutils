package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const idPlaceholder = "any"

// resourceID builds a google_bigquery_routine / google_bigquery_table-style id.
// kind is "routines" or "tables". Empty project or dataset segments become "any"
// for uniqueness only; callers must not expose that placeholder on project/dataset_id attributes.
func resourceID(kind, project, dataset, object string) string {
	if project == "" {
		project = idPlaceholder
	}
	if dataset == "" {
		dataset = idPlaceholder
	}
	return fmt.Sprintf("projects/%s/datasets/%s/%s/%s", project, dataset, kind, object)
}

// referenceID builds the dependency-join key "<dataset_id>.<object_id>".
// Returns null when either segment is empty (unqualified CREATE names).
func referenceID(dataset, object string) types.String {
	if dataset == "" || object == "" {
		return types.StringNull()
	}
	return types.StringValue(dataset + "." + object)
}
