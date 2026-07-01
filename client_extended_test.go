package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ── ListImages ────────────────────────────────────────────────────────────────

func TestClient_ListImages(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	expected := []Image{
		{ID: "img-1", Name: "debian-12-generic-amd64", Distro: "debian", Version: "12", Arch: "amd64", Format: "qcow2", SizeBytes: 2_000_000_000, CreatedAt: now},
		{ID: "img-2", Name: "ubuntu-26.04-server-amd64", Distro: "ubuntu", Version: "26.04", Arch: "amd64", Format: "qcow2", SizeBytes: 3_000_000_000, CreatedAt: now.Add(-time.Hour)},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/images" {
			t.Errorf("expected /images, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	images, err := client.ListImages()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}
	if images[0].ID != "img-1" {
		t.Errorf("expected img-1, got %s", images[0].ID)
	}
	if images[1].Distro != "ubuntu" {
		t.Errorf("expected ubuntu distro, got %s", images[1].Distro)
	}
}

func TestClient_ListImages_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Image{})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	images, err := client.ListImages()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(images) != 0 {
		t.Errorf("expected empty slice, got %d images", len(images))
	}
}

func TestClient_ListImages_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	_, err := client.ListImages()
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

// ── ListKernels ───────────────────────────────────────────────────────────────

func TestClient_ListKernels(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	expected := []Kernel{
		{
			ID:          "kern-1",
			Name:        "alpine-3.24.1",
			Distro:      "alpine",
			Version:     "6.12.9",
			Arch:        "amd64",
			VmlinuzPath: "/kernels/alpine/vmlinuz",
			SizeBytes:   10_000_000,
			CreatedAt:   now,
		},
		{
			ID:          "kern-2",
			Name:        "ubuntu-26.04",
			Distro:      "ubuntu",
			Version:     "6.14.0",
			Arch:        "arm64",
			VmlinuzPath: "/kernels/ubuntu/vmlinuz",
			SizeBytes:   12_000_000,
			CreatedAt:   now.Add(-time.Hour),
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/kernels" {
			t.Errorf("expected /kernels, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	kernels, err := client.ListKernels()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(kernels) != 2 {
		t.Fatalf("expected 2 kernels, got %d", len(kernels))
	}
	if kernels[0].Arch != "amd64" {
		t.Errorf("expected Arch=amd64, got %s", kernels[0].Arch)
	}
	if kernels[1].Version != "6.14.0" {
		t.Errorf("expected Version=6.14.0, got %s", kernels[1].Version)
	}
}

func TestClient_ListKernels_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Kernel{})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	kernels, err := client.ListKernels()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(kernels) != 0 {
		t.Errorf("expected empty slice, got %d kernels", len(kernels))
	}
}

func TestClient_ListKernels_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	_, err := client.ListKernels()
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

// ── isNotFound ────────────────────────────────────────────────────────────────

func TestIsNotFound_Cases(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"HTTP 404: not found", true},
		{"404", true},
		{"not found", true},
		{"does not exist", true},
		{"HTTP 500: internal server error", false},
		{"connection refused", false},
	}
	for _, tc := range cases {
		err := fmt.Errorf("%s", tc.msg)
		if got := isNotFound(err); got != tc.want {
			t.Errorf("isNotFound(%q) = %v, want %v", tc.msg, got, tc.want)
		}
	}
}

func TestIsNotFound_Nil(t *testing.T) {
	if isNotFound(nil) {
		t.Error("isNotFound(nil) should return false")
	}
}

// ── ListNodes ─────────────────────────────────────────────────────────────────

func TestClient_ListNodes(t *testing.T) {
	expected := []Node{
		{ID: "node-1", Name: "kvm-amd-01", Arch: "amd64", Status: "online", CPUs: 32, MemoryMB: 65536, StorageGB: 2000},
		{ID: "node-2", Name: "kvm-arm-01", Arch: "arm64", Status: "online", CPUs: 16, MemoryMB: 32768, StorageGB: 1000},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/nodes" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	nodes, err := client.ListNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Arch != "amd64" {
		t.Errorf("expected amd64, got %s", nodes[0].Arch)
	}
	if nodes[1].Arch != "arm64" {
		t.Errorf("expected arm64, got %s", nodes[1].Arch)
	}
}

