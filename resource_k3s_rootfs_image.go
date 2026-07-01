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

var _ resource.Resource = &K3sRootfsImageResource{}
var _ resource.ResourceWithConfigure = &K3sRootfsImageResource{}

// K3sRootfsImageResource imports a pinned or latest k3s rootfs build for an
// architecture. Import happens at resource creation; arch and version changes
// replace the image so the dependent cluster can run its rolling upgrade.
type K3sRootfsImageResource struct {
	client *Client
}

type K3sRootfsImageResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Arch        types.String `tfsdk:"arch"`
	Version     types.String `tfsdk:"version"`
	Name        types.String `tfsdk:"name"`
	DownloadURL types.String `tfsdk:"download_url"`
	SizeBytes   types.Int64  `tfsdk:"size_bytes"`
}

func NewK3sRootfsImageResource() resource.Resource {
	return &K3sRootfsImageResource{}
}

func (r *K3sRootfsImageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_k3s_rootfs_image"
}

func (r *K3sRootfsImageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Imports a pinned or latest k3s rootfs build for `arch` from latticeve-k3s-images. Use the resulting `id` as `rootfs_id` in `lattice_kube_cluster`; changing `version` replaces the image and drives the cluster's safe rolling upgrade.",
		Attributes: map[string]schema.Attribute{
			"arch": schema.StringAttribute{
				MarkdownDescription: "Architecture to track: `amd64` or `arm64`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "UUID of the imported rootfs image. Use this as `rootfs_id`.",
				Computed:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Optional exact k3s image version to import, e.g. `v1.36.2+k3s1-r23`. When omitted, imports the newest discovered build for the architecture.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the imported rootfs image.",
				Computed:            true,
			},
			"download_url": schema.StringAttribute{
				MarkdownDescription: "GitHub release asset URL the image was downloaded from.",
				Computed:            true,
			},
			"size_bytes": schema.Int64Attribute{
				MarkdownDescription: "Size of the imported rootfs image in bytes.",
				Computed:            true,
			},
		},
	}
}

func (r *K3sRootfsImageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// discoverAndImport finds the requested (or latest) k3s rootfs asset and
// imports it, returning the resulting registry entry plus the discovery
// entry's download URL (which the import response itself doesn't echo back).
func (r *K3sRootfsImageResource) discoverAndImport(arch, version string) (*RootfsImage, string, error) {
	entries, err := r.client.DiscoverK3sRootfs()
	if err != nil {
		return nil, "", fmt.Errorf("discovering k3s rootfs: %w", err)
	}
	for _, e := range entries {
		if e.Arch != arch || (version != "" && e.Version != version) {
			continue
		}
		img, err := r.client.ImportK3sRootfs(e)
		if err != nil {
			return nil, "", fmt.Errorf("importing k3s rootfs: %w", err)
		}
		return img, e.DownloadURL, nil
	}
	if version != "" {
		return nil, "", fmt.Errorf("latticeve-k3s-images has no rootfs asset for arch %q and version %q", arch, version)
	}
	return nil, "", fmt.Errorf("latticeve-k3s-images has no rootfs asset for arch %q", arch)
}

func (r *K3sRootfsImageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan K3sRootfsImageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	version := ""
	if !plan.Version.IsNull() && !plan.Version.IsUnknown() {
		version = plan.Version.ValueString()
	}
	img, downloadURL, err := r.discoverAndImport(plan.Arch.ValueString(), version)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing k3s Rootfs", err.Error())
		return
	}
	k3sRootfsToState(*img, downloadURL, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *K3sRootfsImageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state K3sRootfsImageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	images, err := r.client.ListRootfsImages()
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Rootfs Images", err.Error())
		return
	}
	for _, img := range images {
		if img.ID == state.ID.ValueString() {
			k3sRootfsToState(img, state.DownloadURL.ValueString(), &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *K3sRootfsImageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// arch is RequiresReplace and every other attribute is Computed, so
	// Terraform never has a plan diff that calls Update for this resource —
	// changes always go through Delete+Create instead.
}

func (r *K3sRootfsImageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state K3sRootfsImageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRootfsImage(state.ID.ValueString()); err != nil && !strings.Contains(err.Error(), "404") {
		resp.Diagnostics.AddError("Error Deleting Rootfs Image", err.Error())
	}
}

func k3sRootfsToState(img RootfsImage, downloadURL string, state *K3sRootfsImageResourceModel) {
	state.ID = types.StringValue(img.ID)
	state.Arch = types.StringValue(img.Arch)
	state.Version = types.StringValue(img.Version)
	state.Name = types.StringValue(img.Name)
	state.SizeBytes = types.Int64Value(img.SizeBytes)
	state.DownloadURL = types.StringValue(downloadURL)
}
