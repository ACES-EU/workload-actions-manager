package actions

import (
	"context"
	"fmt"

	"github.com/ACES-EU/workload-actions-manager/logger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

type ParentInfo struct {
	Type      logger.PodParentTypeEnum
	Name      string
	Namespace string
	UID       string
}

func GetPodOwner(pod *v1.Pod, k8sClient *clientset.Clientset) (ParentInfo, error) {
	obj := metav1.Object(pod)
	owner := metav1.GetControllerOf(obj)

	if owner == nil {
		return ParentInfo{}, fmt.Errorf("no owner of pod: %s found", pod.Name)
	}

	for {
		var parent metav1.Object
		var err error

		namespace := obj.GetNamespace()

		switch owner.Kind {
		case "ReplicaSet":
			parent, err = k8sClient.AppsV1().ReplicaSets(namespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
		case "StatefulSet":
			parent, err = k8sClient.AppsV1().StatefulSets(namespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
		case "DaemonSet":
			parent, err = k8sClient.AppsV1().DaemonSets(namespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
		case "Job":
			parent, err = k8sClient.BatchV1().Jobs(namespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
		case "Deployment":
			parent, err = k8sClient.AppsV1().Deployments(namespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
		case "CronJob":
			parent, err = k8sClient.BatchV1().CronJobs(namespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
		default:
			parentType, err := k8sKindToParentType(owner.Kind)
			if err != nil {
				return ParentInfo{}, err
			}
			return ParentInfo{
				Type:      parentType,
				Name:      owner.Name,
				Namespace: namespace,
				UID:       string(owner.UID),
			}, nil
		}

		if err != nil {
			return ParentInfo{}, fmt.Errorf("failed to get owner %s %s: %w", owner.Kind, owner.Name, err)
		}

		nextOwner := metav1.GetControllerOf(parent)
		if nextOwner == nil {
			parentType, err := k8sKindToParentType(owner.Kind)
			if err != nil {
				return ParentInfo{}, err
			}
			return ParentInfo{
				Type:      parentType,
				Name:      owner.Name,
				Namespace: parent.GetNamespace(),
				UID:       string(parent.GetUID()),
			}, nil
		}

		obj = parent
		owner = nextOwner
	}
}

func k8sKindToParentType(kind string) (logger.PodParentTypeEnum, error) {
	switch kind {
	case "ReplicaSet":
		return logger.PodParentTypeEnumReplicaset, nil
	case "StatefulSet":
		return logger.PodParentTypeEnumStatefulset, nil
	case "DaemonSet":
		return logger.PodParentTypeEnumDaemonset, nil
	case "Job":
		return logger.PodParentTypeEnumJob, nil
	case "CronJob":
		return logger.PodParentTypeEnumCronjob, nil
	case "Deployment":
		return logger.PodParentTypeEnumDeployment, nil
	default:
		return "", fmt.Errorf("unknown kind: %s", kind)
	}
}

func getPodsDeployment(pod *v1.Pod, k8sClient *clientset.Clientset) (*metav1.OwnerReference, error) {
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "ReplicaSet" {
			rs, err := k8sClient.AppsV1().ReplicaSets(pod.Namespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("error getting replicaset: %w", err)
			}
			for _, owner := range rs.OwnerReferences {
				if owner.Kind == "Deployment" {
					return &owner, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("error getting deployment")
}

type Workload struct {
	Namespace  string  `json:"namespace"`
	APIVersion *string `json:"apiVersion,omitempty"`
	Kind       string  `json:"kind"`
	Name       string  `json:"name"`
}

type Pod struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type Node struct {
	Name string `json:"name"`
}