func TestClient_ListNodes_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Node{})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	nodes, err := client.ListNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected empty, got %d", len(nodes))
	}
}

func TestClient_GetNode_ByName(t *testing.T) {
	nodes := []Node{
		{ID: "node-1", Name: "kvm-amd-01", Arch: "amd64", Status: "online"},
		{ID: "node-2", Name: "kvm-arm-01", Arch: "arm64", Status: "online"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(nodes)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	got, err := client.GetNode("kvm-arm-01")
	if err != nil {
		t.Fatalf("GetNode error: %v", err)
	}
	if got.Arch != "arm64" {
		t.Errorf("expected arm64, got %s", got.Arch)
	}
}

func TestClient_GetNode_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Node{})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	_, err := client.GetNode("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent node, got nil")
	}
}

// ── VPC client ────────────────────────────────────────────────────────────────

func TestClient_VPCCreateAndGet(t *testing.T) {
	vpc := VPC{
		ID:     "vpc-1",
		Name:   "test-vpc",
		CIDR:   "10.10.0.0/24",
		Bridge: "br-vpc1",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/vpc":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(vpc)
		case r.Method == http.MethodGet && r.URL.Path == "/vpc":
			// GetVPC calls ListVPCs internally
			_ = json.NewEncoder(w).Encode([]VPC{vpc})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)

	created, err := client.CreateVPC("test-vpc", "10.10.0.0/24", "")
	if err != nil {
		t.Fatalf("CreateVPC error: %v", err)
	}
	if created.ID != "vpc-1" {
		t.Errorf("expected vpc-1, got %s", created.ID)
	}

	got, err := client.GetVPC("vpc-1")
	if err != nil {
		t.Fatalf("GetVPC error: %v", err)
	}
	if got.Bridge != "br-vpc1" {
		t.Errorf("expected bridge br-vpc1, got %s", got.Bridge)
	}
}

func TestClient_VPCDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/vpc/vpc-1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if err := client.DeleteVPC("vpc-1"); err != nil {
		t.Errorf("DeleteVPC error: %v", err)
	}
}

// ── PublicIP client ───────────────────────────────────────────────────────────

func TestClient_PublicIPCreateAndDelete(t *testing.T) {
	pip := PublicIP{
		ID:     "pip-1",
		PoolID: "pool-1",
		IP:     "192.168.100.10",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/network/public-ips":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(pip)
		case r.Method == http.MethodDelete && r.URL.Path == "/network/public-ips/pip-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)

	created, err := client.AllocatePublicIP("pool-1", "")
	if err != nil {
		t.Fatalf("AllocatePublicIP error: %v", err)
	}
	if created.IP != "192.168.100.10" {
		t.Errorf("expected 192.168.100.10, got %s", created.IP)
	}

	if err := client.ReleasePublicIP("pip-1"); err != nil {
		t.Errorf("ReleasePublicIP error: %v", err)
	}
}

// ── Kernel catalog ────────────────────────────────────────────────────────────

func TestClient_ListKernelCatalog(t *testing.T) {
	expected := []KernelCatalogEntry{
		{
			ID:            "firecracker-6.1.141-amd64",
			Name:          "Firecracker Kernel 6.1.141 (amd64)",
			Distro:        "firecracker",
			Version:       "6.1.141",
			Arch:          "amd64",
			VmlinuzURL:    "https://example.com/vmlinuz-6.1.141",
			VmlinuzSizeMB: 8,
			Imported:      false,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/kernel-catalog" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	entries, err := client.ListKernelCatalog()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "firecracker-6.1.141-amd64" {
		t.Errorf("expected 1 entry with id firecracker-6.1.141-amd64, got %+v", entries)
	}
}

func TestClient_ListKernelCatalog_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if _, err := client.ListKernelCatalog(); err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestClient_ImportKernelCatalogEntry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/kernel-catalog/firecracker-6.1.141-amd64/import" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if err := client.ImportKernelCatalogEntry("firecracker-6.1.141-amd64"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_ImportKernelCatalogEntry_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "import already in progress"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if err := client.ImportKernelCatalogEntry("firecracker-6.1.141-amd64"); err == nil {
		t.Fatal("expected error for 409 response, got nil")
	}
}

