package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &KernelCatalogDataSource{}
var _ datasource.DataSourceWithConfigure = &KernelCatalogDataSource{}

type KernelCatalogDataSource struct {
	client *Client
}

type KernelCatalogDataSourceModel struct {
	// filters
	Distro      types.String `tfsdk:"distro"`
	Name        types.String `tfsdk:"name"`
	Version     types.String `tfsdk:"version"`
	VersionGlob types.String `tfsdk:"version_glob"`
	Arch        types.String `tfsdk:"arch"`
	// computed
	ID            types.String `tfsdk:"id"`
	VmlinuzURL    types.String `tfsdk:"vmlinuz_url"`
	VmlinuzSizeMB types.Int64  `tfsdk:"vmlinuz_size_mb"`
	Description   types.String `tfsdk:"description"`
	Imported      types.Bool   `tfsdk:"imported"`
}

func NewKernelCatalogDataSource() datasource.DataSource {
	return &KernelCatalogDataSource{}
}

func (d *KernelCatalogDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kernel_catalog"
}

func (d *KernelCatalogDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a kernel from LatticeVE's Kernel Catalog (`GET /kernel-catalog`) — built-in entries plus anything discovered from Firecracker's CI bucket. The catalog entry's `id` is not usable as `kernel_id` until imported; use `lattice_kernel_catalog_import` to import it, or look up `lattice_kernel` afterwards. At least one filter must be set.",
		Attributes: map[string]schema.Attribute{
			"distro": schema.StringAttribute{
				MarkdownDescription: "Filter by distro name, e.g. `firecracker`.",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Filter by exact catalog entry name.",
				Optional:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Filter by exact kernel version. Mutually exclusive with `version_glob`.",
				Optional:            true,
			},
			"version_glob": schema.StringAttribute{
				MarkdownDescription: "Glob pattern matched against the kernel version, e.g. `6.1.*`. Returns the newest match. Mutually exclusive with `version`.",
				Optional:            true,
			},
			"arch": schema.StringAttribute{
				MarkdownDescription: "Filter by architecture: `amd64` or `arm64`.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Catalog entry ID — pass this as `entry_id` to `lattice_kernel_catalog_import`.",
				Computed:            true,
			},
			"vmlinuz_url": schema.StringAttribute{
				MarkdownDescription: "Source URL the kernel is downloaded from on import.",
				Computed:            true,
			},
			"vmlinuz_size_mb": schema.Int64Attribute{
				MarkdownDescription: "Approximate kernel image size in MB.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Catalog entry description.",
				Computed:            true,
			},
			"imported": schema.BoolAttribute{
				MarkdownDescription: "Whether this catalog entry has already been imported into the kernels table on this controller.",
				Computed:            true,
			},
		},
	}
}

func (d *KernelCatalogDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", fmt.Sprintf("Expected *Client, got: %T", req.ProviderData))
		return
	}
	d.client = client
}

func (d *KernelCatalogDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KernelCatalogDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Distro.IsNull() && data.Name.IsNull() && data.Version.IsNull() && data.VersionGlob.IsNull() && data.Arch.IsNull() {
		resp.Diagnostics.AddError("No Filter Specified", "At least one of distro, name, version, version_glob, or arch must be set.")
		return
	}
	if !data.Version.IsNull() && !data.VersionGlob.IsNull() {
		resp.Diagnostics.AddError("Conflicting Filters", "version and version_glob are mutually exclusive.")
		return
	}

	entries, err := d.client.ListKernelCatalog()
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Kernel Catalog", err.Error())
		return
	}

	var matched []KernelCatalogEntry
	for _, e := range entries {
		if !data.Distro.IsNull() && e.Distro != data.Distro.ValueString() {
			continue
		}
		if !data.Name.IsNull() && e.Name != data.Name.ValueString() {
			continue
		}
		if !data.Version.IsNull() && e.Version != data.Version.ValueString() {
			continue
		}
		if !data.Arch.IsNull() && e.Arch != data.Arch.ValueString() {
			continue
		}
		if !data.VersionGlob.IsNull() {
			ok, _ := filepath.Match(data.VersionGlob.ValueString(), e.Version)
			if !ok {
				continue
			}
		}
		matched = append(matched, e)
	}

	if len(matched) == 0 {
		resp.Diagnostics.AddError("No Kernel Catalog Entry Found", fmt.Sprintf(
			"No catalog entry matched filters (distro=%q name=%q version=%q version_glob=%q arch=%q).",
			data.Distro.ValueString(), data.Name.ValueString(),
			data.Version.ValueString(), data.VersionGlob.ValueString(), data.Arch.ValueString()))
		return
	}

	// Lexically-newest version wins when multiple entries match (catalog
	// entries don't carry a timestamp to sort by).
	best := matched[0]
	for _, e := range matched[1:] {
		if e.Version > best.Version {
			best = e
		}
	}

	data.ID = types.StringValue(best.ID)
	data.VmlinuzURL = types.StringValue(best.VmlinuzURL)
	data.VmlinuzSizeMB = types.Int64Value(int64(best.VmlinuzSizeMB))
	data.Description = types.StringValue(best.Description)
	data.Imported = types.BoolValue(best.Imported)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
