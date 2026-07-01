package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RootfsImageDataSource{}
var _ datasource.DataSourceWithConfigure = &RootfsImageDataSource{}

type RootfsImageDataSource struct {
	client *Client
}

type RootfsImageDataSourceModel struct {
	// filters
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Arch    types.String `tfsdk:"arch"`
	Source  types.String `tfsdk:"source"`
	Version types.String `tfsdk:"version"`
	// computed
	Description types.String `tfsdk:"description"`
	RootfsPath  types.String `tfsdk:"rootfs_path"`
	SizeBytes   types.Int64  `tfsdk:"size_bytes"`
	SHA256      types.String `tfsdk:"sha256"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func NewRootfsImageDataSource() datasource.DataSource {
	return &RootfsImageDataSource{}
}

func (d *RootfsImageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rootfs_image"
}

func (d *RootfsImageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a Firecracker rootfs image from LatticeVE's general-purpose rootfs registry (`GET /rootfs-images`) — covers both manual uploads and images imported via the k3s auto-fetch flow (`source = \"latticeve-k3s-images\"`). At least one filter must be set. If multiple images match, the most recently created one is returned.",
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
			"arch": schema.StringAttribute{
				MarkdownDescription: "Filter by architecture: `amd64` or `arm64`.",
				Optional:            true,
				Computed:            true,
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "Filter by import source, e.g. `latticeve-k3s-images` for images imported via the k3s auto-fetch flow. Empty for manual uploads.",
				Optional:            true,
				Computed:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Filter by exact version (only set for images imported via the k3s auto-fetch flow).",
				Optional:            true,
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Image description.",
				Computed:            true,
			},
			"rootfs_path": schema.StringAttribute{
				MarkdownDescription: "Host path to the rootfs image — use as `rootfs_id` lookups resolve to this via id, not this path directly.",
				Computed:            true,
			},
			"size_bytes": schema.Int64Attribute{
				MarkdownDescription: "Image size in bytes.",
				Computed:            true,
			},
			"sha256": schema.StringAttribute{
				MarkdownDescription: "SHA-256 checksum of the rootfs image.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "ISO 8601 timestamp when the image was uploaded or imported.",
				Computed:            true,
			},
		},
	}
}

func (d *RootfsImageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RootfsImageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RootfsImageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() && data.Name.IsNull() && data.Arch.IsNull() && data.Source.IsNull() && data.Version.IsNull() {
		resp.Diagnostics.AddError("No Filter Specified", "At least one of id, name, arch, source, or version must be set.")
		return
	}

	images, err := d.client.ListRootfsImages()
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Rootfs Images", err.Error())
		return
	}

	if !data.ID.IsNull() {
		for _, img := range images {
			if img.ID == data.ID.ValueString() {
				d.toState(&data, img)
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
		}
		resp.Diagnostics.AddError("Rootfs Image Not Found", fmt.Sprintf("No rootfs image with id %q.", data.ID.ValueString()))
		return
	}

	var matched []RootfsImage
	for _, img := range images {
		if !data.Name.IsNull() && img.Name != data.Name.ValueString() {
			continue
		}
		if !data.Arch.IsNull() && img.Arch != data.Arch.ValueString() {
			continue
		}
		if !data.Source.IsNull() && img.Source != data.Source.ValueString() {
			continue
		}
		if !data.Version.IsNull() && img.Version != data.Version.ValueString() {
			continue
		}
		matched = append(matched, img)
	}

	if len(matched) == 0 {
		resp.Diagnostics.AddError("No Rootfs Image Found", fmt.Sprintf(
			"No rootfs image matched filters (name=%q arch=%q source=%q version=%q).",
			data.Name.ValueString(), data.Arch.ValueString(), data.Source.ValueString(), data.Version.ValueString()))
		return
	}

	best := matched[0]
	for _, img := range matched[1:] {
		if img.CreatedAt.After(best.CreatedAt) {
			best = img
		}
	}

	d.toState(&data, best)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *RootfsImageDataSource) toState(data *RootfsImageDataSourceModel, img RootfsImage) {
	data.ID = types.StringValue(img.ID)
	data.Name = types.StringValue(img.Name)
	data.Arch = types.StringValue(img.Arch)
	data.Source = types.StringValue(img.Source)
	data.Version = types.StringValue(img.Version)
	data.Description = types.StringValue(img.Description)
	data.RootfsPath = types.StringValue(img.RootfsPath)
	data.SizeBytes = types.Int64Value(img.SizeBytes)
	data.SHA256 = types.StringValue(img.SHA256)
	data.CreatedAt = types.StringValue(img.CreatedAt.Format("2006-01-02T15:04:05Z"))
}
