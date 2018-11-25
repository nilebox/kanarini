package controller

import (
	apps "k8s.io/api/apps/v1"
)

const timedOutReason = "ProgressDeadlineExceeded"

func IsDeploymentReady(deployment *apps.Deployment) bool {
	replicas := deployment.Spec.Replicas

	generation := deployment.Generation
	observedGeneration := deployment.Status.ObservedGeneration
	updatedReplicas := deployment.Status.UpdatedReplicas
	availableReplicas := deployment.Status.AvailableReplicas

	if generation <= observedGeneration {
		progressingCond := GetDeploymentCondition(&deployment.Status, apps.DeploymentProgressing)
		if progressingCond != nil && progressingCond.Reason == timedOutReason {
			// Deployment exceeded its progress deadline
			// TODO return some final error
			return false
		}

		if replicas != nil && updatedReplicas < *replicas {
			return false
		}

		if deployment.Status.Replicas > updatedReplicas {
			return false
		}

		if availableReplicas < updatedReplicas {
			return false
		}

		return true
	}

	return false
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
