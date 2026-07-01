package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// This file rounds out client_extended_test.go with the client methods that
// had no coverage at all: VPC sub-resources, LB certificates, public IP
// pools/IPs, storage backends/volumes, kube clusters, security groups, and
// IPAM pools. One happy-path test per method, mirroring the existing
// httptest.NewServer style.

// ── VPC sub-resources ─────────────────────────────────────────────────────────

func TestClient_UpdateVPC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/vpc/vpc-1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(VPC{ID: "vpc-1", Name: "renamed"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	vpc, err := client.UpdateVPC("vpc-1", "renamed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vpc.Name != "renamed" {
		t.Errorf("expected name=renamed, got %s", vpc.Name)
	}
}

func TestClient_AddPortForward(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/vpc/vpc-1/portforward" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(PortForward{ID: "pf-1", Proto: "tcp", ExtPort: 8080, DestIP: "10.0.0.5", DestPort: 80})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	pf, err := client.AddPortForward("vpc-1", PortForward{Proto: "tcp", ExtPort: 8080, DestIP: "10.0.0.5", DestPort: 80})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pf.ID != "pf-1" {
		t.Errorf("expected pf-1, got %s", pf.ID)
	}
}

func TestClient_RemovePortForward(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/vpc/vpc-1/portforward/pf-1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if err := client.RemovePortForward("vpc-1", "pf-1"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_AddFirewallRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/vpc/vpc-1/firewall" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(FirewallRule{ID: "fw-1", Direction: "ingress", Proto: "tcp", CIDR: "0.0.0.0/0", Action: "accept"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	rule, err := client.AddFirewallRule("vpc-1", FirewallRule{Direction: "ingress", Proto: "tcp", CIDR: "0.0.0.0/0", Action: "accept"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.ID != "fw-1" {
		t.Errorf("expected fw-1, got %s", rule.ID)
	}
}

func TestClient_RemoveFirewallRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/vpc/vpc-1/firewall/fw-1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if err := client.RemoveFirewallRule("vpc-1", "fw-1"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_SetFirewallDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/vpc/vpc-1/firewall/default" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["action"] != "drop" {
			t.Errorf("expected action=drop, got %v", body)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if err := client.SetFirewallDefault("vpc-1", "drop"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_AddLoadBalancer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/vpc/vpc-1/lb" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(LoadBalancer{ID: "lb-1", Name: "web", Port: 443, Protocol: "https"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	lb, err := client.AddLoadBalancer("vpc-1", LoadBalancer{Name: "web", Port: 443, Protocol: "https"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lb.ID != "lb-1" {
		t.Errorf("expected lb-1, got %s", lb.ID)
	}
}

func TestClient_RemoveLoadBalancer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/vpc/vpc-1/lb/lb-1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if err := client.RemoveLoadBalancer("vpc-1", "lb-1"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_GetVPCLoadBalancer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vpc" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]VPC{
			{ID: "vpc-1", LoadBalancers: []LoadBalancer{{ID: "lb-1", Name: "web"}}},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	lb, err := client.GetVPCLoadBalancer("vpc-1", "lb-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lb.Name != "web" {
		t.Errorf("expected name=web, got %s", lb.Name)
	}
}

func TestClient_GetVPCLoadBalancer_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]VPC{{ID: "vpc-1"}})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if _, err := client.GetVPCLoadBalancer("vpc-1", "missing"); err == nil {
		t.Fatal("expected error for missing load balancer, got nil")
	}
}

// ── LB certificates ───────────────────────────────────────────────────────────

