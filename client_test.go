package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestClient_ListVMs(t *testing.T) {
	expectedVMs := []VM{
		{
			ID:     "vm-1",
			Name:   "test-vm-1",
			CPUs:   2,
			Memory: 2048,
			Status: StatusRunning,
		},
		{
			ID:     "vm-2",
			Name:   "test-vm-2",
			CPUs:   4,
			Memory: 4096,
			Status: StatusStopped,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/vm" {
			t.Errorf("expected path /vm, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected auth header Bearer test-key, got %s", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(expectedVMs)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	vms, err := client.ListVMs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(vms, expectedVMs) {
		t.Errorf("expected %v, got %v", expectedVMs, vms)
	}
}

func TestClient_GetVM(t *testing.T) {
	expectedVMs := []VM{
		{ID: "vm-1", Name: "test-vm-1", Status: StatusRunning},
		{ID: "vm-2", Name: "test-vm-2", Status: StatusStopped},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expectedVMs)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	vm, err := client.GetVM("vm-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vm.Name != "test-vm-2" {
		t.Errorf("expected test-vm-2, got %s", vm.Name)
	}
}

func TestClient_GetVM_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]VM{})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if _, err := client.GetVM("missing"); err == nil {
		t.Fatal("expected error for missing VM, got nil")
	}
}

func TestClient_CreateVM(t *testing.T) {
	inputName := "new-vm"
	inputCPUs := 2
	inputMemory := 4096
	inputBootDiskGB := 30
	inputDiskInterface := "scsi"
	inputISO := "/path/to/iso"
	inputCloudInit := &CloudInitConfig{
		UserData: "user-data",
		MetaData: "meta-data",
	}
	inputDisks := []ExtraDisk{{SizeGB: 15, Interface: "scsi"}}
	inputNICs := []NIC{{Bridge: "br0", Model: "e1000"}}

	expectedVM := VM{
		ID:            "new-uuid",
		Name:          inputName,
		CPUs:          inputCPUs,
		Memory:        inputMemory,
		Status:        StatusStopped,
		ISOPath:       inputISO,
		CloudInit:     inputCloudInit,
		ExtraDisks:    []ExtraDisk{{Index: 1, SizeGB: 15, DiskPath: "/vm/disk1.qcow2", Interface: "scsi"}},
		NICs:          []NIC{{Index: 1, Bridge: "br0", MACAddr: "aa:bb:cc:dd:ee:ff", DeviceID: "nic1", Model: "e1000"}},
		DiskPath:      "/vm/disk.qcow2",
		BootDiskGB:    inputBootDiskGB,
		DiskInterface: inputDiskInterface,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/vm" {
			t.Errorf("expected path /vm, got %s", r.URL.Path)
		}

		var req vmCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if req.Name != inputName || req.CPUs != inputCPUs || req.MemoryMB != inputMemory || req.ISOPath != inputISO || req.BootDiskGB != inputBootDiskGB || req.DiskInterface != inputDiskInterface {
			t.Errorf("unexpected request payload: %+v", req)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(expectedVM)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	vm, err := client.CreateVM(inputName, inputCPUs, inputMemory, inputBootDiskGB, inputDiskInterface, inputISO, inputCloudInit, inputDisks, inputNICs, "", "", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(*vm, expectedVM) {
		t.Errorf("expected %+v, got %+v", expectedVM, *vm)
	}
}

func TestClient_VMStateTransitions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req vmActionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if req.ID != "test-id" {
			t.Errorf("expected ID test-id, got %s", req.ID)
		}

		if r.URL.Path == "/vm/start" {
			w.WriteHeader(http.StatusNoContent)
		} else if r.URL.Path == "/vm/stop" {
			w.WriteHeader(http.StatusNoContent)
		} else {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)

	if err := client.StartVM("test-id"); err != nil {
		t.Errorf("start VM failed: %v", err)
	}

	if err := client.StopVM("test-id"); err != nil {
		t.Errorf("stop VM failed: %v", err)
	}
}

func TestClient_DeleteVM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}
		if r.URL.Path != "/vm/test-id" {
			t.Errorf("expected path /vm/test-id, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("stop"); got != "true" {
			t.Errorf("expected stop=true, got %q", got)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if err := client.DeleteVM("test-id", DeleteVMOptions{Stop: true}); err != nil {
		t.Errorf("delete VM failed: %v", err)
	}
}

func TestClient_DeleteVMForce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}
		if r.URL.Path != "/vm/test-id" {
			t.Errorf("expected path /vm/test-id, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("force"); got != "true" {
			t.Errorf("expected force=true, got %q", got)
		}
		if got := r.URL.Query().Get("stop"); got != "" {
			t.Errorf("expected stop to be omitted for force delete, got %q", got)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if err := client.DeleteVM("test-id", DeleteVMOptions{Force: true}); err != nil {
		t.Errorf("force delete VM failed: %v", err)
	}
}
