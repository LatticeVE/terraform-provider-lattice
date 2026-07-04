package main

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestVMResourceSchemaIncludesStorageHAAndAllocationFields(t *testing.T) {
	var resp resource.SchemaResponse
	NewVMResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{
		"storage",
		"ha",
		"boot_disk_allocation",
		"boot_disk_allocation_policy",
	} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Fatalf("VM resource schema missing %q", name)
		}
	}
}
