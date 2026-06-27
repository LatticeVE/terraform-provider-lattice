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
			ID:            "kern-1",
			Name:          "alpine-3.24.1",
			Distro:        "alpine",
			DistroVersion: "3.24.1",
			Version:       "6.12.9",
			VmlinuzPath:   "/kernels/alpine/vmlinuz",
			InitramfsPath: "/kernels/alpine/initramfs",
			SizeBytes:     10_000_000,
			BuiltAt:       now,
		},
		{
			ID:            "kern-2",
			Name:          "ubuntu-26.04",
			Distro:        "ubuntu",
			DistroVersion: "26.04",
			Version:       "6.14.0",
			VmlinuzPath:   "/kernels/ubuntu/vmlinuz",
			InitramfsPath: "/kernels/ubuntu/initramfs",
			SizeBytes:     12_000_000,
			BuiltAt:       now.Add(-time.Hour),
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
	if kernels[0].DistroVersion != "3.24.1" {
		t.Errorf("expected DistroVersion=3.24.1, got %s", kernels[0].DistroVersion)
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
