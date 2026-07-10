package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSelectK3sKernelDiscoveryEntry(t *testing.T) {
	entries := []K3sKernelDiscoveryEntry{
		{Name: "k3s-kernel-6.1.175-arm64", Version: "6.1.175", Arch: "arm64"},
		{Name: "k3s-kernel-6.1.175-amd64", Version: "6.1.175", Arch: "amd64"},
		{Name: "k3s-kernel-6.1.176-amd64", Version: "6.1.176", Arch: "amd64"},
	}

	got, ok := selectK3sKernelDiscoveryEntry(entries, "amd64", "6.1.175")
	if !ok || got.Name != "k3s-kernel-6.1.175-amd64" {
		t.Fatalf("select exact = %+v, %v", got, ok)
	}

	got, ok = selectK3sKernelDiscoveryEntry(entries, "amd64", "")
	if !ok || got.Name != "k3s-kernel-6.1.175-amd64" {
		t.Fatalf("select first arch match = %+v, %v", got, ok)
	}

	if _, ok := selectK3sKernelDiscoveryEntry(entries, "amd64", "6.1.999"); ok {
		t.Fatal("unexpected match for missing version")
	}
}

func TestFindExistingK3sKernelPrefersNameMatch(t *testing.T) {
	entry := K3sKernelDiscoveryEntry{Name: "k3s-kernel-6.1.175-amd64", Version: "6.1.175", Arch: "amd64"}
	kernels := []Kernel{
		{ID: "wrong-distro", Name: entry.Name, Distro: "firecracker", Version: entry.Version, Arch: entry.Arch},
		{ID: "version-match", Name: "different-name", Distro: "latticeve-k3s", Version: entry.Version, Arch: entry.Arch},
		{ID: "name-match", Name: entry.Name, Distro: "latticeve-k3s", Version: "different", Arch: entry.Arch},
	}

	got, ok := findExistingK3sKernel(kernels, entry)
	if !ok || got.ID != "name-match" {
		t.Fatalf("findExistingK3sKernel = %+v, %v; want name-match", got, ok)
	}
}

func TestFindExistingK3sKernelFallsBackToVersionMatch(t *testing.T) {
	entry := K3sKernelDiscoveryEntry{Name: "k3s-kernel-6.1.175-amd64", Version: "6.1.175", Arch: "amd64"}
	kernels := []Kernel{
		{ID: "wrong-arch", Name: entry.Name, Distro: "latticeve-k3s", Version: entry.Version, Arch: "arm64"},
		{ID: "version-match", Name: "renamed-kernel", Distro: "latticeve-k3s", Version: entry.Version, Arch: entry.Arch},
	}

	got, ok := findExistingK3sKernel(kernels, entry)
	if !ok || got.ID != "version-match" {
		t.Fatalf("findExistingK3sKernel = %+v, %v; want version-match", got, ok)
	}
}

func TestFindExistingK3sKernelNoMatch(t *testing.T) {
	entry := K3sKernelDiscoveryEntry{Name: "k3s-kernel-6.1.175-amd64", Version: "6.1.175", Arch: "amd64"}
	kernels := []Kernel{
		{ID: "generic", Name: entry.Name, Distro: "firecracker", Version: entry.Version, Arch: entry.Arch},
	}

	if got, ok := findExistingK3sKernel(kernels, entry); ok {
		t.Fatalf("unexpected match: %+v", got)
	}
}

func TestK3sKernelShouldDeleteOnDestroy(t *testing.T) {
	cases := []struct {
		name    string
		managed types.Bool
		want    bool
	}{
		{name: "managed true", managed: types.BoolValue(true), want: true},
		{name: "managed false", managed: types.BoolValue(false), want: false},
		{name: "legacy null", managed: types.BoolNull(), want: true},
		{name: "legacy unknown", managed: types.BoolUnknown(), want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := k3sKernelShouldDeleteOnDestroy(tc.managed); got != tc.want {
				t.Fatalf("k3sKernelShouldDeleteOnDestroy() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEnsureK3sKernelReusesExistingKernelWithoutImport(t *testing.T) {
	importCalled := false
	entry := K3sKernelDiscoveryEntry{
		Name:        "k3s-kernel-6.1.175-amd64",
		Version:     "6.1.175",
		Arch:        "amd64",
		DownloadURL: "https://example.invalid/k3s-kernel-6.1.175-amd64",
		SizeBytes:   1234,
	}
	existing := Kernel{
		ID:        "existing-kernel",
		Name:      entry.Name,
		Distro:    "latticeve-k3s",
		Version:   entry.Version,
		Arch:      entry.Arch,
		SizeBytes: entry.SizeBytes,
	}
	client := newK3sKernelTestClient(t, entry, []Kernel{existing}, &importCalled)

	got, downloadURL, managed, err := ensureK3sKernel(client, "amd64", "6.1.175")
	if err != nil {
		t.Fatalf("ensureK3sKernel returned error: %v", err)
	}
	if got.ID != existing.ID || downloadURL != entry.DownloadURL || managed {
		t.Fatalf("ensureK3sKernel = kernel=%+v url=%q managed=%v", got, downloadURL, managed)
	}
	if importCalled {
		t.Fatal("import endpoint was called even though an existing kernel matched")
	}
}

func TestEnsureK3sKernelImportsWhenMissing(t *testing.T) {
	importCalled := false
	entry := K3sKernelDiscoveryEntry{
		Name:        "k3s-kernel-6.1.175-amd64",
		Version:     "6.1.175",
		Arch:        "amd64",
		DownloadURL: "https://example.invalid/k3s-kernel-6.1.175-amd64",
		SizeBytes:   1234,
	}
	client := newK3sKernelTestClient(t, entry, nil, &importCalled)

	got, downloadURL, managed, err := ensureK3sKernel(client, "amd64", "6.1.175")
	if err != nil {
		t.Fatalf("ensureK3sKernel returned error: %v", err)
	}
	if got.ID != "imported-kernel" || downloadURL != entry.DownloadURL || !managed {
		t.Fatalf("ensureK3sKernel = kernel=%+v url=%q managed=%v", got, downloadURL, managed)
	}
	if !importCalled {
		t.Fatal("import endpoint was not called for missing kernel")
	}
}

func newK3sKernelTestClient(t *testing.T, entry K3sKernelDiscoveryEntry, kernels []Kernel, importCalled *bool) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/kernels/discover/k3s":
			_ = json.NewEncoder(w).Encode([]K3sKernelDiscoveryEntry{entry})
		case r.Method == http.MethodGet && r.URL.Path == "/kernels":
			_ = json.NewEncoder(w).Encode(kernels)
		case r.Method == http.MethodPost && r.URL.Path == "/kernels/discover/k3s/import":
			*importCalled = true
			var requested K3sKernelDiscoveryEntry
			if err := json.NewDecoder(r.Body).Decode(&requested); err != nil {
				t.Fatalf("decode import request: %v", err)
			}
			if requested.Name != entry.Name || requested.Arch != entry.Arch || requested.DownloadURL != entry.DownloadURL {
				t.Fatalf("unexpected import request: %+v", requested)
			}
			_ = json.NewEncoder(w).Encode(Kernel{
				ID:        "imported-kernel",
				Name:      entry.Name,
				Distro:    "latticeve-k3s",
				Version:   entry.Version,
				Arch:      entry.Arch,
				SizeBytes: entry.SizeBytes,
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return NewClient(server.URL, "test-key", true)
}
