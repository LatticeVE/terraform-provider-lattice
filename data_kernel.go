package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &KernelDataSource{}
var _ datasource.DataSourceWithConfigure = &KernelDataSource{}

type KernelDataSource struct {
	client *Client
}

type KernelDataSourceModel struct {
	// filters
	Distro      types.String `tfsdk:"distro"`
	Name        types.String `tfsdk:"name"`
	Version     types.String `tfsdk:"version"`
	VersionGlob types.String `tfsdk:"version_glob"`
	Arch        types.String `tfsdk:"arch"`
	// computed
	ID          types.String `tfsdk:"id"`
	CreatedAt   types.String `tfsdk:"created_at"`
	SizeBytes   types.Int64  `tfsdk:"size_bytes"`
	VmlinuzPath types.String `tfsdk:"vmlinuz_path"`
}

func NewKernelDataSource() datasource.DataSource {
	return &KernelDataSource{}
}

func (d *KernelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kernel"
}

func (d *KernelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up an already-imported kernel from `GET /kernels`. At least one filter must be set. If multiple kernels match, the most recently imported one is returned. To browse kernels available to import but not yet imported, use `lattice_kernel_catalog` instead.",
		Attributes: map[string]schema.Attribute{
			"distro": schema.StringAttribute{
				MarkdownDescription: "Filter by distro name, e.g. `alpine`, `firecracker`.",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Filter by exact kernel name.",
				Optional:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Filter by exact kernel version. Mutually exclusive with `version_glob`.",
				Optional:            true,
			},
			"version_glob": schema.StringAttribute{
				MarkdownDescription: "Glob pattern matched against the kernel version, e.g. `6.12.*` or `6.*`. Returns the newest match. Mutually exclusive with `version`.",
				Optional:            true,
			},
			"arch": schema.StringAttribute{
				MarkdownDescription: "Filter by architecture: `amd64` or `arm64`.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Kernel UUID — use this as `kernel_id` in `lattice_vm` or `lattice_kube_cluster`.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "ISO 8601 timestamp when the kernel was imported.",
				Computed:            true,
			},
			"size_bytes": schema.Int64Attribute{
				MarkdownDescription: "Size of the kernel image in bytes.",
				Computed:            true,
			},
			"vmlinuz_path": schema.StringAttribute{
				MarkdownDescription: "Host path to the kernel image.",
				Computed:            true,
			},
		},
	}
}

func (d *KernelDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *KernelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KernelDataSourceModel
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

	kernels, err := d.client.ListKernels()
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Kernels", err.Error())
		return
	}

	var matched []Kernel
	for _, k := range kernels {
		if !data.Distro.IsNull() && k.Distro != data.Distro.ValueString() {
			continue
		}
		if !data.Name.IsNull() && k.Name != data.Name.ValueString() {
			continue
		}
		if !data.Version.IsNull() && k.Version != data.Version.ValueString() {
			continue
		}
		if !data.Arch.IsNull() && k.Arch != data.Arch.ValueString() {
			continue
		}
		if !data.VersionGlob.IsNull() {
			ok, _ := filepath.Match(data.VersionGlob.ValueString(), k.Version)
			if !ok {
				continue
			}
		}
		matched = append(matched, k)
	}

	if len(matched) == 0 {
		resp.Diagnostics.AddError("No Kernel Found", fmt.Sprintf(
			"No kernel matched filters (distro=%q name=%q version=%q version_glob=%q arch=%q).",
			data.Distro.ValueString(), data.Name.ValueString(),
			data.Version.ValueString(), data.VersionGlob.ValueString(), data.Arch.ValueString()))
		return
	}

	// Pick most recently imported.
	best := matched[0]
	for _, k := range matched[1:] {
		if k.CreatedAt.After(best.CreatedAt) {
			best = k
		}
	}

	data.ID = types.StringValue(best.ID)
	data.CreatedAt = types.StringValue(best.CreatedAt.Format("2006-01-02T15:04:05Z"))
	data.SizeBytes = types.Int64Value(best.SizeBytes)
	data.VmlinuzPath = types.StringValue(best.VmlinuzPath)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
