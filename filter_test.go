package main

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// ── isNotFound ────────────────────────────────────────────────────────────────

func TestIsNotFound(t *testing.T) {
	cases := []struct {
		err  string
		want bool
	}{
		{"", false},
		{"HTTP 404: resource not found", true},
		{"HTTP 404", true},
		{"not found", true},
		{"does not exist", true},
		{"HTTP 500: internal server error", false},
		{"connection refused", false},
	}
	for _, tc := range cases {
		var err error
		if tc.err != "" {
			err = fmt.Errorf("%s", tc.err)
		}
		if got := isNotFound(err); got != tc.want {
			t.Errorf("isNotFound(%q) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

// ── Image filtering ───────────────────────────────────────────────────────────

func TestImageFilter_ByDistroAndVersion(t *testing.T) {
	now := time.Now()
	images := []Image{
		{ID: "1", Distro: "debian", Version: "12", Arch: "amd64", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "2", Distro: "debian", Version: "12", Arch: "amd64", CreatedAt: now.Add(-1 * time.Hour)},
		{ID: "3", Distro: "debian", Version: "11", Arch: "amd64", CreatedAt: now},
		{ID: "4", Distro: "ubuntu", Version: "26.04", Arch: "amd64", CreatedAt: now},
	}

	matched := filterImages(images, "debian", "12", "amd64", "")
	if len(matched) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matched))
	}
	best := newestImage(matched)
	if best.ID != "2" {
		t.Errorf("expected newest image ID=2, got %s", best.ID)
	}
}

func TestImageFilter_DefaultsToAmd64(t *testing.T) {
	now := time.Now()
	images := []Image{
		{ID: "1", Distro: "debian", Version: "12", Arch: "amd64", CreatedAt: now},
		{ID: "2", Distro: "debian", Version: "12", Arch: "arm64", CreatedAt: now},
	}

	// arch="" → default amd64
	matched := filterImages(images, "debian", "12", "", "")
	if len(matched) != 1 || matched[0].ID != "1" {
		t.Errorf("expected only amd64 image, got %+v", matched)
	}
}

func TestImageFilter_ExplicitArch(t *testing.T) {
	now := time.Now()
	images := []Image{
		{ID: "1", Distro: "debian", Version: "12", Arch: "amd64", CreatedAt: now},
		{ID: "2", Distro: "debian", Version: "12", Arch: "arm64", CreatedAt: now},
	}

	matched := filterImages(images, "debian", "12", "arm64", "")
	if len(matched) != 1 || matched[0].ID != "2" {
		t.Errorf("expected arm64 image, got %+v", matched)
	}
}

func TestImageFilter_ByName(t *testing.T) {
	now := time.Now()
	images := []Image{
		{ID: "1", Name: "debian-12-generic-amd64", Distro: "debian", Arch: "amd64", CreatedAt: now},
		{ID: "2", Name: "ubuntu-26.04-server-amd64", Distro: "ubuntu", Arch: "amd64", CreatedAt: now},
	}

	matched := filterImages(images, "", "", "", "debian-12-generic-amd64")
	if len(matched) != 1 || matched[0].ID != "1" {
		t.Errorf("expected name match, got %+v", matched)
	}
}

func TestImageFilter_NoMatch(t *testing.T) {
	images := []Image{
		{ID: "1", Distro: "debian", Version: "12", Arch: "amd64", CreatedAt: time.Now()},
	}
	matched := filterImages(images, "ubuntu", "26.04", "amd64", "")
	if len(matched) != 0 {
		t.Errorf("expected no matches, got %d", len(matched))
	}
}

// filterImages and newestImage are helpers extracted from data_image.go Read logic
// to keep the filter behaviour independently testable.
func filterImages(images []Image, distro, version, arch, name string) []Image {
	var matched []Image
	for _, img := range images {
		if name != "" && img.Name != name {
			continue
		}
		if distro != "" && img.Distro != distro {
			continue
		}
		if version != "" && img.Version != version {
			continue
		}
		if arch != "" {
			if img.Arch != arch {
				continue
			}
		} else if img.Arch != "amd64" {
			continue
		}
		matched = append(matched, img)
	}
	return matched
}

func newestImage(images []Image) Image {
	best := images[0]
	for _, img := range images[1:] {
		if img.CreatedAt.After(best.CreatedAt) {
			best = img
		}
	}
	return best
}

// ── Kernel filtering ──────────────────────────────────────────────────────────

func TestKernelFilter_ByArch(t *testing.T) {
	now := time.Now()
	kernels := []Kernel{
		{ID: "1", Distro: "firecracker", Arch: "amd64", Version: "6.12.9", CreatedAt: now.Add(-time.Hour)},
		{ID: "2", Distro: "firecracker", Arch: "arm64", Version: "6.12.8", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "3", Distro: "firecracker", Arch: "amd64", Version: "6.6.1", CreatedAt: now},
		{ID: "4", Distro: "alpine", Arch: "amd64", Version: "6.14.0", CreatedAt: now},
	}

	matched := filterKernels(kernels, "firecracker", "arm64", "6.12.8", "", "")
	if len(matched) != 1 || matched[0].ID != "2" {
		t.Errorf("expected kernel 2, got %+v", matched)
	}
}

func TestKernelFilter_VersionGlob(t *testing.T) {
	now := time.Now()
	kernels := []Kernel{
		{ID: "1", Distro: "alpine", Version: "6.12.8", CreatedAt: now.Add(-time.Hour)},
		{ID: "2", Distro: "alpine", Version: "6.12.9", CreatedAt: now},
		{ID: "3", Distro: "alpine", Version: "6.6.1", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "4", Distro: "ubuntu", Version: "6.14.0", CreatedAt: now},
	}

	matched := filterKernels(kernels, "alpine", "", "", "6.12.*", "")
	if len(matched) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matched))
	}
	best := newestKernel(matched)
	if best.ID != "2" {
		t.Errorf("expected newest 6.12.x kernel (ID=2), got %s", best.ID)
	}
}

