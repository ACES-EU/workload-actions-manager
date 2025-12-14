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

func validateRedeployReq(args *RedeployArgs) error {
	if args.Pod.Namespace == "" {
		args.Pod.Namespace = "default"
	}

	if args.Pod.Name == "" {
		return fmt.Errorf("pod's name must be specified")
	}

	return nil
}

func (as *ActionService) RedeployHandler(id uuid.UUID, args *RedeployArgs) error {
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

		return err
	}

	owner, err := GetPodOwner(pod, as.k8sClient)
	if err != nil {
		log.Printf("error getting %s's owner: %s\n", pod.Name, err)
	} else {
		logTime := time.Now()
		_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), id, walog.WorkloadActionUpdate{
			PodParentType: &owner.Type,
			PodParentUID:  &owner.UID,
			PodParentName: &owner.Name,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}
	}

	err = as.k8sClient.CoreV1().Pods(args.Pod.Namespace).Delete(context.TODO(), args.Pod.Name, metav1.DeleteOptions{})
	if err != nil {
		log.Printf("error deleting pod %s\n", pod.Name)

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
			ActionType:     walog.WorkloadActionTypeEnumRedeploy,
			DecisionStatus: status,
		})
		if logErr != nil {
			log.Printf("error updating decision status log: %v\n", logErr)
		}

		return err
	}

	log.Println("redeploy action successful")

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
		ActionType:     walog.WorkloadActionTypeEnumRedeploy,
		DecisionStatus: status,
	})
	if logErr != nil {
		log.Printf("error updating decision status log: %v\n", logErr)
	}

	return nil
}

type RedeployArgs struct {
	Pod `json:"pod"`
}

type RedeployReply struct {
	Message string `json:"message"`
}
