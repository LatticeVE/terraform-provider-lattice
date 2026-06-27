package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &KubeClusterResource{}
var _ resource.ResourceWithConfigure = &KubeClusterResource{}

type KubeClusterResource struct {
	client *Client
}

type KubeClusterResourceModel struct {
	ID             types.String     `tfsdk:"id"`
	Name           types.String     `tfsdk:"name"`
	TalosImage     types.String     `tfsdk:"talos_image"`
	TalosVersion   types.String     `tfsdk:"talos_version"`
	K8sVersion     types.String     `tfsdk:"k8s_version"`
	CNI            types.String     `tfsdk:"cni"`
	LBMode         types.String     `tfsdk:"lb_mode"`
	PoolID         types.String     `tfsdk:"pool_id"`
	CPCount        types.Int64      `tfsdk:"cp_count"`
	WorkerCount    types.Int64      `tfsdk:"worker_count"`
	CPVCPUs        types.Int64      `tfsdk:"cp_vcpus"`
	CPMemoryMB     types.Int64      `tfsdk:"cp_memory_mb"`
	CPDiskGB       types.Int64      `tfsdk:"cp_disk_gb"`
	WorkerVCPUs    types.Int64      `tfsdk:"worker_vcpus"`
	WorkerMemoryMB types.Int64      `tfsdk:"worker_memory_mb"`
	WorkerDiskGB   types.Int64      `tfsdk:"worker_disk_gb"`
	Status         types.String     `tfsdk:"status"`
	Endpoint       types.String     `tfsdk:"endpoint"`
	PublicIP       types.String     `tfsdk:"public_ip"`
	VPCID          types.String     `tfsdk:"vpc_id"`
	VPCCIDR        types.String     `tfsdk:"vpc_cidr"`
	Kubeconfig     types.String     `tfsdk:"kubeconfig"`
	Talosconfig    types.String     `tfsdk:"talosconfig"`
	Nodes          []KubeNodeModel  `tfsdk:"nodes"`
}

type KubeNodeModel struct {
	ID     types.String `tfsdk:"id"`
	VMID   types.String `tfsdk:"vm_id"`
	Role   types.String `tfsdk:"role"`
	IP     types.String `tfsdk:"ip"`
	Status types.String `tfsdk:"status"`
}

func NewKubeClusterResource() resource.Resource {
	return &KubeClusterResource{}
}

func (r *KubeClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kube_cluster"
}

