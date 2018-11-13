package v1alpha1

import (
	"github.com/nilebox/kanarini/pkg/apis/kanarini"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CanaryDeploymentResourceSingular = "canarydeployment"
	CanaryDeploymentResourcePlural   = "canarydeployments"
	CanaryDeploymentResourceVersion  = "v1alpha1"
	CanaryDeploymentResourceKind     = "CanaryDeployment"
	CanaryDeploymentResourceListKind = CanaryDeploymentResourceKind + "List"

	CanaryDeploymentResourceName = CanaryDeploymentResourcePlural + "." + kanarini.GroupName
)

var (
	CanaryDeploymentGVK = SchemeGroupVersion.WithKind(CanaryDeploymentResourceKind)
)

// +k8s:deepcopy-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CanaryDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []CanaryDeployment `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CanaryDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              CanaryDeploymentSpec   `json:"spec"`
	Status            CanaryDeploymentStatus `json:"status,omitempty"`
}

const (
	TemplateHashAnnotationKey string = kanarini.GroupName + "/template-hash"
)

type CanaryDeploymentSpec struct {

	// Label selector for pods. Existing ReplicaSets whose pods are
	// selected by this will be the ones affected by this deployment.
	// It must match the pod template's labels.
	Selector *metav1.LabelSelector `json:"selector"`

	// Template describes the pods that will be created.
	Template corev1.PodTemplateSpec `json:"template"`
	Tracks          CanaryDeploymentTracks `json:"tracks"`

	// Minimum number of seconds for which a newly created pod should be ready
	// without any of its container crashing, for it to be considered available.
	// Defaults to 0 (pod will be considered available as soon as it is ready)
	// +optional
	MinReadySeconds int32 `json:"minReadySeconds,omitempty"`

	// The maximum time in seconds for a deployment to make progress before it
	// is considered to be failed. The deployment controller will continue to
	// process failed deployments and a condition with a ProgressDeadlineExceeded
	// reason will be surfaced in the deployment status. Note that progress will
	// not be estimated during the time a deployment is paused. Defaults to 600s.
	ProgressDeadlineSeconds *int32 `json:"progressDeadlineSeconds,omitempty"`
}

// DeploymentStatus is the most recently observed status of the CanaryDeployment.
type CanaryDeploymentStatus struct {
	// The generation observed by the deployment controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Represents the latest available observations of a deployment's current state.
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []CanaryDeploymentCondition `json:"conditions,omitempty"`

	// Checkpoint used to calculate delay to check metric for canary Deployment
	CanaryDeploymentReadyStatusCheckpoint *DeploymentReadyStatusCheckpoint `json:"deploymentReadyStatusCheckpoint,omitempty"`
}

type DeploymentReadyStatusCheckpoint struct {
	TemplateHash string `json:"templateHash,omitempty"`
	LatestReadyTimestamp metav1.Time `json:"latestReadyTimestamp,omitempty"`
	MetricCheckResult MetricCheckResult `json:"metricCheckResult,omitempty"`
}

type MetricCheckResult string

const (
	MetricCheckResultUnknown MetricCheckResult = ""
	MetricCheckResultSuccess MetricCheckResult = "Success"
	MetricCheckResultFailure MetricCheckResult = "Failure"
)

type CanaryDeploymentConditionType string

// These are valid conditions of a deployment.
const (
	// Available means the deployment is available, ie. at least the minimum available
	// replicas required are up and running for at least minReadySeconds.
	CanaryDeploymentAvailable CanaryDeploymentConditionType = "Available"
	// Progressing means the deployment is progressing. Progress for a deployment is
	// considered when a new replica set is created or adopted, and when new pods scale
	// up or old pods scale down. Progress is not estimated for paused deployments or
	// when progressDeadlineSeconds is not specified.
	CanaryDeploymentProgressing CanaryDeploymentConditionType = "Progressing"
	// Failure is added in a deployment when one of its pods fails to be created
	// or deleted.
	CanaryDeploymentFailure CanaryDeploymentConditionType = "Failure"
)

