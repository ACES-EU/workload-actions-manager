package actions

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"log"
	"time"
)

func (ma *MoveArgs) toCreateArgs(k8sClient *clientset.Clientset) (*CreateArgs, error) {
	pod, err := k8sClient.CoreV1().Pods(ma.Pod.Namespace).Get(context.TODO(), ma.Pod.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var deploymentAPIVersion string
	var deploymentKind string
	var deploymentName string

	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "ReplicaSet" {
			rs, err := k8sClient.AppsV1().ReplicaSets(ma.Pod.Namespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			for _, owner := range rs.OwnerReferences {
				if owner.Kind == "Deployment" {
					deploymentAPIVersion = owner.APIVersion
					deploymentKind = owner.Kind
					deploymentName = owner.Name
					break
				}
			}
		}
	}

	return &CreateArgs{
		Workload: Workload{
			Namespace:  ma.Pod.Namespace,
			APIVersion: deploymentAPIVersion,
			Kind:       deploymentKind,
			Name:       deploymentName,
		},
		Node: ma.Node,
	}, nil
}

func (ma *MoveArgs) toDeleteArgs() *DeleteArgs {
	return &DeleteArgs{
		Pod: ma.Pod,
	}
}

func (as *ActionService) waitToBeReady(namespace string, schedulingSuggestion *SchedulingSuggestion, timeout time.Duration) error {
	watch, err := as.k8sClient.CoreV1().Pods(namespace).Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}
	defer watch.Stop()

	// Channel to signal when the pod is ready
	podReady := make(chan struct{})

	// Start watching for events
	go func() {
		for event := range watch.ResultChan() {
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				continue
			}

			if hasSchedulingSuggestionID(pod, string(schedulingSuggestion.ID)) {
				switch event.Type {
				case "ADDED", "MODIFIED":
					if isPodReady(pod) {
						close(podReady)
					}
				}
			}
		}
	}()

	select {
	case <-podReady:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("waiting for pod with %s exceeded timeout", string(schedulingSuggestion.ID))
	}
}

func hasSchedulingSuggestionID(pod *corev1.Pod, ID string) bool {
	val, ok := pod.Annotations["example.com/scheduling-suggestion-id"]
	return ok && val == ID
}

func isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (as *ActionService) MoveHandler(args *MoveArgs) {
	createArgs, err := args.toCreateArgs(as.k8sClient)
	if err != nil {
		fmt.Printf("%s: move action failed at determining the workload of %s", err.Error(), args.Pod.Name)
	}

	schedulingSuggestion, err := as.CreateHandler(createArgs)
	if err != nil {
		fmt.Printf("move action failed at create step: %s\n", err.Error())
	}

	log.Printf("waiting for pod of workload %s on node %s to become ready\n", createArgs.Workload.Name, createArgs.Node.Name)

	// todo: this can takes a while, so consider a better architecture than keeping a goroutine alive for so long
	err = as.waitToBeReady(args.Pod.Namespace, schedulingSuggestion, 5*time.Minute)
	if err != nil {
		fmt.Printf("move action failed at wait step: %s\n", err.Error())
	}

	log.Printf("done waiting, proceeding with delete\n")

	as.DeleteHandler(args.toDeleteArgs())
}

type MoveArgs struct {
	Pod  `json:"pod"`
	Node `json:"node"`
}

type MoveReply struct {
	Message string `json:"message"`
}

func validateMoveReq(args *MoveArgs) error {
	if args.Pod.Namespace == "" {
		return fmt.Errorf("pod's namespace must be specified")
	}

	if args.Pod.Name == "" {
		return fmt.Errorf("pod's name must be specified")
	}

	if args.Node.Name == "" {
		return fmt.Errorf("node name is required")
	}

	return nil
}