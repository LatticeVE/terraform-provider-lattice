package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &IPAMPoolResource{}
var _ resource.ResourceWithConfigure = &IPAMPoolResource{}

type IPAMPoolResource struct {
	client *Client
}

type IPAMPoolModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Bridge     types.String `tfsdk:"bridge"`
	Subnet     types.String `tfsdk:"subnet"`
	Gateway    types.String `tfsdk:"gateway"`
	RangeStart types.String `tfsdk:"range_start"`
	RangeEnd   types.String `tfsdk:"range_end"`
	DNS        types.List   `tfsdk:"dns"`
	CreatedAt  types.String `tfsdk:"created_at"`
}

func NewIPAMPoolResource() resource.Resource {
	return &IPAMPoolResource{}
}

func (r *IPAMPoolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ipam_pool"
}

func (r *IPAMPoolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an IPAM pool on LatticeVE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"bridge": schema.StringAttribute{
				Required: true,
			},
			"subnet": schema.StringAttribute{
				Required: true,
			},
			"gateway": schema.StringAttribute{
				Required: true,
			},
			"range_start": schema.StringAttribute{
				Required: true,
			},
			"range_end": schema.StringAttribute{
				Required: true,
			},
			"dns": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *IPAMPoolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", fmt.Sprintf("Expected *Client, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *IPAMPoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IPAMPoolModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var dnsSlice []string
	if !data.DNS.IsNull() && !data.DNS.IsUnknown() {
		resp.Diagnostics.Append(data.DNS.ElementsAs(ctx, &dnsSlice, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	pool, err := r.client.CreateIPAMPool(IPAMPool{
		Name:       data.Name.ValueString(),
		Bridge:     data.Bridge.ValueString(),
		Subnet:     data.Subnet.ValueString(),
		Gateway:    data.Gateway.ValueString(),
		RangeStart: data.RangeStart.ValueString(),
		RangeEnd:   data.RangeEnd.ValueString(),
		DNS:        dnsSlice,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Creating IPAM Pool", err.Error())
		return
	}

	resp.Diagnostics.Append(ipamPoolToModel(ctx, pool, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IPAMPoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IPAMPoolModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pool, err := r.client.GetIPAMPool(data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading IPAM Pool", err.Error())
		return
	}

	resp.Diagnostics.Append(ipamPoolToModel(ctx, pool, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IPAMPoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IPAMPoolModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var dns []string
	resp.Diagnostics.Append(data.DNS.ElementsAs(ctx, &dns, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pool := IPAMPool{ID: data.ID.ValueString(), Name: data.Name.ValueString(), Bridge: data.Bridge.ValueString(), Subnet: data.Subnet.ValueString(), Gateway: data.Gateway.ValueString(), RangeStart: data.RangeStart.ValueString(), RangeEnd: data.RangeEnd.ValueString(), DNS: dns}
	if err := r.client.UpdateIPAMPool(pool.ID, pool); err != nil {
		resp.Diagnostics.AddError("Error Updating IPAM Pool", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IPAMPoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IPAMPoolModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteIPAMPool(data.ID.ValueString()); err != nil {
		if !isNotFound(err) {
			resp.Diagnostics.AddError("Error Deleting IPAM Pool", err.Error())
		}
	}
}

func ipamPoolToModel(ctx context.Context, pool *IPAMPool, data *IPAMPoolModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(pool.ID)
	data.Name = types.StringValue(pool.Name)
	data.Bridge = types.StringValue(pool.Bridge)
	data.Subnet = types.StringValue(pool.Subnet)
	data.Gateway = types.StringValue(pool.Gateway)
	data.RangeStart = types.StringValue(pool.RangeStart)
	data.RangeEnd = types.StringValue(pool.RangeEnd)
	data.CreatedAt = types.StringValue(pool.CreatedAt)

	dnsList, d := types.ListValueFrom(ctx, types.StringType, pool.DNS)
	diags.Append(d...)
	if !diags.HasError() {
		data.DNS = dnsList
	}

	return diags
}
