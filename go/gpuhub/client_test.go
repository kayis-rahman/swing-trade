package gpuhub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGpuHubCreateAndList parity.
func TestGpuHubCreateAndList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"code": "Success", "data": map[string]any{"deploymentUUID": "u1", "list": []map[string]any{{"deploymentUUID": "u1", "status": "Running"}}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "key")
	resp, err := c.CreateDeployment(context.Background(), CreateDeploymentRequest{Name: "n"})
	if err != nil || resp.DeploymentUUID != "u1" {
		t.Fatalf("create err %v resp %+v", err, resp)
	}
	list, err := c.ListDeployments(context.Background(), 1, 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("list err %v len %d", err, len(list))
	}
	svc := NewService(c)
	ok, _ := svc.IsRunning("u1")
	if !ok {
		t.Fatal("expected running")
	}
}

// TestGpuHubPrivateImages parity.
func TestGpuHubPrivateImages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":"Success","data":{"list":[{"imageUUID":"img1"}]}}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "key")
	imgs, err := c.ListPrivateImages(context.Background(), 1, 10)
	if err != nil || len(imgs) != 1 {
		t.Fatalf("err %v len %d", err, len(imgs))
	}
}

// TestGpuHubMoreOps parity.
func TestGpuHubMoreOps(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/dev/deployment" && r.Method == http.MethodDelete {
			_, _ = w.Write([]byte(`{"code":"Success","data":null}`))
			return
		}
		if r.URL.Path == "/api/v1/dev/deployment/container/list" {
			_, _ = w.Write([]byte(`{"code":"Success","data":{"list":[{"deploymentContainerUUID":"c1"}]}}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":"Success","data":null}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "key")
	ctx := context.Background()
	if err := c.StopDeployment(ctx, "u1"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteDeployment(ctx, "u1"); err != nil {
		t.Fatal(err)
	}
	if containers, err := c.ListContainers(ctx, "u1"); err != nil || len(containers) != 1 {
		t.Fatalf("containers %v %d", err, len(containers))
	}
	if err := c.StopContainer(ctx, "c1"); err != nil {
		t.Fatal(err)
	}
	by, _ := c.ListDeploymentsByUUID(ctx, "u1")
	if by == nil {
		t.Fatal("nil")
	}
	svc := NewService(c)
	if err := svc.Stop("u1"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete("u1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ListContainers("u1"); err != nil {
		t.Fatal(err)
	}
	if err := svc.StopContainer("c1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ListPrivateImages(1, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Status("u1"); err != nil {
		t.Fatal(err)
	}
}
