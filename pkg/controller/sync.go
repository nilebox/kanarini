package controller

import (
	apps "k8s.io/api/apps/v1"
	kanarini "github.com/nilebox/kanarini/pkg/apis/kanarini/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"fmt"
)

// syncStatusOnly only updates Deployments Status and doesn't take any mutating actions.
func (c *CanaryDeploymentController) syncStatusOnly(cd *kanarini.CanaryDeployment, dList []*apps.Deployment, sList []*corev1.Service) error {
	// TODO
	return fmt.Errorf("Not implemented")
}
