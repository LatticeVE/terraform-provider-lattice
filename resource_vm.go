package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &VMResource{}
var _ resource.ResourceWithConfigure = &VMResource{}

type VMResource struct {
	client *Client
}

type VMResourceModel struct {
	ID            types.String          `tfsdk:"id"`
	Name          types.String          `tfsdk:"name"`
	CPUs          types.Int64           `tfsdk:"cpus"`
	MemoryMB      types.Int64           `tfsdk:"memory_mb"`
	ISOPath       types.String          `tfsdk:"iso_path"`
	Status        types.String          `tfsdk:"status"`
	DiskPath      types.String          `tfsdk:"disk_path"`
	BootDiskGB    types.Int64           `tfsdk:"boot_disk_gb"`
	DiskInterface types.String          `tfsdk:"disk_interface"`
	VMType        types.String          `tfsdk:"vm_type"`
	KernelID      types.String          `tfsdk:"kernel_id"`
	KernelCmdline types.String          `tfsdk:"kernel_cmdline"`
	ImageID       types.String          `tfsdk:"image_id"`
	Arch          types.String          `tfsdk:"arch"`
	Node          types.String          `tfsdk:"node"`
	ForceDestroy  types.Bool            `tfsdk:"force_destroy"`
	CloudInit     *CloudInitConfigModel `tfsdk:"cloud_init"`
	ExtraDisks    []ExtraDiskModel      `tfsdk:"extra_disks"`
	NICs          []NICModel            `tfsdk:"nics"`
}

type CloudInitConfigModel struct {
	UserData      types.String `tfsdk:"user_data"`
	MetaData      types.String `tfsdk:"meta_data"`
	NetworkConfig types.String `tfsdk:"network_config"`
}

type ExtraDiskModel struct {
	Index     types.Int64  `tfsdk:"index"`
	SizeGB    types.Int64  `tfsdk:"size_gb"`
	DiskPath  types.String `tfsdk:"disk_path"`
	Interface types.String `tfsdk:"interface"`
}

type NICModel struct {
	Index    types.Int64  `tfsdk:"index"`
	Bridge   types.String `tfsdk:"bridge"`
	MACAddr  types.String `tfsdk:"mac_addr"`
	DeviceID types.String `tfsdk:"device_id"`
	Model    types.String `tfsdk:"model"`
}

func NewVMResource() resource.Resource {
	return &VMResource{}
}

