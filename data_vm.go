package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &VMDataSource{}
var _ datasource.DataSourceWithConfigure = &VMDataSource{}

type VMDataSource struct {
	client *Client
}

type VMDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	CPUs       types.Int64  `tfsdk:"cpus"`
	MemoryMB   types.Int64  `tfsdk:"memory_mb"`
	Status     types.String `tfsdk:"status"`
	DiskPath   types.String `tfsdk:"disk_path"`
	BootDiskGB types.Int64  `tfsdk:"boot_disk_gb"`
}

func NewVMDataSource() datasource.DataSource {
	return &VMDataSource{}
}

func (d *VMDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (d *VMDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a VM by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"cpus": schema.Int64Attribute{
				Computed: true,
			},
			"memory_mb": schema.Int64Attribute{
				Computed: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"disk_path": schema.StringAttribute{
				Computed: true,
			},
			"boot_disk_gb": schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (d *VMDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *VMDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VMDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() && data.Name.IsNull() {
		resp.Diagnostics.AddError("Missing Filter", "At least one of 'id' or 'name' must be set.")
		return
	}

	vms, err := d.client.ListVMs()
	if err != nil {
		resp.Diagnostics.AddError("Error Listing VMs", err.Error())
		return
	}

	var found *VM
	for i := range vms {
		vm := &vms[i]
		if !data.ID.IsNull() && vm.ID == data.ID.ValueString() {
			found = vm
			break
		}
		if !data.Name.IsNull() && vm.Name == data.Name.ValueString() {
			found = vm
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError("VM Not Found", fmt.Sprintf("No VM matched the given filter (id=%q, name=%q).", data.ID.ValueString(), data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(found.ID)
	data.Name = types.StringValue(found.Name)
	data.CPUs = types.Int64Value(int64(found.CPUs))
	data.MemoryMB = types.Int64Value(int64(found.Memory))
	data.Status = types.StringValue(string(found.Status))
	data.DiskPath = types.StringValue(found.DiskPath)
	data.BootDiskGB = types.Int64Value(int64(found.BootDiskGB))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
