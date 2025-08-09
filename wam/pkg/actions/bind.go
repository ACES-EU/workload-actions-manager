package actions

import (
	"context"
	"errors"
	"fmt"
	walog "github.com/ACES-EU/workload-actions-manager/logger"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"time"
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

func (as *ActionService) BindHandler(id uuid.UUID, args *BindArgs) error {
	pod, err := as.k8sClient.CoreV1().Pods(args.Pod.Namespace).Get(context.TODO(), args.Pod.Name, metav1.GetOptions{})
	if err != nil {
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

		return fmt.Errorf("error getting pod for binding %s: %w", pod, err)
	}

	logTime := time.Now()
	_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), id, walog.WorkloadActionUpdate{
		BoundPodName:      &args.Pod.Name,
		BoundPodNamespace: &args.Pod.Namespace,
		UpdatedAt:         &logTime,
	})
	if logErr != nil {
		log.Printf("error updating action log: %v\n", logErr)
	}

	dep, err := getPodsDeployment(pod, as.k8sClient)
	if err == nil {
		depUid, _ := uuid.Parse(string(dep.UID))
		depType := walog.PodParentTypeEnumDeployment

		logTime := time.Now()
		_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), id, walog.WorkloadActionUpdate{
			PodParentUID:  &depUid,
			PodParentType: &depType,
			PodParentName: &dep.Name,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}

	} else {
		fmt.Printf("could not find deployment for pod, not logging this info to db: %v\n", err)
	}

	if pod.Status.Phase != corev1.PodPending {
		log.Printf("Warning: Pod is not in Pending state (%s).\n", pod.Status.Phase)
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

		return errors.New("pod cannot be bound since it is not in pending state")
	}
	if pod.Spec.NodeName != "" {
		log.Printf("Warning: Pod already has NodeName set to %s.\n", pod.Spec.NodeName)
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

	log.Printf("Attempting to bind Pod %s (UID: %s) to Node %s...\n", pod.Name, pod.UID, args.Node.Name)

	err = as.k8sClient.CoreV1().Pods(pod.Namespace).Bind(context.TODO(), binding, metav1.CreateOptions{})
	if err != nil {
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

		return fmt.Errorf("failed to bind Pod %s to Node %s: %w", pod.Name, args.Node.Name, err)
	}

	status := walog.WorkloadActionStatusEnumSucceeded
	logTime = time.Now()
	_, _, logErr = as.log.UpdateWorkloadAction(context.TODO(), id, walog.WorkloadActionUpdate{
		ActionStatus:  &status,
		BoundNodeName: &args.Node.Name,
		ActionEndTime: &logTime,
		UpdatedAt:     &logTime,
	})
	if logErr != nil {
		log.Printf("error updating action log: %v\n", logErr)
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
