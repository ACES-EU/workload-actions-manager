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

func validateUpdateResourcesReq(args *UpdateResourcesArgs) error {
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

	if args.Resources.ContainerName == "" {
		return fmt.Errorf("container name of which resources will be updated must be specified")
	}

	return nil
}

func (as *ActionService) UpdateResourcesHandler(id uuid.UUID, args *UpdateResourcesArgs) error {

	deploymentName := args.Workload.Name
	namespace := args.Workload.Namespace

	if args.Workload.Kind == "Pod" {
		pod, err := as.k8sClient.CoreV1().Pods(namespace).Get(context.TODO(), args.Workload.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting pod for updating resources %s: %w", pod, err)
		}

		deploymentRef, err := getPodsDeployment(pod, as.k8sClient)
		if err != nil {
			return fmt.Errorf("error getting pod's deployment name for updating resources %s: %w", pod, err)
		}

		deploymentName = deploymentRef.Name
	}

	resourcesPatch := buildResourcesPatch(args.Resources)

	if len(resourcesPatch) == 0 {
		return fmt.Errorf("no resources were provided to be updated")
	}

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":      args.Resources.ContainerName,
							"resources": resourcesPatch,
						},
					},
				},
			},
		},
	}

	fmt.Println(patch)

	patchBytes, _ := json.Marshal(patch)

	_, err := as.k8sClient.AppsV1().Deployments(namespace).Patch(
		context.TODO(),
		deploymentName,
		types.StrategicMergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("error updating resources: %w", err)
	}

	log.Printf("update resources action successful")

	return nil
}

func buildResourcesPatch(args Resources) map[string]interface{} {
	resources := make(map[string]interface{})
	limits := make(map[string]string)
	requests := make(map[string]string)

	if args.CpuLimit != nil {
		limits["cpu"] = *args.CpuLimit
	}
	if args.MemoryLimit != nil {
		limits["memory"] = *args.MemoryLimit
	}

	if args.CpuRequest != nil {
		requests["cpu"] = *args.CpuRequest
	}
	if args.MemoryRequest != nil {
		requests["memory"] = *args.MemoryRequest
	}

	if len(limits) > 0 {
		resources["limits"] = limits
	}

	if len(requests) > 0 {
		resources["requests"] = requests
	}

	return resources
}

type UpdateResourcesArgs struct {
	Workload  `json:"workload"`
	Resources `json:"resources"`
}

type Resources struct {
	ContainerName string  `json:"container_name"`
	CpuRequest    *string `json:"cpu_request"`
	CpuLimit      *string `json:"cpu_limit"`
	MemoryRequest *string `json:"memory_request"`
	MemoryLimit   *string `json:"memory_limit"`
}

type UpdateResourceReply struct {
	Message string `json:"message"`
}
