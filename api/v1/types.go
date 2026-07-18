package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DataPipelineSpec defines the desired state of DataPipeline.
type DataPipelineSpec struct {
	// Image is the container image for the pipeline Job.
	Image string `json:"image"`

	// Schedule is an optional cron expression for periodic runs.
	// +optional
	Schedule string `json:"schedule,omitempty"`

	// Retries is the number of times to retry a failed Job (default 0).
	// +optional
	Retries int32 `json:"retries,omitempty"`

	// Timeout is the duration after which an active Job is deleted.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

// DataPipelinePhase is the lifecycle phase of a DataPipeline.
type DataPipelinePhase string

const (
	DataPipelinePending   DataPipelinePhase = "Pending"
	DataPipelineRunning   DataPipelinePhase = "Running"
	DataPipelineSucceeded DataPipelinePhase = "Succeeded"
	DataPipelineFailed    DataPipelinePhase = "Failed"
)

// DataPipelineStatus defines the observed state of DataPipeline.
type DataPipelineStatus struct {
	// Phase is the current lifecycle phase.
	Phase DataPipelinePhase `json:"phase,omitempty"`

	// Runs is the total number of Job executions attempted.
	Runs int32 `json:"runs,omitempty"`

	// LastRunTime records when the most recent Job was started.
	// +optional
	LastRunTime *metav1.Time `json:"lastRunTime,omitempty"`

	// LastSuccessTime records when the most recent successful Job completed.
	// +optional
	LastSuccessTime *metav1.Time `json:"lastSuccessTime,omitempty"`

	// Message is a human-readable description of the current status.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=dp
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Runs",type=integer,JSONPath=`.status.runs`
// +kubebuilder:printcolumn:name="Last-Run",type=string,JSONPath=`.status.lastRunTime`

// DataPipeline is the Schema for the datapipelines API.
type DataPipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataPipelineSpec   `json:"spec,omitempty"`
	Status DataPipelineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DataPipelineList contains a list of DataPipeline.
type DataPipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataPipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataPipeline{}, &DataPipelineList{})
}
