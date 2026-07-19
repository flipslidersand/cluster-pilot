package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
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
	// Metrics
	reconcileTotal   prometheus.Counter
	pipelineDuration prometheus.Histogram
	tracer           trace.Tracer
}

// +kubebuilder:rbac:groups=clusterpilot.dev,resources=datapipelines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clusterpilot.dev,resources=datapipelines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

func (r *DataPipelineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if r.reconcileTotal != nil {
			r.reconcileTotal.Inc()
		}
		if r.pipelineDuration != nil {
			r.pipelineDuration.Observe(duration)
		}
	}()

	if r.tracer != nil {
		var span trace.Span
		ctx, span = r.tracer.Start(ctx, "Reconcile")
		defer span.End()
	}

	pipeline := &pilotv1.DataPipeline{}
	if err := r.Get(ctx, req.NamespacedName, pipeline); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Terminal state: only Succeeded (and not scheduled). Failed may have retries, Scheduled continue.
	if pipeline.Status.Phase == pilotv1.DataPipelineSucceeded && pipeline.Spec.Schedule == "" {
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

	// Job exists — check timeout, retries, then sync status.
	if err := r.checkTimeout(ctx, pipeline, job); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.checkRetry(ctx, pipeline, job); err != nil {
		return ctrl.Result{}, err
	}

	result, err := r.syncStatus(ctx, logger, pipeline, job)
	if err != nil {
		return result, err
	}

	// If scheduled and terminal, schedule next run.
	if pipeline.Spec.Schedule != "" && (pipeline.Status.Phase == pilotv1.DataPipelineSucceeded || pipeline.Status.Phase == pilotv1.DataPipelineFailed) {
		return r.scheduleNext(ctx, pipeline)
	}

	return result, nil
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

func (r *DataPipelineReconciler) checkTimeout(ctx context.Context, p *pilotv1.DataPipeline, job *batchv1.Job) error {
	if p.Spec.Timeout == nil || p.Status.LastRunTime == nil {
		return nil
	}
	expiry := p.Status.LastRunTime.Add(p.Spec.Timeout.Duration)
	if time.Now().After(expiry) && job.Status.Active > 0 {
		if err := r.Delete(ctx, job); err != nil {
			return err
		}
		p.Status.Phase = pilotv1.DataPipelineFailed
		p.Status.Message = "timeout exceeded"
		return r.Status().Update(ctx, p)
	}
	return nil
}

func (r *DataPipelineReconciler) checkRetry(ctx context.Context, p *pilotv1.DataPipeline, job *batchv1.Job) error {
	if job.Status.Failed == 0 {
		return nil
	}
	maxAttempts := int32(p.Spec.Retries) + 1
	if p.Status.Runs < maxAttempts {
		if err := r.Delete(ctx, job); err != nil {
			return err
		}
		newJobName := fmt.Sprintf("%s-run-%d", p.Name, p.Status.Runs+1)
		_, err := r.createJob(ctx, p, newJobName)
		return err
	}
	return nil
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

func (r *DataPipelineReconciler) scheduleNext(ctx context.Context, p *pilotv1.DataPipeline) (ctrl.Result, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(p.Spec.Schedule)
	if err != nil {
		p.Status.Message = fmt.Sprintf("invalid cron expression: %v", err)
		_ = r.Status().Update(ctx, p)
		return ctrl.Result{}, nil
	}

	nextRun := schedule.Next(time.Now())
	nextTime := metav1.NewTime(nextRun)
	p.Status.NextRunTime = &nextTime
	p.Status.Phase = pilotv1.DataPipelinePending
	p.Status.Runs = 0
	p.Status.Message = fmt.Sprintf("scheduled for %s", nextRun.Format(time.RFC3339))
	if err := r.Status().Update(ctx, p); err != nil {
		return ctrl.Result{}, err
	}

	waitDur := time.Until(nextRun)
	return ctrl.Result{RequeueAfter: waitDur}, nil
}

func (r *DataPipelineReconciler) desiredJob(p *pilotv1.DataPipeline, name string) *batchv1.Job {
	job := &batchv1.Job{
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
	if p.Spec.Timeout != nil {
		secs := int64(p.Spec.Timeout.Seconds())
		job.Spec.ActiveDeadlineSeconds = &secs
	}
	return job
}

func (r *DataPipelineReconciler) SetupMetrics() error {
	var err error
	r.reconcileTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "datapipeline_reconcile_total",
		Help: "Total number of DataPipeline reconciliations",
	})
	if err = prometheus.Register(r.reconcileTotal); err != nil {
		// Counter already registered, that's okay
	}

	r.pipelineDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "datapipeline_reconcile_duration_seconds",
		Help:    "Reconciliation duration in seconds",
		Buckets: prometheus.ExponentialBuckets(0.01, 2, 10),
	})
	if err = prometheus.Register(r.pipelineDuration); err != nil {
		// Histogram already registered, that's okay
	}

	r.tracer = otel.Tracer("cluster-pilot")
	return nil
}

func (r *DataPipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	_ = r.SetupMetrics()
	return ctrl.NewControllerManagedBy(mgr).
		For(&pilotv1.DataPipeline{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
