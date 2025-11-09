package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func validateScaleReq(args *ScalesArgs) error {
	if args.Workload.Namespace == "" {
		args.Workload.Namespace = "default"
	}

	if args.Workload.Name == "" {
		return fmt.Errorf("workload's name must be specified")
	}

	if args.Workload.Kind == "" {
		return fmt.Errorf("workload's name must be specified")
	}

	if args.Workload.Kind != "Pod" && args.Workload.Kind != "Deployment" {
		return fmt.Errorf("workload's kind must be 'Pod' or 'Deployment'")
	}

	if args.Replicas == nil || (args.Replicas != nil && *args.Replicas < 0) {
		return fmt.Errorf("replicas is required and must be >= 0")
	}

	return nil
}

func (as *ActionService) ScaleHandler(id uuid.UUID, args *ScalesArgs) error {
	deploymentName := args.Workload.Name
	namespace := args.Workload.Namespace

	if args.Workload.Kind == "Pod" {
		pod, err := as.k8sClient.CoreV1().Pods(namespace).Get(context.TODO(), args.Workload.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting pod for updating scale %s: %w", pod, err)
		}

		deploymentRef, err := getPodsDeployment(pod, as.k8sClient)
		if err != nil {
			return fmt.Errorf("error getting pod's deployment name for updating scale %s: %w", pod, err)
		}

		deploymentName = deploymentRef.Name
	}

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": *args.Replicas,
		},
	}

	patchBytes, _ := json.Marshal(patch)

	_, err := as.k8sClient.AppsV1().Deployments(namespace).Patch(
		context.TODO(),
		deploymentName,
		types.StrategicMergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("error updating scale: %w", err)
	}

	log.Printf("update scale action successful")

	return nil
}

type ScalesArgs struct {
	Workload `json:"workload"`
	Replicas *int32 `json:"replicas"`
}

type ScaleReply struct {
	Message string `json:"message"`
}
