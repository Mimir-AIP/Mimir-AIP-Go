package k8s

import (
	"context"
	"fmt"
	"path/filepath"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// Client provides Kubernetes API operations
type Client struct {
	clientset *kubernetes.Clientset
	namespace string
	ctx       context.Context
}

// NewClient creates a new Kubernetes client
func NewClient(namespace string) (*Client, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	if namespace == "" {
		namespace = "default"
	}

	return &Client{
		clientset: clientset,
		namespace: namespace,
		ctx:       context.Background(),
	}, nil
}

// getKubeConfig returns the Kubernetes configuration
func getKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig file
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// CreateWorkerJob creates a Kubernetes Job for a worker to execute a work task
func (c *Client) CreateWorkerJob(task *models.WorkTask, workerImage string) error {
	jobName := fmt.Sprintf("worker-task-%s", task.ID)

	// Parse resource requirements
	cpuRequest := task.ResourceRequirements.CPU
	if cpuRequest == "" {
		cpuRequest = "500m"
	}
	memoryRequest := task.ResourceRequirements.Memory
	if memoryRequest == "" {
		memoryRequest = "1Gi"
	}

	// Calculate limits (2x requests)
	cpuLimit := "2000m"
	memoryLimit := "4Gi"

	// Build environment variables
	envVars := []corev1.EnvVar{
		{Name: "WORKTASK_ID", Value: task.ID},
		{Name: "WORKTASK_TYPE", Value: string(task.Type)},
		{Name: "ORCHESTRATOR_URL", Value: "http://orchestrator:8080"},
	}

	// Create Job specification
	k8sJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: c.namespace,
			Labels: map[string]string{
				"app":           "mimir-worker",
				"worktask-type": string(task.Type),
				"worktask-id":   task.ID,
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: int32Ptr(300),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":           "mimir-worker",
						"worktask-type": string(task.Type),
						"worktask-id":   task.ID,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "worker-service-account",
					Containers: []corev1.Container{
						{
							Name:            "worker",
							Image:           workerImage,
							ImagePullPolicy: corev1.PullAlways,
							Env:             envVars,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    parseQuantity(cpuRequest),
									corev1.ResourceMemory: parseQuantity(memoryRequest),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    parseQuantity(cpuLimit),
									corev1.ResourceMemory: parseQuantity(memoryLimit),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "job-data", MountPath: "/app/data"},
								{Name: "model-cache", MountPath: "/app/models"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "job-data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "model-cache",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	// Create the job
	_, err := c.clientset.BatchV1().Jobs(c.namespace).Create(c.ctx, k8sJob, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create kubernetes job: %w", err)
	}

	return nil
}

// GetJobStatus retrieves the status of a Kubernetes Job
func (c *Client) GetJobStatus(jobName string) (string, error) {
	job, err := c.clientset.BatchV1().Jobs(c.namespace).Get(c.ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get job status: %w", err)
	}

	if job.Status.Succeeded > 0 {
		return "completed", nil
	}
	if job.Status.Failed > 0 {
		return "failed", nil
	}
	if job.Status.Active > 0 {
		return "running", nil
	}

	return "pending", nil
}

// DeleteJob deletes a Kubernetes Job
func (c *Client) DeleteJob(jobName string) error {
	propagationPolicy := metav1.DeletePropagationBackground
	err := c.clientset.BatchV1().Jobs(c.namespace).Delete(c.ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}
	return nil
}

// GetActiveWorkerCount returns the number of active worker jobs
func (c *Client) GetActiveWorkerCount() (int, error) {
	jobs, err := c.clientset.BatchV1().Jobs(c.namespace).List(c.ctx, metav1.ListOptions{
		LabelSelector: "app=mimir-worker",
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list jobs: %w", err)
	}

	activeCount := 0
	for _, job := range jobs.Items {
		if job.Status.Active > 0 {
			activeCount++
		}
	}

	return activeCount, nil
}

// Helper functions
func int32Ptr(i int32) *int32 {
	return &i
}

func parseQuantity(s string) resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		// Return a default value if parsing fails
		return resource.MustParse("0")
	}
	return q
}
