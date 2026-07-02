package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &StorageBackendsDataSource{}
var _ datasource.DataSourceWithConfigure = &StorageBackendsDataSource{}

type StorageBackendsDataSource struct {
	client *Client
}

type StorageBackendsModel struct {
	Backends types.List `tfsdk:"backends"`
}

var storageBackendSummaryAttrTypes = map[string]attr.Type{
	"id":                    types.StringType,
	"name":                  types.StringType,
	"type":                  types.StringType,
	"is_default":            types.BoolType,
	"allocation_policy":     types.StringType,
	"disk_overcommit_ratio": types.Float64Type,
}

func NewStorageBackendsDataSource() datasource.DataSource {
	return &StorageBackendsDataSource{}
}

func (d *StorageBackendsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_backends"
}

func (d *StorageBackendsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all storage backends.",
		Attributes: map[string]schema.Attribute{
			"backends": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"type": schema.StringAttribute{
							Computed: true,
						},
						"is_default": schema.BoolAttribute{
							Computed: true,
						},
						"allocation_policy":     schema.StringAttribute{Computed: true},
						"disk_overcommit_ratio": schema.Float64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *StorageBackendsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", fmt.Sprintf("Expected *Client, got: %T", req.ProviderData))
		return
	}
	d.client = client
}

func (d *StorageBackendsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data StorageBackendsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backends, err := d.client.ListStorageBackends()
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Storage Backends", err.Error())
		return
	}

	backendVals := make([]attr.Value, 0, len(backends))
	for _, backend := range backends {
		policy := "thin"
		if value, ok := backend.Config["allocation_policy"].(string); ok && value != "" {
			policy = value
		}
		ratio := 1.0
		if value, ok := backend.Config["disk_overcommit_ratio"].(float64); ok && value != 0 {
			ratio = value
		}
		obj, diags := types.ObjectValue(storageBackendSummaryAttrTypes, map[string]attr.Value{
			"id":                    types.StringValue(backend.ID),
			"name":                  types.StringValue(backend.Name),
			"type":                  types.StringValue(backend.Type),
			"is_default":            types.BoolValue(backend.IsDefault),
			"allocation_policy":     types.StringValue(policy),
			"disk_overcommit_ratio": types.Float64Value(ratio),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		backendVals = append(backendVals, obj)
	}

	listVal, diags := types.ListValue(types.ObjectType{AttrTypes: storageBackendSummaryAttrTypes}, backendVals)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Backends = listVal
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
