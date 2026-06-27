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
	Distro        types.String `tfsdk:"distro"`
	DistroVersion types.String `tfsdk:"distro_version"`
	Name          types.String `tfsdk:"name"`
	Version       types.String `tfsdk:"version"`
	VersionGlob   types.String `tfsdk:"version_glob"`
	// computed
	ID                    types.String `tfsdk:"id"`
	ResolvedDistroVersion types.String `tfsdk:"resolved_distro_version"`
	BuiltAt               types.String `tfsdk:"built_at"`
	SizeBytes             types.Int64  `tfsdk:"size_bytes"`
	VmlinuzPath           types.String `tfsdk:"vmlinuz_path"`
	InitramfsPath         types.String `tfsdk:"initramfs_path"`
}

func NewKernelDataSource() datasource.DataSource {
	return &KernelDataSource{}
}

func (d *KernelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kernel"
}

func (d *KernelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a kernel from the LatticeVE kernel catalog. At least one filter must be set. If multiple kernels match, the most recently built one is returned.",
		Attributes: map[string]schema.Attribute{
			"distro": schema.StringAttribute{
				MarkdownDescription: "Filter by distro name, e.g. `alpine`, `ubuntu`, `debian`, `fedora-coreos`, `talos`.",
				Optional:            true,
			},
			"distro_version": schema.StringAttribute{
				MarkdownDescription: "Filter by distro release version, e.g. `3.24.1` for Alpine or `26.04` for Ubuntu. This is the distro's own version number, not the upstream kernel version.",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Filter by exact kernel name.",
				Optional:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Filter by exact upstream kernel version, e.g. `6.12.9`. Mutually exclusive with `version_glob`.",
				Optional:            true,
			},
			"version_glob": schema.StringAttribute{
				MarkdownDescription: "Glob pattern matched against the upstream kernel version, e.g. `6.12.*` or `6.*`. Returns the newest match. Mutually exclusive with `version`.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Kernel UUID — use this as `kernel_id` in `lattice_vm`.",
				Computed:            true,
			},
			"resolved_distro_version": schema.StringAttribute{
				MarkdownDescription: "Distro release version of the selected kernel, e.g. `3.24.1` or `26.04`.",
				Computed:            true,
			},
			"built_at": schema.StringAttribute{
				MarkdownDescription: "ISO 8601 timestamp when the kernel was built or imported.",
				Computed:            true,
			},
			"size_bytes": schema.Int64Attribute{
				MarkdownDescription: "Combined size of vmlinuz + initramfs in bytes.",
				Computed:            true,
			},
			"vmlinuz_path": schema.StringAttribute{
				MarkdownDescription: "Host path to the kernel image.",
				Computed:            true,
			},
			"initramfs_path": schema.StringAttribute{
				MarkdownDescription: "Host path to the initramfs image.",
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

	if data.Distro.IsNull() && data.DistroVersion.IsNull() && data.Name.IsNull() && data.Version.IsNull() && data.VersionGlob.IsNull() {
		resp.Diagnostics.AddError("No Filter Specified", "At least one of distro, distro_version, name, version, or version_glob must be set.")
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
		if !data.DistroVersion.IsNull() && k.DistroVersion != data.DistroVersion.ValueString() {
			continue
		}
		if !data.Name.IsNull() && k.Name != data.Name.ValueString() {
			continue
		}
		if !data.Version.IsNull() && k.Version != data.Version.ValueString() {
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
			"No kernel matched filters (distro=%q name=%q version=%q version_glob=%q).",
			data.Distro.ValueString(), data.Name.ValueString(),
			data.Version.ValueString(), data.VersionGlob.ValueString()))
		return
	}

	// Pick most recently built.
	best := matched[0]
	for _, k := range matched[1:] {
		if k.BuiltAt.After(best.BuiltAt) {
			best = k
		}
	}

	data.ID = types.StringValue(best.ID)
	data.ResolvedDistroVersion = types.StringValue(best.DistroVersion)
	data.BuiltAt = types.StringValue(best.BuiltAt.Format("2006-01-02T15:04:05Z"))
	data.SizeBytes = types.Int64Value(best.SizeBytes)
	data.VmlinuzPath = types.StringValue(best.VmlinuzPath)
	data.InitramfsPath = types.StringValue(best.InitramfsPath)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
