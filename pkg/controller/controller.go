package controller

import (
	kanariniclientset "github.com/nilebox/kanarini/pkg/client/clientset_generated/clientset/typed/kanarini/v1alpha1"
	"k8s.io/client-go/kubernetes"
)

type Controller interface {
	Run(workers int, stopCh <-chan struct{})
}

// controller is a concrete Controller.
type controller struct {
	kubeClient kubernetes.Interface
	kanariniClient        kanariniclientset.KanariniV1alpha1Interface
}
