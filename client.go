package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type VMStatus string

const (
	StatusStopped VMStatus = "stopped"
	StatusRunning VMStatus = "running"
	StatusPaused  VMStatus = "paused"
	StatusError   VMStatus = "error"
)

type CloudInitConfig struct {
	UserData      string `json:"user_data"`
	MetaData      string `json:"meta_data"`
	NetworkConfig string `json:"network_config,omitempty"`
}

type ExtraDisk struct {
	Index     int    `json:"index,omitempty"`
	SizeGB    int    `json:"size_gb"`
	DiskPath  string `json:"disk_path,omitempty"`
	Interface string `json:"interface,omitempty"`
}

type NIC struct {
	Index    int    `json:"index,omitempty"`
	Bridge   string `json:"bridge"`
	MACAddr  string `json:"mac_addr,omitempty"`
	DeviceID string `json:"device_id,omitempty"`
	Model    string `json:"model,omitempty"`
}

type VM struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	CPUs          int              `json:"cpus"`
	Memory        int              `json:"memory_mb"`
	Disks         []string         `json:"disks,omitempty"`
	Status        VMStatus         `json:"status"`
	ISOPath       string           `json:"iso_path,omitempty"`
	CloudInit     *CloudInitConfig `json:"cloud_init,omitempty"`
	ExtraDisks    []ExtraDisk      `json:"extra_disks,omitempty"`
	NICs          []NIC            `json:"nics,omitempty"`
	PID           int              `json:"pid,omitempty"`
	QMPSocket     string           `json:"qmp_socket,omitempty"`
	VNCSocket     string           `json:"vnc_socket,omitempty"`
	DiskPath      string           `json:"disk_path,omitempty"`
	BootDiskGB    int              `json:"boot_disk_gb,omitempty"`
	DiskInterface string           `json:"disk_interface,omitempty"`
	VMType        string           `json:"vm_type,omitempty"`
	KernelID      string           `json:"kernel_id,omitempty"`
	KernelCmdline string           `json:"kernel_cmdline,omitempty"`
	ImageID       string           `json:"image_id,omitempty"`
	Arch          string           `json:"arch,omitempty"`
	Node          string           `json:"node,omitempty"`
}

type Client struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

func NewClient(endpoint, apiKey string, insecure bool) *Client {
	endpoint = strings.TrimSuffix(endpoint, "/")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}
	return &Client{
		endpoint: endpoint,
		apiKey:   apiKey,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}

func (c *Client) ListVMs() ([]VM, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/vm", c.endpoint), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("list vms failed with status %d: %s", resp.StatusCode, errResp["error"])
	}

	var vms []VM
	if err := json.NewDecoder(resp.Body).Decode(&vms); err != nil {
		return nil, err
	}

	return vms, nil
}

func (c *Client) GetVM(id string) (*VM, error) {
	// The API doesn't have a direct GET /vm/{id} endpoint based on FEATURES.md or server.go route definitions.
	// But it does have handleVMList which returns all VMs.
	// We'll retrieve all VMs and filter by ID to find the requested VM.
	vms, err := c.ListVMs()
	if err != nil {
		return nil, err
	}

	for _, vm := range vms {
		if vm.ID == id {
			return &vm, nil
		}
	}

	return nil, fmt.Errorf("vm with ID %s not found", id)
}

type vmCreateRequest struct {
	Name          string           `json:"name"`
	CPUs          int              `json:"cpus"`
	MemoryMB      int              `json:"memory_mb"`
	BootDiskGB    int              `json:"boot_disk_gb,omitempty"`
	DiskInterface string           `json:"disk_interface,omitempty"`
	ISOPath       string           `json:"iso_path,omitempty"`
	CloudInit     *CloudInitConfig `json:"cloud_init,omitempty"`
	ExtraDisks    []ExtraDisk      `json:"extra_disks,omitempty"`
	NICs          []NIC            `json:"nics,omitempty"`
	VMType        string           `json:"vm_type,omitempty"`
	KernelID      string           `json:"kernel_id,omitempty"`
	KernelCmdline string           `json:"kernel_cmdline,omitempty"`
	ImageID       string           `json:"image_id,omitempty"`
	Arch          string           `json:"arch,omitempty"`
	Node          string           `json:"node,omitempty"`
}

func (c *Client) CreateVM(name string, cpus, memory int, bootDiskGB int, diskInterface string, isoPath string, cloudInit *CloudInitConfig, extraDisks []ExtraDisk, nics []NIC, vmType, kernelID, kernelCmdline, imageID, arch, node string) (*VM, error) {
	reqBody := vmCreateRequest{
		Name:          name,
		CPUs:          cpus,
		MemoryMB:      memory,
		BootDiskGB:    bootDiskGB,
		DiskInterface: diskInterface,
		ISOPath:       isoPath,
		CloudInit:     cloudInit,
		ExtraDisks:    extraDisks,
		NICs:          nics,
		VMType:        vmType,
		KernelID:      kernelID,
		KernelCmdline: kernelCmdline,
		ImageID:       imageID,
		Arch:          arch,
		Node:          node,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/vm", c.endpoint), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("create vm failed with status %d: %s", resp.StatusCode, errResp["error"])
	}

	var vm VM
	if err := json.NewDecoder(resp.Body).Decode(&vm); err != nil {
		return nil, err
	}

	return &vm, nil
}

type vmActionRequest struct {
	ID string `json:"id"`
}

func (c *Client) StartVM(id string) error {
	reqBody := vmActionRequest{ID: id}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/vm/start", c.endpoint), bytes.NewReader(body))
	if err != nil {
		return err
	}

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("start vm failed with status %d: %s", resp.StatusCode, errResp["error"])
	}

	return nil
}

func (c *Client) StopVM(id string) error {
	reqBody := vmActionRequest{ID: id}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/vm/stop", c.endpoint), bytes.NewReader(body))
	if err != nil {
		return err
	}

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("stop vm failed with status %d: %s", resp.StatusCode, errResp["error"])
	}

	return nil
}

type DeleteVMOptions struct {
	Stop  bool
	Force bool
}

func (c *Client) DeleteVM(id string, opts DeleteVMOptions) error {
	u, err := url.Parse(fmt.Sprintf("%s/vm/%s", c.endpoint, id))
	if err != nil {
		return err
	}
	q := u.Query()
	if opts.Force {
		q.Set("force", "true")
	} else if opts.Stop {
		q.Set("stop", "true")
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodDelete, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("delete vm failed with status %d: %s", resp.StatusCode, errResp["error"])
	}

	return nil
}
