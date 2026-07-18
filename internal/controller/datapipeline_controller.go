package controller

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pilotv1 "github.com/flipslidersand/cluster-pilot/api/v1"
)

// DataPipelineReconciler reconciles a DataPipeline object.
type DataPipelineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=clusterpilot.dev,resources=datapipelines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clusterpilot.dev,resources=datapipelines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

func (r *DataPipelineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	pipeline := &pilotv1.DataPipeline{}
	if err := r.Get(ctx, req.NamespacedName, pipeline); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Terminal states — nothing to do.
	if pipeline.Status.Phase == pilotv1.DataPipelineSucceeded ||
		pipeline.Status.Phase == pilotv1.DataPipelineFailed {
		return ctrl.Result{}, nil
	}

	jobName := fmt.Sprintf("%s-run-%d", pipeline.Name, pipeline.Status.Runs)
	job := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Name: jobName, Namespace: pipeline.Namespace}, job)

	if errors.IsNotFound(err) {
		return r.createJob(ctx, pipeline, jobName)
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	// Job exists — mirror its outcome into Status.
	return r.syncStatus(ctx, logger, pipeline, job)
}

func (r *DataPipelineReconciler) createJob(ctx context.Context, p *pilotv1.DataPipeline, name string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	job := r.desiredJob(p, name)
	if err := ctrl.SetControllerReference(p, job, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Create(ctx, job); err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("created Job", "job", name)

	now := metav1.Now()
	p.Status.Phase = pilotv1.DataPipelineRunning
	p.Status.Runs++
	p.Status.LastRunTime = &now
	return ctrl.Result{}, r.Status().Update(ctx, p)
}

func (r *DataPipelineReconciler) syncStatus(
	ctx context.Context,
	logger interface{ Info(string, ...interface{}) },
	p *pilotv1.DataPipeline,
	job *batchv1.Job,
) (ctrl.Result, error) {
	if job.Status.Succeeded > 0 {
		now := metav1.Now()
		p.Status.Phase = pilotv1.DataPipelineSucceeded
		p.Status.LastSuccessTime = &now
		p.Status.Message = "completed successfully"
		return ctrl.Result{}, r.Status().Update(ctx, p)
	}

	if job.Status.Failed > 0 {
		p.Status.Phase = pilotv1.DataPipelineFailed
		p.Status.Message = fmt.Sprintf("failed after %d attempt(s)", job.Status.Failed)
		return ctrl.Result{}, r.Status().Update(ctx, p)
	}

	return ctrl.Result{}, nil
}

func (r *DataPipelineReconciler) desiredJob(p *pilotv1.DataPipeline, name string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: p.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "cluster-pilot",
				"datapipeline":                 p.Name,
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "pipeline",
							Image: p.Spec.Image,
						},
					},
				},
			},
		},
	}
}

func (r *DataPipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pilotv1.DataPipeline{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
