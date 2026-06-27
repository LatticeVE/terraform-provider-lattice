package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &PublicIPPoolsDataSource{}
var _ datasource.DataSourceWithConfigure = &PublicIPPoolsDataSource{}

type PublicIPPoolsDataSource struct {
	client *Client
}

type PublicIPPoolsModel struct {
	Pools types.List `tfsdk:"pools"`
}

var publicIPPoolAttrTypes = map[string]attr.Type{
	"id":        types.StringType,
	"name":      types.StringType,
	"interface": types.StringType,
	"cidr":      types.StringType,
}

func NewPublicIPPoolsDataSource() datasource.DataSource {
	return &PublicIPPoolsDataSource{}
}

func (d *PublicIPPoolsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_public_ip_pools"
}

func (d *PublicIPPoolsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all public IP pools.",
		Attributes: map[string]schema.Attribute{
			"pools": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"interface": schema.StringAttribute{
							Computed: true,
						},
						"cidr": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *PublicIPPoolsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PublicIPPoolsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PublicIPPoolsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pools, err := d.client.ListPublicIPPools()
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Public IP Pools", err.Error())
		return
	}

	poolVals := make([]attr.Value, 0, len(pools))
	for _, pool := range pools {
		obj, diags := types.ObjectValue(publicIPPoolAttrTypes, map[string]attr.Value{
			"id":        types.StringValue(pool.ID),
			"name":      types.StringValue(pool.Name),
			"interface": types.StringValue(pool.Interface),
			"cidr":      types.StringValue(pool.CIDR),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		poolVals = append(poolVals, obj)
	}

	listVal, diags := types.ListValue(types.ObjectType{AttrTypes: publicIPPoolAttrTypes}, poolVals)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Pools = listVal
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
