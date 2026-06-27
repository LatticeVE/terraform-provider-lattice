package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "404") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "does not exist")
}

// ── Nodes ─────────────────────────────────────────────────────────────────────

type Node struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Arch          string `json:"arch"`
	Status        string `json:"status"`
	CPUs          int    `json:"cpus"`
	MemoryMB      int64  `json:"memory_mb"`
	MemoryUsedMB  int64  `json:"memory_used_mb"`
	StorageGB     int64  `json:"storage_gb"`
	StorageUsedGB int64  `json:"storage_used_gb"`
}

func (c *Client) ListNodes() ([]Node, error) {
	var nodes []Node
	if err := c.getJSON(fmt.Sprintf("%s/nodes", c.endpoint), &nodes); err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	return nodes, nil
}

func (c *Client) GetNode(id string) (*Node, error) {
	nodes, err := c.ListNodes()
	if err != nil {
		return nil, err
	}
	for _, n := range nodes {
		if n.ID == id || n.Name == id {
			return &n, nil
		}
	}
	return nil, fmt.Errorf("node %s not found", id)
}

// ── Image catalog ─────────────────────────────────────────────────────────────

type Image struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Distro      string    `json:"distro"`
	Version     string    `json:"version"`
	Arch        string    `json:"arch"`
	Format      string    `json:"format"`
	SizeBytes   int64     `json:"size_bytes"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (c *Client) ListImages() ([]Image, error) {
	var images []Image
	return images, c.getJSON(c.endpoint+"/images", &images)
}

// ── Kernel catalog ────────────────────────────────────────────────────────────

type Kernel struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Distro        string    `json:"distro"`
	DistroVersion string    `json:"distro_version,omitempty"`
	Version       string    `json:"version"`
	VmlinuzPath   string    `json:"vmlinuz_path"`
	InitramfsPath string    `json:"initramfs_path"`
	SourceURL     string    `json:"source_url,omitempty"`
	SizeBytes     int64     `json:"size_bytes"`
	BuiltAt       time.Time `json:"built_at"`
}

func (c *Client) ListKernels() ([]Kernel, error) {
	var kernels []Kernel
	return kernels, c.getJSON(c.endpoint+"/kernels", &kernels)
}

// ── Helper methods ────────────────────────────────────────────────────────────

func (c *Client) apiError(resp *http.Response) error {
	var e map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&e)
	if msg := e["error"]; msg != "" {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
	}
	return fmt.Errorf("HTTP %d", resp.StatusCode)
}

func (c *Client) getJSON(url string, out any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("building GET request for %s: %w", url, err)
	}
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.apiError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding GET %s response: %w", url, err)
	}
	return nil
}

func (c *Client) postJSON(url string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling POST body for %s: %w", url, err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("building POST request for %s: %w", url, err)
	}
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return c.apiError(resp)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decoding POST %s response: %w", url, err)
		}
	}
	return nil
}

func (c *Client) patchJSON(url string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling PATCH body for %s: %w", url, err)
	}
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("building PATCH request for %s: %w", url, err)
	}
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("PATCH %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.apiError(resp)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decoding PATCH %s response: %w", url, err)
		}
	}
	return nil
}

func (c *Client) putJSON(url string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling PUT body for %s: %w", url, err)
	}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("building PUT request for %s: %w", url, err)
	}
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("PUT %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.apiError(resp)
	}
	return nil
}

func (c *Client) deleteReq(url string) error {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("building DELETE request for %s: %w", url, err)
	}
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.apiError(resp)
	}
	return nil
}

// ── VPC ───────────────────────────────────────────────────────────────────────

type VPC struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	CIDR          string         `json:"cidr,omitempty"`
	CIDR6         string         `json:"cidr_v6,omitempty"`
	Bridge        string         `json:"bridge"`
	Gateway       string         `json:"gateway,omitempty"`
	GatewayV6     string         `json:"gateway_v6,omitempty"`
	Status        string         `json:"status"`
	DefaultAction string         `json:"default_action,omitempty"`
	PortForwards  []PortForward  `json:"port_forwards,omitempty"`
	FirewallRules []FirewallRule `json:"firewall_rules,omitempty"`
	LoadBalancers []LoadBalancer `json:"load_balancers,omitempty"`
}

type PortForward struct {
	ID       string `json:"id"`
	Proto    string `json:"proto"`
	ExtPort  int    `json:"ext_port"`
	DestIP   string `json:"dest_ip"`
	DestPort int    `json:"dest_port"`
	Desc     string `json:"desc,omitempty"`
}

type FirewallRule struct {
	ID        string `json:"id"`
	Direction string `json:"direction"`
	Proto     string `json:"proto"`
	Port      string `json:"port,omitempty"`
	CIDR      string `json:"cidr"`
	Action    string `json:"action"`
	Desc      string `json:"desc,omitempty"`
}

type LoadBalancer struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Port            int         `json:"port"`
	Protocol        string      `json:"protocol"`
	CertificateID   string      `json:"certificate_id,omitempty"`
	BackendProtocol string      `json:"backend_protocol,omitempty"`
	Backends        []LBBackend `json:"backends,omitempty"`
}

type LBBackend struct {
	ID      string `json:"id,omitempty"`
	Address string `json:"address"`
	Weight  int    `json:"weight,omitempty"`
}

type LBCertificate struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CertPEM     string    `json:"cert_pem,omitempty"`
	ChainPEM    string    `json:"chain_pem,omitempty"`
	Subject     string    `json:"subject,omitempty"`
	DNSNames    []string  `json:"dns_names,omitempty"`
	NotBefore   time.Time `json:"not_before"`
	NotAfter    time.Time `json:"not_after"`
	Fingerprint string    `json:"fingerprint,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type LBCertificateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CertPEM     string `json:"cert_pem"`
	KeyPEM      string `json:"key_pem"`
	ChainPEM    string `json:"chain_pem,omitempty"`
}

