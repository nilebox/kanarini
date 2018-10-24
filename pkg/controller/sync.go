package controller

import (
	"fmt"

	kanarini "github.com/nilebox/kanarini/pkg/apis/kanarini/v1alpha1"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// syncStatusOnly only updates Deployments Status and doesn't take any mutating actions.
func (c *CanaryDeploymentController) syncStatusOnly(cd *kanarini.CanaryDeployment, dList []*apps.Deployment, sList []*corev1.Service) error {
	// TODO
	return fmt.Errorf("Not implemented")
}