func (r *VMResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (r *VMResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a LatticeVE virtual machine.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique UUID of the virtual machine.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the virtual machine.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cpus": schema.Int64Attribute{
				MarkdownDescription: "The number of CPU cores allocated to the VM.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"memory_mb": schema.Int64Attribute{
				MarkdownDescription: "The memory size allocated to the VM in MB.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"iso_path": schema.StringAttribute{
				MarkdownDescription: "Optional path to an ISO file to boot from.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the VM: `running` or `stopped`. Defaults to `running`.",
				Optional:            true,
				Computed:            true,
			},
			"disk_path": schema.StringAttribute{
				MarkdownDescription: "The absolute host path to the main boot disk image (managed by LatticeVE).",
				Computed:            true,
			},
			"boot_disk_gb": schema.Int64Attribute{
				MarkdownDescription: "Optional boot disk size in GB. Defaults to 20.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"disk_interface": schema.StringAttribute{
				MarkdownDescription: "Optional boot disk interface type: `virtio`, `scsi`, `ide`, `sata`. Defaults to `virtio`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cloud_init": schema.SingleNestedAttribute{
				MarkdownDescription: "Optional cloud-init configuration block.",
				Optional:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"user_data": schema.StringAttribute{
						MarkdownDescription: "Cloud-init user-data YAML script.",
						Required:            true,
					},
					"meta_data": schema.StringAttribute{
						MarkdownDescription: "Cloud-init meta-data YAML script.",
						Required:            true,
					},
					"network_config": schema.StringAttribute{
						MarkdownDescription: "Cloud-init network-config YAML script.",
						Optional:            true,
					},
				},
			},
			"extra_disks": schema.ListNestedAttribute{
				MarkdownDescription: "Optional list of additional data volumes to attach to the VM.",
				Optional:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"index": schema.Int64Attribute{
							MarkdownDescription: "The index of the extra disk (1-based).",
							Computed:            true,
						},
						"size_gb": schema.Int64Attribute{
							MarkdownDescription: "The size of the extra disk in GB.",
							Required:            true,
						},
						"disk_path": schema.StringAttribute{
							MarkdownDescription: "The host path to the extra disk image.",
							Computed:            true,
						},
						"interface": schema.StringAttribute{
							MarkdownDescription: "Optional virtual interface type for this disk: `virtio`, `scsi`, `sata`, `ide`. Defaults to `virtio`.",
							Optional:            true,
							Computed:            true,
						},
					},
				},
			},
			"image_id": schema.StringAttribute{
				MarkdownDescription: "ID of a LatticeVE image to use as the boot disk. The image is cloned for this VM — like an AMI in AWS. Use the `lattice_image` data source to look one up. Mutually exclusive with `iso_path`.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vm_type": schema.StringAttribute{
				MarkdownDescription: "VM backend: `qemu` (default) or `firecracker`. Firecracker requires `kernel_id`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"kernel_id": schema.StringAttribute{
				MarkdownDescription: "Kernel UUID from the LatticeVE kernel catalog. Required when `vm_type` is `firecracker`.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"kernel_cmdline": schema.StringAttribute{
				MarkdownDescription: "Extra kernel command-line arguments appended to the default Firecracker cmdline.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"arch": schema.StringAttribute{
				MarkdownDescription: "CPU architecture required for placement: `amd64` or `arm64`. The scheduler picks a node whose CPU matches. Computed from the actual node if not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"node": schema.StringAttribute{
				MarkdownDescription: "Name of the specific host node to run this VM on. Pin to a node when you need locality or want to bypass the scheduler. Computed to reflect actual placement.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"force_destroy": schema.BoolAttribute{
				MarkdownDescription: "If true, Terraform destroy asks LatticeVE to force-delete the VM. Defaults to false; normal destroy asks LatticeVE to stop the VM before deleting it.",
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"nics": schema.ListNestedAttribute{
				MarkdownDescription: "Optional list of bridge network interfaces to attach to the VM.",
				Optional:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"index": schema.Int64Attribute{
							MarkdownDescription: "The index of the interface.",
							Computed:            true,
						},
						"bridge": schema.StringAttribute{
							MarkdownDescription: "The host bridge name (e.g. `virbr0`, `br0`).",
							Required:            true,
						},
						"mac_addr": schema.StringAttribute{
							MarkdownDescription: "The MAC address of the interface. Auto-generated if left empty.",
							Optional:            true,
							Computed:            true,
						},
						"device_id": schema.StringAttribute{
							MarkdownDescription: "The QEMU device ID of the interface.",
							Computed:            true,
						},
						"model": schema.StringAttribute{
							MarkdownDescription: "Optional network interface card model: `virtio-net-pci`, `e1000`, `rtl8139`. Defaults to `virtio-net-pci`.",
							Optional:            true,
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (r *VMResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *VMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VMResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var cloudInit *CloudInitConfig
	if plan.CloudInit != nil {
		cloudInit = &CloudInitConfig{
			UserData: plan.CloudInit.UserData.ValueString(),
			MetaData: plan.CloudInit.MetaData.ValueString(),
		}
		if !plan.CloudInit.NetworkConfig.IsNull() && !plan.CloudInit.NetworkConfig.IsUnknown() {
			cloudInit.NetworkConfig = plan.CloudInit.NetworkConfig.ValueString()
		}
	}

	bootDiskGB := int(20)
	if !plan.BootDiskGB.IsNull() && !plan.BootDiskGB.IsUnknown() {
		bootDiskGB = int(plan.BootDiskGB.ValueInt64())
	}

	diskInterface := "virtio"
	if !plan.DiskInterface.IsNull() && !plan.DiskInterface.IsUnknown() {
		diskInterface = plan.DiskInterface.ValueString()
	}

	extraDisks := make([]ExtraDisk, len(plan.ExtraDisks))
	for i, d := range plan.ExtraDisks {
		extraDisks[i] = ExtraDisk{
			SizeGB: int(d.SizeGB.ValueInt64()),
		}
		if !d.Interface.IsNull() && !d.Interface.IsUnknown() {
			extraDisks[i].Interface = d.Interface.ValueString()
		}
	}

	nics := make([]NIC, len(plan.NICs))
	for i, n := range plan.NICs {
		nics[i] = NIC{
			Bridge: n.Bridge.ValueString(),
		}
		if !n.MACAddr.IsNull() && !n.MACAddr.IsUnknown() {
			nics[i].MACAddr = n.MACAddr.ValueString()
		}
		if !n.Model.IsNull() && !n.Model.IsUnknown() {
			nics[i].Model = n.Model.ValueString()
		}
	}

	isoPath := ""
	if !plan.ISOPath.IsNull() && !plan.ISOPath.IsUnknown() {
		isoPath = plan.ISOPath.ValueString()
	}

	vmType := ""
	if !plan.VMType.IsNull() && !plan.VMType.IsUnknown() {
		vmType = plan.VMType.ValueString()
	}
	kernelID := ""
	if !plan.KernelID.IsNull() && !plan.KernelID.IsUnknown() {
		kernelID = plan.KernelID.ValueString()
	}
	kernelCmdline := ""
	if !plan.KernelCmdline.IsNull() && !plan.KernelCmdline.IsUnknown() {
		kernelCmdline = plan.KernelCmdline.ValueString()
	}

	imageID := ""
	if !plan.ImageID.IsNull() && !plan.ImageID.IsUnknown() {
		imageID = plan.ImageID.ValueString()
	}

	arch := ""
	if !plan.Arch.IsNull() && !plan.Arch.IsUnknown() {
		arch = plan.Arch.ValueString()
	}
	node := ""
	if !plan.Node.IsNull() && !plan.Node.IsUnknown() {
		node = plan.Node.ValueString()
	}

	vm, err := r.client.CreateVM(
		plan.Name.ValueString(),
		int(plan.CPUs.ValueInt64()),
		int(plan.MemoryMB.ValueInt64()),
		bootDiskGB,
		diskInterface,
		isoPath,
		cloudInit,
		extraDisks,
		nics,
		vmType,
		kernelID,
		kernelCmdline,
		imageID,
		arch,
		node,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating VM", err.Error())
		return
	}

	status := "running"
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		status = plan.Status.ValueString()
	}

	if status == "running" {
		if err := r.client.StartVM(vm.ID); err != nil {
			resp.Diagnostics.AddError("Error Starting VM", err.Error())
			return
		}
		vm.Status = StatusRunning
	}

	plan.ID = types.StringValue(vm.ID)
	plan.DiskPath = types.StringValue(vm.DiskPath)
	plan.Status = types.StringValue(string(vm.Status))
	plan.BootDiskGB = types.Int64Value(int64(vm.BootDiskGB))
	plan.DiskInterface = types.StringValue(vm.DiskInterface)
	plan.VMType = types.StringValue(vm.VMType)
	if vm.KernelID != "" {
		plan.KernelID = types.StringValue(vm.KernelID)
	}
	if vm.KernelCmdline != "" {
		plan.KernelCmdline = types.StringValue(vm.KernelCmdline)
	}
	if vm.ImageID != "" {
		plan.ImageID = types.StringValue(vm.ImageID)
	}
	if vm.Arch != "" {
		plan.Arch = types.StringValue(vm.Arch)
	}
	if vm.Node != "" {
		plan.Node = types.StringValue(vm.Node)
	}

	plan.ExtraDisks = make([]ExtraDiskModel, len(vm.ExtraDisks))
	for i, d := range vm.ExtraDisks {
		plan.ExtraDisks[i] = ExtraDiskModel{
			Index:     types.Int64Value(int64(d.Index)),
			SizeGB:    types.Int64Value(int64(d.SizeGB)),
			DiskPath:  types.StringValue(d.DiskPath),
			Interface: types.StringValue(d.Interface),
		}
	}

	plan.NICs = make([]NICModel, len(vm.NICs))
	for i, n := range vm.NICs {
		plan.NICs[i] = NICModel{
			Index:    types.Int64Value(int64(n.Index)),
			Bridge:   types.StringValue(n.Bridge),
			MACAddr:  types.StringValue(n.MACAddr),
			DeviceID: types.StringValue(n.DeviceID),
			Model:    types.StringValue(n.Model),
		}
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *VMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VMResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	vm, err := r.client.GetVM(state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading VM", err.Error())
		return
	}

	state.Name = types.StringValue(vm.Name)
	state.CPUs = types.Int64Value(int64(vm.CPUs))
	state.MemoryMB = types.Int64Value(int64(vm.Memory))
	state.DiskPath = types.StringValue(vm.DiskPath)
	state.Status = types.StringValue(string(vm.Status))
	state.BootDiskGB = types.Int64Value(int64(vm.BootDiskGB))
	state.DiskInterface = types.StringValue(vm.DiskInterface)
	state.VMType = types.StringValue(vm.VMType)
	if vm.KernelID != "" {
		state.KernelID = types.StringValue(vm.KernelID)
	}
	if vm.KernelCmdline != "" {
		state.KernelCmdline = types.StringValue(vm.KernelCmdline)
	}
	if vm.ImageID != "" {
		state.ImageID = types.StringValue(vm.ImageID)
	}
	if vm.Arch != "" {
		state.Arch = types.StringValue(vm.Arch)
	}
	if vm.Node != "" {
		state.Node = types.StringValue(vm.Node)
	}

	if vm.ISOPath != "" {
		state.ISOPath = types.StringValue(vm.ISOPath)
	} else {
		state.ISOPath = types.StringNull()
	}

	if vm.CloudInit != nil {
		state.CloudInit = &CloudInitConfigModel{
			UserData: types.StringValue(vm.CloudInit.UserData),
			MetaData: types.StringValue(vm.CloudInit.MetaData),
		}
		if vm.CloudInit.NetworkConfig != "" {
			state.CloudInit.NetworkConfig = types.StringValue(vm.CloudInit.NetworkConfig)
		} else {
			state.CloudInit.NetworkConfig = types.StringNull()
		}
	} else {
		state.CloudInit = nil
	}

	state.ExtraDisks = make([]ExtraDiskModel, len(vm.ExtraDisks))
	for i, d := range vm.ExtraDisks {
		state.ExtraDisks[i] = ExtraDiskModel{
			Index:     types.Int64Value(int64(d.Index)),
			SizeGB:    types.Int64Value(int64(d.SizeGB)),
			DiskPath:  types.StringValue(d.DiskPath),
			Interface: types.StringValue(d.Interface),
		}
	}

	state.NICs = make([]NICModel, len(vm.NICs))
	for i, n := range vm.NICs {
		state.NICs[i] = NICModel{
			Index:    types.Int64Value(int64(n.Index)),
			Bridge:   types.StringValue(n.Bridge),
			MACAddr:  types.StringValue(n.MACAddr),
			DeviceID: types.StringValue(n.DeviceID),
			Model:    types.StringValue(n.Model),
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *VMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state VMResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Status.Equal(state.Status) {
		id := state.ID.ValueString()
		newStatus := plan.Status.ValueString()

		if newStatus == "running" {
			if err := r.client.StartVM(id); err != nil {
				resp.Diagnostics.AddError("Error Starting VM", err.Error())
				return
			}
			state.Status = types.StringValue(string(StatusRunning))
		} else if newStatus == "stopped" {
			if err := r.client.StopVM(id); err != nil {
				resp.Diagnostics.AddError("Error Stopping VM", err.Error())
				return
			}
			state.Status = types.StringValue(string(StatusStopped))
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *VMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VMResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	force := false
	if !state.ForceDestroy.IsNull() && !state.ForceDestroy.IsUnknown() {
		force = state.ForceDestroy.ValueBool()
	}

	err := r.client.DeleteVM(state.ID.ValueString(), DeleteVMOptions{
		Stop:  !force,
		Force: force,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting VM", err.Error())
		return
	}
}
