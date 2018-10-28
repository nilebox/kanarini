package controller

import (
	kanarini "github.com/nilebox/kanarini/pkg/apis/kanarini/v1alpha1"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"github.com/nilebox/kanarini/pkg/kubernetes/pkg/controller"
	labelsutil "github.com/nilebox/kanarini/pkg/kubernetes/pkg/util/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"fmt"
	"github.com/pkg/errors"
)

// rolloutRolling implements the logic for rolling a new replica set.
func (c *CanaryDeploymentController) rolloutCanary(cd *kanarini.CanaryDeployment, dList []*apps.Deployment, rsList []*corev1.Service) error {
	// Create a canary track deployment
	canaryTrackDeployment, err := c.createTrackDeployment(cd, dList, &cd.Spec.Tracks.Canary, kanarini.CanaryTrackName)
	if err != nil {
		return err
	}
	// Wait for a canary track deployment to succeed
	if !IsReady(canaryTrackDeployment) {
		// TODO exponential delay + watch on deployment objects
		return errors.New("Canary track deployment is not ready")
	}
	// Create a canary Service
	// TODO
	// Wait for metric delay to expire
	// TODO
	// Check the metric value and decide whether Service is healthy
	// TODO
	// Create a stable track deployment
	stableTrackDeployment, err := c.createTrackDeployment(cd, dList, &cd.Spec.Tracks.Stable, kanarini.CanaryTrackName)
	// Wait for a canary track deployment to succeed
	// Wait for a canary track deployment to succeed
	if !IsReady(stableTrackDeployment) {
		// TODO exponential delay + watch on deployment objects
		return errors.New("Canary track deployment is not ready")
	}
	// Create a stable Service
	// TODO
	// Done
	return nil
}

func (c *CanaryDeploymentController) createTrackDeployment(cd *kanarini.CanaryDeployment, dList []*apps.Deployment, trackSpec *kanarini.DeploymentTrackSpec, trackName kanarini.CanaryDeploymentTrackName) (*apps.Deployment, error) {
	newDeploymentTemplate := *cd.Spec.PodTemplate.DeepCopy()
	podTemplateSpecHash := controller.ComputeHash(&newDeploymentTemplate, nil)
	newDeploymentTemplate.Labels = labelsutil.CloneAndAddLabel(newDeploymentTemplate.Labels, "track", string(trackName))
	newDeploymentTemplate.Labels = labelsutil.CloneAndAddLabel(newDeploymentTemplate.Labels, kanarini.PodTemplateHashLabelKey, podTemplateSpecHash)
	newDeployment := apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			// Make the name deterministic, to ensure idempotence
			Name:            cd.Name + "-" + string(trackName),
			Namespace:       cd.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(cd, kanarini.CanaryDeploymentGVK)},
			Labels:          newDeploymentTemplate.Labels,
		},
		Spec: apps.DeploymentSpec{
			Template: newDeploymentTemplate,
			Replicas: trackSpec.Replicas,
			Selector: cd.Spec.Selector,
			MinReadySeconds: cd.Spec.MinReadySeconds,
			ProgressDeadlineSeconds: cd.Spec.ProgressDeadlineSeconds,
		},
	}
	// TODO this means we ignore selector from CD spec, we should extend the selector separately instead
	newDeployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: newDeploymentTemplate.Labels,
	}

	// Create the new Deployment. If it already exists, then we need to check for possible
	// conflicts. If there is any other error, we need to report it in the status of
	// the CanaryDeployment.
	alreadyExists := false
	createdDeployment, err := c.kubeClient.AppsV1().Deployments(cd.Namespace).Create(&newDeployment)
	switch {
	// We may end up hitting this due to a slow cache or a fast resync of the Deployment.
	case apierrors.IsAlreadyExists(err):
		alreadyExists = true

		// Fetch a copy of the Deployment.
		d, dErr := c.dLister.Deployments(newDeployment.Namespace).Get(newDeployment.Name)
		if dErr != nil {
			return nil, dErr
		}

		controllerRef := metav1.GetControllerOf(d)
		if controllerRef != nil && controllerRef.UID == cd.UID {
			createdDeployment = d
			err = nil
			// TODO: also need to check contents to make sure that there were no manual changes to deployment
			if createdDeployment.Labels[kanarini.PodTemplateHashLabelKey] != newDeployment.Labels[kanarini.PodTemplateHashLabelKey] {
				// Pod template hashes are different; need to update the deployment
				createdDeployment := createdDeployment.DeepCopy()
				createdDeployment.Labels = newDeployment.Labels
				createdDeployment.Spec = newDeployment.Spec
				createdDeployment, err = c.kubeClient.AppsV1().Deployments(createdDeployment.Namespace).Update(createdDeployment)
				if err != nil {
					return nil, err
				}
			}
			break
		}

		msg := fmt.Sprintf("New Deployment conflicts with existing one: %q", newDeployment.Name)
		if HasProgressDeadline(cd) {
			cond := NewCanaryDeploymentCondition(kanarini.CanaryDeploymentProgressing, corev1.ConditionFalse, FailedDeploymentCreateReason, msg)
			SetCanaryDeploymentCondition(&cd.Status, *cond)
			// We don't really care about this error at this point, since we have a bigger issue to report.
			_, _ = c.kanariniClient.CanaryDeployments(cd.Namespace).UpdateStatus(cd)
		}
		c.eventRecorder.Eventf(cd, corev1.EventTypeWarning, FailedDeploymentCreateReason, msg)
		return nil, fmt.Errorf("new Deployment conflicts with existing one: %q", newDeployment.Name)
	case err != nil:
		msg := fmt.Sprintf("Failed to create new Deployment %q: %v", newDeployment.Name, err)
		if HasProgressDeadline(cd) {
			cond := NewCanaryDeploymentCondition(kanarini.CanaryDeploymentProgressing, corev1.ConditionFalse, FailedDeploymentCreateReason, msg)
			SetCanaryDeploymentCondition(&cd.Status, *cond)
			// We don't really care about this error at this point, since we have a bigger issue to report.
			_, _ = c.kanariniClient.CanaryDeployments(cd.Namespace).UpdateStatus(cd)
		}
		c.eventRecorder.Eventf(cd, corev1.EventTypeWarning, FailedDeploymentCreateReason, msg)
		return nil, err
	}

	needsUpdate := false
	if !alreadyExists && HasProgressDeadline(cd) {
		msg := fmt.Sprintf("Created new Deployment %q", createdDeployment.Name)
		condition := NewCanaryDeploymentCondition(kanarini.CanaryDeploymentProgressing, corev1.ConditionTrue, NewDeploymentReason, msg)
		SetCanaryDeploymentCondition(&cd.Status, *condition)
		needsUpdate = true
	}
	if needsUpdate {
		_, err = c.kanariniClient.CanaryDeployments(cd.Namespace).UpdateStatus(cd)
	}
	return createdDeployment, err
}
