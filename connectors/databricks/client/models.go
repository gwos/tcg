package client

type RunsResponse struct {
	Runs          []Run  `json:"runs"`
	NextPageToken string `json:"next_page_token,omitempty"`
	HasMore       bool   `json:"has_more,omitempty"`
}

type RunResponse struct {
	JobID int64  `json:"job_id"`
	Tasks []Task `json:"tasks"`
}

type Run struct {
	JobID   int64  `json:"job_id,"`
	RunID   int64  `json:"run_id"`
	RunName string `json:"run_name"`
	Status  struct {
		State              string `json:"state,omitempty"` // BLOCKED | PENDING | QUEUED | RUNNING | TERMINATING | TERMINATED
		TerminationDetails struct {
			Code    string `json:"code,omitempty"` // SUCCESS | CANCELED | DRIVER_ERROR | CLUSTER_ERROR | REPOSITORY_CHECKOUT_FAILED | INVALID_CLUSTER_REQUEST | WORKSPACE_RUN_LIMIT_EXCEEDED | FEATURE_DISABLED | CLUSTER_REQUEST_LIMIT_EXCEEDED | STORAGE_ACCESS_ERROR | RUN_EXECUTION_ERROR | UNAUTHORIZED_ERROR | LIBRARY_INSTALLATION_ERROR | MAX_CONCURRENT_RUNS_EXCEEDED | MAX_SPARK_CONTEXTS_EXCEEDED | RESOURCE_NOT_FOUND | INVALID_RUN_CONFIGURATION | INTERNAL_ERROR | CLOUD_FAILURE | MAX_JOB_QUEUE_SIZE_EXCEEDED | SKIPPED | USER_CANCELED
			Type    string `json:"type,omitempty"` // SUCCESS | INTERNAL_ERROR | CLIENT_ERROR | CLOUD_FAILURE
			Message string `json:"message,omitempty"`
		} `json:"termination_details,omitempty"`
	} `json:"status,omitempty"`
	RunDuration float64 `json:"run_duration,omitempty"`
	StartTime   int64   `json:"start_time,omitempty"`
	EndTime     int64   `json:"end_time,omitempty"`
}

type ClustersResponse struct {
	Clusters      []Cluster `json:"clusters"`
	NextPageToken string    `json:"next_page_token,omitempty"`
	HasMore       bool      `json:"has_more,omitempty"`
}

type Cluster struct {
	Name              string `json:"cluster_name,omitempty"`
	State             string `json:"state,omitempty"` // PENDING | RUNNING | RESTARTING | RESIZING | TERMINATING | TERMINATED | ERROR | UNKNOWN
	StateMessage      string `json:"state_message,omitempty"`
	TerminationReason struct {
		Code       string                 `json:"code"`
		Type       string                 `json:"type"`
		Parameters map[string]interface{} `json:"parameters,omitempty"`
	} `json:"termination_reason,omitempty"`
}

type Task struct {
	TaskKey string `json:"task_key"`
	State   struct {
		StateMessage string `json:"state_message,omitempty"`
	} `json:"state,omitempty"`
	StartTime int64 `json:"start_time,omitempty"`
	EndTime   int64 `json:"end_time,omitempty"`
	Status    struct {
		State              string `json:"state,omitempty"`
		TerminationDetails struct {
			Code    string `json:"code,omitempty"` // SUCCESS | CANCELED | DRIVER_ERROR | CLUSTER_ERROR | REPOSITORY_CHECKOUT_FAILED | INVALID_CLUSTER_REQUEST | WORKSPACE_RUN_LIMIT_EXCEEDED | FEATURE_DISABLED | CLUSTER_REQUEST_LIMIT_EXCEEDED | STORAGE_ACCESS_ERROR | RUN_EXECUTION_ERROR | UNAUTHORIZED_ERROR | LIBRARY_INSTALLATION_ERROR | MAX_CONCURRENT_RUNS_EXCEEDED | MAX_SPARK_CONTEXTS_EXCEEDED | RESOURCE_NOT_FOUND | INVALID_RUN_CONFIGURATION | INTERNAL_ERROR | CLOUD_FAILURE | MAX_JOB_QUEUE_SIZE_EXCEEDED | SKIPPED | USER_CANCELED
			Type    string `json:"type,omitempty"` // SUCCESS | INTERNAL_ERROR | CLIENT_ERROR | CLOUD_FAILURE
			Message string `json:"message,omitempty"`
		} `json:"termination_details,omitempty"`
	} `json:"status,omitempty"`
}
