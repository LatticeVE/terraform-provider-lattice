package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SecurityGroupResource{}
var _ resource.ResourceWithConfigure = &SecurityGroupResource{}

type SecurityGroupResource struct {
	client *Client
}

type SecurityGroupModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
	Rules       types.List   `tfsdk:"rules"`
}

type SGRuleModel struct {
	ID        types.String `tfsdk:"id"`
	Direction types.String `tfsdk:"direction"`
	Protocol  types.String `tfsdk:"protocol"`
	PortFrom  types.Int64  `tfsdk:"port_from"`
	PortTo    types.Int64  `tfsdk:"port_to"`
	CIDR      types.String `tfsdk:"cidr"`
	Action    types.String `tfsdk:"action"`
	Priority  types.Int64  `tfsdk:"priority"`
}

var sgRuleAttrTypes = map[string]attr.Type{
	"id":        types.StringType,
	"direction": types.StringType,
	"protocol":  types.StringType,
	"port_from": types.Int64Type,
	"port_to":   types.Int64Type,
	"cidr":      types.StringType,
	"action":    types.StringType,
	"priority":  types.Int64Type,
}

func NewSecurityGroupResource() resource.Resource {
	return &SecurityGroupResource{}
}

func (r *SecurityGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_group"
}

func (r *SecurityGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a security group on LatticeVE.",
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
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"rules": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"direction": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "\"ingress\" or \"egress\".",
						},
						"protocol": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "\"tcp\", \"udp\", \"icmp\", or \"all\".",
						},
						"port_from": schema.Int64Attribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Start of the port range; 0 means all ports.",
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"port_to": schema.Int64Attribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"cidr": schema.StringAttribute{
							Required: true,
						},
						"action": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "\"accept\" or \"drop\".",
						},
						"priority": schema.Int64Attribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
		},
	}
}

func (r *SecurityGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SecurityGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SecurityGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sg, err := r.client.CreateSecurityGroup(data.Name.ValueString(), data.Description.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Security Group", err.Error())
		return
	}

	// Add rules.
	var planRules []SGRuleModel
	resp.Diagnostics.Append(data.Rules.ElementsAs(ctx, &planRules, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createdRules := make([]SGRule, 0, len(planRules))
	for _, rm := range planRules {
		rule := SGRule{
			Direction: rm.Direction.ValueString(),
			Protocol:  rm.Protocol.ValueString(),
			PortFrom:  int(rm.PortFrom.ValueInt64()),
			PortTo:    int(rm.PortTo.ValueInt64()),
			CIDR:      rm.CIDR.ValueString(),
			Action:    rm.Action.ValueString(),
			Priority:  int(rm.Priority.ValueInt64()),
		}
		created, addErr := r.client.AddSGRule(sg.ID, rule)
		if addErr != nil {
			resp.Diagnostics.AddError("Error Adding Security Group Rule", addErr.Error())
			return
		}
		createdRules = append(createdRules, *created)
	}
	sg.Rules = createdRules

	resp.Diagnostics.Append(sgToModel(ctx, sg, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecurityGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SecurityGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sg, err := r.client.GetSecurityGroup(data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Security Group", err.Error())
		return
	}

	resp.Diagnostics.Append(sgToModel(ctx, sg, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecurityGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan SecurityGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read current state from API.
	sg, err := r.client.GetSecurityGroup(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Security Group", err.Error())
		return
	}

	// Build a set of existing rule IDs.
	existingByID := make(map[string]struct{}, len(sg.Rules))
	for _, r := range sg.Rules {
		existingByID[r.ID] = struct{}{}
	}

	// Desired rules from plan.
	var planRules []SGRuleModel
	resp.Diagnostics.Append(plan.Rules.ElementsAs(ctx, &planRules, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build set of desired rule IDs (non-empty means it already exists in state).
	desiredIDs := make(map[string]struct{})
	for _, rm := range planRules {
		if !rm.ID.IsNull() && !rm.ID.IsUnknown() && rm.ID.ValueString() != "" {
			desiredIDs[rm.ID.ValueString()] = struct{}{}
		}
	}

	// Remove rules that are in current state but not in plan.
	for id := range existingByID {
		if _, keep := desiredIDs[id]; !keep {
			if removeErr := r.client.RemoveSGRule(sg.ID, id); removeErr != nil {
				resp.Diagnostics.AddError("Error Removing Security Group Rule", removeErr.Error())
				return
			}
		}
	}

	// Add rules that have no ID yet (new rules).
	for _, rm := range planRules {
		if rm.ID.IsNull() || rm.ID.IsUnknown() || rm.ID.ValueString() == "" {
			rule := SGRule{
				Direction: rm.Direction.ValueString(),
				Protocol:  rm.Protocol.ValueString(),
				PortFrom:  int(rm.PortFrom.ValueInt64()),
				PortTo:    int(rm.PortTo.ValueInt64()),
				CIDR:      rm.CIDR.ValueString(),
				Action:    rm.Action.ValueString(),
				Priority:  int(rm.Priority.ValueInt64()),
			}
			if _, addErr := r.client.AddSGRule(sg.ID, rule); addErr != nil {
				resp.Diagnostics.AddError("Error Adding Security Group Rule", addErr.Error())
				return
			}
		}
	}

	// Re-read to get final state.
	sg, err = r.client.GetSecurityGroup(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Security Group After Update", err.Error())
		return
	}

	resp.Diagnostics.Append(sgToModel(ctx, sg, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SecurityGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SecurityGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteSecurityGroup(data.ID.ValueString()); err != nil {
		if !isNotFound(err) {
			resp.Diagnostics.AddError("Error Deleting Security Group", err.Error())
		}
	}
}

func sgToModel(ctx context.Context, sg *SecurityGroup, data *SecurityGroupModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(sg.ID)
	data.Name = types.StringValue(sg.Name)
	data.Description = types.StringValue(sg.Description)
	data.CreatedAt = types.StringValue(sg.CreatedAt)

	ruleVals := make([]attr.Value, 0, len(sg.Rules))
	for _, rule := range sg.Rules {
		obj, d := types.ObjectValue(sgRuleAttrTypes, map[string]attr.Value{
			"id":        types.StringValue(rule.ID),
			"direction": types.StringValue(rule.Direction),
			"protocol":  types.StringValue(rule.Protocol),
			"port_from": types.Int64Value(int64(rule.PortFrom)),
			"port_to":   types.Int64Value(int64(rule.PortTo)),
			"cidr":      types.StringValue(rule.CIDR),
			"action":    types.StringValue(rule.Action),
			"priority":  types.Int64Value(int64(rule.Priority)),
		})
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		ruleVals = append(ruleVals, obj)
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: sgRuleAttrTypes}, ruleVals)
	diags.Append(d...)
	if !diags.HasError() {
		data.Rules = listVal
	}

	return diags
}
