package controller

import (
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func IsReady(deployment *apps.Deployment) bool {
	status := &deployment.Status

	if deployment.Generation > status.ObservedGeneration {
		return false
	}

	availableCondition := GetDeploymentCondition(status, apps.DeploymentAvailable)
	progressingCondition := GetDeploymentCondition(status, apps.DeploymentProgressing)
	failureCondition := GetDeploymentCondition(status, apps.DeploymentReplicaFailure)

	if progressingCondition != nil && progressingCondition.Status != corev1.ConditionFalse {
		return false
	}

	if failureCondition != nil && failureCondition.Status != corev1.ConditionFalse {
		return false
	}

	if availableCondition == nil || failureCondition.Status != corev1.ConditionTrue {
		return false
	}

	return true
}

// GetDeploymentCondition returns the condition with the provided type.
func GetDeploymentCondition(status *apps.DeploymentStatus, condType apps.DeploymentConditionType) *apps.DeploymentCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}