func TestClient_KernelCatalogStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/kernel-catalog/firecracker-6.1.141-amd64/status" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(kernelCatalogStatus{
			EntryID:  "firecracker-6.1.141-amd64",
			Status:   "done",
			Progress: 100,
			Imported: true,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	status, err := client.KernelCatalogStatus("firecracker-6.1.141-amd64")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.Imported || status.Status != "done" {
		t.Errorf("expected imported done status, got %+v", status)
	}
}

func TestClient_DeleteKernel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/kernels/kern-1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if err := client.DeleteKernel("kern-1"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ── Rootfs images ─────────────────────────────────────────────────────────────

func TestClient_ListRootfsImages(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	expected := []RootfsImage{
		{
			ID:         "rootfs-1",
			Name:       "k3s v1.32.0+k3s1 rootfs (amd64)",
			Arch:       "amd64",
			RootfsPath: "/var/lib/latticeve/rootfs/rootfs-1-k3s-v1.32.0+k3s1-amd64.ext4",
			SizeBytes:  500_000_000,
			SHA256:     "deadbeef",
			Source:     "latticeve-k3s-images",
			Version:    "v1.32.0+k3s1",
			CreatedAt:  now,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/rootfs-images" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	images, err := client.ListRootfsImages()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(images) != 1 || images[0].Source != "latticeve-k3s-images" {
		t.Errorf("expected 1 k3s-sourced image, got %+v", images)
	}
}

func TestClient_ListRootfsImages_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if _, err := client.ListRootfsImages(); err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestClient_DeleteRootfsImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rootfs-images/rootfs-1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if err := client.DeleteRootfsImage("rootfs-1"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_DiscoverK3sRootfs(t *testing.T) {
	expected := []K3sRootfsDiscoveryEntry{
		{Version: "v1.32.0+k3s1", Arch: "amd64", DownloadURL: "https://github.com/LatticeVE/latticeve-k3s-images/releases/download/k3s-v1.32.0%2Bk3s1-r1/k3s-v1.32.0+k3s1-amd64.ext4", SizeBytes: 500_000_000},
		{Version: "v1.32.0+k3s1", Arch: "arm64", DownloadURL: "https://github.com/LatticeVE/latticeve-k3s-images/releases/download/k3s-v1.32.0%2Bk3s1-r1/k3s-v1.32.0+k3s1-arm64.ext4", SizeBytes: 480_000_000},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/rootfs-images/discover/k3s" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	entries, err := client.DiscoverK3sRootfs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Arch != "amd64" || entries[1].Arch != "arm64" {
		t.Errorf("expected amd64 then arm64, got %+v", entries)
	}
}

func TestClient_DiscoverK3sRootfs_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no rootfs assets found", http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if _, err := client.DiscoverK3sRootfs(); err == nil {
		t.Fatal("expected error for 502 response, got nil")
	}
}

func TestClient_ImportK3sRootfs(t *testing.T) {
	entry := K3sRootfsDiscoveryEntry{
		Version:     "v1.32.0+k3s1",
		Arch:        "amd64",
		DownloadURL: "https://github.com/LatticeVE/latticeve-k3s-images/releases/download/k3s-v1.32.0%2Bk3s1-r1/k3s-v1.32.0+k3s1-amd64.ext4",
		SizeBytes:   500_000_000,
	}
	want := RootfsImage{
		ID:        "rootfs-2",
		Name:      "k3s v1.32.0+k3s1 rootfs (amd64)",
		Arch:      "amd64",
		Source:    "latticeve-k3s-images",
		Version:   "v1.32.0+k3s1",
		SizeBytes: 500_000_000,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rootfs-images/discover/k3s/import" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var got K3sRootfsDiscoveryEntry
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if got != entry {
			t.Errorf("expected request body %+v, got %+v", entry, got)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	img, err := client.ImportK3sRootfs(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if img.ID != "rootfs-2" || img.Version != "v1.32.0+k3s1" {
		t.Errorf("expected rootfs-2 v1.32.0+k3s1, got %+v", img)
	}
}

func TestClient_ImportK3sRootfs_RejectsNonDiscoveredEntry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "rootfs image must match a discovered latticeve-k3s-images release asset"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	_, err := client.ImportK3sRootfs(K3sRootfsDiscoveryEntry{Version: "bogus", Arch: "amd64", DownloadURL: "https://example.com/bogus.ext4"})
	if err == nil {
		t.Fatal("expected error for 400 response, got nil")
	}
}
