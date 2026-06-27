package main

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type LatticeProvider struct {
	version string
}

type LatticeProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	APIKey   types.String `tfsdk:"api_key"`
	Insecure types.Bool   `tfsdk:"insecure"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &LatticeProvider{
			version: version,
		}
	}
}

func (p *LatticeProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "lattice"
	resp.Version = p.version
}

func (p *LatticeProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The endpoint of the LatticeVE controller (e.g. `https://localhost:8006`). Can also be set via the `LATTICE_ENDPOINT` environment variable.",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "The API Key for LatticeVE authentication. Can also be set via the `LATTICE_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"insecure": schema.BoolAttribute{
				MarkdownDescription: "Controls whether the client verifies the server's TLS certificate. Defaults to `true` to accommodate self-signed certs in homelabs.",
				Optional:            true,
			},
		},
	}
}

func (p *LatticeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data LatticeProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv("LATTICE_ENDPOINT")
	apiKey := os.Getenv("LATTICE_API_KEY")
	insecureStr := os.Getenv("LATTICE_INSECURE")

	if !data.Endpoint.IsNull() {
		endpoint = data.Endpoint.ValueString()
	}
	if !data.APIKey.IsNull() {
		apiKey = data.APIKey.ValueString()
	}

	insecure := true
	if !data.Insecure.IsNull() {
		insecure = data.Insecure.ValueBool()
	} else if insecureStr != "" {
		insecure = insecureStr == "true"
	}

	if endpoint == "" {
		resp.Diagnostics.AddError(
			"Missing Endpoint Configuration",
			"The LatticeVE endpoint is required. Please set it in the provider configuration or via the LATTICE_ENDPOINT environment variable.",
		)
		return
	}

	client := NewClient(endpoint, apiKey, insecure)

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *LatticeProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewVMResource,
		NewVPCResource,
		NewPublicIPPoolResource,
		NewPublicIPResource,
		NewStorageBackendResource,
		NewStorageVolumeResource,
		NewKubeClusterResource,
		NewSecurityGroupResource,
		NewIPAMPoolResource,
		NewLBCertificateResource,
		NewVPCLoadBalancerResource,
	}
}

func (p *LatticeProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewVMDataSource,
		NewVPCDataSource,
		NewKubeReleasesDataSource,
		NewPublicIPPoolsDataSource,
		NewStorageBackendsDataSource,
		NewKernelDataSource,
		NewImageDataSource,
		NewNodesDataSource,
	}
}