func TestClient_LBCertificateCRUD(t *testing.T) {
	cert := LBCertificate{ID: "cert-1", Name: "web-cert"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/lb/certificates":
			_ = json.NewEncoder(w).Encode([]LBCertificate{cert})
		case r.Method == http.MethodGet && r.URL.Path == "/lb/certificates/cert-1":
			_ = json.NewEncoder(w).Encode(cert)
		case r.Method == http.MethodPost && r.URL.Path == "/lb/certificates":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(cert)
		case r.Method == http.MethodPut && r.URL.Path == "/lb/certificates/cert-1":
			_ = json.NewEncoder(w).Encode(LBCertificate{ID: "cert-1", Name: "web-cert-renewed"})
		case r.Method == http.MethodDelete && r.URL.Path == "/lb/certificates/cert-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)

	certs, err := client.ListLBCertificates()
	if err != nil || len(certs) != 1 {
		t.Fatalf("ListLBCertificates: err=%v certs=%+v", err, certs)
	}
	got, err := client.GetLBCertificate("cert-1")
	if err != nil || got.ID != "cert-1" {
		t.Fatalf("GetLBCertificate: err=%v got=%+v", err, got)
	}
	created, err := client.CreateLBCertificate(LBCertificateRequest{Name: "web-cert", CertPEM: "pem", KeyPEM: "key"})
	if err != nil || created.ID != "cert-1" {
		t.Fatalf("CreateLBCertificate: err=%v created=%+v", err, created)
	}
	updated, err := client.UpdateLBCertificate("cert-1", LBCertificateRequest{Name: "web-cert-renewed", CertPEM: "pem2", KeyPEM: "key2"})
	if err != nil || updated.Name != "web-cert-renewed" {
		t.Fatalf("UpdateLBCertificate: err=%v updated=%+v", err, updated)
	}
	if err := client.DeleteLBCertificate("cert-1"); err != nil {
		t.Fatalf("DeleteLBCertificate: %v", err)
	}
}

// ── Public IP pools / IPs ─────────────────────────────────────────────────────

func TestClient_PublicIPPoolCRUD(t *testing.T) {
	pool := PublicIPPool{ID: "pool-1", Name: "kube-pool", Interface: "eth0", CIDR: "192.168.100.0/27"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/network/public-ip-pools":
			_ = json.NewEncoder(w).Encode([]PublicIPPool{pool})
		case r.Method == http.MethodPost && r.URL.Path == "/network/public-ip-pools":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(pool)
		case r.Method == http.MethodDelete && r.URL.Path == "/network/public-ip-pools/pool-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)

	got, err := client.GetPublicIPPool("pool-1")
	if err != nil || got.ID != "pool-1" {
		t.Fatalf("GetPublicIPPool: err=%v got=%+v", err, got)
	}
	created, err := client.CreatePublicIPPool("kube-pool", "eth0", "192.168.100.0/27")
	if err != nil || created.ID != "pool-1" {
		t.Fatalf("CreatePublicIPPool: err=%v created=%+v", err, created)
	}
	if err := client.DeletePublicIPPool("pool-1"); err != nil {
		t.Fatalf("DeletePublicIPPool: %v", err)
	}
}

func TestClient_PublicIPPool_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]PublicIPPool{})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if _, err := client.GetPublicIPPool("missing"); err == nil {
		t.Fatal("expected error for missing pool, got nil")
	}
}

func TestClient_PublicIPCRUD(t *testing.T) {
	ip := PublicIP{ID: "ip-1", PoolID: "pool-1", IP: "192.168.100.10"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/network/public-ips":
			_ = json.NewEncoder(w).Encode([]PublicIP{ip})
		case r.Method == http.MethodPost && r.URL.Path == "/network/public-ips/ip-1/static-nat":
			_ = json.NewEncoder(w).Encode(PublicIP{ID: "ip-1", PoolID: "pool-1", IP: "192.168.100.10", PrivateIP: "10.0.0.5"})
		case r.Method == http.MethodDelete && r.URL.Path == "/network/public-ips/ip-1/static-nat":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)

	got, err := client.GetPublicIP("ip-1")
	if err != nil || got.ID != "ip-1" {
		t.Fatalf("GetPublicIP: err=%v got=%+v", err, got)
	}
	nat, err := client.EnableStaticNAT("ip-1", "10.0.0.5")
	if err != nil || nat.PrivateIP != "10.0.0.5" {
		t.Fatalf("EnableStaticNAT: err=%v nat=%+v", err, nat)
	}
	if err := client.DisableStaticNAT("ip-1"); err != nil {
		t.Fatalf("DisableStaticNAT: %v", err)
	}
}

// ── Storage backends / volumes ────────────────────────────────────────────────