func (r *KubeClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a LatticeVE Kubernetes cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique UUID of the Kubernetes cluster.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Kubernetes cluster.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"talos_image": schema.StringAttribute{
				MarkdownDescription: "Path to the Talos disk image on the host.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"talos_version": schema.StringAttribute{
				MarkdownDescription: "Talos version string, e.g. \"v1.9.0\". Can be upgraded in-place.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"k8s_version": schema.StringAttribute{
				MarkdownDescription: "Kubernetes version string, e.g. \"v1.32.0\". Can be upgraded in-place.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cni": schema.StringAttribute{
				MarkdownDescription: "CNI plugin to use: \"flannel\", \"cilium\", or \"none\".",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"lb_mode": schema.StringAttribute{
				MarkdownDescription: "Load-balancer mode: \"ccm\", \"metallb\", or \"cilium\".",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"pool_id": schema.StringAttribute{
				MarkdownDescription: "Public IP pool ID for the control plane endpoint.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cp_count": schema.Int64Attribute{
				MarkdownDescription: "Number of control plane nodes.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"worker_count": schema.Int64Attribute{
				MarkdownDescription: "Number of worker nodes. Can be scaled without replacement.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"cp_vcpus": schema.Int64Attribute{
				MarkdownDescription: "vCPUs per control plane node.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"cp_memory_mb": schema.Int64Attribute{
				MarkdownDescription: "Memory in MB per control plane node.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"cp_disk_gb": schema.Int64Attribute{
				MarkdownDescription: "Disk size in GB per control plane node.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"worker_vcpus": schema.Int64Attribute{
				MarkdownDescription: "vCPUs per worker node.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"worker_memory_mb": schema.Int64Attribute{
				MarkdownDescription: "Memory in MB per worker node.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"worker_disk_gb": schema.Int64Attribute{
				MarkdownDescription: "Disk size in GB per worker node.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Cluster status: \"provisioning\", \"ready\", \"failed\", or \"deleting\".",
				Computed:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Kubernetes API endpoint URL.",
				Computed:            true,
			},
			"public_ip": schema.StringAttribute{
				MarkdownDescription: "Public IP address assigned to the control plane, if pool_id was provided.",
				Computed:            true,
			},
			"vpc_id": schema.StringAttribute{
				MarkdownDescription: "VPC ID of the cluster network.",
				Computed:            true,
			},
			"vpc_cidr": schema.StringAttribute{
				MarkdownDescription: "VPC CIDR block of the cluster network.",
				Computed:            true,
			},
			"kubeconfig": schema.StringAttribute{
				MarkdownDescription: "Kubeconfig YAML for authenticating to the cluster.",
				Computed:            true,
				Sensitive:           true,
			},
			"talosconfig": schema.StringAttribute{
				MarkdownDescription: "Talosconfig YAML for managing Talos nodes.",
				Computed:            true,
				Sensitive:           true,
			},
			"nodes": schema.ListNestedAttribute{
				MarkdownDescription: "List of nodes in the cluster.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Node UUID.",
							Computed:            true,
						},
						"vm_id": schema.StringAttribute{
							MarkdownDescription: "VM UUID backing this node.",
							Computed:            true,
						},
						"role": schema.StringAttribute{
							MarkdownDescription: "Node role: \"controlplane\" or \"worker\".",
							Computed:            true,
						},
						"ip": schema.StringAttribute{
							MarkdownDescription: "Node IP address.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "Node status.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (r *KubeClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *KubeClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan KubeClusterResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := KubeCreateRequest{
		Name:       plan.Name.ValueString(),
		TalosImage: plan.TalosImage.ValueString(),
	}

	if !plan.CPCount.IsNull() && !plan.CPCount.IsUnknown() {
		createReq.CPCount = int(plan.CPCount.ValueInt64())
	}
	if !plan.WorkerCount.IsNull() && !plan.WorkerCount.IsUnknown() {
		createReq.WorkerCount = int(plan.WorkerCount.ValueInt64())
	}
	if !plan.CPVCPUs.IsNull() && !plan.CPVCPUs.IsUnknown() {
		createReq.CPVCPUs = int(plan.CPVCPUs.ValueInt64())
	}
	if !plan.CPMemoryMB.IsNull() && !plan.CPMemoryMB.IsUnknown() {
		createReq.CPMemoryMB = int(plan.CPMemoryMB.ValueInt64())
	}
	if !plan.CPDiskGB.IsNull() && !plan.CPDiskGB.IsUnknown() {
		createReq.CPDiskGB = int(plan.CPDiskGB.ValueInt64())
	}
	if !plan.WorkerVCPUs.IsNull() && !plan.WorkerVCPUs.IsUnknown() {
		createReq.WorkerVCPUs = int(plan.WorkerVCPUs.ValueInt64())
	}
	if !plan.WorkerMemoryMB.IsNull() && !plan.WorkerMemoryMB.IsUnknown() {
		createReq.WorkerMemMB = int(plan.WorkerMemoryMB.ValueInt64())
	}
	if !plan.WorkerDiskGB.IsNull() && !plan.WorkerDiskGB.IsUnknown() {
		createReq.WorkerDiskGB = int(plan.WorkerDiskGB.ValueInt64())
	}
	if !plan.CNI.IsNull() && !plan.CNI.IsUnknown() {
		createReq.CNI = plan.CNI.ValueString()
	}
	if !plan.LBMode.IsNull() && !plan.LBMode.IsUnknown() {
		createReq.LBMode = plan.LBMode.ValueString()
	}
	if !plan.PoolID.IsNull() && !plan.PoolID.IsUnknown() {
		createReq.PoolID = plan.PoolID.ValueString()
	}
	if !plan.TalosVersion.IsNull() && !plan.TalosVersion.IsUnknown() {
		createReq.TalosVersion = plan.TalosVersion.ValueString()
	}
	if !plan.K8sVersion.IsNull() && !plan.K8sVersion.IsUnknown() {
		createReq.K8sVersion = plan.K8sVersion.ValueString()
	}

	cluster, err := r.client.CreateKubeCluster(createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Kubernetes Cluster", err.Error())
		return
	}

	clusterID := cluster.ID

	pollCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	for {
		select {
		case <-pollCtx.Done():
			resp.Diagnostics.AddError("Timeout", "cluster did not become ready within 30 minutes")
			return
		case <-time.After(10 * time.Second):
			cluster, err = r.client.GetKubeCluster(clusterID)
			if err != nil {
				resp.Diagnostics.AddError("Error Polling Kubernetes Cluster", err.Error())
				return
			}
			if cluster.Status == "ready" {
				goto ready
			}
			if cluster.Status == "failed" {
				resp.Diagnostics.AddError(
					"Kubernetes Cluster Provisioning Failed",
					fmt.Sprintf("cluster %s failed: %s", clusterID, cluster.ErrorMsg),
				)
				return
			}
		}
	}

ready:
	kubeconfig, err := r.client.GetKubeconfig(clusterID)
	if err != nil {
		resp.Diagnostics.AddError("Error Fetching Kubeconfig", err.Error())
		return
	}

	talosconfig, err := r.client.GetTalosconfig(clusterID)
	if err != nil {
		resp.Diagnostics.AddError("Error Fetching Talosconfig", err.Error())
		return
	}

	kubeClusterToState(cluster, kubeconfig, talosconfig, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *KubeClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state KubeClusterResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cluster, err := r.client.GetKubeCluster(state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Kubernetes Cluster", err.Error())
		return
	}

	kubeconfig := state.Kubeconfig.ValueString()
	talosconfig := state.Talosconfig.ValueString()

	if cluster.Status != "provisioning" {
		kubeconfig, err = r.client.GetKubeconfig(cluster.ID)
		if err != nil {
			resp.Diagnostics.AddError("Error Fetching Kubeconfig", err.Error())
			return
		}

		talosconfig, err = r.client.GetTalosconfig(cluster.ID)
		if err != nil {
			resp.Diagnostics.AddError("Error Fetching Talosconfig", err.Error())
			return
		}
	}

	kubeClusterToState(cluster, kubeconfig, talosconfig, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *KubeClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state KubeClusterResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	patchReq := KubePatchRequest{}

	if !plan.WorkerCount.Equal(state.WorkerCount) {
		wc := int(plan.WorkerCount.ValueInt64())
		patchReq.WorkerCount = &wc
	}
	if !plan.TalosVersion.Equal(state.TalosVersion) && !plan.TalosVersion.IsNull() && !plan.TalosVersion.IsUnknown() {
		patchReq.TalosVersion = plan.TalosVersion.ValueString()
	}
	if !plan.K8sVersion.Equal(state.K8sVersion) && !plan.K8sVersion.IsNull() && !plan.K8sVersion.IsUnknown() {
		patchReq.K8sVersion = plan.K8sVersion.ValueString()
	}

	cluster, err := r.client.PatchKubeCluster(state.ID.ValueString(), patchReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Kubernetes Cluster", err.Error())
		return
	}

	kubeconfig := state.Kubeconfig.ValueString()
	talosconfig := state.Talosconfig.ValueString()

	kubeClusterToState(cluster, kubeconfig, talosconfig, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *KubeClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state KubeClusterResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteKubeCluster(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Kubernetes Cluster", err.Error())
		return
	}
}

func kubeClusterToState(cluster *KubeCluster, kubeconfig, talosconfig string, state *KubeClusterResourceModel) {
	state.ID = types.StringValue(cluster.ID)
	state.Name = types.StringValue(cluster.Name)
	state.Status = types.StringValue(cluster.Status)
	state.TalosVersion = types.StringValue(cluster.TalosVersion)
	state.K8sVersion = types.StringValue(cluster.K8sVersion)
	state.CNI = types.StringValue(cluster.CNI)
	state.LBMode = types.StringValue(cluster.LBMode)
	state.Endpoint = types.StringValue(cluster.Endpoint)
	state.PublicIP = types.StringValue(cluster.PublicIP)
	state.VPCID = types.StringValue(cluster.VPCID)
	state.VPCCIDR = types.StringValue(cluster.VPCCIDR)
	state.CPCount = types.Int64Value(int64(cluster.CPCount))
	state.WorkerCount = types.Int64Value(int64(cluster.WorkerCount))
	state.CPVCPUs = types.Int64Value(int64(cluster.CPVCPUs))
	state.CPMemoryMB = types.Int64Value(int64(cluster.CPMemoryMB))
	state.CPDiskGB = types.Int64Value(int64(cluster.CPDiskGB))
	state.WorkerVCPUs = types.Int64Value(int64(cluster.WorkerVCPUs))
	state.WorkerMemoryMB = types.Int64Value(int64(cluster.WorkerMemMB))
	state.WorkerDiskGB = types.Int64Value(int64(cluster.WorkerDiskGB))
	state.Kubeconfig = types.StringValue(kubeconfig)
	state.Talosconfig = types.StringValue(talosconfig)

	state.Nodes = make([]KubeNodeModel, len(cluster.Nodes))
	for i, n := range cluster.Nodes {
		state.Nodes[i] = KubeNodeModel{
			ID:     types.StringValue(n.ID),
			VMID:   types.StringValue(n.VMID),
			Role:   types.StringValue(n.Role),
			IP:     types.StringValue(n.IP),
			Status: types.StringValue(n.Status),
		}
	}
}