func (c *Client) ListVPCs() ([]VPC, error) {
	var vpcs []VPC
	if err := c.getJSON(fmt.Sprintf("%s/vpc", c.endpoint), &vpcs); err != nil {
		return nil, fmt.Errorf("list vpcs: %w", err)
	}
	return vpcs, nil
}

func (c *Client) GetVPC(id string) (*VPC, error) {
	vpcs, err := c.ListVPCs()
	if err != nil {
		return nil, err
	}
	for _, v := range vpcs {
		if v.ID == id {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("vpc with ID %s not found", id)
}

func (c *Client) CreateVPC(name, cidr, cidr6 string) (*VPC, error) {
	body := map[string]string{"name": name, "cidr": cidr, "cidr6": cidr6}
	var vpc VPC
	if err := c.postJSON(fmt.Sprintf("%s/vpc", c.endpoint), body, &vpc); err != nil {
		return nil, fmt.Errorf("create vpc: %w", err)
	}
	return &vpc, nil
}

func (c *Client) UpdateVPC(id, name string) (*VPC, error) {
	body := map[string]string{"name": name}
	var vpc VPC
	if err := c.patchJSON(fmt.Sprintf("%s/vpc/%s", c.endpoint, id), body, &vpc); err != nil {
		return nil, fmt.Errorf("update vpc %s: %w", id, err)
	}
	return &vpc, nil
}

func (c *Client) DeleteVPC(id string) error {
	if err := c.deleteReq(fmt.Sprintf("%s/vpc/%s", c.endpoint, id)); err != nil {
		return fmt.Errorf("delete vpc %s: %w", id, err)
	}
	return nil
}

func (c *Client) AddPortForward(vpcID string, pf PortForward) (*PortForward, error) {
	var result PortForward
	if err := c.postJSON(fmt.Sprintf("%s/vpc/%s/portforward", c.endpoint, vpcID), pf, &result); err != nil {
		return nil, fmt.Errorf("add port forward to vpc %s: %w", vpcID, err)
	}
	return &result, nil
}

func (c *Client) RemovePortForward(vpcID, pfID string) error {
	if err := c.deleteReq(fmt.Sprintf("%s/vpc/%s/portforward/%s", c.endpoint, vpcID, pfID)); err != nil {
		return fmt.Errorf("remove port forward %s from vpc %s: %w", pfID, vpcID, err)
	}
	return nil
}

func (c *Client) AddFirewallRule(vpcID string, rule FirewallRule) (*FirewallRule, error) {
	var result FirewallRule
	if err := c.postJSON(fmt.Sprintf("%s/vpc/%s/firewall", c.endpoint, vpcID), rule, &result); err != nil {
		return nil, fmt.Errorf("add firewall rule to vpc %s: %w", vpcID, err)
	}
	return &result, nil
}

func (c *Client) RemoveFirewallRule(vpcID, ruleID string) error {
	if err := c.deleteReq(fmt.Sprintf("%s/vpc/%s/firewall/%s", c.endpoint, vpcID, ruleID)); err != nil {
		return fmt.Errorf("remove firewall rule %s from vpc %s: %w", ruleID, vpcID, err)
	}
	return nil
}

func (c *Client) SetFirewallDefault(vpcID, action string) error {
	body := map[string]string{"action": action}
	if err := c.putJSON(fmt.Sprintf("%s/vpc/%s/firewall/default", c.endpoint, vpcID), body); err != nil {
		return fmt.Errorf("set firewall default for vpc %s: %w", vpcID, err)
	}
	return nil
}

func (c *Client) AddLoadBalancer(vpcID string, lb LoadBalancer) (*LoadBalancer, error) {
	var result LoadBalancer
	if err := c.postJSON(fmt.Sprintf("%s/vpc/%s/lb", c.endpoint, vpcID), lb, &result); err != nil {
		return nil, fmt.Errorf("add load balancer to vpc %s: %w", vpcID, err)
	}
	return &result, nil
}

func (c *Client) RemoveLoadBalancer(vpcID, lbID string) error {
	if err := c.deleteReq(fmt.Sprintf("%s/vpc/%s/lb/%s", c.endpoint, vpcID, lbID)); err != nil {
		return fmt.Errorf("remove load balancer %s from vpc %s: %w", lbID, vpcID, err)
	}
	return nil
}

func (c *Client) GetVPCLoadBalancer(vpcID, lbID string) (*LoadBalancer, error) {
	vpc, err := c.GetVPC(vpcID)
	if err != nil {
		return nil, err
	}
	for _, lb := range vpc.LoadBalancers {
		if lb.ID == lbID {
			return &lb, nil
		}
	}
	return nil, fmt.Errorf("load balancer with ID %s not found in vpc %s", lbID, vpcID)
}

// ── LB Certificates ──────────────────────────────────────────────────────────

func (c *Client) ListLBCertificates() ([]LBCertificate, error) {
	var certs []LBCertificate
	if err := c.getJSON(fmt.Sprintf("%s/lb/certificates", c.endpoint), &certs); err != nil {
		return nil, fmt.Errorf("list lb certificates: %w", err)
	}
	return certs, nil
}

func (c *Client) GetLBCertificate(id string) (*LBCertificate, error) {
	var cert LBCertificate
	if err := c.getJSON(fmt.Sprintf("%s/lb/certificates/%s", c.endpoint, id), &cert); err != nil {
		return nil, fmt.Errorf("get lb certificate %s: %w", id, err)
	}
	return &cert, nil
}

func (c *Client) CreateLBCertificate(req LBCertificateRequest) (*LBCertificate, error) {
	var cert LBCertificate
	if err := c.postJSON(fmt.Sprintf("%s/lb/certificates", c.endpoint), req, &cert); err != nil {
		return nil, fmt.Errorf("create lb certificate: %w", err)
	}
	return &cert, nil
}

func (c *Client) UpdateLBCertificate(id string, req LBCertificateRequest) (*LBCertificate, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling PUT body for lb certificate %s: %w", id, err)
	}
	httpReq, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/lb/certificates/%s", c.endpoint, id), bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("building PUT request for lb certificate %s: %w", id, err)
	}
	resp, err := c.do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("update lb certificate %s: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.apiError(resp)
	}
	var cert LBCertificate
	if err := json.NewDecoder(resp.Body).Decode(&cert); err != nil {
		return nil, fmt.Errorf("decoding update lb certificate %s response: %w", id, err)
	}
	return &cert, nil
}

