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

type IPAMLeaseResource struct{ client *Client }
type IPAMLeaseModel struct {
	ID        types.String `tfsdk:"id"`
	PoolID    types.String `tfsdk:"pool_id"`
	MAC       types.String `tfsdk:"mac"`
	IP        types.String `tfsdk:"ip"`
	Hostname  types.String `tfsdk:"hostname"`
	VMID      types.String `tfsdk:"vm_id"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func NewIPAMLeaseResource() resource.Resource { return &IPAMLeaseResource{} }
func (r *IPAMLeaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ipam_lease"
}
func (r *IPAMLeaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{MarkdownDescription: "Manages a static DHCP lease in an IPAM pool.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "pool_id": schema.StringAttribute{Required: true, PlanModifiers: replace},
		"mac": schema.StringAttribute{Required: true, PlanModifiers: replace}, "ip": schema.StringAttribute{Required: true, PlanModifiers: replace},
		"hostname": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replace}, "vm_id": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replace},
		"created_at": schema.StringAttribute{Computed: true},
	}}
}
func (r *IPAMLeaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", fmt.Sprintf("Expected *Client, got %T", req.ProviderData))
	}
}
func (r *IPAMLeaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var d IPAMLeaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateIPAMLease(d.PoolID.ValueString(), IPAMLease{PoolID: d.PoolID.ValueString(), MAC: d.MAC.ValueString(), IP: d.IP.ValueString(), Hostname: d.Hostname.ValueString(), VMID: d.VMID.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error Creating IPAM Lease", err.Error())
		return
	}
	ipamLeaseToState(*created, &d)
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
}
func (r *IPAMLeaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var d IPAMLeaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	leases, err := r.client.ListIPAMLeases(d.PoolID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading IPAM Lease", err.Error())
		return
	}
	for _, l := range leases {
		if l.ID == d.ID.ValueString() {
			ipamLeaseToState(l, &d)
			resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}
func (r *IPAMLeaseResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}
func (r *IPAMLeaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var d IPAMLeaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteIPAMLease(d.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting IPAM Lease", err.Error())
	}
}
func ipamLeaseToState(l IPAMLease, d *IPAMLeaseModel) {
	d.ID = types.StringValue(l.ID)
	d.PoolID = types.StringValue(l.PoolID)
	d.MAC = types.StringValue(l.MAC)
	d.IP = types.StringValue(l.IP)
	d.Hostname = types.StringValue(l.Hostname)
	d.VMID = types.StringValue(l.VMID)
	d.CreatedAt = types.StringValue(l.CreatedAt)
}
