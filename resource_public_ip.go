package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &PublicIPResource{}
var _ resource.ResourceWithConfigure = &PublicIPResource{}

type PublicIPResource struct {
	client *Client
}

type PublicIPModel struct {
	ID          types.String `tfsdk:"id"`
	PoolID      types.String `tfsdk:"pool_id"`
	Description types.String `tfsdk:"description"`
	IP          types.String `tfsdk:"ip"`
	PrivateIP   types.String `tfsdk:"private_ip"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func NewPublicIPResource() resource.Resource {
	return &PublicIPResource{}
}

func (r *PublicIPResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_public_ip"
}

func (r *PublicIPResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a public IP allocation on LatticeVE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pool_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"ip": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The allocated public IP address.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_ip": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "If set, static NAT is enabled mapping this public IP to the given private IP.",
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

func (r *PublicIPResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PublicIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PublicIPModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pip, err := r.client.AllocatePublicIP(data.PoolID.ValueString(), data.Description.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Allocating Public IP", err.Error())
		return
	}

	if !data.PrivateIP.IsNull() && !data.PrivateIP.IsUnknown() && data.PrivateIP.ValueString() != "" {
		pip, err = r.client.EnableStaticNAT(pip.ID, data.PrivateIP.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error Enabling Static NAT", err.Error())
			// Attempt cleanup to avoid orphaned allocation.
			_ = r.client.ReleasePublicIP(pip.ID)
			return
		}
	}

	r.pipToModel(pip, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PublicIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PublicIPModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pip, err := r.client.GetPublicIP(data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Public IP", err.Error())
		return
	}

	r.pipToModel(pip, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PublicIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan PublicIPModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	statePrivateIP := state.PrivateIP.ValueString()
	planPrivateIP := plan.PrivateIP.ValueString()

	var pip *PublicIP
	var err error

	switch {
	case statePrivateIP == "" && planPrivateIP != "":
		pip, err = r.client.EnableStaticNAT(state.ID.ValueString(), planPrivateIP)
		if err != nil {
			resp.Diagnostics.AddError("Error Enabling Static NAT", err.Error())
			return
		}
	case statePrivateIP != "" && planPrivateIP == "":
		if err = r.client.DisableStaticNAT(state.ID.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error Disabling Static NAT", err.Error())
			return
		}
		pip, err = r.client.GetPublicIP(state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error Reading Public IP After NAT Change", err.Error())
			return
		}
	case statePrivateIP != planPrivateIP:
		// Changed from one private IP to another: disable then re-enable.
		if err = r.client.DisableStaticNAT(state.ID.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error Disabling Static NAT", err.Error())
			return
		}
		pip, err = r.client.EnableStaticNAT(state.ID.ValueString(), planPrivateIP)
		if err != nil {
			resp.Diagnostics.AddError("Error Enabling Static NAT", err.Error())
			return
		}
	default:
		pip, err = r.client.GetPublicIP(state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error Reading Public IP", err.Error())
			return
		}
	}

	r.pipToModel(pip, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PublicIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PublicIPModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.ReleasePublicIP(data.ID.ValueString()); err != nil {
		if !isNotFound(err) {
			resp.Diagnostics.AddError("Error Releasing Public IP", err.Error())
		}
	}
}

func (r *PublicIPResource) pipToModel(pip *PublicIP, data *PublicIPModel) {
	data.ID = types.StringValue(pip.ID)
	data.PoolID = types.StringValue(pip.PoolID)
	data.IP = types.StringValue(pip.IP)
	data.Description = types.StringValue(pip.Description)
	data.PrivateIP = types.StringValue(pip.PrivateIP)
	data.CreatedAt = types.StringValue(pip.CreatedAt.Format("2006-01-02T15:04:05Z"))
}
