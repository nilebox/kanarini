package controller

import (
	"math"

	kanarini "github.com/nilebox/kanarini/pkg/apis/kanarini/v1alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CreatedDeploymentReason        = "CreatedDeployment"
	UpdatedDeploymentReason        = "UpdatedDeployment"
	DelayMetricsCheckReason        = "DelayMetricsCheck"
	MetricsCheckResultReason       = "MetricsCheckResult"
	MetricsCheckUnsuccessfulReason = "MetricsCheckUnsuccessful"
	DoneProcessingReason           = "DoneProcessing"

	DoneProcessingMessage = "Finished reconciling update"
)

const (
	// Reasons for deployment conditions
	//
	// Progressing:
	//
	// FailedDeploymentCreateReason is added in a canary deployment when it cannot create a new deployment.
	FailedDeploymentCreateReason = "DeploymentCreateError"
	// NewDeploymentReason is added in a canary deployment when it creates a new deployment.
	NewDeploymentReason = "NewDeploymentCreated"

	// Reasons for deployment conditions
	//
	// Progressing:
	//
	// FailedServiceCreateReason is added in a canary deployment when it cannot create a new service.
	FailedServiceCreateReason = "DeploymentCreateError"
	// NewServiceReason is added in a canary deployment when it creates a new service.
	NewServiceReason = "NewServiceCreated"
)

func HasProgressDeadline(d *kanarini.CanaryDeployment) bool {
	return d.Spec.ProgressDeadlineSeconds != nil && *d.Spec.ProgressDeadlineSeconds != math.MaxInt32
}

// NewCanaryDeploymentCondition creates a new deployment condition.
func NewCanaryDeploymentCondition(condType kanarini.CanaryDeploymentConditionType, status v1.ConditionStatus, reason, message string) *kanarini.CanaryDeploymentCondition {
	return &kanarini.CanaryDeploymentCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetCanaryDeploymentCondition returns the condition with the provided type.
func GetCanaryDeploymentCondition(status kanarini.CanaryDeploymentStatus, condType kanarini.CanaryDeploymentConditionType) *kanarini.CanaryDeploymentCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetCanaryDeploymentCondition updates the deployment to include the provided condition. If the condition that
// we are about to add already exists and has the same status and reason then we are not going to update.
func SetCanaryDeploymentCondition(status *kanarini.CanaryDeploymentStatus, condition kanarini.CanaryDeploymentCondition) {
	currentCond := GetCanaryDeploymentCondition(*status, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}
	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}
	newConditions := filterOutCanaryDeploymentCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// RemoveCanaryDeploymentCondition removes the deployment condition with the provided type.
func RemoveCanaryDeploymentCondition(status *kanarini.CanaryDeploymentStatus, condType kanarini.CanaryDeploymentConditionType) {
	status.Conditions = filterOutCanaryDeploymentCondition(status.Conditions, condType)
}

// filterOutCanaryDeploymentCondition returns a new slice of deployment conditions without conditions with the provided type.
func filterOutCanaryDeploymentCondition(conditions []kanarini.CanaryDeploymentCondition, condType kanarini.CanaryDeploymentConditionType) []kanarini.CanaryDeploymentCondition {
	var newConditions []kanarini.CanaryDeploymentCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
