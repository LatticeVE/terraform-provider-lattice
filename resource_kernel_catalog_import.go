package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &KernelCatalogImportResource{}
var _ resource.ResourceWithConfigure = &KernelCatalogImportResource{}

// KernelCatalogImportResource imports a Kernel Catalog entry (built-in or
// discovered from Firecracker's CI bucket) into the kernels table, so it can
// be used as kernel_id elsewhere. The imported kernel keeps the catalog
// entry's id, so this resource's id doubles as the resulting kernel_id.
type KernelCatalogImportResource struct {
	client *Client
}

type KernelCatalogImportResourceModel struct {
	ID          types.String `tfsdk:"id"`
	EntryID     types.String `tfsdk:"entry_id"`
	Name        types.String `tfsdk:"name"`
	Distro      types.String `tfsdk:"distro"`
	Version     types.String `tfsdk:"version"`
	Arch        types.String `tfsdk:"arch"`
	VmlinuzPath types.String `tfsdk:"vmlinuz_path"`
	SizeBytes   types.Int64  `tfsdk:"size_bytes"`
}

func NewKernelCatalogImportResource() resource.Resource {
	return &KernelCatalogImportResource{}
}

func (r *KernelCatalogImportResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kernel_catalog_import"
}

func (r *KernelCatalogImportResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Imports a `lattice_kernel_catalog` entry into the kernels table. The resulting `id` can be used as `kernel_id` in `lattice_vm` or `lattice_kube_cluster`.",
		Attributes: map[string]schema.Attribute{
			"entry_id": schema.StringAttribute{
				MarkdownDescription: "ID of the catalog entry to import (from `lattice_kernel_catalog.id`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Resulting kernel ID — equal to `entry_id`. Use this as `kernel_id`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Kernel name.",
				Computed:            true,
			},
			"distro": schema.StringAttribute{
				MarkdownDescription: "Kernel distro.",
				Computed:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Kernel version.",
				Computed:            true,
			},
			"arch": schema.StringAttribute{
				MarkdownDescription: "Kernel architecture.",
				Computed:            true,
			},
			"vmlinuz_path": schema.StringAttribute{
				MarkdownDescription: "Host path to the imported kernel image.",
				Computed:            true,
			},
			"size_bytes": schema.Int64Attribute{
				MarkdownDescription: "Size of the imported kernel image in bytes.",
				Computed:            true,
			},
		},
	}
}

func (r *KernelCatalogImportResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *KernelCatalogImportResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan KernelCatalogImportResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entryID := plan.EntryID.ValueString()
	if err := r.client.ImportKernelCatalogEntry(entryID); err != nil {
		resp.Diagnostics.AddError("Error Importing Kernel Catalog Entry", err.Error())
		return
	}

	pollCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	for {
		status, err := r.client.KernelCatalogStatus(entryID)
		if err != nil {
			resp.Diagnostics.AddError("Error Polling Kernel Catalog Import", err.Error())
			return
		}
		if status.Imported {
			break
		}
		if status.Error != "" {
			resp.Diagnostics.AddError("Kernel Catalog Import Failed", status.Error)
			return
		}
		select {
		case <-pollCtx.Done():
			resp.Diagnostics.AddError("Timeout", fmt.Sprintf("kernel catalog entry %s did not finish importing within 10 minutes", entryID))
			return
		case <-time.After(3 * time.Second):
		}
	}

	kernels, err := r.client.ListKernels()
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Kernels", err.Error())
		return
	}
	for _, k := range kernels {
		if k.ID == entryID {
			kernelCatalogImportToState(k, &plan)
			resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
			return
		}
	}
	resp.Diagnostics.AddError("Kernel Not Found After Import", fmt.Sprintf("kernel %s did not appear in the kernels list after import", entryID))
}

func (r *KernelCatalogImportResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state KernelCatalogImportResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	kernels, err := r.client.ListKernels()
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Kernels", err.Error())
		return
	}
	for _, k := range kernels {
		if k.ID == state.ID.ValueString() {
			kernelCatalogImportToState(k, &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *KernelCatalogImportResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// entry_id is RequiresReplace and nothing else is settable, so Update is
	// never actually invoked by Terraform for this resource.
}

func (r *KernelCatalogImportResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state KernelCatalogImportResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteKernel(state.ID.ValueString()); err != nil && !strings.Contains(err.Error(), "404") {
		resp.Diagnostics.AddError("Error Deleting Kernel", err.Error())
	}
}

func kernelCatalogImportToState(k Kernel, state *KernelCatalogImportResourceModel) {
	state.ID = types.StringValue(k.ID)
	state.EntryID = types.StringValue(k.ID)
	state.Name = types.StringValue(k.Name)
	state.Distro = types.StringValue(k.Distro)
	state.Version = types.StringValue(k.Version)
	state.Arch = types.StringValue(k.Arch)
	state.VmlinuzPath = types.StringValue(k.VmlinuzPath)
	state.SizeBytes = types.Int64Value(k.SizeBytes)
}
