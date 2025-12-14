package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	walog "github.com/ACES-EU/workload-actions-manager/logger"
	"github.com/google/uuid"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

func (as *ActionService) UpdateResourcesHandler(id uuid.UUID, args *UpdateResourcesArgs) {
	deploymentName := args.Workload.Name
	namespace := args.Workload.Namespace

	actionStartTime := time.Now()

	if args.Workload.Kind == "Pod" {
		pod, err := as.k8sClient.CoreV1().Pods(namespace).Get(context.TODO(), args.Workload.Name, metav1.GetOptions{})
		if err != nil {
			log.Printf("error getting pod for updating resources %s: %s\n", pod, err)

			actionEndTime := time.Now()
			_, _, logErr := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
				ActionType:      walog.WorkloadActionTypeEnumUpdateResources,
				ActionStatus:    walog.WorkloadActionStatusEnumFailed,
				ActionStartTime: actionStartTime,
				ActionEndTime:   &actionEndTime,
				CreatedAt:       actionEndTime,
				UpdatedAt:       actionEndTime,
			})

			if logErr != nil {
				log.Printf("error updating action log: %v\n", logErr)
			}

			return
		}

		owner, err := getPodsDeployment(pod, as.k8sClient)
		if err != nil {
			log.Printf("error getting pod's deployment name for updating resources %s: %s\n", pod, err)

			actionEndTime := time.Now()
			_, _, logErr := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
				ActionType:      walog.WorkloadActionTypeEnumUpdateResources,
				ActionStatus:    walog.WorkloadActionStatusEnumFailed,
				ActionStartTime: actionStartTime,
				ActionEndTime:   &actionEndTime,
				CreatedAt:       actionEndTime,
				UpdatedAt:       actionEndTime,
			})

			if logErr != nil {
				log.Printf("error updating action log: %v\n", logErr)
			}

			return
		}

		deploymentName = owner.Name
	}

	deploy, err := as.k8sClient.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		log.Printf("error getting deployment for updating resources %s: %s\n", deploymentName, err)

		actionEndTime := time.Now()
		_, _, logErr := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
			ActionType:      walog.WorkloadActionTypeEnumUpdateResources,
			ActionStatus:    walog.WorkloadActionStatusEnumFailed,
			ActionStartTime: actionStartTime,
			ActionEndTime:   &actionEndTime,
			CreatedAt:       actionEndTime,
			UpdatedAt:       actionEndTime,
		})

		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}

		return
	}

	selector := labels.Set(deploy.Spec.Selector.MatchLabels).AsSelector().String()

	deletedPods, err := as.k8sClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		log.Printf("error getting pods for deployment %s: %s\n", deploymentName, err)
	}

	resourcesPatch := buildResourcesPatch(args.Resources)

	if len(resourcesPatch) == 0 {
		log.Printf("no resources were provided to be updated\n")

		actionEndTime := time.Now()
		_, _, logErr := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
			ActionType:      walog.WorkloadActionTypeEnumUpdateResources,
			ActionStatus:    walog.WorkloadActionStatusEnumFailed,
			ActionStartTime: actionStartTime,
			ActionEndTime:   &actionEndTime,
			CreatedAt:       actionEndTime,
			UpdatedAt:       actionEndTime,
		})

		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}
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

	patchBytes, _ := json.Marshal(patch)

	deploy2, err := as.k8sClient.AppsV1().Deployments(namespace).Patch(
		context.TODO(),
		deploymentName,
		types.StrategicMergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		log.Printf("error updating resources: %s\n", err)

		actionEndTime := time.Now()
		_, _, logErr := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
			ActionType:      walog.WorkloadActionTypeEnumUpdateResources,
			ActionStatus:    walog.WorkloadActionStatusEnumFailed,
			ActionStartTime: actionStartTime,
			ActionEndTime:   &actionEndTime,
			CreatedAt:       actionEndTime,
			UpdatedAt:       actionEndTime,
		})

		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}

		return
	}

	if compareResources(deploy, deploy2) {
		log.Printf("resources are the same, no update needed")

		actionEndTime := time.Now()
		_, _, logErr := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
			ActionType:      walog.WorkloadActionTypeEnumUpdateResources,
			ActionStatus:    walog.WorkloadActionStatusEnumSucceeded,
			ActionStartTime: actionStartTime,
			ActionEndTime:   &actionEndTime,
			CreatedAt:       actionEndTime,
			UpdatedAt:       actionEndTime,
		})

		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}
		return
	}

	log.Printf("update resources action successful")

	actionEndTime := time.Now()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		d, err := as.k8sClient.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				log.Printf("Deployment %s no longer exists. Stopping watch loop for logging.\n", deploymentName)
				break
			}

			log.Printf("Error getting deployment: %v\n", err)
			continue
		}

		if isRolloutComplete(d) {
			log.Println("Rollout finished")

			time.Sleep(2 * time.Second)

			createdPods, err := as.k8sClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
				LabelSelector: selector,
			})
			if err != nil {
				log.Printf("error getting pods for deployment %s: %s\n", deploymentName, err)
			}

			for _, pod := range deletedPods.Items {
				log.Printf("Deleted pod %s\n", pod.Name)

				parentType := walog.PodParentTypeEnumDeployment
				parentUid := string(deploy.UID)
				_, _, logErr := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
					ActionType:          walog.WorkloadActionTypeEnumUpdateResources,
					ActionStatus:        walog.WorkloadActionStatusEnumSucceeded,
					ActionStartTime:     actionStartTime,
					ActionEndTime:       &actionEndTime,
					ActionReason:        nil,
					PodParentName:       &deploymentName,
					PodParentType:       &parentType,
					PodParentUID:        &parentUid,
					CreatedPodName:      nil,
					CreatedPodNamespace: nil,
					CreatedNodeName:     nil,
					DeletedPodName:      &pod.Name,
					DeletedPodNamespace: &pod.Namespace,
					DeletedNodeName:     &pod.Spec.NodeName,
					BoundPodName:        nil,
					BoundPodNamespace:   nil,
					BoundNodeName:       nil,
					CreatedAt:           actionEndTime,
					UpdatedAt:           actionEndTime,
				})

				if logErr != nil {
					log.Printf("error updating action log: %v\n", logErr)
				}
			}

			for _, pod := range createdPods.Items {
				log.Printf("Created pod %s\n", pod.Name)

				parentType := walog.PodParentTypeEnumDeployment
				parentUid := string(deploy.UID)
				_, _, logErr := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
					ActionType:          walog.WorkloadActionTypeEnumUpdateResources,
					ActionStatus:        walog.WorkloadActionStatusEnumSucceeded,
					ActionStartTime:     actionStartTime,
					ActionEndTime:       &actionEndTime,
					ActionReason:        nil,
					PodParentName:       &deploymentName,
					PodParentType:       &parentType,
					PodParentUID:        &parentUid,
					CreatedPodName:      &pod.Name,
					CreatedPodNamespace: &pod.Namespace,
					CreatedNodeName:     nil,
					DeletedPodName:      nil,
					DeletedPodNamespace: nil,
					DeletedNodeName:     nil,
					BoundPodName:        nil,
					BoundPodNamespace:   nil,
					BoundNodeName:       nil,
					CreatedAt:           actionEndTime,
					UpdatedAt:           actionEndTime,
				})

				if logErr != nil {
					log.Printf("error updating action log: %v\n", logErr)
				}
			}

			break
		}
	}
}

func compareResources(d1, d2 *v1.Deployment) bool {
	res1 := d1.Spec.Template.Spec.Containers[0].Resources
	res2 := d2.Spec.Template.Spec.Containers[0].Resources

	isEqual := func(list1, list2 corev1.ResourceList) bool {
		keys := []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory}
		for _, key := range keys {
			qty1 := list1[key]
			qty2 := list2[key]
			if qty1.Cmp(qty2) != 0 {
				return false
			}
		}
		return true
	}

	requestsMatch := isEqual(res1.Requests, res2.Requests)
	limitsMatch := isEqual(res1.Limits, res2.Limits)

	return requestsMatch && limitsMatch
}

func isRolloutComplete(d *v1.Deployment) bool {
	if d.Generation > d.Status.ObservedGeneration {
		return false
	}
	if d.Status.UpdatedReplicas < *d.Spec.Replicas {
		return false
	}
	if d.Status.Replicas > d.Status.UpdatedReplicas {
		return false
	}

	return true
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
