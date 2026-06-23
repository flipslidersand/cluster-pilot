# Data Model — ClusterPilot

## CRD 型定義

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type DataPipeline struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec   DataPipelineSpec   `json:"spec,omitempty"`
    Status DataPipelineStatus `json:"status,omitempty"`
}

type DataPipelineSpec struct {
    Schedule string `json:"schedule,omitempty"` // cron 式
    Image    string `json:"image"`
    Retries  int32  `json:"retries,omitempty"`
    Timeout  string `json:"timeout,omitempty"`  // "10m"
    Env      []corev1.EnvVar `json:"env,omitempty"`
}

type DataPipelineStatus struct {
    Phase      PipelinePhase `json:"phase"`
    RunCount   int32         `json:"runCount"`
    LastRunAt  *metav1.Time  `json:"lastRunAt,omitempty"`
    LastResult string        `json:"lastResult,omitempty"` // "Succeeded" | "Failed"
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type PipelinePhase string
const (
    PhasePending  PipelinePhase = "Pending"
    PhaseRunning  PipelinePhase = "Running"
    PhaseSucceeded PipelinePhase = "Succeeded"
    PhaseFailed   PipelinePhase = "Failed"
)
```

## Reconcile 状態遷移

```
DataPipeline 作成
  → Pending
  → Job 生成 → Running
  → Job 完了 → Succeeded
  → Job 失敗 → retries 残あり → Running (再実行)
             → retries 上限  → Failed
  → timeout  → Job 削除 → Failed
```

## 生成する Kubernetes Job

```go
job := &batchv1.Job{
    ObjectMeta: metav1.ObjectMeta{
        Name:      fmt.Sprintf("%s-%d", pipeline.Name, pipeline.Status.RunCount),
        Namespace: pipeline.Namespace,
        OwnerReferences: []metav1.OwnerReference{
            *metav1.NewControllerRef(pipeline, GroupVersion.WithKind("DataPipeline")),
        },
    },
    Spec: batchv1.JobSpec{
        Template: corev1.PodTemplateSpec{
            Spec: corev1.PodSpec{
                Containers: []corev1.Container{{
                    Name:  "pipeline",
                    Image: pipeline.Spec.Image,
                    Env:   pipeline.Spec.Env,
                }},
                RestartPolicy: corev1.RestartPolicyNever,
            },
        },
        BackoffLimit: pointer.Int32(0),
    },
}
```
