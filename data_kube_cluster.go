package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type KubeClusterDataSource struct{ client *Client }

type KubeClusterDataSourceModel struct {
	ID            types.String    `tfsdk:"id"`
	Name          types.String    `tfsdk:"name"`
	Status        types.String    `tfsdk:"status"`
	K8sVersion    types.String    `tfsdk:"k8s_version"`
	KernelID      types.String    `tfsdk:"kernel_id"`
	KernelVersion types.String    `tfsdk:"kernel_version"`
	RootfsID      types.String    `tfsdk:"rootfs_id"`
	Endpoint      types.String    `tfsdk:"endpoint"`
	PublicIP      types.String    `tfsdk:"public_ip"`
	VPCID         types.String    `tfsdk:"vpc_id"`
	VPCCIDR       types.String    `tfsdk:"vpc_cidr"`
	CPCount       types.Int64     `tfsdk:"cp_count"`
	WorkerCount   types.Int64     `tfsdk:"worker_count"`
	Kubeconfig    types.String    `tfsdk:"kubeconfig"`
	Nodes         []KubeNodeModel `tfsdk:"nodes"`
}

func NewKubeClusterDataSource() datasource.DataSource { return &KubeClusterDataSource{} }
func (d *KubeClusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kube_cluster"
}
func (d *KubeClusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	nodeAttrs := map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "vm_id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Computed: true},
		"role": schema.StringAttribute{Computed: true}, "ip": schema.StringAttribute{Computed: true}, "status": schema.StringAttribute{Computed: true},
		"kubelet_version": schema.StringAttribute{Computed: true}, "upgrade_error": schema.StringAttribute{Computed: true},
	}
	resp.Schema = schema.Schema{MarkdownDescription: "Looks up a LatticeKube cluster by ID or name.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Optional: true, Computed: true}, "name": schema.StringAttribute{Optional: true, Computed: true},
		"status": schema.StringAttribute{Computed: true}, "k8s_version": schema.StringAttribute{Computed: true},
		"kernel_id": schema.StringAttribute{Computed: true}, "kernel_version": schema.StringAttribute{Computed: true}, "rootfs_id": schema.StringAttribute{Computed: true},
		"endpoint": schema.StringAttribute{Computed: true}, "public_ip": schema.StringAttribute{Computed: true}, "vpc_id": schema.StringAttribute{Computed: true}, "vpc_cidr": schema.StringAttribute{Computed: true},
		"cp_count": schema.Int64Attribute{Computed: true}, "worker_count": schema.Int64Attribute{Computed: true},
		"kubeconfig": schema.StringAttribute{Computed: true, Sensitive: true},
		"nodes":      schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: nodeAttrs}},
	}}
}
func (d *KubeClusterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	d.client, ok = req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", fmt.Sprintf("Expected *Client, got %T", req.ProviderData))
	}
}
func (d *KubeClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state KubeClusterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var cluster *KubeCluster
	var err error
	if !state.ID.IsNull() && !state.ID.IsUnknown() && state.ID.ValueString() != "" {
		cluster, err = d.client.GetKubeCluster(state.ID.ValueString())
	} else if !state.Name.IsNull() && !state.Name.IsUnknown() && state.Name.ValueString() != "" {
		var clusters []KubeCluster
		clusters, err = d.client.ListKubeClusters()
		if err == nil {
			for i := range clusters {
				if clusters[i].Name == state.Name.ValueString() {
					cluster = &clusters[i]
					break
				}
			}
		}
		if err == nil && cluster == nil {
			err = fmt.Errorf("Kubernetes cluster %q not found", state.Name.ValueString())
		}
	} else {
		resp.Diagnostics.AddError("Missing Cluster Selector", "Set either id or name.")
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Kubernetes Cluster", err.Error())
		return
	}
	kubeconfig, err := d.client.GetKubeconfig(cluster.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error Fetching Kubeconfig", err.Error())
		return
	}
	state.ID = types.StringValue(cluster.ID)
	state.Name = types.StringValue(cluster.Name)
	state.Status = types.StringValue(cluster.Status)
	state.K8sVersion = types.StringValue(cluster.K8sVersion)
	state.KernelID = types.StringValue(cluster.KernelID)
	state.KernelVersion = types.StringValue(cluster.KernelVersion)
	state.RootfsID = types.StringValue(cluster.RootfsID)
	state.Endpoint = types.StringValue(cluster.Endpoint)
	state.PublicIP = types.StringValue(cluster.PublicIP)
	state.VPCID = types.StringValue(cluster.VPCID)
	state.VPCCIDR = types.StringValue(cluster.VPCCIDR)
	state.CPCount = types.Int64Value(int64(cluster.CPCount))
	state.WorkerCount = types.Int64Value(int64(cluster.WorkerCount))
	state.Kubeconfig = types.StringValue(kubeconfig)
	state.Nodes = make([]KubeNodeModel, len(cluster.Nodes))
	for i, n := range cluster.Nodes {
		state.Nodes[i] = KubeNodeModel{ID: types.StringValue(n.ID), VMID: types.StringValue(n.VMID), Name: types.StringValue(n.Name), Role: types.StringValue(n.Role), IP: types.StringValue(n.IP), Status: types.StringValue(n.Status), KubeletVersion: types.StringValue(n.KubeletVersion), UpgradeError: types.StringValue(n.UpgradeError)}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
