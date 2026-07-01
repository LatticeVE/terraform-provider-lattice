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

type VMAffinityGroupResource struct{ client *Client }
type VMAffinityGroupModel struct {
	ID              types.String `tfsdk:"id"`
	VMID            types.String `tfsdk:"vm_id"`
	AffinityGroupID types.String `tfsdk:"affinity_group_id"`
}

func NewVMAffinityGroupResource() resource.Resource { return &VMAffinityGroupResource{} }
func (r *VMAffinityGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_affinity_group"
}
func (r *VMAffinityGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{MarkdownDescription: "Assigns a VM to an affinity or anti-affinity group.", Attributes: map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "vm_id": schema.StringAttribute{Required: true, PlanModifiers: replace}, "affinity_group_id": schema.StringAttribute{Required: true, PlanModifiers: replace}}}
}
func (r *VMAffinityGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", fmt.Sprintf("Expected *Client, got %T", req.ProviderData))
	}
}
func (r *VMAffinityGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var d VMAffinityGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.AssignVMAffinityGroup(d.VMID.ValueString(), d.AffinityGroupID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error Assigning Affinity Group", err.Error())
		return
	}
	d.ID = types.StringValue(d.VMID.ValueString() + ":" + d.AffinityGroupID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
}
func (r *VMAffinityGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var d VMAffinityGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	groups, err := r.client.ListVMAffinityGroups(d.VMID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Affinity Assignment", err.Error())
		return
	}
	for _, g := range groups {
		if g.ID == d.AffinityGroupID.ValueString() {
			return
		}
	}
	resp.State.RemoveResource(ctx)
}
func (r *VMAffinityGroupResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}
func (r *VMAffinityGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var d VMAffinityGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.UnassignVMAffinityGroup(d.VMID.ValueString(), d.AffinityGroupID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Removing Affinity Assignment", err.Error())
	}
}
