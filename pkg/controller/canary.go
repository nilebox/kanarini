package controller

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	kanarini "github.com/nilebox/kanarini/pkg/apis/kanarini/v1alpha1"
	"github.com/nilebox/kanarini/pkg/kubernetes/pkg/controller"
	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// rolloutRolling implements the logic for rolling a new replica set.
func (c *CanaryDeploymentController) rolloutCanary(cd *kanarini.CanaryDeployment, dList []*apps.Deployment) error {
	cdBytes, _ := json.Marshal(cd)
	glog.V(4).Infof("CanaryDeployment: %v", string(cdBytes))

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
	glog.V(4).Info("Canary track deployment is ready!")
	// Wait for metric delay to expire
	// TODO wait for metric delay to expire
	time.Sleep(2*time.Minute)
	// Check the metric value and decide whether Service is healthy
	healthy, err := c.isServiceHealthy(cd, &cd.Spec.Tracks.Canary)
	if err != nil {
		return err
	}
	if !healthy {
		// TODO rollback canary deployment and don't proceed for stable deployment
		glog.V(4).Info("Canary track deployment is not healthy. Stopping propagation")
		return nil
	}
	// Create a stable track deployment
	stableTrackDeployment, err := c.createTrackDeployment(cd, dList, &cd.Spec.Tracks.Stable, kanarini.StableTrackName)
	// Wait for a canary track deployment to succeed
	// Wait for a canary track deployment to succeed
	if !IsReady(stableTrackDeployment) {
		// TODO exponential delay + watch on deployment objects
		return errors.New("Stable track deployment is not ready")
	}
	glog.V(4).Info("Stable track deployment is ready!")
	// Done
	return nil
}

func (c *CanaryDeploymentController) isServiceHealthy(cd *kanarini.CanaryDeployment, trackSpec *kanarini.DeploymentTrackSpec) (bool, error) {
	for _, metricSpec := range trackSpec.Metrics {
		switch metricSpec.Type {
		case kanarini.ObjectMetricSourceType:
			metricSelector, err := metav1.LabelSelectorAsSelector(metricSpec.Object.Metric.Selector)
			if err != nil {
				return false, err
			}
			val, _, err := c.metricsClient.GetObjectMetric(metricSpec.Object.Metric.Name, cd.Namespace, &metricSpec.Object.DescribedObject, metricSelector)
			glog.V(4).Infof("Custom metric value: %v", val)
			glog.V(4).Infof("Custom metric target value: %v", metricSpec.Object.Target.Value.MilliValue())
			// If metric value is equal or greater than target value, it's considered unhealthy
			if val >= metricSpec.Object.Target.Value.MilliValue() {
				return false, nil
			}
		case kanarini.ExternalMetricSourceType:
			metricSelector, err := metav1.LabelSelectorAsSelector(metricSpec.External.Metric.Selector)
			if err != nil {
				return false, err
			}
			metrics, _, err := c.metricsClient.GetExternalMetric(metricSpec.Object.Metric.Name, cd.Namespace, metricSelector)
			var sum int64 = 0
			for _, val := range metrics {
				sum = sum + val
			}
			// If metric value is equal or greater than target value, it's considered unhealthy
			if sum >= metricSpec.External.Target.Value.MilliValue() {
				return false, nil
			}
		default:
			return false, errors.New(fmt.Sprintf("Unexpected metric source type: %v", metricSpec.Type))
		}
	}

	return true, nil
}

func (c *CanaryDeploymentController) createTrackDeployment(cd *kanarini.CanaryDeployment, dList []*apps.Deployment, trackSpec *kanarini.DeploymentTrackSpec, trackName kanarini.CanaryDeploymentTrackName) (*apps.Deployment, error) {
	newDeploymentTemplate := *cd.Spec.Template.DeepCopy()
	templateHash := controller.ComputeHash(&newDeploymentTemplate, nil)
	_ = templateHash // Useful for rollback implementation
	annotations := newDeploymentTemplate.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[kanarini.TemplateHashAnnotationKey] = templateHash
	labels := newDeploymentTemplate.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	for k, v := range trackSpec.Labels {
		labels[k] = v
	}
	newDeployment := apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			// Make the name deterministic, to ensure idempotence
			Name:            cd.Name + "-" + string(trackName),
			Namespace:       cd.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(cd, kanarini.CanaryDeploymentGVK)},
			Annotations: annotations,
			Labels:          labels,
		},
		Spec: apps.DeploymentSpec{
			Template:                newDeploymentTemplate,
			Replicas:                trackSpec.Replicas,
			Selector:                cd.Spec.Selector,
			MinReadySeconds:         cd.Spec.MinReadySeconds,
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
			if createdDeployment.Annotations[kanarini.TemplateHashAnnotationKey] != newDeployment.Annotations[kanarini.TemplateHashAnnotationKey] {
				// Pod template hashes are different; need to update the deployment
				createdDeployment := createdDeployment.DeepCopy()
				createdDeployment.Annotations = newDeployment.Annotations
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
