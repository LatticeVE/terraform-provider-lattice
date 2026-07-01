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

type AffinityGroupResource struct{ client *Client }
type AffinityGroupModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Policy    types.String `tfsdk:"policy"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func NewAffinityGroupResource() resource.Resource { return &AffinityGroupResource{} }
func (r *AffinityGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_affinity_group"
}
func (r *AffinityGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{MarkdownDescription: "Manages an affinity or anti-affinity placement group.", Attributes: map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Required: true, PlanModifiers: replace}, "policy": schema.StringAttribute{Required: true, PlanModifiers: replace, MarkdownDescription: "`affinity` or `anti-affinity`."}, "created_at": schema.StringAttribute{Computed: true}}}
}
func (r *AffinityGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", fmt.Sprintf("Expected *Client, got %T", req.ProviderData))
	}
}
func (r *AffinityGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var d AffinityGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	g, err := r.client.CreateAffinityGroup(d.Name.ValueString(), d.Policy.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Affinity Group", err.Error())
		return
	}
	affinityGroupToState(*g, &d)
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
}
func (r *AffinityGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var d AffinityGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	groups, err := r.client.ListAffinityGroups()
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Affinity Group", err.Error())
		return
	}
	for _, g := range groups {
		if g.ID == d.ID.ValueString() {
			affinityGroupToState(g, &d)
			resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}
func (r *AffinityGroupResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}
func (r *AffinityGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var d AffinityGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAffinityGroup(d.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Affinity Group", err.Error())
	}
}
func affinityGroupToState(g AffinityGroup, d *AffinityGroupModel) {
	d.ID = types.StringValue(g.ID)
	d.Name = types.StringValue(g.Name)
	d.Policy = types.StringValue(g.Policy)
	d.CreatedAt = types.StringValue(g.CreatedAt)
}
