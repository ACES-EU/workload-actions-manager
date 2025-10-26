package actions

import (
	"context"
	"fmt"
	"log"
	"time"

	walog "github.com/ACES-EU/workload-actions-manager/logger"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func validateDeleteReq(args *DeleteArgs) error {
	if args.Pod.Namespace == "" {
		args.Pod.Namespace = "default"
	}

	if args.Pod.Name == "" {
		return fmt.Errorf("pod's name must be specified")
	}

	return nil
}

func (as *ActionService) DeleteHandler(id uuid.UUID, actionType walog.WorkloadActionTypeEnum, args *DeleteArgs) {
	pod, err := as.k8sClient.CoreV1().Pods(args.Pod.Namespace).Get(context.TODO(), args.Pod.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("error getting pod: %v\n", err)

		status := walog.WorkloadActionStatusEnumFailed
		logTime := time.Now()
		_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), id, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}

		return
	}

	logTime := time.Now()
	_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), id, walog.WorkloadActionUpdate{
		DeletedPodNamespace: &pod.Namespace,
		DeletedPodName:      &pod.Name,
		DeletedNodeName:     &pod.Spec.NodeName,
		UpdatedAt:           &logTime,
	})
	if logErr != nil {
		log.Printf("error updating action log: %v\n", logErr)
	}

	deployment, err := getPodsDeployment(pod, as.k8sClient)
	if err != nil {
		log.Printf("error getting %s's deployment\n", pod.Name)

		status := walog.WorkloadActionStatusEnumFailed
		logTime := time.Now()
		_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), id, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}
		_, logErr = as.log.UpdateWorkloadDecisionStatus(context.TODO(), walog.WorkloadDecisionStatusUpdate{
			PodName:        pod.Name,
			Namespace:      pod.Namespace,
			NodeName:       pod.Spec.NodeName,
			ActionType:     actionType,
			DecisionStatus: status,
		})
		if logErr != nil {
			log.Printf("error updating decision status log: %v\n", logErr)
		}

		return
	}

	depUid, _ := uuid.Parse(string(deployment.UID))

	logTime = time.Now()
	parentType := walog.PodParentTypeEnumDeployment
	_, _, logErr = as.log.UpdateWorkloadAction(context.TODO(), id, walog.WorkloadActionUpdate{
		PodParentType: &parentType,
		PodParentUID:  &depUid,
		PodParentName: &deployment.Name,
		UpdatedAt:     &logTime,
	})
	if logErr != nil {
		log.Printf("error updating action log: %v\n", logErr)
	}

	// Prefer removing this pod. It is not guaranteed though.
	// https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/#pod-deletion-cost
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations["controller.kubernetes.io/pod-deletion-cost"] = "-1000"

	_, err = as.k8sClient.CoreV1().Pods(args.Pod.Namespace).Update(context.TODO(), pod, metav1.UpdateOptions{})
	if err != nil {
		log.Println(err)

		status := walog.WorkloadActionStatusEnumFailed
		logTime := time.Now()
		_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), id, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}
		_, logErr = as.log.UpdateWorkloadDecisionStatus(context.TODO(), walog.WorkloadDecisionStatusUpdate{
			PodName:        pod.Name,
			Namespace:      pod.Namespace,
			NodeName:       pod.Spec.NodeName,
			ActionType:     actionType,
			DecisionStatus: status,
		})
		if logErr != nil {
			log.Printf("error updating decision status log: %v\n", logErr)
		}

		return
	}

	// todo: race conditions here, think about a distributed lock
	scale, err := as.k8sClient.AppsV1().
		Deployments(args.Pod.Namespace).
		GetScale(context.TODO(), deployment.Name, metav1.GetOptions{})
	if err != nil {
		log.Println(err)

		status := walog.WorkloadActionStatusEnumFailed
		logTime := time.Now()
		_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), id, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}
		_, logErr = as.log.UpdateWorkloadDecisionStatus(context.TODO(), walog.WorkloadDecisionStatusUpdate{
			PodName:        pod.Name,
			Namespace:      pod.Namespace,
			NodeName:       pod.Spec.NodeName,
			ActionType:     actionType,
			DecisionStatus: status,
		})
		if logErr != nil {
			log.Printf("error updating decision status log: %v\n", logErr)
		}

		return
	}

	log.Printf("got current scale for %s: %d\n", deployment.Name, scale.Spec.Replicas)

	s := *scale
	s.Spec.Replicas -= 1

	_, err = as.k8sClient.AppsV1().
		Deployments(args.Pod.Namespace).UpdateScale(context.TODO(),
		deployment.Name, &s, metav1.UpdateOptions{})
	if err != nil {
		log.Println(err)

		status := walog.WorkloadActionStatusEnumFailed
		logTime := time.Now()
		_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), id, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}
		_, logErr = as.log.UpdateWorkloadDecisionStatus(context.TODO(), walog.WorkloadDecisionStatusUpdate{
			PodName:        pod.Name,
			Namespace:      pod.Namespace,
			NodeName:       pod.Spec.NodeName,
			ActionType:     actionType,
			DecisionStatus: status,
		})
		if logErr != nil {
			log.Printf("error updating decision status log: %v\n", logErr)
		}

		return
	}

	log.Printf("updated new scale of %s to: %d\n", deployment.Name, s.Spec.Replicas)

	log.Printf("pod %s will be preferentially deleted\n", args.Pod.Name)

	log.Println("delete action successful")

	if actionType == walog.WorkloadActionTypeEnumDelete || actionType == walog.WorkloadActionTypeEnumMove {
		status := walog.WorkloadActionStatusEnumSucceeded
		logTime := time.Now()
		_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), id, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}

		_, logErr = as.log.UpdateWorkloadDecisionStatus(context.TODO(), walog.WorkloadDecisionStatusUpdate{
			PodName:        pod.Name,
			Namespace:      pod.Namespace,
			NodeName:       pod.Spec.NodeName,
			ActionType:     actionType,
			DecisionStatus: status,
		})
		if logErr != nil {
			log.Printf("error updating decision status log: %v\n", logErr)
		}
	}
}

type DeleteArgs struct {
	Pod `json:"pod"`
}

type DeleteReply struct {
	Message string `json:"message"`
}