func TestClient_StorageBackendCRUD(t *testing.T) {
	backend := StorageBackend{ID: "sb-1", Name: "linstor-default", Type: "linstor"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/storage/backends":
			_ = json.NewEncoder(w).Encode([]StorageBackend{backend})
		case r.Method == http.MethodPost && r.URL.Path == "/storage/backends":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(backend)
		case r.Method == http.MethodDelete && r.URL.Path == "/storage/backends/sb-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)

	got, err := client.GetStorageBackend("sb-1")
	if err != nil || got.ID != "sb-1" {
		t.Fatalf("GetStorageBackend: err=%v got=%+v", err, got)
	}
	created, err := client.CreateStorageBackend("linstor-default", "linstor", map[string]any{"controller": "linstor://10.0.0.1:3370"})
	if err != nil || created.ID != "sb-1" {
		t.Fatalf("CreateStorageBackend: err=%v created=%+v", err, created)
	}
	if err := client.DeleteStorageBackend("sb-1"); err != nil {
		t.Fatalf("DeleteStorageBackend: %v", err)
	}
}

func TestClient_StorageBackend_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]StorageBackend{})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if _, err := client.GetStorageBackend("missing"); err == nil {
		t.Fatal("expected error for missing backend, got nil")
	}
}

func TestClient_StorageVolumeCRUD(t *testing.T) {
	vol := StorageVolume{ID: "vol-1", Name: "prod-data-0", SizeBytes: 200 << 30, BackendID: "sb-1"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/storage/volumes":
			_ = json.NewEncoder(w).Encode([]StorageVolume{vol})
		case r.Method == http.MethodPost && r.URL.Path == "/storage/volumes":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(vol)
		case r.Method == http.MethodPut && r.URL.Path == "/storage/volumes/vol-1":
			_ = json.NewEncoder(w).Encode(StorageVolume{ID: "vol-1", Name: "prod-data-0", SizeBytes: 400 << 30, BackendID: "sb-1"})
		case r.Method == http.MethodDelete && r.URL.Path == "/storage/volumes/vol-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)

	got, err := client.GetStorageVolume("vol-1")
	if err != nil || got.ID != "vol-1" {
		t.Fatalf("GetStorageVolume: err=%v got=%+v", err, got)
	}
	created, err := client.CreateStorageVolume("prod-data-0", 200<<30, "sb-1")
	if err != nil || created.ID != "vol-1" {
		t.Fatalf("CreateStorageVolume: err=%v created=%+v", err, created)
	}
	resized, err := client.ResizeStorageVolume("vol-1", 400<<30)
	if err != nil || resized.SizeBytes != 400<<30 {
		t.Fatalf("ResizeStorageVolume: err=%v resized=%+v", err, resized)
	}
	if err := client.DeleteStorageVolume("vol-1"); err != nil {
		t.Fatalf("DeleteStorageVolume: %v", err)
	}
}

func TestClient_StorageVolume_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]StorageVolume{})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if _, err := client.GetStorageVolume("missing"); err == nil {
		t.Fatal("expected error for missing volume, got nil")
	}
}

// ── Kube clusters ─────────────────────────────────────────────────────────────

func TestClient_KubeClusterCRUD(t *testing.T) {
	cluster := KubeCluster{ID: "kube-1", Name: "prod", Status: "ready", Runtime: "firecracker"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/kube/clusters":
			_ = json.NewEncoder(w).Encode([]KubeCluster{cluster})
		case r.Method == http.MethodGet && r.URL.Path == "/kube/clusters/kube-1":
			_ = json.NewEncoder(w).Encode(cluster)
		case r.Method == http.MethodPost && r.URL.Path == "/kube/clusters":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(cluster)
		case r.Method == http.MethodPatch && r.URL.Path == "/kube/clusters/kube-1":
			_ = json.NewEncoder(w).Encode(KubeCluster{ID: "kube-1", Name: "prod", Status: "ready", WorkerCount: 5})
		case r.Method == http.MethodDelete && r.URL.Path == "/kube/clusters/kube-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)

	clusters, err := client.ListKubeClusters()
	if err != nil || len(clusters) != 1 {
		t.Fatalf("ListKubeClusters: err=%v clusters=%+v", err, clusters)
	}
	got, err := client.GetKubeCluster("kube-1")
	if err != nil || got.ID != "kube-1" {
		t.Fatalf("GetKubeCluster: err=%v got=%+v", err, got)
	}
	created, err := client.CreateKubeCluster(KubeCreateRequest{Name: "prod"})
	if err != nil || created.ID != "kube-1" {
		t.Fatalf("CreateKubeCluster: err=%v created=%+v", err, created)
	}
	wc := 5
	patched, err := client.PatchKubeCluster("kube-1", KubePatchRequest{WorkerCount: &wc})
	if err != nil || patched.WorkerCount != 5 {
		t.Fatalf("PatchKubeCluster: err=%v patched=%+v", err, patched)
	}
	if err := client.DeleteKubeCluster("kube-1"); err != nil {
		t.Fatalf("DeleteKubeCluster: %v", err)
	}
}

func TestClient_GetKubeconfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/kube/clusters/kube-1/kubeconfig" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte("apiVersion: v1\nkind: Config\n"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	kubeconfig, err := client.GetKubeconfig("kube-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kubeconfig == "" {
		t.Error("expected non-empty kubeconfig")
	}
}