func (c *Client) DeleteLBCertificate(id string) error {
	if err := c.deleteReq(fmt.Sprintf("%s/lb/certificates/%s", c.endpoint, id)); err != nil {
		return fmt.Errorf("delete lb certificate %s: %w", id, err)
	}
	return nil
}

// ── Public IP Pool ────────────────────────────────────────────────────────────

type PublicIPPool struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Interface string    `json:"interface"`
	CIDR      string    `json:"cidr"`
	CreatedAt time.Time `json:"created_at"`
}

type PublicIP struct {
	ID          string    `json:"id"`
	PoolID      string    `json:"pool_id"`
	IP          string    `json:"ip"`
	PrivateIP   string    `json:"private_ip,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (c *Client) ListPublicIPPools() ([]PublicIPPool, error) {
	var pools []PublicIPPool
	if err := c.getJSON(fmt.Sprintf("%s/network/public-ip-pools", c.endpoint), &pools); err != nil {
		return nil, fmt.Errorf("list public ip pools: %w", err)
	}
	return pools, nil
}

func (c *Client) GetPublicIPPool(id string) (*PublicIPPool, error) {
	pools, err := c.ListPublicIPPools()
	if err != nil {
		return nil, err
	}
	for _, p := range pools {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("public ip pool with ID %s not found", id)
}

func (c *Client) CreatePublicIPPool(name, iface, cidr string) (*PublicIPPool, error) {
	body := map[string]string{"name": name, "interface": iface, "cidr": cidr}
	var pool PublicIPPool
	if err := c.postJSON(fmt.Sprintf("%s/network/public-ip-pools", c.endpoint), body, &pool); err != nil {
		return nil, fmt.Errorf("create public ip pool: %w", err)
	}
	return &pool, nil
}

func (c *Client) DeletePublicIPPool(id string) error {
	if err := c.deleteReq(fmt.Sprintf("%s/network/public-ip-pools/%s", c.endpoint, id)); err != nil {
		return fmt.Errorf("delete public ip pool %s: %w", id, err)
	}
	return nil
}

func (c *Client) ListPublicIPs() ([]PublicIP, error) {
	var ips []PublicIP
	if err := c.getJSON(fmt.Sprintf("%s/network/public-ips", c.endpoint), &ips); err != nil {
		return nil, fmt.Errorf("list public ips: %w", err)
	}
	return ips, nil
}

func (c *Client) GetPublicIP(id string) (*PublicIP, error) {
	ips, err := c.ListPublicIPs()
	if err != nil {
		return nil, err
	}
	for _, ip := range ips {
		if ip.ID == id {
			return &ip, nil
		}
	}
	return nil, fmt.Errorf("public ip with ID %s not found", id)
}

func (c *Client) AllocatePublicIP(poolID, description string) (*PublicIP, error) {
	body := map[string]string{"pool_id": poolID, "description": description}
	var ip PublicIP
	if err := c.postJSON(fmt.Sprintf("%s/network/public-ips", c.endpoint), body, &ip); err != nil {
		return nil, fmt.Errorf("allocate public ip from pool %s: %w", poolID, err)
	}
	return &ip, nil
}

func (c *Client) ReleasePublicIP(id string) error {
	if err := c.deleteReq(fmt.Sprintf("%s/network/public-ips/%s", c.endpoint, id)); err != nil {
		return fmt.Errorf("release public ip %s: %w", id, err)
	}
	return nil
}

func (c *Client) EnableStaticNAT(id, privateIP string) (*PublicIP, error) {
	body := map[string]string{"private_ip": privateIP}
	var ip PublicIP
	if err := c.postJSON(fmt.Sprintf("%s/network/public-ips/%s/static-nat", c.endpoint, id), body, &ip); err != nil {
		return nil, fmt.Errorf("enable static nat for public ip %s: %w", id, err)
	}
	return &ip, nil
}

func (c *Client) DisableStaticNAT(id string) error {
	if err := c.deleteReq(fmt.Sprintf("%s/network/public-ips/%s/static-nat", c.endpoint, id)); err != nil {
		return fmt.Errorf("disable static nat for public ip %s: %w", id, err)
	}
	return nil
}

// ── Storage ───────────────────────────────────────────────────────────────────

type StorageBackend struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	Config    map[string]any `json:"config"`
	IsDefault bool           `json:"is_default"`
	CreatedAt time.Time      `json:"created_at"`
}

type StorageVolume struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	SizeBytes     int64     `json:"size_bytes"`
	BackendID     string    `json:"backend_id,omitempty"`
	DiskfulNodes  []string  `json:"diskful_nodes,omitempty"`
	DisklessNodes []string  `json:"diskless_nodes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func (c *Client) ListStorageBackends() ([]StorageBackend, error) {
	var backends []StorageBackend
	if err := c.getJSON(fmt.Sprintf("%s/storage/backends", c.endpoint), &backends); err != nil {
		return nil, fmt.Errorf("list storage backends: %w", err)
	}
	return backends, nil
}

