package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ImageDataSource{}
var _ datasource.DataSourceWithConfigure = &ImageDataSource{}

type ImageDataSource struct {
	client *Client
}

type ImageDataSourceModel struct {
	// filters
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Distro  types.String `tfsdk:"distro"`
	Version types.String `tfsdk:"version"`
	Arch    types.String `tfsdk:"arch"`
	// computed
	Format      types.String `tfsdk:"format"`
	SizeBytes   types.Int64  `tfsdk:"size_bytes"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func NewImageDataSource() datasource.DataSource {
	return &ImageDataSource{}
}

func (d *ImageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image"
}

func (d *ImageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a LatticeVE image (like an AWS AMI) to use as the boot disk for a `lattice_vm`. At least one filter must be set. If multiple images match, the most recently created one is returned.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Filter by exact image UUID. When set, all other filters are ignored.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Filter by exact image name.",
				Optional:            true,
				Computed:            true,
			},
			"distro": schema.StringAttribute{
				MarkdownDescription: "Filter by distro, e.g. `debian`, `ubuntu`, `alpine`, `fedora`, `rocky`.",
				Optional:            true,
				Computed:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Filter by distro version, e.g. `12` for Debian 12, `26.04` for Ubuntu 26.04, `3.24` for Alpine 3.24.",
				Optional:            true,
				Computed:            true,
			},
			"arch": schema.StringAttribute{
				MarkdownDescription: "Filter by architecture: `amd64` or `arm64`. Defaults to `amd64` when not set.",
				Optional:            true,
				Computed:            true,
			},
			"format": schema.StringAttribute{
				MarkdownDescription: "Image format: `qcow2` or `raw`.",
				Computed:            true,
			},
			"size_bytes": schema.Int64Attribute{
				MarkdownDescription: "Image size in bytes.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Image description.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "ISO 8601 timestamp when the image was imported.",
				Computed:            true,
			},
		},
	}
}

func (d *ImageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ImageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ImageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() && data.Name.IsNull() && data.Distro.IsNull() && data.Version.IsNull() && data.Arch.IsNull() {
		resp.Diagnostics.AddError("No Filter Specified", "At least one of id, name, distro, version, or arch must be set.")
		return
	}

	images, err := d.client.ListImages()
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Images", err.Error())
		return
	}

	// ID lookup short-circuits everything else.
	if !data.ID.IsNull() {
		for _, img := range images {
			if img.ID == data.ID.ValueString() {
				d.toState(&data, img)
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
		}
		resp.Diagnostics.AddError("Image Not Found", fmt.Sprintf("No image with id %q.", data.ID.ValueString()))
		return
	}

	var matched []Image
	for _, img := range images {
		if !data.Name.IsNull() && img.Name != data.Name.ValueString() {
			continue
		}
		if !data.Distro.IsNull() && img.Distro != data.Distro.ValueString() {
			continue
		}
		if !data.Version.IsNull() && img.Version != data.Version.ValueString() {
			continue
		}
		if !data.Arch.IsNull() && img.Arch != data.Arch.ValueString() {
			continue
		}
		// Default arch to amd64 when caller omits it
		if data.Arch.IsNull() && img.Arch != "amd64" {
			continue
		}
		matched = append(matched, img)
	}

	if len(matched) == 0 {
		resp.Diagnostics.AddError("No Image Found", fmt.Sprintf(
			"No image matched filters (distro=%q version=%q arch=%q name=%q).",
			data.Distro.ValueString(), data.Version.ValueString(),
			data.Arch.ValueString(), data.Name.ValueString()))
		return
	}

	// Most recently created wins.
	best := matched[0]
	for _, img := range matched[1:] {
		if img.CreatedAt.After(best.CreatedAt) {
			best = img
		}
	}

	d.toState(&data, best)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *ImageDataSource) toState(data *ImageDataSourceModel, img Image) {
	data.ID = types.StringValue(img.ID)
	data.Name = types.StringValue(img.Name)
	data.Distro = types.StringValue(img.Distro)
	data.Version = types.StringValue(img.Version)
	data.Arch = types.StringValue(img.Arch)
	data.Format = types.StringValue(img.Format)
	data.SizeBytes = types.Int64Value(img.SizeBytes)
	data.Description = types.StringValue(img.Description)
	data.CreatedAt = types.StringValue(img.CreatedAt.Format("2006-01-02T15:04:05Z"))
}
