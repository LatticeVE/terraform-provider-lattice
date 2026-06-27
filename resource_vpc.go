package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &VPCResource{}
var _ resource.ResourceWithConfigure = &VPCResource{}

type VPCResource struct {
	client *Client
}

type VPCResourceModel struct {
	ID            types.String        `tfsdk:"id"`
	Name          types.String        `tfsdk:"name"`
	CIDR          types.String        `tfsdk:"cidr"`
	CIDRV6        types.String        `tfsdk:"cidr_v6"`
	Bridge        types.String        `tfsdk:"bridge"`
	Gateway       types.String        `tfsdk:"gateway"`
	GatewayV6     types.String        `tfsdk:"gateway_v6"`
	Status        types.String        `tfsdk:"status"`
	DefaultAction types.String        `tfsdk:"default_action"`
	PortForwards  []PortForwardModel  `tfsdk:"port_forwards"`
	FirewallRules []FirewallRuleModel `tfsdk:"firewall_rules"`
}

type PortForwardModel struct {
	ID       types.String `tfsdk:"id"`
	Proto    types.String `tfsdk:"proto"`
	ExtPort  types.Int64  `tfsdk:"ext_port"`
	DestIP   types.String `tfsdk:"dest_ip"`
	DestPort types.Int64  `tfsdk:"dest_port"`
	Desc     types.String `tfsdk:"desc"`
}

type FirewallRuleModel struct {
	ID        types.String `tfsdk:"id"`
	Direction types.String `tfsdk:"direction"`
	Proto     types.String `tfsdk:"proto"`
	Port      types.String `tfsdk:"port"`
	CIDR      types.String `tfsdk:"cidr"`
	Action    types.String `tfsdk:"action"`
	Desc      types.String `tfsdk:"desc"`
}

func NewVPCResource() resource.Resource {
	return &VPCResource{}
}

func (r *VPCResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc"
}