func (c *Client) GetStorageBackend(id string) (*StorageBackend, error) {
	backends, err := c.ListStorageBackends()
	if err != nil {
		return nil, err
	}
	for _, b := range backends {
		if b.ID == id {
			return &b, nil
		}
	}
	return nil, fmt.Errorf("storage backend with ID %s not found", id)
}

func (c *Client) CreateStorageBackend(name, backendType string, config map[string]any) (*StorageBackend, error) {
	body := map[string]any{"name": name, "type": backendType, "config": config}
	var backend StorageBackend
	if err := c.postJSON(fmt.Sprintf("%s/storage/backends", c.endpoint), body, &backend); err != nil {
		return nil, fmt.Errorf("create storage backend: %w", err)
	}
	return &backend, nil
}

func (c *Client) DeleteStorageBackend(id string) error {
	if err := c.deleteReq(fmt.Sprintf("%s/storage/backends/%s", c.endpoint, id)); err != nil {
		return fmt.Errorf("delete storage backend %s: %w", id, err)
	}
	return nil
}

func (c *Client) ListStorageVolumes() ([]StorageVolume, error) {
	var volumes []StorageVolume
	if err := c.getJSON(fmt.Sprintf("%s/storage/volumes", c.endpoint), &volumes); err != nil {
		return nil, fmt.Errorf("list storage volumes: %w", err)
	}
	return volumes, nil
}