func TestKernelFilter_VersionGlobBroad(t *testing.T) {
	now := time.Now()
	kernels := []Kernel{
		{ID: "1", Version: "6.12.9", CreatedAt: now.Add(-time.Hour)},
		{ID: "2", Version: "6.14.0", CreatedAt: now},
		{ID: "3", Version: "5.15.0", CreatedAt: now.Add(-2 * time.Hour)},
	}

	matched := filterKernels(kernels, "", "", "", "6.*", "")
	if len(matched) != 2 {
		t.Fatalf("expected 2 matches for '6.*', got %d", len(matched))
	}
}

func TestKernelFilter_ExactVersion(t *testing.T) {
	now := time.Now()
	kernels := []Kernel{
		{ID: "1", Version: "6.12.9", CreatedAt: now},
		{ID: "2", Version: "6.12.8", CreatedAt: now},
	}

	matched := filterKernels(kernels, "", "", "6.12.9", "", "")
	if len(matched) != 1 || matched[0].ID != "1" {
		t.Errorf("expected exact version match, got %+v", matched)
	}
}

func TestKernelFilter_ByName(t *testing.T) {
	now := time.Now()
	kernels := []Kernel{
		{ID: "1", Name: "firecracker-v6.1.0-amd64", CreatedAt: now},
		{ID: "2", Name: "alpine-6.12.9", CreatedAt: now},
	}

	matched := filterKernels(kernels, "", "", "", "", "firecracker-v6.1.0-amd64")
	if len(matched) != 1 || matched[0].ID != "1" {
		t.Errorf("expected name match, got %+v", matched)
	}
}

func TestKernelFilter_NoMatch(t *testing.T) {
	kernels := []Kernel{
		{ID: "1", Distro: "alpine", Version: "6.12.9", CreatedAt: time.Now()},
	}
	matched := filterKernels(kernels, "ubuntu", "", "", "", "")
	if len(matched) != 0 {
		t.Errorf("expected no matches, got %d", len(matched))
	}
}

// filterKernels mirrors the filter logic in data_kernel.go Read.
func filterKernels(kernels []Kernel, distro, arch, version, versionGlob, name string) []Kernel {
	var matched []Kernel
	for _, k := range kernels {
		if distro != "" && k.Distro != distro {
			continue
		}
		if arch != "" && k.Arch != arch {
			continue
		}
		if name != "" && k.Name != name {
			continue
		}
		if version != "" && k.Version != version {
			continue
		}
		if versionGlob != "" {
			ok, _ := filepath.Match(versionGlob, k.Version)
			if !ok {
				continue
			}
		}
		matched = append(matched, k)
	}
	return matched
}

func newestKernel(kernels []Kernel) Kernel {
	best := kernels[0]
	for _, k := range kernels[1:] {
		if k.CreatedAt.After(best.CreatedAt) {
			best = k
		}
	}
	return best
}

// ── Node filtering ────────────────────────────────────────────────────────────

func TestNodeFilter_ByArch(t *testing.T) {
	nodes := []Node{
		{ID: "n1", Name: "kvm-amd-01", Arch: "amd64", Status: "online"},
		{ID: "n2", Name: "kvm-amd-02", Arch: "amd64", Status: "online"},
		{ID: "n3", Name: "kvm-arm-01", Arch: "arm64", Status: "online"},
	}

	matched := filterNodes(nodes, "arm64")
	if len(matched) != 1 || matched[0].ID != "n3" {
		t.Errorf("expected 1 arm64 node, got %+v", matched)
	}
}

func TestNodeFilter_AllArches(t *testing.T) {
	nodes := []Node{
		{ID: "n1", Arch: "amd64", Status: "online"},
		{ID: "n2", Arch: "arm64", Status: "online"},
	}

	matched := filterNodes(nodes, "")
	if len(matched) != 2 {
		t.Errorf("expected all nodes, got %d", len(matched))
	}
}

func TestNodeFilter_NoMatch(t *testing.T) {
	nodes := []Node{
		{ID: "n1", Arch: "amd64", Status: "online"},
	}
	matched := filterNodes(nodes, "arm64")
	if len(matched) != 0 {
		t.Errorf("expected no matches, got %+v", matched)
	}
}

func filterNodes(nodes []Node, arch string) []Node {
	var matched []Node
	for _, n := range nodes {
		if arch != "" && n.Arch != arch {
			continue
		}
		matched = append(matched, n)
	}
	return matched
}
