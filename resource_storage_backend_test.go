package main

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestStorageBackendSchemaIncludesAllocationPolicy(t *testing.T) {
	var resp resource.SchemaResponse
	NewStorageBackendResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	for _, name := range []string{"allocation_policy", "disk_overcommit_ratio"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Fatalf("storage backend schema missing %q", name)
		}
	}
}

func TestStorageBackendToModelAllocationDefaultsAndValues(t *testing.T) {
	ctx := context.Background()
	var defaults StorageBackendModel
	diags := storageBackendToModel(ctx, &StorageBackend{Config: map[string]any{}}, &defaults)
	if diags.HasError() {
		t.Fatalf("default conversion diagnostics: %v", diags)
	}
	if defaults.AllocationPolicy.ValueString() != "thin" || defaults.DiskOvercommitRatio.ValueFloat64() != 1 {
		t.Fatalf("defaults = %q %.1f, want thin 1.0", defaults.AllocationPolicy.ValueString(), defaults.DiskOvercommitRatio.ValueFloat64())
	}

	var configured StorageBackendModel
	diags = storageBackendToModel(ctx, &StorageBackend{Config: map[string]any{
		"allocation_policy":     "thin",
		"disk_overcommit_ratio": 1.5,
	}}, &configured)
	if diags.HasError() {
		t.Fatalf("configured conversion diagnostics: %v", diags)
	}
	if configured.AllocationPolicy.ValueString() != "thin" || configured.DiskOvercommitRatio.ValueFloat64() != 1.5 {
		t.Fatalf("configured = %q %.1f, want thin 1.5", configured.AllocationPolicy.ValueString(), configured.DiskOvercommitRatio.ValueFloat64())
	}
	if _, present := configured.Config.Elements()["allocation_policy"]; present {
		t.Fatal("allocation_policy should not be duplicated in generic config state")
	}
}