func TestClient_GetKubeconfig_NotReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "cluster not ready"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if _, err := client.GetKubeconfig("kube-1"); err == nil {
		t.Fatal("expected error for 503 response, got nil")
	}
}

func TestClient_KubePatchIncludesScaleAndUpgradeImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/kube/clusters/kube-1" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body KubePatchRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.CPCount == nil || *body.CPCount != 3 || body.WorkerCount == nil || *body.WorkerCount != 5 || body.RootfsID != "rootfs-2" {
			t.Fatalf("unexpected patch body: %+v", body)
		}
		_ = json.NewEncoder(w).Encode(KubeCluster{ID: "kube-1", Status: "upgrading"})
	}))
	defer server.Close()
	cp, workers := 3, 5
	_, err := NewClient(server.URL, "test-key", true).PatchKubeCluster("kube-1", KubePatchRequest{CPCount: &cp, WorkerCount: &workers, RootfsID: "rootfs-2"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_DeclarativeNetworkRelationships(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /vm/vm-1/security-groups", "POST /vm/vm-1/affinity-groups":
			w.WriteHeader(http.StatusNoContent)
		case "GET /vm/vm-1/security-groups":
			_ = json.NewEncoder(w).Encode([]SecurityGroup{{ID: "sg-1"}})
		case "GET /vm/vm-1/affinity-groups":
			_ = json.NewEncoder(w).Encode([]AffinityGroup{{ID: "ag-1"}})
		case "DELETE /vm/vm-1/security-groups/sg-1", "DELETE /vm/vm-1/affinity-groups/ag-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	c := NewClient(server.URL, "test-key", true)
	if err := c.AssignVMSecurityGroup("vm-1", "sg-1"); err != nil {
		t.Fatal(err)
	}
	if groups, err := c.ListVMSecurityGroups("vm-1"); err != nil || len(groups) != 1 {
		t.Fatalf("groups=%v err=%v", groups, err)
	}
	if err := c.UnassignVMSecurityGroup("vm-1", "sg-1"); err != nil {
		t.Fatal(err)
	}
	if err := c.AssignVMAffinityGroup("vm-1", "ag-1"); err != nil {
		t.Fatal(err)
	}
	if groups, err := c.ListVMAffinityGroups("vm-1"); err != nil || len(groups) != 1 {
		t.Fatalf("groups=%v err=%v", groups, err)
	}
	if err := c.UnassignVMAffinityGroup("vm-1", "ag-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 6 {
		t.Fatalf("calls=%v", calls)
	}
}

func TestClient_IPAMLeaseAndAffinityGroupCRUD(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /ipam/pools/pool-1/leases":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(IPAMLease{ID: "lease-1", PoolID: "pool-1", MAC: "02:00:00:00:00:01", IP: "10.0.0.10"})
		case "GET /ipam/pools/pool-1/leases":
			_ = json.NewEncoder(w).Encode([]IPAMLease{{ID: "lease-1", PoolID: "pool-1"}})
		case "DELETE /ipam/leases/lease-1", "DELETE /affinity-groups/ag-1":
			w.WriteHeader(http.StatusNoContent)
		case "POST /affinity-groups":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(AffinityGroup{ID: "ag-1", Name: "cp", Policy: "anti-affinity"})
		case "GET /affinity-groups":
			_ = json.NewEncoder(w).Encode([]AffinityGroup{{ID: "ag-1"}})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	c := NewClient(server.URL, "test-key", true)
	if _, err := c.CreateIPAMLease("pool-1", IPAMLease{MAC: "02:00:00:00:00:01", IP: "10.0.0.10"}); err != nil {
		t.Fatal(err)
	}
	if leases, err := c.ListIPAMLeases("pool-1"); err != nil || len(leases) != 1 {
		t.Fatalf("leases=%v err=%v", leases, err)
	}
	if err := c.DeleteIPAMLease("lease-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.CreateAffinityGroup("cp", "anti-affinity"); err != nil {
		t.Fatal(err)
	}
	if groups, err := c.ListAffinityGroups(); err != nil || len(groups) != 1 {
		t.Fatalf("groups=%v err=%v", groups, err)
	}
	if err := c.DeleteAffinityGroup("ag-1"); err != nil {
		t.Fatal(err)
	}
}

// ── Security groups ───────────────────────────────────────────────────────────

func TestClient_SecurityGroupCRUD(t *testing.T) {
	sg := SecurityGroup{ID: "sg-1", Name: "web"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/security-groups":
			_ = json.NewEncoder(w).Encode([]SecurityGroup{sg})
		case r.Method == http.MethodPost && r.URL.Path == "/security-groups":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(sg)
		case r.Method == http.MethodPost && r.URL.Path == "/security-groups/sg-1/rules":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(SGRule{ID: "rule-1", Direction: "ingress", Protocol: "tcp", PortFrom: 22, PortTo: 22, CIDR: "10.0.0.0/8", Action: "accept"})
		case r.Method == http.MethodDelete && r.URL.Path == "/security-groups/sg-1/rules/rule-1":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/security-groups/sg-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)

	got, err := client.GetSecurityGroup("sg-1")
	if err != nil || got.ID != "sg-1" {
		t.Fatalf("GetSecurityGroup: err=%v got=%+v", err, got)
	}
	created, err := client.CreateSecurityGroup("web", "")
	if err != nil || created.ID != "sg-1" {
		t.Fatalf("CreateSecurityGroup: err=%v created=%+v", err, created)
	}
	rule, err := client.AddSGRule("sg-1", SGRule{Direction: "ingress", Protocol: "tcp", PortFrom: 22, PortTo: 22, CIDR: "10.0.0.0/8", Action: "accept"})
	if err != nil || rule.ID != "rule-1" {
		t.Fatalf("AddSGRule: err=%v rule=%+v", err, rule)
	}
	if err := client.RemoveSGRule("sg-1", "rule-1"); err != nil {
		t.Fatalf("RemoveSGRule: %v", err)
	}
	if err := client.DeleteSecurityGroup("sg-1"); err != nil {
		t.Fatalf("DeleteSecurityGroup: %v", err)
	}
}

func TestClient_SecurityGroup_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]SecurityGroup{})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if _, err := client.GetSecurityGroup("missing"); err == nil {
		t.Fatal("expected error for missing security group, got nil")
	}
}

