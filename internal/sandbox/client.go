package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// VMCreateRequest is the payload for creating a new microVM.
type VMCreateRequest struct {
	AgentName    string `json:"agent_name"`
	WorkspaceDir string `json:"workspace_dir"`
}

// VMInfo represents a microVM returned by the sandboxd API.
type VMInfo struct {
	VMID       string `json:"vm_id"`
	Name       string `json:"name"`
	SocketPath string `json:"socketPath"`
	StateDir   string `json:"stateDir"`
	CACertPath string `json:"ca_cert_path"`
}

// Client communicates with the sandboxd Unix socket API.
type Client struct {
	httpClient *http.Client
	socketPath string
}

// NewClient creates a sandboxd client connecting to the given Unix socket.
func NewClient(socketPath string) *Client {
	if socketPath == "" {
		homeDir, _ := os.UserHomeDir()
		socketPath = filepath.Join(homeDir, ".docker", "sandboxes", "sandboxd.sock")
	}

	return &Client{
		socketPath: socketPath,
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
		},
	}
}

// CreateVM creates a new microVM via POST /vm.
func (c *Client) CreateVM(ctx context.Context, req VMCreateRequest) (*VMInfo, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost/vm", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request to sandboxd: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sandboxd returned %d: %s", resp.StatusCode, string(respBody))
	}

	var vm VMInfo
	if err := json.NewDecoder(resp.Body).Decode(&vm); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &vm, nil
}

// ListVMs lists all microVMs via GET /vm.
func (c *Client) ListVMs(ctx context.Context) ([]VMInfo, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/vm", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request to sandboxd: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sandboxd returned %d: %s", resp.StatusCode, string(respBody))
	}

	var vms []VMInfo
	if err := json.NewDecoder(resp.Body).Decode(&vms); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return vms, nil
}

// DestroyVM destroys a microVM via DELETE /vm/{name}.
func (c *Client) DestroyVM(ctx context.Context, name string) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, "http://localhost/vm/"+name, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("sending request to sandboxd: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sandboxd returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
