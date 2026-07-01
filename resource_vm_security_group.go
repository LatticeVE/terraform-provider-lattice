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

type VMSecurityGroupResource struct{ client *Client }
type VMSecurityGroupModel struct {
	ID              types.String `tfsdk:"id"`
	VMID            types.String `tfsdk:"vm_id"`
	SecurityGroupID types.String `tfsdk:"security_group_id"`
}

func NewVMSecurityGroupResource() resource.Resource { return &VMSecurityGroupResource{} }
func (r *VMSecurityGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_security_group"
}
func (r *VMSecurityGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{MarkdownDescription: "Attaches a security group to a VM.", Attributes: map[string]schema.Attribute{
		"id":                schema.StringAttribute{Computed: true},
		"vm_id":             schema.StringAttribute{Required: true, PlanModifiers: replace},
		"security_group_id": schema.StringAttribute{Required: true, PlanModifiers: replace},
	}}
}
func (r *VMSecurityGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", fmt.Sprintf("Expected *Client, got %T", req.ProviderData))
	}
}
func (r *VMSecurityGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VMSecurityGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.AssignVMSecurityGroup(data.VMID.ValueString(), data.SecurityGroupID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error Attaching Security Group", err.Error())
		return
	}
	data.ID = types.StringValue(data.VMID.ValueString() + ":" + data.SecurityGroupID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *VMSecurityGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VMSecurityGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	groups, err := r.client.ListVMSecurityGroups(data.VMID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Security Group Attachment", err.Error())
		return
	}
	for _, group := range groups {
		if group.ID == data.SecurityGroupID.ValueString() {
			return
		}
	}
	resp.State.RemoveResource(ctx)
}
func (r *VMSecurityGroupResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}
func (r *VMSecurityGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VMSecurityGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.UnassignVMSecurityGroup(data.VMID.ValueString(), data.SecurityGroupID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Detaching Security Group", err.Error())
	}
}
