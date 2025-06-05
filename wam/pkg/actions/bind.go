package actions

import (
	"context"
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
)

func validateBindReq(args *BindArgs) error {
	if args.Pod.Namespace == "" {
		args.Pod.Namespace = "default"
	}

	if args.Pod.Name == "" {
		return fmt.Errorf("pod's name must be specified")
	}

	if args.Node.Name == "" {
		return fmt.Errorf("node name is required")
	}

	return nil
}

func (as *ActionService) BindHandler(args *BindArgs) error {
	pod, err := as.k8sClient.CoreV1().Pods(args.Pod.Namespace).Get(context.TODO(), args.Pod.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting pod for binding %s: %w", pod, err)
	}

	if pod.Status.Phase != corev1.PodPending {
		log.Printf("Warning: Pod is not in Pending state (%s).\n", pod.Status.Phase)
		return errors.New("pod cannot be bound since it is not in pending state")
	}
	if pod.Spec.NodeName != "" {
		log.Printf("Warning: Pod already has NodeName set to %s.\n", pod.Spec.NodeName)
		return errors.New("pod cannot be bound since it has nodeName set")
	}

	binding := &corev1.Binding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			UID:       pod.UID,
		},
		Target: corev1.ObjectReference{
			Kind: "Node",
			Name: args.Node.Name,
		},
	}

	log.Printf("Attempting to bind Pod %s (UID: %s) to Node %s...\n", pod.Name, pod.UID, pod.Name)

	err = as.k8sClient.CoreV1().Pods(pod.Namespace).Bind(context.TODO(), binding, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to bind Pod %s to Node %s: %w", pod.Name, args.Node.Name, err)
	}

	log.Printf("Pod %s successfully bound to Node %s.\n", pod.Name, args.Node.Name)
	return nil
}

type BindArgs struct {
	Pod  `json:"pod"`
	Node `json:"node"`
}

type BindReply struct {
	Message string `json:"message"`
}
