package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &VPCDataSource{}
var _ datasource.DataSourceWithConfigure = &VPCDataSource{}

type VPCDataSource struct {
	client *Client
}

type VPCDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	CIDR    types.String `tfsdk:"cidr"`
	Bridge  types.String `tfsdk:"bridge"`
	Gateway types.String `tfsdk:"gateway"`
	Status  types.String `tfsdk:"status"`
}

func NewVPCDataSource() datasource.DataSource {
	return &VPCDataSource{}
}

func (d *VPCDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc"
}

func (d *VPCDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a VPC by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"cidr": schema.StringAttribute{
				Computed: true,
			},
			"bridge": schema.StringAttribute{
				Computed: true,
			},
			"gateway": schema.StringAttribute{
				Computed: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *VPCDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *VPCDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VPCDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() && data.Name.IsNull() {
		resp.Diagnostics.AddError("Missing Filter", "At least one of 'id' or 'name' must be set.")
		return
	}

	vpcs, err := d.client.ListVPCs()
	if err != nil {
		resp.Diagnostics.AddError("Error Listing VPCs", err.Error())
		return
	}

	var found *VPC
	for i := range vpcs {
		vpc := &vpcs[i]
		if !data.ID.IsNull() && vpc.ID == data.ID.ValueString() {
			found = vpc
			break
		}
		if !data.Name.IsNull() && vpc.Name == data.Name.ValueString() {
			found = vpc
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError("VPC Not Found", fmt.Sprintf("No VPC matched the given filter (id=%q, name=%q).", data.ID.ValueString(), data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(found.ID)
	data.Name = types.StringValue(found.Name)
	data.CIDR = types.StringValue(found.CIDR)
	data.Bridge = types.StringValue(found.Bridge)
	data.Gateway = types.StringValue(found.Gateway)
	data.Status = types.StringValue(found.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
