package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	walog "github.com/ACES-EU/workload-actions-manager/logger"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

func (as *ActionService) ScaleHandler(args *ScalesArgs) {
	deploymentName := args.Workload.Name
	var deployment *v1.Deployment
	namespace := args.Workload.Namespace

	actionStartTime := time.Now()

	if args.Workload.Kind == "Pod" {
		pod, err := as.k8sClient.CoreV1().Pods(namespace).Get(context.TODO(), args.Workload.Name, metav1.GetOptions{})
		if err != nil {
			log.Printf("error getting pod for updating scale %s: %s\n", pod, err)

			actionEndTime := time.Now()
			_, _, logErr := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
				ActionType:      walog.WorkloadActionTypeEnumScale,
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
			log.Printf("error getting pod's deployment name for updating scale %s: %s\n", pod, err)

			actionEndTime := time.Now()
			_, _, logErr := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
				ActionType:      walog.WorkloadActionTypeEnumScale,
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

	deployment, err := as.k8sClient.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		log.Printf("error getting deployment for updating scale %s: %s\n", deploymentName, err)

		actionEndTime := time.Now()
		_, _, logErr := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
			ActionType:      walog.WorkloadActionTypeEnumScale,
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

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": *args.Replicas,
		},
	}

	patchBytes, _ := json.Marshal(patch)

	selector := labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels).String()

	oldPodsMap := make(map[string]bool)

	oldPods, err := as.k8sClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		log.Printf("failed fetching current pods: %s\n", err)
		return
	}

	for _, pod := range oldPods.Items {
		oldPodsMap[pod.Name] = true
	}

	deployment, err = as.k8sClient.AppsV1().Deployments(namespace).Patch(
		context.TODO(),
		deploymentName,
		types.StrategicMergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		log.Printf("error updating scale: %s\n", err)

		actionEndTime := time.Now()
		_, _, logErr := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
			ActionType:      walog.WorkloadActionTypeEnumScale,
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

	log.Printf("update scale action successful")

	actionEndTime := time.Now()

	time.Sleep(5 * time.Second) // make sure pods are created/deleted

	newPods, _ := as.k8sClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector,
	})

	for _, pod := range newPods.Items {
		if !oldPodsMap[pod.Name] {
			fmt.Printf("pod created due to scale action: %s\n", pod.Name)
			parentType := walog.PodParentTypeEnumDeployment
			parentUid := string(deployment.UID)
			_, _, logErr := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
				ActionType:          walog.WorkloadActionTypeEnumScale,
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
	}

	newPodsMap := make(map[string]bool)
	for _, pod := range newPods.Items {
		newPodsMap[pod.Name] = true
	}

	for podName := range oldPodsMap {
		if !newPodsMap[podName] {
			fmt.Printf("pod deleted due to scale action: %s\n", podName)
			parentType := walog.PodParentTypeEnumDeployment
			parentUid := string(deployment.UID)
			_, _, logErr := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
				ActionType:          walog.WorkloadActionTypeEnumScale,
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
				DeletedPodName:      &podName,
				DeletedPodNamespace: &deployment.Namespace,
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
	}
}

type ScalesArgs struct {
	Workload `json:"workload"`
	Replicas *int32 `json:"replicas"`
}

type ScaleReply struct {
	Message string `json:"message"`
}
