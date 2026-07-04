package main

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestNodesDataSourceSchemaIncludesPausedAndDraining(t *testing.T) {
	var resp datasource.SchemaResponse
	NewNodesDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	nodesAttr, ok := resp.Schema.Attributes["nodes"]
	if !ok {
		t.Fatal("nodes data source schema missing nodes attribute")
	}
	nodesList, ok := nodesAttr.(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("nodes attribute type = %T, want schema.ListNestedAttribute", nodesAttr)
	}

	for _, name := range []string{"paused", "draining"} {
		if _, ok := nodesList.NestedObject.Attributes[name]; !ok {
			t.Fatalf("nodes nested schema missing %q", name)
		}
	}
}