// CanaryDeploymentCondition describes the state of a deployment at a certain point.
type CanaryDeploymentCondition struct {
	// Type of deployment condition.
	Type CanaryDeploymentConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

type CanaryDeploymentTracks struct {
	Canary DeploymentTrackSpec `json:"canary,omitempty"`
	Stable DeploymentTrackSpec `json:"stable,omitempty"`
}

type DeploymentTrackSpec struct {
	// Number of desired pods. This is a pointer to distinguish between explicit
	// zero and not specified. Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Labels to add to pods to distinguish between tracks
	Labels map[string]string `json:"labels,omitempty"`

	// Delay to wait before checking metric values
	MetricCheckDelaySeconds int32 `json:"metricCheckDelaySeconds,omitempty"`

	// Metrics contains the specifications for which to use to determine whether
	// the service is healthy.
	// +optional
	Metrics []MetricSpec `json:"metrics,omitempty"`
}

// MetricSpec specifies how to scale based on a single metric
// (only `type` and one other matching field should be set at once).
type MetricSpec struct {
	// type is the type of metric source.  It should be one of "Object"
	// or "External", each mapping to a matching field in the object.
	Type MetricSourceType `json:"type"`

	// object refers to a metric describing a single kubernetes object
	// (for example, hits-per-second on an Ingress object).
	// +optional
	Object *ObjectMetricSource `json:"object,omitempty"`

	// External refers to a global metric that is not associated
	// with any Kubernetes object. It allows making decision based on information
	// coming from components running outside of cluster
	// (for example length of queue in cloud messaging service, or
	// QPS from loadbalancer running outside of cluster).
	// +optional
	External *ExternalMetricSource
}

// MetricSourceType indicates the type of metric.
type MetricSourceType string

var (
	// ObjectMetricSourceType is a metric describing a kubernetes object
	// (for example, hits-per-second on an Ingress object).
	ObjectMetricSourceType MetricSourceType = "Object"
	// ExternalMetricSourceType is a global metric that is not associated
	// with any Kubernetes object. It allows autoscaling based on information
	// coming from components running outside of cluster
	// (for example length of queue in cloud messaging service, or
	// QPS from loadbalancer running outside of cluster).
	ExternalMetricSourceType MetricSourceType = "External"
)

// ObjectMetricSource indicates how to scale on a metric describing a
// kubernetes object (for example, hits-per-second on an Ingress object).
type ObjectMetricSource struct {
	DescribedObject CrossVersionObjectReference `json:"describedObject"`
	// target specifies the target value for the given metric
	Target MetricTarget `json:"target"`
	// metric identifies the target metric by name and selector
	Metric MetricIdentifier `json:"metric"`
}

// ExternalMetricSource indicates how to scale on a metric not associated with
// any Kubernetes object (for example length of queue in cloud
// messaging service, or QPS from loadbalancer running outside of cluster).
type ExternalMetricSource struct {
	// Metric identifies the target metric by name and selector
	Metric MetricIdentifier
	// Target specifies the target value for the given metric
	Target MetricTarget
}

// CrossVersionObjectReference contains enough information to let you identify the referred resource.
type CrossVersionObjectReference struct {
	// Kind of the referent; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds"
	Kind string `json:"kind"`
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name"`
	// API version of the referent
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}

// MetricTarget defines the target value of a specific metric
type MetricTarget struct {
	// type represents the metric type
	Type MetricTargetType `json:"type"`
	// value is the target value of the metric (as a quantity).
	// +optional
	Value *resource.Quantity `json:"value,omitempty"`
}

// MetricTargetType specifies the type of metric being targeted, only
// "Value" is supported
type MetricTargetType string

var (
	// ValueMetricType declares a MetricTarget is a raw value
	ValueMetricType MetricTargetType = "Value"
)

// MetricIdentifier defines the name and optionally selector for a metric
type MetricIdentifier struct {
	// name is the name of the given metric
	Name string `json:"name"`
	// selector is the string-encoded form of a standard kubernetes label selector for the given metric
	// When set, it is passed as an additional parameter to the metrics server for more specific metrics scoping.
	// When unset, just the metricName will be used to gather metrics.
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

type CanaryDeploymentTrackName string

const (
	CanaryTrackName CanaryDeploymentTrackName = "canary"
	StableTrackName CanaryDeploymentTrackName = "stable"
)
