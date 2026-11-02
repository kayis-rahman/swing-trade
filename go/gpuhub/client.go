package gpuhub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client parity.
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// NewClient parity.
func NewClient(baseURL, apiKey string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://www.gpuhub.com"
	}
	return &Client{BaseURL: strings.TrimSpace(baseURL), APIKey: apiKey, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any) (map[string]any, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var node map[string]any
	if err := json.Unmarshal(raw, &node); err != nil {
		return nil, err
	}
	code, _ := node["code"].(string)
	if code != "" && code != "Success" {
		msg, _ := node["msg"].(string)
		return nil, fmt.Errorf("%s: %s", code, msg)
	}
	return node, nil
}

// ListPrivateImages parity.
func (c *Client) ListPrivateImages(ctx context.Context, page, pageSize int) ([]PrivateImage, error) {
	node, err := c.doJSON(ctx, http.MethodPost, "/api/v1/dev/image/private/list", map[string]any{"page_index": page, "page_size": pageSize})
	if err != nil {
		return nil, err
	}
	data, _ := node["data"].(map[string]any)
	list, _ := data["list"].([]any)
	out := make([]PrivateImage, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			img := PrivateImage{}
			if v, ok := m["imageUUID"].(string); ok {
				img.ImageUUID = v
			}
			out = append(out, img)
		}
	}
	return out, nil
}

// CreateDeployment parity.
func (c *Client) CreateDeployment(ctx context.Context, req CreateDeploymentRequest) (*CreateDeploymentResponse, error) {
	node, err := c.doJSON(ctx, http.MethodPost, "/api/v1/dev/deployment", req)
	if err != nil {
		return nil, err
	}
	data, _ := node["data"].(map[string]any)
	resp := &CreateDeploymentResponse{}
	if v, ok := data["deploymentUUID"].(string); ok {
		resp.DeploymentUUID = v
	}
	return resp, nil
}

// ListDeployments parity.
func (c *Client) ListDeployments(ctx context.Context, page, pageSize int) ([]DeploymentInfo, error) {
	return c.listDeployments(ctx, map[string]any{"page_index": page, "page_size": pageSize})
}

// ListDeploymentsByUUID parity.
func (c *Client) ListDeploymentsByUUID(ctx context.Context, uuid string) ([]DeploymentInfo, error) {
	return c.listDeployments(ctx, map[string]any{"page_index": 1, "page_size": 1, "deployment_uuid": uuid})
}

func (c *Client) listDeployments(ctx context.Context, body map[string]any) ([]DeploymentInfo, error) {
	node, err := c.doJSON(ctx, http.MethodPost, "/api/v1/dev/deployment/list", body)
	if err != nil {
		return nil, err
	}
	data, _ := node["data"].(map[string]any)
	list, _ := data["list"].([]any)
	out := make([]DeploymentInfo, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			info := DeploymentInfo{}
			if v, ok := m["deploymentUUID"].(string); ok {
				info.DeploymentUUID = v
			}
			if v, ok := m["status"].(string); ok {
				info.Status = v
			}
			out = append(out, info)
		}
	}
	return out, nil
}

// StopDeployment parity.
func (c *Client) StopDeployment(ctx context.Context, uuid string) error {
	_, err := c.doJSON(ctx, http.MethodPut, "/api/v1/dev/deployment/operate", map[string]any{"deployment_uuid": uuid, "operation": "stop"})
	return err
}

// DeleteDeployment parity.
func (c *Client) DeleteDeployment(ctx context.Context, uuid string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.BaseURL+"/api/v1/dev/deployment", bytes.NewReader(mustMarshal(map[string]any{"deployment_uuid": uuid})))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

// ListContainers parity.
func (c *Client) ListContainers(ctx context.Context, uuid string) ([]ContainerInfo, error) {
	node, err := c.doJSON(ctx, http.MethodPost, "/api/v1/dev/deployment/container/list", map[string]any{"deployment_uuid": uuid, "page_index": 1, "page_size": 100})
	if err != nil {
		return nil, err
	}
	data, _ := node["data"].(map[string]any)
	list, _ := data["list"].([]any)
	out := make([]ContainerInfo, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			ci := ContainerInfo{}
			if v, ok := m["deploymentContainerUUID"].(string); ok {
				ci.DeploymentContainerUUID = v
			}
			out = append(out, ci)
		}
	}
	return out, nil
}

// StopContainer parity.
func (c *Client) StopContainer(ctx context.Context, uuid string) error {
	_, err := c.doJSON(ctx, http.MethodPut, "/api/v1/dev/deployment/container/stop", map[string]any{"deployment_container_uuid": uuid})
	return err
}

// mustMarshal parity.
func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
