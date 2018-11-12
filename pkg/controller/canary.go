package controller

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	kanarini "github.com/nilebox/kanarini/pkg/apis/kanarini/v1alpha1"
	"github.com/nilebox/kanarini/pkg/kubernetes/pkg/controller"
	labelsutil "github.com/nilebox/kanarini/pkg/kubernetes/pkg/util/labels"
	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// rolloutRolling implements the logic for rolling a new replica set.
func (c *CanaryDeploymentController) rolloutCanary(cd *kanarini.CanaryDeployment, dList []*apps.Deployment, sList []*corev1.Service) error {
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
	// Create a canary Service
	_, err = c.createTrackService(cd, sList, &cd.Spec.Tracks.Canary, kanarini.CanaryTrackName)
	if err != nil {
		return err
	}
	// Wait for metric delay to expire
	// TODO wait for metric delay to expire
	// Check the metric value and decide whether Service is healthy
	healthy, err := c.isServiceHealthy(cd, &cd.Spec.Tracks.Canary)
	if err != nil {
		return err
	}
	if !healthy {
		// TODO rollback canary deployment and don't proceed for stable deployment
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
	// Create a stable Service
	_, err = c.createTrackService(cd, sList, &cd.Spec.Tracks.Stable, kanarini.StableTrackName)
	if err != nil {
		return err
	}
	// Done
	return nil
}

func (c *CanaryDeploymentController) isServiceHealthy(cd *kanarini.CanaryDeployment, trackSpec *kanarini.DeploymentTrackSpec) (bool, error) {
	for _, metricSpec := range trackSpec.Metrics {
		var metricVal int64 = 0
		switch metricSpec.Type {
		case kanarini.ObjectMetricSourceType:
			metricSelector, err := metav1.LabelSelectorAsSelector(metricSpec.Object.Metric.Selector)
			if err != nil {
				return false, err
			}
			val, _, err := c.metricsClient.GetObjectMetric(metricSpec.Object.Metric.Name, cd.Namespace, &metricSpec.Object.DescribedObject, metricSelector)
			glog.V(4).Infof("Custom metric value: %v", val)
			metricVal = val
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
			metricVal = sum
		default:
			return false, errors.New(fmt.Sprintf("Unexpected metric source type: %v", metricSpec.Type))
		}

		// If metric value is equal or greater than target value, it's considered unhealthy
		if metricVal >= metricSpec.External.Target.Value.MilliValue() {
			return false, nil
		}
	}

	return true, nil
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

func (c *CanaryDeploymentController) createTrackService(cd *kanarini.CanaryDeployment, sList []*corev1.Service, trackSpec *kanarini.DeploymentTrackSpec, trackName kanarini.CanaryDeploymentTrackName) (*corev1.Service, error) {
	newServiceObjectMeta := *cd.Spec.ServiceTemplate.ObjectMeta.DeepCopy()
	newServiceSpec := *cd.Spec.ServiceTemplate.Spec.DeepCopy()
	serviceSpecHash := controller.ComputeHash(&newServiceSpec, nil)
	newServiceObjectMeta.Labels = labelsutil.CloneAndAddLabel(newServiceObjectMeta.Labels, "track", string(trackName))
	newServiceObjectMeta.Labels = labelsutil.CloneAndAddLabel(newServiceObjectMeta.Labels, kanarini.ServiceTemplateHashLabelKey, serviceSpecHash)
	newService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			// Make the name deterministic, to ensure idempotence
			Name:            cd.Name + "-" + string(trackName),
			Namespace:       cd.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(cd, kanarini.CanaryDeploymentGVK)},
			Labels:          newServiceObjectMeta.Labels,
			Annotations:     newServiceObjectMeta.Annotations,
		},
		// TODO: modify selector labels to include "canary"/"stable" tracks
		Spec: newServiceSpec,
	}

	bytes, _ := json.Marshal(&newService)
	glog.V(4).Infof("Service spec: %v", string(bytes))

	// Create the new Service. If it already exists, then we need to check for possible
	// conflicts. If there is any other error, we need to report it in the status of
	// the CanaryDeployment.
	alreadyExists := false
	createdService, err := c.kubeClient.CoreV1().Services(cd.Namespace).Create(&newService)
	switch {
	// We may end up hitting this due to a slow cache or a fast resync of the Service.
	case apierrors.IsAlreadyExists(err):
		alreadyExists = true

		// Fetch a copy of the Service.
		s, sErr := c.sLister.Services(newService.Namespace).Get(newService.Name)
		if sErr != nil {
			return nil, sErr
		}

		controllerRef := metav1.GetControllerOf(s)
		if controllerRef != nil && controllerRef.UID == cd.UID {
			createdService = s
			err = nil
			if createdService.Labels[kanarini.ServiceTemplateHashLabelKey] != newService.Labels[kanarini.ServiceTemplateHashLabelKey] {
				// Service template hashes are different; need to update the deployment
				createdService := createdService.DeepCopy()
				createdService.Labels = newService.Labels
				createdService.Spec = newService.Spec
				createdService, err = c.kubeClient.CoreV1().Services(cd.Namespace).Update(createdService)
				if err != nil {
					return nil, err
				}
			}
			break
		}

		msg := fmt.Sprintf("New Service conflicts with existing one: %q", newService.Name)
		if HasProgressDeadline(cd) {
			cond := NewCanaryDeploymentCondition(kanarini.CanaryDeploymentProgressing, corev1.ConditionFalse, FailedServiceCreateReason, msg)
			SetCanaryDeploymentCondition(&cd.Status, *cond)
			// We don't really care about this error at this point, since we have a bigger issue to report.
			_, _ = c.kanariniClient.CanaryDeployments(cd.Namespace).Update(cd)
		}
		c.eventRecorder.Eventf(cd, corev1.EventTypeWarning, FailedServiceCreateReason, msg)
		return nil, fmt.Errorf("new Service conflicts with existing one: %q", newService.Name)
	case err != nil:
		msg := fmt.Sprintf("Failed to create new Service %q: %v", newService.Name, err)
		if HasProgressDeadline(cd) {
			cond := NewCanaryDeploymentCondition(kanarini.CanaryDeploymentProgressing, corev1.ConditionFalse, FailedServiceCreateReason, msg)
			SetCanaryDeploymentCondition(&cd.Status, *cond)
			// We don't really care about this error at this point, since we have a bigger issue to report.
			_, _ = c.kanariniClient.CanaryDeployments(cd.Namespace).Update(cd)
		}
		c.eventRecorder.Eventf(cd, corev1.EventTypeWarning, FailedServiceCreateReason, msg)
		return nil, err
	}

	needsUpdate := false
	if !alreadyExists && HasProgressDeadline(cd) {
		msg := fmt.Sprintf("Created new Service %q", createdService.Name)
		condition := NewCanaryDeploymentCondition(kanarini.CanaryDeploymentProgressing, corev1.ConditionTrue, NewServiceReason, msg)
		SetCanaryDeploymentCondition(&cd.Status, *condition)
		needsUpdate = true
	}
	if needsUpdate {
		_, err = c.kanariniClient.CanaryDeployments(cd.Namespace).UpdateStatus(cd)
	}
	return createdService, err
}
