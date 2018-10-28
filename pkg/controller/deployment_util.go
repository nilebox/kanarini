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
	failureCondition := GetDeploymentCondition(status, apps.DeploymentReplicaFailure)

	if failureCondition != nil && failureCondition.Status != corev1.ConditionFalse {
		return false
	}

	if availableCondition == nil || availableCondition.Status != corev1.ConditionTrue {
		return false
	}

	var desiredReplicas int32 = 1
	if deployment.Spec.Replicas != nil {
		desiredReplicas = *deployment.Spec.Replicas
	}
	// TODO also check updatedReplicas?
	if status.ReadyReplicas != desiredReplicas {
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
