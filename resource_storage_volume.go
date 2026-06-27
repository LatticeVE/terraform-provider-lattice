package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &StorageVolumeResource{}
var _ resource.ResourceWithConfigure = &StorageVolumeResource{}

type StorageVolumeResource struct {
	client *Client
}

type StorageVolumeModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	SizeGB       types.Int64  `tfsdk:"size_gb"`
	BackendID    types.String `tfsdk:"backend_id"`
	SizeBytes    types.Int64  `tfsdk:"size_bytes"`
	DiskfulNodes types.List   `tfsdk:"diskful_nodes"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func NewStorageVolumeResource() resource.Resource {
	return &StorageVolumeResource{}
}

func (r *StorageVolumeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_volume"
}

func (r *StorageVolumeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a storage volume on LatticeVE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"size_gb": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Volume size in GiB. Can be increased (grow only); shrinking is not supported.",
			},
			"backend_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"size_bytes": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Actual allocated size in bytes as reported by the backend.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"diskful_nodes": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *StorageVolumeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *StorageVolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data StorageVolumeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sizeBytes := data.SizeGB.ValueInt64() * 1073741824
	vol, err := r.client.CreateStorageVolume(data.Name.ValueString(), sizeBytes, data.BackendID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Storage Volume", err.Error())
		return
	}

	resp.Diagnostics.Append(storageVolumeToModel(ctx, vol, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StorageVolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data StorageVolumeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vol, err := r.client.GetStorageVolume(data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Storage Volume", err.Error())
		return
	}

	resp.Diagnostics.Append(storageVolumeToModel(ctx, vol, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StorageVolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan StorageVolumeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.SizeGB.ValueInt64() < state.SizeGB.ValueInt64() {
		resp.Diagnostics.AddError(
			"Volume Shrinking Not Supported",
			fmt.Sprintf("Cannot shrink volume from %d GiB to %d GiB; only growing is supported.", state.SizeGB.ValueInt64(), plan.SizeGB.ValueInt64()),
		)
		return
	}

	var vol *StorageVolume
	var err error

	if plan.SizeGB.ValueInt64() > state.SizeGB.ValueInt64() {
		newSizeBytes := plan.SizeGB.ValueInt64() * 1073741824
		vol, err = r.client.ResizeStorageVolume(state.ID.ValueString(), newSizeBytes)
		if err != nil {
			resp.Diagnostics.AddError("Error Resizing Storage Volume", err.Error())
			return
		}
	} else {
		vol, err = r.client.GetStorageVolume(state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error Reading Storage Volume", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(storageVolumeToModel(ctx, vol, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *StorageVolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data StorageVolumeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteStorageVolume(data.ID.ValueString()); err != nil {
		if !isNotFound(err) {
			resp.Diagnostics.AddError("Error Deleting Storage Volume", err.Error())
		}
	}
}

func storageVolumeToModel(ctx context.Context, vol *StorageVolume, data *StorageVolumeModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(vol.ID)
	data.Name = types.StringValue(vol.Name)
	data.SizeBytes = types.Int64Value(vol.SizeBytes)
	data.BackendID = types.StringValue(vol.BackendID)
	data.CreatedAt = types.StringValue(vol.CreatedAt.Format("2006-01-02T15:04:05Z"))

	nodes, d := types.ListValueFrom(ctx, types.StringType, vol.DiskfulNodes)
	diags.Append(d...)
	if !diags.HasError() {
		data.DiskfulNodes = nodes
	}

	return diags
}
