package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &NodesDataSource{}
var _ datasource.DataSourceWithConfigure = &NodesDataSource{}

type NodesDataSource struct {
	client *Client
}

type NodesDataSourceModel struct {
	Arch  types.String `tfsdk:"arch"`
	Nodes []NodeModel  `tfsdk:"nodes"`
}

type NodeModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Arch          types.String `tfsdk:"arch"`
	Status        types.String `tfsdk:"status"`
	CPUs          types.Int64  `tfsdk:"cpus"`
	MemoryMB      types.Int64  `tfsdk:"memory_mb"`
	MemoryUsedMB  types.Int64  `tfsdk:"memory_used_mb"`
	StorageGB     types.Int64  `tfsdk:"storage_gb"`
	StorageUsedGB types.Int64  `tfsdk:"storage_used_gb"`
}

func NewNodesDataSource() datasource.DataSource {
	return &NodesDataSource{}
}

func (d *NodesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nodes"
}

func (d *NodesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists LatticeVE host nodes. Filter by `arch` to discover nodes suitable for placement of VMs with a specific CPU architecture.",
		Attributes: map[string]schema.Attribute{
			"arch": schema.StringAttribute{
				MarkdownDescription: "Filter nodes by CPU architecture: `amd64` or `arm64`. Omit to return all nodes.",
				Optional:            true,
			},
			"nodes": schema.ListNestedAttribute{
				MarkdownDescription: "The list of nodes matching the filter.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Node UUID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Node hostname.",
							Computed:            true,
						},
						"arch": schema.StringAttribute{
							MarkdownDescription: "CPU architecture: `amd64` or `arm64`.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "Node status: `online`, `offline`, or `maintenance`.",
							Computed:            true,
						},
						"cpus": schema.Int64Attribute{
							MarkdownDescription: "Total logical CPU count on the node.",
							Computed:            true,
						},
						"memory_mb": schema.Int64Attribute{
							MarkdownDescription: "Total RAM on the node in MiB.",
							Computed:            true,
						},
						"memory_used_mb": schema.Int64Attribute{
							MarkdownDescription: "RAM currently in use in MiB.",
							Computed:            true,
						},
						"storage_gb": schema.Int64Attribute{
							MarkdownDescription: "Total local storage in GiB.",
							Computed:            true,
						},
						"storage_used_gb": schema.Int64Attribute{
							MarkdownDescription: "Local storage currently in use in GiB.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *NodesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Client, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *NodesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config NodesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nodes, err := d.client.ListNodes()
	if err != nil {
		resp.Diagnostics.AddError("Error listing nodes", err.Error())
		return
	}

	archFilter := config.Arch.ValueString()

	var matched []NodeModel
	for _, n := range nodes {
		if archFilter != "" && n.Arch != archFilter {
			continue
		}
		matched = append(matched, NodeModel{
			ID:            types.StringValue(n.ID),
			Name:          types.StringValue(n.Name),
			Arch:          types.StringValue(n.Arch),
			Status:        types.StringValue(n.Status),
			CPUs:          types.Int64Value(int64(n.CPUs)),
			MemoryMB:      types.Int64Value(n.MemoryMB),
			MemoryUsedMB:  types.Int64Value(n.MemoryUsedMB),
			StorageGB:     types.Int64Value(n.StorageGB),
			StorageUsedGB: types.Int64Value(n.StorageUsedGB),
		})
	}
	if matched == nil {
		matched = []NodeModel{}
	}

	config.Nodes = matched
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