// ── IPAM pools ────────────────────────────────────────────────────────────────

func TestClient_IPAMPoolCRUD(t *testing.T) {
	pool := IPAMPool{ID: "ipam-1", Name: "main-pool", Bridge: "br-main", Subnet: "10.10.0.0/24"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/ipam/pools":
			_ = json.NewEncoder(w).Encode([]IPAMPool{pool})
		case r.Method == http.MethodPost && r.URL.Path == "/ipam/pools":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(pool)
		case r.Method == http.MethodDelete && r.URL.Path == "/ipam/pools/ipam-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)

	got, err := client.GetIPAMPool("ipam-1")
	if err != nil || got.ID != "ipam-1" {
		t.Fatalf("GetIPAMPool: err=%v got=%+v", err, got)
	}
	created, err := client.CreateIPAMPool(pool)
	if err != nil || created.ID != "ipam-1" {
		t.Fatalf("CreateIPAMPool: err=%v created=%+v", err, created)
	}
	if err := client.DeleteIPAMPool("ipam-1"); err != nil {
		t.Fatalf("DeleteIPAMPool: %v", err)
	}
}

func TestClient_IPAMPool_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]IPAMPool{})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", true)
	if _, err := client.GetIPAMPool("missing"); err == nil {
		t.Fatal("expected error for missing pool, got nil")
	}
}
