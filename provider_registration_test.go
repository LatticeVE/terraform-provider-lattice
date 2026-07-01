package main

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestProviderRegistersDeclarativeResources(t *testing.T) {
	p := &LatticeProvider{}
	want := map[string]bool{
		"lattice_vm": false, "lattice_vpc": false, "lattice_public_ip_pool": false, "lattice_public_ip": false,
		"lattice_storage_backend": false, "lattice_storage_volume": false, "lattice_kube_cluster": false,
		"lattice_security_group": false, "lattice_vm_security_group": false, "lattice_ipam_pool": false, "lattice_ipam_lease": false,
		"lattice_affinity_group": false, "lattice_vm_affinity_group": false, "lattice_lb_certificate": false,
		"lattice_vpc_load_balancer": false, "lattice_kernel_catalog_import": false, "lattice_k3s_rootfs_image": false, "lattice_k3s_kernel": false,
	}
	for _, factory := range p.Resources(context.Background()) {
		var resp resource.MetadataResponse
		factory().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "lattice"}, &resp)
		if _, ok := want[resp.TypeName]; !ok {
			t.Fatalf("unexpected resource %q", resp.TypeName)
		}
		want[resp.TypeName] = true
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("resource %s not registered", name)
		}
	}
}

func TestProviderRegistersDataSources(t *testing.T) {
	p := &LatticeProvider{}
	want := map[string]bool{"lattice_vm": false, "lattice_vpc": false, "lattice_public_ip_pools": false, "lattice_storage_backends": false, "lattice_kernel": false, "lattice_kernel_catalog": false, "lattice_image": false, "lattice_rootfs_image": false, "lattice_nodes": false, "lattice_kube_cluster": false}
	for _, factory := range p.DataSources(context.Background()) {
		var resp datasource.MetadataResponse
		factory().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "lattice"}, &resp)
		if _, ok := want[resp.TypeName]; !ok {
			t.Fatalf("unexpected data source %q", resp.TypeName)
		}
		want[resp.TypeName] = true
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("data source %s not registered", name)
		}
	}
}
