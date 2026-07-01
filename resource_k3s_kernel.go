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

var _ resource.Resource = &K3sKernelResource{}
var _ resource.ResourceWithConfigure = &K3sKernelResource{}

type K3sKernelResource struct{ client *Client }

type K3sKernelResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Arch        types.String `tfsdk:"arch"`
	Name        types.String `tfsdk:"name"`
	Version     types.String `tfsdk:"version"`
	DownloadURL types.String `tfsdk:"download_url"`
	SizeBytes   types.Int64  `tfsdk:"size_bytes"`
}

func NewK3sKernelResource() resource.Resource { return &K3sKernelResource{} }

func (r *K3sKernelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_k3s_kernel"
}

func (r *K3sKernelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Imports the latest Kubernetes-compatible Firecracker kernel for an architecture from latticeve-k3s-images. Replace this resource explicitly to adopt a newer published kernel.",
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Computed: true, MarkdownDescription: "UUID of the imported kernel."},
			"arch":         schema.StringAttribute{Required: true, MarkdownDescription: "Kernel architecture: `amd64` or `arm64`.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"name":         schema.StringAttribute{Computed: true, MarkdownDescription: "Imported kernel filename/name."},
			"version":      schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Optional exact Linux kernel version. When omitted, imports the newest discovered kernel for the architecture.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"download_url": schema.StringAttribute{Computed: true, MarkdownDescription: "Verified GitHub release asset URL."},
			"size_bytes":   schema.Int64Attribute{Computed: true, MarkdownDescription: "Kernel size in bytes."},
		},
	}
}

func (r *K3sKernelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *Client, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *K3sKernelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan K3sKernelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	entries, err := r.client.DiscoverK3sKernels()
	if err != nil {
		resp.Diagnostics.AddError("Error Discovering k3s Kernels", err.Error())
		return
	}
	version := ""
	if !plan.Version.IsNull() && !plan.Version.IsUnknown() {
		version = plan.Version.ValueString()
	}
	for _, entry := range entries {
		if entry.Arch != plan.Arch.ValueString() || (version != "" && entry.Version != version) {
			continue
		}
		kernel, err := r.client.ImportK3sKernel(entry)
		if err != nil {
			resp.Diagnostics.AddError("Error Importing k3s Kernel", err.Error())
			return
		}
		k3sKernelToState(*kernel, entry.DownloadURL, &plan)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}
	resp.Diagnostics.AddError("No Matching k3s Kernel", fmt.Sprintf("no Kubernetes-compatible kernel was found for architecture %q and version %q", plan.Arch.ValueString(), version))
}

func (r *K3sKernelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state K3sKernelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	kernels, err := r.client.ListKernels()
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Kernels", err.Error())
		return
	}
	for _, kernel := range kernels {
		if kernel.ID == state.ID.ValueString() {
			k3sKernelToState(kernel, state.DownloadURL.ValueString(), &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *K3sKernelResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}

func (r *K3sKernelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state K3sKernelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteKernel(state.ID.ValueString()); err != nil && !strings.Contains(err.Error(), "404") {
		resp.Diagnostics.AddError("Error Deleting k3s Kernel", err.Error())
	}
}

func k3sKernelToState(kernel Kernel, downloadURL string, state *K3sKernelResourceModel) {
	state.ID = types.StringValue(kernel.ID)
	state.Arch = types.StringValue(kernel.Arch)
	state.Name = types.StringValue(kernel.Name)
	state.Version = types.StringValue(kernel.Version)
	state.DownloadURL = types.StringValue(downloadURL)
	state.SizeBytes = types.Int64Value(kernel.SizeBytes)
}