func (c *Client) GetStorageVolume(id string) (*StorageVolume, error) {
	volumes, err := c.ListStorageVolumes()
	if err != nil {
		return nil, err
	}
	for _, v := range volumes {
		if v.ID == id {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("storage volume with ID %s not found", id)
}

func (c *Client) CreateStorageVolume(name string, sizeBytes int64, backendID string) (*StorageVolume, error) {
	body := map[string]any{"name": name, "size_bytes": sizeBytes, "backend_id": backendID}
	var vol StorageVolume
	if err := c.postJSON(fmt.Sprintf("%s/storage/volumes", c.endpoint), body, &vol); err != nil {
		return nil, fmt.Errorf("create storage volume: %w", err)
	}
	return &vol, nil
}

func (c *Client) DeleteStorageVolume(id string) error {
	if err := c.deleteReq(fmt.Sprintf("%s/storage/volumes/%s", c.endpoint, id)); err != nil {
		return fmt.Errorf("delete storage volume %s: %w", id, err)
	}
	return nil
}

func (c *Client) ResizeStorageVolume(id string, newSizeBytes int64) (*StorageVolume, error) {
	body := map[string]any{"size_bytes": newSizeBytes}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling resize body for storage volume %s: %w", id, err)
	}
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/storage/volumes/%s", c.endpoint, id), bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("building PUT request for storage volume %s: %w", id, err)
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("resize storage volume %s: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.apiError(resp)
	}
	var vol StorageVolume
	if err := json.NewDecoder(resp.Body).Decode(&vol); err != nil {
		return nil, fmt.Errorf("decoding resize storage volume %s response: %w", id, err)
	}
	return &vol, nil
}

// ── Kube cluster ──────────────────────────────────────────────────────────────

type KubeCluster struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Status       string     `json:"status"`
	TalosVersion string     `json:"talos_version"`
	K8sVersion   string     `json:"k8s_version"`
	CNI          string     `json:"cni"`
	LBMode       string     `json:"lb_mode"`
	VPCID        string     `json:"vpc_id,omitempty"`
	VPCCIDR      string     `json:"vpc_cidr,omitempty"`
	PublicIPID   string     `json:"public_ip_id,omitempty"`
	PublicIP     string     `json:"public_ip,omitempty"`
	Endpoint     string     `json:"endpoint,omitempty"`
	CPCount      int        `json:"cp_count"`
	WorkerCount  int        `json:"worker_count"`
	TalosImage   string     `json:"talos_image"`
	CPVCPUs      int        `json:"cp_vcpus"`
	CPMemoryMB   int        `json:"cp_memory_mb"`
	CPDiskGB     int        `json:"cp_disk_gb"`
	WorkerVCPUs  int        `json:"worker_vcpus"`
	WorkerMemMB  int        `json:"worker_memory_mb"`
	WorkerDiskGB int        `json:"worker_disk_gb"`
	ErrorMsg     string     `json:"error,omitempty"`
	Nodes        []KubeNode `json:"nodes,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type KubeNode struct {
	ID        string `json:"id"`
	ClusterID string `json:"cluster_id"`
	VMID      string `json:"vm_id"`
	Role      string `json:"role"`
	IP        string `json:"ip,omitempty"`
	Status    string `json:"status"`
}

type TalosRelease struct {
	Version     string    `json:"version"`
	K8sVersion  string    `json:"k8s_version,omitempty"`
	PublishedAt time.Time `json:"published_at"`
}

type KubeCreateRequest struct {
	Name         string `json:"name"`
	TalosImage   string `json:"talos_image"`
	CPCount      int    `json:"cp_count"`
	WorkerCount  int    `json:"worker_count"`
	CPVCPUs      int    `json:"cp_vcpus"`
	CPMemoryMB   int    `json:"cp_memory_mb"`
	CPDiskGB     int    `json:"cp_disk_gb"`
	WorkerVCPUs  int    `json:"worker_vcpus"`
	WorkerMemMB  int    `json:"worker_memory_mb"`
	WorkerDiskGB int    `json:"worker_disk_gb"`
	CNI          string `json:"cni"`
	LBMode       string `json:"lb_mode"`
	PoolID       string `json:"pool_id,omitempty"`
	TalosVersion string `json:"talos_version"`
	K8sVersion   string `json:"k8s_version"`
}

type KubePatchRequest struct {
	WorkerCount  *int   `json:"worker_count,omitempty"`
	TalosVersion string `json:"talos_version,omitempty"`
	K8sVersion   string `json:"k8s_version,omitempty"`
}

func (c *Client) ListKubeClusters() ([]KubeCluster, error) {
	var clusters []KubeCluster
	if err := c.getJSON(fmt.Sprintf("%s/kube/clusters", c.endpoint), &clusters); err != nil {
		return nil, fmt.Errorf("list kube clusters: %w", err)
	}
	return clusters, nil
}

func (c *Client) GetKubeCluster(id string) (*KubeCluster, error) {
	var cluster KubeCluster
	if err := c.getJSON(fmt.Sprintf("%s/kube/clusters/%s", c.endpoint, id), &cluster); err != nil {
		return nil, fmt.Errorf("get kube cluster %s: %w", id, err)
	}
	return &cluster, nil
}

func (c *Client) CreateKubeCluster(req KubeCreateRequest) (*KubeCluster, error) {
	var cluster KubeCluster
	if err := c.postJSON(fmt.Sprintf("%s/kube/clusters", c.endpoint), req, &cluster); err != nil {
		return nil, fmt.Errorf("create kube cluster: %w", err)
	}
	return &cluster, nil
}

func (c *Client) PatchKubeCluster(id string, req KubePatchRequest) (*KubeCluster, error) {
	var cluster KubeCluster
	if err := c.patchJSON(fmt.Sprintf("%s/kube/clusters/%s", c.endpoint, id), req, &cluster); err != nil {
		return nil, fmt.Errorf("patch kube cluster %s: %w", id, err)
	}
	return &cluster, nil
}

func (c *Client) DeleteKubeCluster(id string) error {
	if err := c.deleteReq(fmt.Sprintf("%s/kube/clusters/%s", c.endpoint, id)); err != nil {
		return fmt.Errorf("delete kube cluster %s: %w", id, err)
	}
	return nil
}

func (c *Client) ListTalosReleases() ([]TalosRelease, error) {
	var releases []TalosRelease
	if err := c.getJSON(fmt.Sprintf("%s/kube/releases", c.endpoint), &releases); err != nil {
		return nil, fmt.Errorf("list talos releases: %w", err)
	}
	return releases, nil
}

func (c *Client) GetKubeconfig(id string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/kube/clusters/%s/kubeconfig", c.endpoint, id), nil)
	if err != nil {
		return "", fmt.Errorf("building GET kubeconfig request for cluster %s: %w", id, err)
	}
	resp, err := c.do(req)
	if err != nil {
		return "", fmt.Errorf("get kubeconfig for cluster %s: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", c.apiError(resp)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return "", fmt.Errorf("reading kubeconfig for cluster %s: %w", id, err)
	}
	return buf.String(), nil
}

func (c *Client) GetTalosconfig(id string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/kube/clusters/%s/talosconfig", c.endpoint, id), nil)
	if err != nil {
		return "", fmt.Errorf("building GET talosconfig request for cluster %s: %w", id, err)
	}
	resp, err := c.do(req)
	if err != nil {
		return "", fmt.Errorf("get talosconfig for cluster %s: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", c.apiError(resp)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return "", fmt.Errorf("reading talosconfig for cluster %s: %w", id, err)
	}
	return buf.String(), nil
}

// ── Security group ────────────────────────────────────────────────────────────

type SecurityGroup struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Rules       []SGRule `json:"rules,omitempty"`
	CreatedAt   string   `json:"created_at"`
}

type SGRule struct {
	ID        string `json:"id"`
	Direction string `json:"direction"` // "ingress"|"egress"
	Protocol  string `json:"protocol"`  // "tcp"|"udp"|"icmp"|"all"
	PortFrom  int    `json:"port_from"`
	PortTo    int    `json:"port_to"`
	CIDR      string `json:"cidr"`
	Action    string `json:"action"` // "accept"|"drop"
	Priority  int    `json:"priority"`
}

func (c *Client) ListSecurityGroups() ([]SecurityGroup, error) {
	var sgs []SecurityGroup
	if err := c.getJSON(fmt.Sprintf("%s/security-groups", c.endpoint), &sgs); err != nil {
		return nil, fmt.Errorf("list security groups: %w", err)
	}
	return sgs, nil
}

func (c *Client) GetSecurityGroup(id string) (*SecurityGroup, error) {
	sgs, err := c.ListSecurityGroups()
	if err != nil {
		return nil, err
	}
	for _, sg := range sgs {
		if sg.ID == id {
			return &sg, nil
		}
	}
	return nil, fmt.Errorf("security group with ID %s not found", id)
}

func (c *Client) CreateSecurityGroup(name, description string) (*SecurityGroup, error) {
	body := map[string]string{"name": name, "description": description}
	var sg SecurityGroup
	if err := c.postJSON(fmt.Sprintf("%s/security-groups", c.endpoint), body, &sg); err != nil {
		return nil, fmt.Errorf("create security group: %w", err)
	}
	return &sg, nil
}

func (c *Client) DeleteSecurityGroup(id string) error {
	if err := c.deleteReq(fmt.Sprintf("%s/security-groups/%s", c.endpoint, id)); err != nil {
		return fmt.Errorf("delete security group %s: %w", id, err)
	}
	return nil
}

func (c *Client) AddSGRule(sgID string, rule SGRule) (*SGRule, error) {
	var result SGRule
	if err := c.postJSON(fmt.Sprintf("%s/security-groups/%s/rules", c.endpoint, sgID), rule, &result); err != nil {
		return nil, fmt.Errorf("add rule to security group %s: %w", sgID, err)
	}
	return &result, nil
}

func (c *Client) RemoveSGRule(sgID, ruleID string) error {
	if err := c.deleteReq(fmt.Sprintf("%s/security-groups/%s/rules/%s", c.endpoint, sgID, ruleID)); err != nil {
		return fmt.Errorf("remove rule %s from security group %s: %w", ruleID, sgID, err)
	}
	return nil
}

// ── IPAM pool ─────────────────────────────────────────────────────────────────

type IPAMPool struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Bridge     string   `json:"bridge"`
	Subnet     string   `json:"subnet"`
	Gateway    string   `json:"gateway"`
	RangeStart string   `json:"range_start"`
	RangeEnd   string   `json:"range_end"`
	DNS        []string `json:"dns"`
	CreatedAt  string   `json:"created_at"`
}

func (c *Client) ListIPAMPools() ([]IPAMPool, error) {
	var pools []IPAMPool
	if err := c.getJSON(fmt.Sprintf("%s/ipam/pools", c.endpoint), &pools); err != nil {
		return nil, fmt.Errorf("list ipam pools: %w", err)
	}
	return pools, nil
}

func (c *Client) GetIPAMPool(id string) (*IPAMPool, error) {
	pools, err := c.ListIPAMPools()
	if err != nil {
		return nil, err
	}
	for _, p := range pools {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("ipam pool with ID %s not found", id)
}

func (c *Client) CreateIPAMPool(pool IPAMPool) (*IPAMPool, error) {
	var result IPAMPool
	if err := c.postJSON(fmt.Sprintf("%s/ipam/pools", c.endpoint), pool, &result); err != nil {
		return nil, fmt.Errorf("create ipam pool: %w", err)
	}
	return &result, nil
}

func (c *Client) DeleteIPAMPool(id string) error {
	if err := c.deleteReq(fmt.Sprintf("%s/ipam/pools/%s", c.endpoint, id)); err != nil {
		return fmt.Errorf("delete ipam pool %s: %w", id, err)
	}
	return nil
}
