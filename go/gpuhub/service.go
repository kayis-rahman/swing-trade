package gpuhub

import (
	"context"
	"time"
)

// Service parity.
type Service struct {
	Client  *Client
	Timeout time.Duration
}

// NewService parity.
func NewService(client *Client) *Service {
	return &Service{Client: client, Timeout: 120 * time.Second}
}

func (s *Service) context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s.Timeout)
}

// Create parity.
func (s *Service) Create(name, dc, gpuSet, imageUUID string, replicas int, reuseContainer bool, cmd string) (*CreateDeploymentResponse, error) {
	ctx, cancel := s.context()
	defer cancel()
	req := CreateDeploymentRequest{
		Name:                name,
		DeploymentType:      "ReplicaSet",
		ReplicaNum:          replicas,
		ReuseContainer:      reuseContainer,
		ReuseContainerScope: "all",
		ContainerTemplate: ContainerTemplate{
			DcList:     []string{dc},
			GpuNameSet: []string{gpuSet},
			ImageUUID:  imageUUID,
			Cmd:        cmd,
		},
	}
	return s.Client.CreateDeployment(ctx, req)
}

// List parity.
func (s *Service) List(page, pageSize int) ([]DeploymentInfo, error) {
	ctx, cancel := s.context()
	defer cancel()
	return s.Client.ListDeployments(ctx, page, pageSize)
}

// Status parity.
func (s *Service) Status(uuid string) (*DeploymentInfo, error) {
	ctx, cancel := s.context()
	defer cancel()
	list, err := s.Client.ListDeploymentsByUUID(ctx, uuid)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

// Stop parity.
func (s *Service) Stop(uuid string) error {
	ctx, cancel := s.context()
	defer cancel()
	return s.Client.StopDeployment(ctx, uuid)
}

// Delete parity.
func (s *Service) Delete(uuid string) error {
	ctx, cancel := s.context()
	defer cancel()
	return s.Client.DeleteDeployment(ctx, uuid)
}

// ListContainers parity.
func (s *Service) ListContainers(uuid string) ([]ContainerInfo, error) {
	ctx, cancel := s.context()
	defer cancel()
	return s.Client.ListContainers(ctx, uuid)
}

// StopContainer parity.
func (s *Service) StopContainer(uuid string) error {
	ctx, cancel := s.context()
	defer cancel()
	return s.Client.StopContainer(ctx, uuid)
}

// ListPrivateImages parity.
func (s *Service) ListPrivateImages(page, pageSize int) ([]PrivateImage, error) {
	ctx, cancel := s.context()
	defer cancel()
	return s.Client.ListPrivateImages(ctx, page, pageSize)
}

// IsRunning parity.
func (s *Service) IsRunning(uuid string) (bool, error) {
	info, err := s.Status(uuid)
	if err != nil || info == nil {
		return false, err
	}
	st := info.Status
	return st != "" && (contains(st, "Run") || contains(st, "Active") || contains(st, "Start")), nil
}

// contains parity.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
