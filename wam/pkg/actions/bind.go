package actions

import (
	"context"
	"errors"
	"fmt"
	"github.com/ACES-EU/workload-actions-manager/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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
	wa, err := as.db.CreateAction(context.TODO(), db.CreateActionParams{
		ID:                  id,
		ActionType:          db.ActionTypeEnumBind,
		ActionStatus:        db.ActionStatusEnumPending,
		ActionEndTime:       nil,
		ActionReason:        pgtype.Text{},
		PodParentName:       pgtype.Text{},
		PodParentType:       pgtype.Text{},
		PodParentUid:        nil,
		CreatedPodName:      pgtype.Text{},
		CreatedPodNamespace: pgtype.Text{},
		CreatedNodeName:     pgtype.Text{},
		DeletedPodName:      pgtype.Text{},
		DeletedPodNamespace: pgtype.Text{},
		DeletedNodeName:     pgtype.Text{},
		BoundPodName:        pgtype.Text{},
		BoundPodNamespace:   pgtype.Text{},
		BoundNodeName:       pgtype.Text{},
	})

	if err != nil {
		log.Printf("error initializing bind action in db: %v\n", err)
	}

	pod, err := as.k8sClient.CoreV1().Pods(args.Pod.Namespace).Get(context.TODO(), args.Pod.Name, metav1.GetOptions{})
	if err != nil {
		wa.ActionStatus = db.ActionStatusEnumFailed
		endTime := time.Now()
		wa.ActionEndTime = &endTime
		wa, err = updateActionLog(as.db, wa)
		if err != nil {
			log.Printf("error updating bind action in db: %v\n", err)
		}

		return fmt.Errorf("error getting pod for binding %s: %w", pod, err)
	}

	wa.BoundPodName = pgtype.Text{String: args.Pod.Name, Valid: true}
	wa.BoundPodNamespace = pgtype.Text{String: args.Pod.Namespace, Valid: true}

	wa, err = updateActionLog(as.db, wa)
	if err != nil {
		log.Printf("error updating bind action in db: %v\n", err)
	}

	dep, err := getPodsDeployment(pod, as.k8sClient)
	if err == nil {
		depUid, err := uuid.Parse(string(dep.UID))
		if err == nil {
			wa.PodParentUid = &depUid
		}
		wa.PodParentType = pgtype.Text{String: dep.Kind, Valid: true}
		wa.PodParentName = pgtype.Text{String: dep.Name, Valid: true}
		wa, err = updateActionLog(as.db, wa)
		if err != nil {
			log.Printf("error updating bind action in db: %v\n", err)
		}
	} else {
		fmt.Printf("could not find deployment for pod, not logging this info to db: %v\n", err)
	}

	if pod.Status.Phase != corev1.PodPending {
		log.Printf("Warning: Pod is not in Pending state (%s).\n", pod.Status.Phase)
		wa.ActionStatus = db.ActionStatusEnumFailed
		endTime := time.Now()
		wa.ActionEndTime = &endTime
		wa, err = updateActionLog(as.db, wa)
		if err != nil {
			log.Printf("error updating bind action in db: %v\n", err)
		}
		return errors.New("pod cannot be bound since it is not in pending state")
	}
	if pod.Spec.NodeName != "" {
		log.Printf("Warning: Pod already has NodeName set to %s.\n", pod.Spec.NodeName)
		wa.ActionStatus = db.ActionStatusEnumFailed
		endTime := time.Now()
		wa.ActionEndTime = &endTime
		wa, err = updateActionLog(as.db, wa)
		if err != nil {
			log.Printf("error updating bind action in db: %v\n", err)
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
		wa.ActionStatus = db.ActionStatusEnumFailed
		endTime := time.Now()
		wa.ActionEndTime = &endTime
		wa, err = updateActionLog(as.db, wa)
		if err != nil {
			log.Printf("error updating bind action in db: %v\n", err)
		}

		return fmt.Errorf("failed to bind Pod %s to Node %s: %w", pod.Name, args.Node.Name, err)
	}

	wa.BoundNodeName = pgtype.Text{String: args.Node.Name, Valid: true}
	wa.ActionStatus = db.ActionStatusEnumSucceeded
	endTime := time.Now()
	wa.ActionEndTime = &endTime
	wa, err = updateActionLog(as.db, wa)
	if err != nil {
		log.Printf("error updating bind action in db: %v\n", err)
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
