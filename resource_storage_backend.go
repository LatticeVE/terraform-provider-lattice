package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &StorageBackendResource{}
var _ resource.ResourceWithConfigure = &StorageBackendResource{}

type StorageBackendResource struct {
	client *Client
}

type StorageBackendModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	Config    types.Map    `tfsdk:"config"`
	IsDefault types.Bool   `tfsdk:"is_default"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func NewStorageBackendResource() resource.Resource {
	return &StorageBackendResource{}
}

func (r *StorageBackendResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_backend"
}

func (r *StorageBackendResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a storage backend on LatticeVE.",
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
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Backend type: \"lvm\", \"linstor\", \"nfs\", \"ceph\", or \"local\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"config": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"is_default": schema.BoolAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
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

func (r *StorageBackendResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *StorageBackendResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data StorageBackendModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfgAny := make(map[string]any)
	if !data.Config.IsNull() && !data.Config.IsUnknown() {
		cfgStr := make(map[string]string)
		resp.Diagnostics.Append(data.Config.ElementsAs(ctx, &cfgStr, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for k, v := range cfgStr {
			cfgAny[k] = v
		}
	}

	backend, err := r.client.CreateStorageBackend(data.Name.ValueString(), data.Type.ValueString(), cfgAny)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Storage Backend", err.Error())
		return
	}

	resp.Diagnostics.Append(storageBackendToModel(ctx, backend, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StorageBackendResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data StorageBackendModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backend, err := r.client.GetStorageBackend(data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Storage Backend", err.Error())
		return
	}

	resp.Diagnostics.Append(storageBackendToModel(ctx, backend, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StorageBackendResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All fields are RequiresReplace; Update is never called.
	var data StorageBackendModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StorageBackendResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data StorageBackendModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteStorageBackend(data.ID.ValueString()); err != nil {
		if !isNotFound(err) {
			resp.Diagnostics.AddError("Error Deleting Storage Backend", err.Error())
		}
	}
}

func storageBackendToModel(ctx context.Context, backend *StorageBackend, data *StorageBackendModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(backend.ID)
	data.Name = types.StringValue(backend.Name)
	data.Type = types.StringValue(backend.Type)
	data.IsDefault = types.BoolValue(backend.IsDefault)
	data.CreatedAt = types.StringValue(backend.CreatedAt.Format("2006-01-02T15:04:05Z"))

	cfgStr := make(map[string]string, len(backend.Config))
	for k, v := range backend.Config {
		if sv, ok := v.(string); ok {
			cfgStr[k] = sv
		} else {
			cfgStr[k] = fmt.Sprintf("%v", v)
		}
	}

	cfgVal, d := types.MapValueFrom(ctx, types.StringType, cfgStr)
	diags.Append(d...)
	if !diags.HasError() {
		data.Config = cfgVal
	}

	return diags
}
