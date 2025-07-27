package actions

import (
	"context"
	"fmt"
	"github.com/ACES-EU/workload-actions-manager/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"time"
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

func (as *ActionService) DeleteHandler(id uuid.UUID, actionType db.ActionTypeEnum, args *DeleteArgs) {
	// todo: swap logging

	var wa db.WorkloadAction
	var err error
	if actionType == db.ActionTypeEnumDelete {
		wa, err = as.db.CreateAction(context.TODO(), db.CreateActionParams{
			ID:                  id,
			ActionType:          actionType,
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
	} else if actionType == db.ActionTypeEnumMove || actionType == db.ActionTypeEnumSwapX || actionType == db.ActionTypeEnumSwapY {
		wa, err = as.db.GetAction(context.TODO(), id)
	}

	pod, err := as.k8sClient.CoreV1().Pods(args.Pod.Namespace).Get(context.TODO(), args.Pod.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("error getting pod: %v\n", err)

		endTime := time.Now()
		wa.ActionEndTime = &endTime
		wa.ActionStatus = db.ActionStatusEnumFailed
		wa, err = updateActionLog(as.db, wa)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}

		return
	}

	wa.DeletedPodNamespace = pgtype.Text{String: pod.Namespace, Valid: true}
	wa.DeletedPodName = pgtype.Text{String: pod.Name, Valid: true}
	wa.DeletedNodeName = pgtype.Text{String: pod.Spec.NodeName, Valid: true}
	wa, err = updateActionLog(as.db, wa)
	if err != nil {
		log.Printf("error updating action in db: %v\n", err)
	}

	deployment, err := getPodsDeployment(pod, as.k8sClient)
	if err != nil {
		log.Printf("error getting %s's deployment\n", pod.Name)

		endTime := time.Now()
		wa.ActionEndTime = &endTime
		wa.ActionStatus = db.ActionStatusEnumFailed
		wa, err = updateActionLog(as.db, wa)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}

		return
	}

	depUid, err := uuid.Parse(string(deployment.UID))
	if err == nil {
		wa.PodParentUid = &depUid
	}
	wa.PodParentType = pgtype.Text{String: deployment.Kind, Valid: true}
	wa.PodParentName = pgtype.Text{String: deployment.Name, Valid: true}
	wa, err = updateActionLog(as.db, wa)
	if err != nil {
		log.Printf("error updating action in db: %v\n", err)
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

		endTime := time.Now()
		wa.ActionEndTime = &endTime
		wa.ActionStatus = db.ActionStatusEnumFailed
		wa, err = updateActionLog(as.db, wa)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}

		return
	}

	// todo: race conditions here, think about a distributed lock
	scale, err := as.k8sClient.AppsV1().
		Deployments(args.Pod.Namespace).
		GetScale(context.TODO(), deployment.Name, metav1.GetOptions{})
	if err != nil {
		log.Println(err)

		endTime := time.Now()
		wa.ActionEndTime = &endTime
		wa.ActionStatus = db.ActionStatusEnumFailed
		wa, err = updateActionLog(as.db, wa)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
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

		endTime := time.Now()
		wa.ActionEndTime = &endTime
		wa.ActionStatus = db.ActionStatusEnumFailed
		wa, err = updateActionLog(as.db, wa)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}
		return
	}

	log.Printf("updated new scale of %s to: %d\n", deployment.Name, s.Spec.Replicas)

	log.Printf("pod %s will be preferentially deleted\n", args.Pod.Name)

	log.Println("delete action successful")

	if wa.ActionType == db.ActionTypeEnumDelete || wa.ActionType == db.ActionTypeEnumMove {
		endTime := time.Now()
		wa.ActionEndTime = &endTime
		wa.ActionStatus = db.ActionStatusEnumSucceeded
		wa, err = updateActionLog(as.db, wa)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}
	}
}

type DeleteArgs struct {
	Pod `json:"pod"`
}

type DeleteReply struct {
	Message string `json:"message"`
}
