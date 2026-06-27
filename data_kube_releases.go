package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &KubeReleasesDataSource{}
var _ datasource.DataSourceWithConfigure = &KubeReleasesDataSource{}

type KubeReleasesDataSource struct {
	client *Client
}

type KubeReleasesModel struct {
	Releases types.List `tfsdk:"releases"`
}

var talosReleaseAttrTypes = map[string]attr.Type{
	"version":      types.StringType,
	"k8s_version":  types.StringType,
	"published_at": types.StringType,
}

func NewKubeReleasesDataSource() datasource.DataSource {
	return &KubeReleasesDataSource{}
}

func (d *KubeReleasesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kube_releases"
}

func (d *KubeReleasesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all available Talos/Kubernetes releases.",
		Attributes: map[string]schema.Attribute{
			"releases": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"version": schema.StringAttribute{
							Computed: true,
						},
						"k8s_version": schema.StringAttribute{
							Computed: true,
						},
						"published_at": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *KubeReleasesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *KubeReleasesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KubeReleasesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	releases, err := d.client.ListTalosReleases()
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Talos Releases", err.Error())
		return
	}

	releaseVals := make([]attr.Value, 0, len(releases))
	for _, rel := range releases {
		obj, diags := types.ObjectValue(talosReleaseAttrTypes, map[string]attr.Value{
			"version":      types.StringValue(rel.Version),
			"k8s_version":  types.StringValue(rel.K8sVersion),
			"published_at": types.StringValue(rel.PublishedAt.Format("2006-01-02T15:04:05Z")),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		releaseVals = append(releaseVals, obj)
	}

	listVal, diags := types.ListValue(types.ObjectType{AttrTypes: talosReleaseAttrTypes}, releaseVals)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Releases = listVal
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
