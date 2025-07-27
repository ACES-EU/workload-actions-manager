package actions

import (
	"context"
	"fmt"
	"github.com/ACES-EU/workload-actions-manager/db"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

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

func updateActionLog(q *db.Queries, wa db.WorkloadAction) (db.WorkloadAction, error) {
	return q.UpdateAction(context.TODO(), db.UpdateActionParams{
		ID:                  wa.ID,
		ActionType:          wa.ActionType,
		ActionStatus:        wa.ActionStatus,
		ActionEndTime:       wa.ActionEndTime,
		ActionReason:        wa.ActionReason,
		PodParentName:       wa.PodParentName,
		PodParentType:       wa.PodParentType,
		PodParentUid:        wa.PodParentUid,
		CreatedPodName:      wa.CreatedPodName,
		CreatedPodNamespace: wa.CreatedPodNamespace,
		CreatedNodeName:     wa.CreatedNodeName,
		DeletedPodName:      wa.DeletedPodName,
		DeletedPodNamespace: wa.DeletedPodNamespace,
		DeletedNodeName:     wa.DeletedNodeName,
		BoundPodName:        wa.BoundPodName,
		BoundPodNamespace:   wa.BoundPodNamespace,
		BoundNodeName:       wa.BoundNodeName,
	})
}

type Workload struct {
	Namespace  string `json:"namespace"`
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
}

type Pod struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type Node struct {
	Name string `json:"name"`
}