func (r *VPCResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a LatticeVE VPC (Virtual Private Cloud) network.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique UUID of the VPC.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the VPC.",
				Required:            true,
			},
			"cidr": schema.StringAttribute{
				MarkdownDescription: "The IPv4 CIDR block for the VPC (e.g. `10.100.1.0/24`).",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cidr_v6": schema.StringAttribute{
				MarkdownDescription: "The IPv6 CIDR block for the VPC.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bridge": schema.StringAttribute{
				MarkdownDescription: "The Linux bridge name associated with this VPC.",
				Computed:            true,
			},
			"gateway": schema.StringAttribute{
				MarkdownDescription: "The IPv4 gateway address of the VPC.",
				Computed:            true,
			},
			"gateway_v6": schema.StringAttribute{
				MarkdownDescription: "The IPv6 gateway address of the VPC.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the VPC.",
				Computed:            true,
			},
			"default_action": schema.StringAttribute{
				MarkdownDescription: "The default firewall action: `accept` or `drop`. Defaults to `accept`.",
				Optional:            true,
				Computed:            true,
			},
			"port_forwards": schema.ListNestedAttribute{
				MarkdownDescription: "Port forwarding rules for the VPC.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The unique ID of the port forward rule.",
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"proto": schema.StringAttribute{
							MarkdownDescription: "The protocol: `tcp` or `udp`.",
							Required:            true,
						},
						"ext_port": schema.Int64Attribute{
							MarkdownDescription: "The external port to forward from.",
							Required:            true,
						},
						"dest_ip": schema.StringAttribute{
							MarkdownDescription: "The destination IP address to forward to.",
							Required:            true,
						},
						"dest_port": schema.Int64Attribute{
							MarkdownDescription: "The destination port to forward to.",
							Required:            true,
						},
						"desc": schema.StringAttribute{
							MarkdownDescription: "An optional description for this port forward rule.",
							Optional:            true,
							Computed:            true,
						},
					},
				},
			},
			"firewall_rules": schema.ListNestedAttribute{
				MarkdownDescription: "Firewall rules for the VPC.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The unique ID of the firewall rule.",
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"direction": schema.StringAttribute{
							MarkdownDescription: "The traffic direction: `ingress`, `egress`, or `both`.",
							Required:            true,
						},
						"proto": schema.StringAttribute{
							MarkdownDescription: "The protocol: `tcp`, `udp`, `icmp`, or `all`.",
							Required:            true,
						},
						"port": schema.StringAttribute{
							MarkdownDescription: "The port or port range (e.g. `80` or `8080-8090`). Empty string matches all ports.",
							Optional:            true,
							Computed:            true,
						},
						"cidr": schema.StringAttribute{
							MarkdownDescription: "The CIDR to match (e.g. `0.0.0.0/0` for all traffic).",
							Required:            true,
						},
						"action": schema.StringAttribute{
							MarkdownDescription: "The action to take: `accept` or `drop`.",
							Required:            true,
						},
						"desc": schema.StringAttribute{
							MarkdownDescription: "An optional description for this firewall rule.",
							Optional:            true,
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (r *VPCResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *VPCResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VPCResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cidr := ""
	if !plan.CIDR.IsNull() && !plan.CIDR.IsUnknown() {
		cidr = plan.CIDR.ValueString()
	}

	cidr6 := ""
	if !plan.CIDRV6.IsNull() && !plan.CIDRV6.IsUnknown() {
		cidr6 = plan.CIDRV6.ValueString()
	}

	vpc, err := r.client.CreateVPC(plan.Name.ValueString(), cidr, cidr6)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating VPC", err.Error())
		return
	}

	for _, pfModel := range plan.PortForwards {
		pf := PortForward{
			Proto:    pfModel.Proto.ValueString(),
			ExtPort:  int(pfModel.ExtPort.ValueInt64()),
			DestIP:   pfModel.DestIP.ValueString(),
			DestPort: int(pfModel.DestPort.ValueInt64()),
		}
		if !pfModel.Desc.IsNull() && !pfModel.Desc.IsUnknown() {
			pf.Desc = pfModel.Desc.ValueString()
		}
		created, err := r.client.AddPortForward(vpc.ID, pf)
		if err != nil {
			resp.Diagnostics.AddError("Error Adding Port Forward", err.Error())
			return
		}
		pfModel.ID = types.StringValue(created.ID)
		pfModel.Desc = types.StringValue(created.Desc)
	}

	for _, ruleModel := range plan.FirewallRules {
		rule := FirewallRule{
			Direction: ruleModel.Direction.ValueString(),
			Proto:     ruleModel.Proto.ValueString(),
			CIDR:      ruleModel.CIDR.ValueString(),
			Action:    ruleModel.Action.ValueString(),
		}
		if !ruleModel.Port.IsNull() && !ruleModel.Port.IsUnknown() {
			rule.Port = ruleModel.Port.ValueString()
		}
		if !ruleModel.Desc.IsNull() && !ruleModel.Desc.IsUnknown() {
			rule.Desc = ruleModel.Desc.ValueString()
		}
		created, err := r.client.AddFirewallRule(vpc.ID, rule)
		if err != nil {
			resp.Diagnostics.AddError("Error Adding Firewall Rule", err.Error())
			return
		}
		ruleModel.ID = types.StringValue(created.ID)
		ruleModel.Port = types.StringValue(created.Port)
		ruleModel.Desc = types.StringValue(created.Desc)
	}

	defaultAction := "accept"
	if !plan.DefaultAction.IsNull() && !plan.DefaultAction.IsUnknown() {
		defaultAction = plan.DefaultAction.ValueString()
	}
	if defaultAction != "accept" {
		if err := r.client.SetFirewallDefault(vpc.ID, defaultAction); err != nil {
			resp.Diagnostics.AddError("Error Setting Firewall Default", err.Error())
			return
		}
	}

	// Re-read to pick up all computed fields including updated port forward and firewall rule IDs.
	vpc, err = r.client.GetVPC(vpc.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading VPC After Create", err.Error())
		return
	}

	plan.ID = types.StringValue(vpc.ID)
	plan.Bridge = types.StringValue(vpc.Bridge)
	plan.Gateway = types.StringValue(vpc.Gateway)
	plan.GatewayV6 = types.StringValue(vpc.GatewayV6)
	plan.Status = types.StringValue(vpc.Status)
	plan.DefaultAction = types.StringValue(vpc.DefaultAction)

	if vpc.CIDR != "" {
		plan.CIDR = types.StringValue(vpc.CIDR)
	} else {
		plan.CIDR = types.StringNull()
	}
	if vpc.CIDR6 != "" {
		plan.CIDRV6 = types.StringValue(vpc.CIDR6)
	} else {
		plan.CIDRV6 = types.StringNull()
	}

	plan.PortForwards = make([]PortForwardModel, len(vpc.PortForwards))
	for i, pf := range vpc.PortForwards {
		plan.PortForwards[i] = PortForwardModel{
			ID:       types.StringValue(pf.ID),
			Proto:    types.StringValue(pf.Proto),
			ExtPort:  types.Int64Value(int64(pf.ExtPort)),
			DestIP:   types.StringValue(pf.DestIP),
			DestPort: types.Int64Value(int64(pf.DestPort)),
			Desc:     types.StringValue(pf.Desc),
		}
	}

	plan.FirewallRules = make([]FirewallRuleModel, len(vpc.FirewallRules))
	for i, rule := range vpc.FirewallRules {
		plan.FirewallRules[i] = FirewallRuleModel{
			ID:        types.StringValue(rule.ID),
			Direction: types.StringValue(rule.Direction),
			Proto:     types.StringValue(rule.Proto),
			Port:      types.StringValue(rule.Port),
			CIDR:      types.StringValue(rule.CIDR),
			Action:    types.StringValue(rule.Action),
			Desc:      types.StringValue(rule.Desc),
		}
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *VPCResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VPCResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpc, err := r.client.GetVPC(state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading VPC", err.Error())
		return
	}

	state.Name = types.StringValue(vpc.Name)
	state.Bridge = types.StringValue(vpc.Bridge)
	state.Gateway = types.StringValue(vpc.Gateway)
	state.GatewayV6 = types.StringValue(vpc.GatewayV6)
	state.Status = types.StringValue(vpc.Status)
	state.DefaultAction = types.StringValue(vpc.DefaultAction)

	if vpc.CIDR != "" {
		state.CIDR = types.StringValue(vpc.CIDR)
	} else {
		state.CIDR = types.StringNull()
	}
	if vpc.CIDR6 != "" {
		state.CIDRV6 = types.StringValue(vpc.CIDR6)
	} else {
		state.CIDRV6 = types.StringNull()
	}

	state.PortForwards = make([]PortForwardModel, len(vpc.PortForwards))
	for i, pf := range vpc.PortForwards {
		state.PortForwards[i] = PortForwardModel{
			ID:       types.StringValue(pf.ID),
			Proto:    types.StringValue(pf.Proto),
			ExtPort:  types.Int64Value(int64(pf.ExtPort)),
			DestIP:   types.StringValue(pf.DestIP),
			DestPort: types.Int64Value(int64(pf.DestPort)),
			Desc:     types.StringValue(pf.Desc),
		}
	}

	state.FirewallRules = make([]FirewallRuleModel, len(vpc.FirewallRules))
	for i, rule := range vpc.FirewallRules {
		state.FirewallRules[i] = FirewallRuleModel{
			ID:        types.StringValue(rule.ID),
			Direction: types.StringValue(rule.Direction),
			Proto:     types.StringValue(rule.Proto),
			Port:      types.StringValue(rule.Port),
			CIDR:      types.StringValue(rule.CIDR),
			Action:    types.StringValue(rule.Action),
			Desc:      types.StringValue(rule.Desc),
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *VPCResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state VPCResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	if !plan.Name.Equal(state.Name) {
		_, err := r.client.UpdateVPC(id, plan.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error Updating VPC", err.Error())
			return
		}
	}

	// Reconcile port forwards: build a set of IDs currently in state.
	statePortForwardIDs := make(map[string]struct{}, len(state.PortForwards))
	for _, pf := range state.PortForwards {
		if !pf.ID.IsNull() && !pf.ID.IsUnknown() && pf.ID.ValueString() != "" {
			statePortForwardIDs[pf.ID.ValueString()] = struct{}{}
		}
	}

	// Build a set of IDs present in the plan (non-empty means it already exists in state).
	planPortForwardIDs := make(map[string]struct{}, len(plan.PortForwards))
	for _, pf := range plan.PortForwards {
		if !pf.ID.IsNull() && !pf.ID.IsUnknown() && pf.ID.ValueString() != "" {
			planPortForwardIDs[pf.ID.ValueString()] = struct{}{}
		}
	}

	// Remove port forwards that are in state but not in plan.
	for pfID := range statePortForwardIDs {
		if _, keep := planPortForwardIDs[pfID]; !keep {
			if err := r.client.RemovePortForward(id, pfID); err != nil {
				resp.Diagnostics.AddError("Error Removing Port Forward", err.Error())
				return
			}
		}
	}

	// Add port forwards that are in plan but have no ID (new entries).
	for i, pfModel := range plan.PortForwards {
		if !pfModel.ID.IsNull() && !pfModel.ID.IsUnknown() && pfModel.ID.ValueString() != "" {
			continue
		}
		pf := PortForward{
			Proto:    pfModel.Proto.ValueString(),
			ExtPort:  int(pfModel.ExtPort.ValueInt64()),
			DestIP:   pfModel.DestIP.ValueString(),
			DestPort: int(pfModel.DestPort.ValueInt64()),
		}
		if !pfModel.Desc.IsNull() && !pfModel.Desc.IsUnknown() {
			pf.Desc = pfModel.Desc.ValueString()
		}
		created, err := r.client.AddPortForward(id, pf)
		if err != nil {
			resp.Diagnostics.AddError("Error Adding Port Forward", err.Error())
			return
		}
		plan.PortForwards[i].ID = types.StringValue(created.ID)
		plan.PortForwards[i].Desc = types.StringValue(created.Desc)
	}

	// Reconcile firewall rules similarly.
	stateFirewallRuleIDs := make(map[string]struct{}, len(state.FirewallRules))
	for _, rule := range state.FirewallRules {
		if !rule.ID.IsNull() && !rule.ID.IsUnknown() && rule.ID.ValueString() != "" {
			stateFirewallRuleIDs[rule.ID.ValueString()] = struct{}{}
		}
	}

	planFirewallRuleIDs := make(map[string]struct{}, len(plan.FirewallRules))
	for _, rule := range plan.FirewallRules {
		if !rule.ID.IsNull() && !rule.ID.IsUnknown() && rule.ID.ValueString() != "" {
			planFirewallRuleIDs[rule.ID.ValueString()] = struct{}{}
		}
	}

	// Remove firewall rules in state but not in plan.
	for ruleID := range stateFirewallRuleIDs {
		if _, keep := planFirewallRuleIDs[ruleID]; !keep {
			if err := r.client.RemoveFirewallRule(id, ruleID); err != nil {
				resp.Diagnostics.AddError("Error Removing Firewall Rule", err.Error())
				return
			}
		}
	}

	// Add firewall rules in plan that have no ID yet.
	for i, ruleModel := range plan.FirewallRules {
		if !ruleModel.ID.IsNull() && !ruleModel.ID.IsUnknown() && ruleModel.ID.ValueString() != "" {
			continue
		}
		rule := FirewallRule{
			Direction: ruleModel.Direction.ValueString(),
			Proto:     ruleModel.Proto.ValueString(),
			CIDR:      ruleModel.CIDR.ValueString(),
			Action:    ruleModel.Action.ValueString(),
		}
		if !ruleModel.Port.IsNull() && !ruleModel.Port.IsUnknown() {
			rule.Port = ruleModel.Port.ValueString()
		}
		if !ruleModel.Desc.IsNull() && !ruleModel.Desc.IsUnknown() {
			rule.Desc = ruleModel.Desc.ValueString()
		}
		created, err := r.client.AddFirewallRule(id, rule)
		if err != nil {
			resp.Diagnostics.AddError("Error Adding Firewall Rule", err.Error())
			return
		}
		plan.FirewallRules[i].ID = types.StringValue(created.ID)
		plan.FirewallRules[i].Port = types.StringValue(created.Port)
		plan.FirewallRules[i].Desc = types.StringValue(created.Desc)
	}

	// Update default firewall action if changed.
	if !plan.DefaultAction.Equal(state.DefaultAction) {
		defaultAction := plan.DefaultAction.ValueString()
		if defaultAction == "" {
			defaultAction = "accept"
		}
		if err := r.client.SetFirewallDefault(id, defaultAction); err != nil {
			resp.Diagnostics.AddError("Error Setting Firewall Default", err.Error())
			return
		}
	}

	// Re-read to sync all computed fields.
	vpc, err := r.client.GetVPC(id)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading VPC After Update", err.Error())
		return
	}

	plan.ID = types.StringValue(vpc.ID)
	plan.Bridge = types.StringValue(vpc.Bridge)
	plan.Gateway = types.StringValue(vpc.Gateway)
	plan.GatewayV6 = types.StringValue(vpc.GatewayV6)
	plan.Status = types.StringValue(vpc.Status)
	plan.DefaultAction = types.StringValue(vpc.DefaultAction)

	if vpc.CIDR != "" {
		plan.CIDR = types.StringValue(vpc.CIDR)
	} else {
		plan.CIDR = types.StringNull()
	}
	if vpc.CIDR6 != "" {
		plan.CIDRV6 = types.StringValue(vpc.CIDR6)
	} else {
		plan.CIDRV6 = types.StringNull()
	}

	plan.PortForwards = make([]PortForwardModel, len(vpc.PortForwards))
	for i, pf := range vpc.PortForwards {
		plan.PortForwards[i] = PortForwardModel{
			ID:       types.StringValue(pf.ID),
			Proto:    types.StringValue(pf.Proto),
			ExtPort:  types.Int64Value(int64(pf.ExtPort)),
			DestIP:   types.StringValue(pf.DestIP),
			DestPort: types.Int64Value(int64(pf.DestPort)),
			Desc:     types.StringValue(pf.Desc),
		}
	}

	plan.FirewallRules = make([]FirewallRuleModel, len(vpc.FirewallRules))
	for i, rule := range vpc.FirewallRules {
		plan.FirewallRules[i] = FirewallRuleModel{
			ID:        types.StringValue(rule.ID),
			Direction: types.StringValue(rule.Direction),
			Proto:     types.StringValue(rule.Proto),
			Port:      types.StringValue(rule.Port),
			CIDR:      types.StringValue(rule.CIDR),
			Action:    types.StringValue(rule.Action),
			Desc:      types.StringValue(rule.Desc),
		}
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *VPCResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VPCResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteVPC(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error Deleting VPC", err.Error())
		return
	}
}
