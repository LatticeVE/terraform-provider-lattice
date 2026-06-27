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

var _ resource.Resource = &LBCertificateResource{}
var _ resource.ResourceWithConfigure = &LBCertificateResource{}

type LBCertificateResource struct {
	client *Client
}

type LBCertificateResourceModel struct {
	ID          types.String   `tfsdk:"id"`
	Name        types.String   `tfsdk:"name"`
	Description types.String   `tfsdk:"description"`
	CertPEM     types.String   `tfsdk:"cert_pem"`
	KeyPEM      types.String   `tfsdk:"key_pem"`
	ChainPEM    types.String   `tfsdk:"chain_pem"`
	Subject     types.String   `tfsdk:"subject"`
	DNSNames    []types.String `tfsdk:"dns_names"`
	NotBefore   types.String   `tfsdk:"not_before"`
	NotAfter    types.String   `tfsdk:"not_after"`
	Fingerprint types.String   `tfsdk:"fingerprint"`
	CreatedAt   types.String   `tfsdk:"created_at"`
	UpdatedAt   types.String   `tfsdk:"updated_at"`
}

func NewLBCertificateResource() resource.Resource {
	return &LBCertificateResource{}
}

func (r *LBCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lb_certificate"
}

func (r *LBCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a TLS certificate used by LatticeVE VPC load balancers.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Certificate name.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional certificate description.",
				Optional:            true,
				Computed:            true,
			},
			"cert_pem": schema.StringAttribute{
				MarkdownDescription: "Leaf certificate PEM.",
				Required:            true,
				Sensitive:           true,
			},
			"key_pem": schema.StringAttribute{
				MarkdownDescription: "Private key PEM. LatticeVE encrypts this at rest and never returns it.",
				Required:            true,
				Sensitive:           true,
			},
			"chain_pem": schema.StringAttribute{
				MarkdownDescription: "Optional intermediate certificate chain PEM.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
			},
			"subject": schema.StringAttribute{
				MarkdownDescription: "Certificate subject parsed by LatticeVE.",
				Computed:            true,
			},
			"dns_names": schema.ListAttribute{
				MarkdownDescription: "DNS SANs parsed by LatticeVE.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"not_before": schema.StringAttribute{
				MarkdownDescription: "Certificate validity start time.",
				Computed:            true,
			},
			"not_after": schema.StringAttribute{
				MarkdownDescription: "Certificate validity end time.",
				Computed:            true,
			},
			"fingerprint": schema.StringAttribute{
				MarkdownDescription: "Certificate fingerprint.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *LBCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LBCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LBCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cert, err := r.client.CreateLBCertificate(lbCertificateRequestFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating LB Certificate", err.Error())
		return
	}

	lbCertificateToModel(cert, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LBCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LBCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cert, err := r.client.GetLBCertificate(state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading LB Certificate", err.Error())
		return
	}

	// Keep sensitive write-only material from state; the API intentionally does not return the private key.
	keyPEM := state.KeyPEM
	certPEM := state.CertPEM
	chainPEM := state.ChainPEM
	lbCertificateToModel(cert, &state)
	state.KeyPEM = keyPEM
	if cert.CertPEM == "" {
		state.CertPEM = certPEM
	}
	if cert.ChainPEM == "" {
		state.ChainPEM = chainPEM
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LBCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state LBCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cert, err := r.client.UpdateLBCertificate(state.ID.ValueString(), lbCertificateRequestFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Updating LB Certificate", err.Error())
		return
	}

	plan.ID = state.ID
	lbCertificateToModel(cert, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LBCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LBCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteLBCertificate(state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting LB Certificate", err.Error())
	}
}

func lbCertificateRequestFromModel(m LBCertificateResourceModel) LBCertificateRequest {
	req := LBCertificateRequest{
		Name:    m.Name.ValueString(),
		CertPEM: m.CertPEM.ValueString(),
		KeyPEM:  m.KeyPEM.ValueString(),
	}
	if !m.Description.IsNull() && !m.Description.IsUnknown() {
		req.Description = m.Description.ValueString()
	}
	if !m.ChainPEM.IsNull() && !m.ChainPEM.IsUnknown() {
		req.ChainPEM = m.ChainPEM.ValueString()
	}
	return req
}

func lbCertificateToModel(cert *LBCertificate, m *LBCertificateResourceModel) {
	m.ID = types.StringValue(cert.ID)
	m.Name = types.StringValue(cert.Name)
	m.Description = types.StringValue(cert.Description)
	if cert.CertPEM != "" {
		m.CertPEM = types.StringValue(cert.CertPEM)
	}
	if cert.ChainPEM != "" {
		m.ChainPEM = types.StringValue(cert.ChainPEM)
	}
	m.Subject = types.StringValue(cert.Subject)
	m.DNSNames = make([]types.String, len(cert.DNSNames))
	for i, name := range cert.DNSNames {
		m.DNSNames[i] = types.StringValue(name)
	}
	m.NotBefore = timeToString(cert.NotBefore)
	m.NotAfter = timeToString(cert.NotAfter)
	m.Fingerprint = types.StringValue(cert.Fingerprint)
	m.CreatedAt = timeToString(cert.CreatedAt)
	m.UpdatedAt = timeToString(cert.UpdatedAt)
}

func timeToString(t time.Time) types.String {
	if t.IsZero() {
		return types.StringNull()
	}
	return types.StringValue(t.Format(time.RFC3339))
}
