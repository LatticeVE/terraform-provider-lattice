package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &VPCLoadBalancerResource{}
var _ resource.ResourceWithConfigure = &VPCLoadBalancerResource{}

type VPCLoadBalancerResource struct {
	client *Client
}

type VPCLoadBalancerResourceModel struct {
	ID              types.String     `tfsdk:"id"`
	VPCID           types.String     `tfsdk:"vpc_id"`
	Name            types.String     `tfsdk:"name"`
	Port            types.Int64      `tfsdk:"port"`
	Protocol        types.String     `tfsdk:"protocol"`
	CertificateID   types.String     `tfsdk:"certificate_id"`
	BackendProtocol types.String     `tfsdk:"backend_protocol"`
	Backends        []LBBackendModel `tfsdk:"backends"`
}

type LBBackendModel struct {
	ID      types.String `tfsdk:"id"`
	Address types.String `tfsdk:"address"`
	Weight  types.Int64  `tfsdk:"weight"`
}

func NewVPCLoadBalancerResource() resource.Resource {
	return &VPCLoadBalancerResource{}
}

func (r *VPCLoadBalancerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc_load_balancer"
}

func (r *VPCLoadBalancerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceString := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	replaceInt := []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a LatticeVE VPC load balancer.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vpc_id": schema.StringAttribute{
				MarkdownDescription: "VPC ID where the load balancer should be created.",
				Required:            true,
				PlanModifiers:       replaceString,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Load balancer name.",
				Required:            true,
				PlanModifiers:       replaceString,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "Frontend listen port.",
				Required:            true,
				PlanModifiers:       replaceInt,
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "Frontend protocol: `tcp`, `http`, or `https`.",
				Required:            true,
				PlanModifiers:       replaceString,
			},
			"certificate_id": schema.StringAttribute{
				MarkdownDescription: "Required when `protocol` is `https`; references `lattice_lb_certificate.id`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       replaceString,
			},
			"backend_protocol": schema.StringAttribute{
				MarkdownDescription: "Backend protocol: `tcp` or `http`. Defaults server-side from the frontend protocol.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       replaceString,
			},
			"backends": schema.ListNestedAttribute{
				MarkdownDescription: "Backend targets as `ip:port` addresses.",
				Required:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"address": schema.StringAttribute{
							MarkdownDescription: "Backend address in `ip:port` form.",
							Required:            true,
						},
						"weight": schema.Int64Attribute{
							MarkdownDescription: "Backend weight. Defaults to 1.",
							Optional:            true,
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (r *VPCLoadBalancerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VPCLoadBalancerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VPCLoadBalancerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lb, err := r.client.AddLoadBalancer(plan.VPCID.ValueString(), loadBalancerFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating VPC Load Balancer", err.Error())
		return
	}

	loadBalancerToModel(lb, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VPCLoadBalancerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VPCLoadBalancerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lb, err := r.client.GetVPCLoadBalancer(state.VPCID.ValueString(), state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading VPC Load Balancer", err.Error())
		return
	}

	loadBalancerToModel(lb, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VPCLoadBalancerResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"VPC Load Balancer Requires Replacement",
		"LatticeVE does not expose an update endpoint for VPC load balancers yet. Terraform should plan replacement for changes to this resource.",
	)
}

func (r *VPCLoadBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VPCLoadBalancerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.RemoveLoadBalancer(state.VPCID.ValueString(), state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting VPC Load Balancer", err.Error())
	}
}

func loadBalancerFromModel(m VPCLoadBalancerResourceModel) LoadBalancer {
	lb := LoadBalancer{
		Name:     m.Name.ValueString(),
		Port:     int(m.Port.ValueInt64()),
		Protocol: m.Protocol.ValueString(),
		Backends: make([]LBBackend, len(m.Backends)),
	}
	if !m.CertificateID.IsNull() && !m.CertificateID.IsUnknown() {
		lb.CertificateID = m.CertificateID.ValueString()
	}
	if !m.BackendProtocol.IsNull() && !m.BackendProtocol.IsUnknown() {
		lb.BackendProtocol = m.BackendProtocol.ValueString()
	}
	for i, be := range m.Backends {
		lb.Backends[i] = LBBackend{
			Address: be.Address.ValueString(),
			Weight:  1,
		}
		if !be.Weight.IsNull() && !be.Weight.IsUnknown() {
			lb.Backends[i].Weight = int(be.Weight.ValueInt64())
		}
	}
	return lb
}

func loadBalancerToModel(lb *LoadBalancer, m *VPCLoadBalancerResourceModel) {
	m.ID = types.StringValue(lb.ID)
	m.Name = types.StringValue(lb.Name)
	m.Port = types.Int64Value(int64(lb.Port))
	m.Protocol = types.StringValue(lb.Protocol)
	m.CertificateID = types.StringValue(lb.CertificateID)
	m.BackendProtocol = types.StringValue(lb.BackendProtocol)
	m.Backends = make([]LBBackendModel, len(lb.Backends))
	for i, be := range lb.Backends {
		m.Backends[i] = LBBackendModel{
			ID:      types.StringValue(be.ID),
			Address: types.StringValue(be.Address),
			Weight:  types.Int64Value(int64(be.Weight)),
		}
	}
}
