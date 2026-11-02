package gpuhub

// ContainerTemplate parity.
type ContainerTemplate struct {
	DcList     []string `json:"dcList,omitempty"`
	GpuNameSet []string `json:"gpuNameSet,omitempty"`
	ImageUUID  string   `json:"imageUUID,omitempty"`
	Cmd        string   `json:"cmd,omitempty"`
}

// CreateDeploymentRequest parity.
type CreateDeploymentRequest struct {
	Name                string            `json:"name"`
	DeploymentType      string            `json:"deploymentType"`
	ReplicaNum          int               `json:"replicaNum"`
	ReuseContainer      bool              `json:"reuseContainer"`
	ReuseContainerScope string            `json:"reuseContainerScope,omitempty"`
	ContainerTemplate   ContainerTemplate `json:"containerTemplate"`
}

// CreateDeploymentResponse parity.
type CreateDeploymentResponse struct {
	DeploymentUUID string `json:"deploymentUUID"`
}

// DeploymentInfo parity.
type DeploymentInfo struct {
	DeploymentUUID string `json:"deploymentUUID"`
	Status         string `json:"status"`
}

// PrivateImage parity.
type PrivateImage struct {
	ImageUUID string `json:"imageUUID"`
}

// ContainerInfo parity.
type ContainerInfo struct {
	DeploymentContainerUUID string `json:"deploymentContainerUUID"`
}
