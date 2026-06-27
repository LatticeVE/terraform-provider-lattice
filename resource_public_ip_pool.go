package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &PublicIPPoolResource{}
var _ resource.ResourceWithConfigure = &PublicIPPoolResource{}

type PublicIPPoolResource struct {
	client *Client
}

type PublicIPPoolModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Interface types.String `tfsdk:"interface"`
	CIDR      types.String `tfsdk:"cidr"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func NewPublicIPPoolResource() resource.Resource {
	return &PublicIPPoolResource{}
}

func (r *PublicIPPoolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_public_ip_pool"
}

func (r *PublicIPPoolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a public IP pool on LatticeVE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"interface": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Host NIC to bind this pool to, e.g. \"eth0\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cidr": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "CIDR block for this pool, e.g. \"192.168.1.200/27\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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

func (r *PublicIPPoolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PublicIPPoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PublicIPPoolModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pool, err := r.client.CreatePublicIPPool(data.Name.ValueString(), data.Interface.ValueString(), data.CIDR.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Public IP Pool", err.Error())
		return
	}

	data.ID = types.StringValue(pool.ID)
	data.Name = types.StringValue(pool.Name)
	data.Interface = types.StringValue(pool.Interface)
	data.CIDR = types.StringValue(pool.CIDR)
	data.CreatedAt = types.StringValue(pool.CreatedAt.Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PublicIPPoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PublicIPPoolModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pool, err := r.client.GetPublicIPPool(data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Public IP Pool", err.Error())
		return
	}

	data.ID = types.StringValue(pool.ID)
	data.Name = types.StringValue(pool.Name)
	data.Interface = types.StringValue(pool.Interface)
	data.CIDR = types.StringValue(pool.CIDR)
	data.CreatedAt = types.StringValue(pool.CreatedAt.Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PublicIPPoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All fields are RequiresReplace; Update is never called.
	var data PublicIPPoolModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PublicIPPoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PublicIPPoolModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeletePublicIPPool(data.ID.ValueString()); err != nil {
		if !isNotFound(err) {
			resp.Diagnostics.AddError("Error Deleting Public IP Pool", err.Error())
		}
	}
}
