package actions

import (
	"context"
	"fmt"
	"github.com/ACES-EU/workload-actions-manager/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"log"
	"time"
)

func (p *Pod) toDeleteArgs() *DeleteArgs {
	return &DeleteArgs{
		Pod: Pod{
			Namespace: p.Namespace,
			Name:      p.Name,
		},
	}
}

func (p *Pod) toCreateArgs(k8sClient *clientset.Clientset, nodeName string) (*CreateArgs, error) {
	pod, err := k8sClient.CoreV1().Pods(p.Namespace).Get(context.TODO(), p.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	deployment, err := getPodsDeployment(pod, k8sClient)
	if err != nil {
		return nil, err
	}

	return &CreateArgs{
		Workload: Workload{
			Namespace:  p.Namespace,
			APIVersion: deployment.APIVersion,
			Kind:       deployment.Kind,
			Name:       deployment.Name,
		},
		Node: Node{
			Name: nodeName,
		},
	}, nil
}

type TargetScaleKey struct {
	Namespace      string
	DeploymentName string
}

func (as *ActionService) SwapHandler(idX uuid.UUID, idY uuid.UUID, args *SwapArgs) {
	startTime := time.Now()

	waX, err := as.db.CreateActionStartTime(context.TODO(), db.CreateActionStartTimeParams{
		ID:                  idX,
		ActionType:          db.ActionTypeEnumSwapX,
		ActionStatus:        db.ActionStatusEnumPending,
		ActionStartTime:     &startTime,
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
		log.Printf("error initializing action in db: %v\n", err)
	}

	waY, err := as.db.CreateActionStartTime(context.TODO(), db.CreateActionStartTimeParams{
		ID:                  idY,
		ActionType:          db.ActionTypeEnumSwapY,
		ActionStatus:        db.ActionStatusEnumPending,
		ActionStartTime:     &startTime,
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
		log.Printf("error initializing action in db: %v\n", err)
	}

	podX, err := as.k8sClient.CoreV1().Pods(args.X.Namespace).Get(context.TODO(), args.X.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("error getting pod X: %v\n", err)

		waX.ActionStatus = db.ActionStatusEnumFailed
		endTime := time.Now()
		waX.ActionEndTime = &endTime
		waX, err = updateActionLog(as.db, waX)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}

		waY.ActionStatus = db.ActionStatusEnumFailed
		waY.ActionEndTime = &endTime
		waY, err = updateActionLog(as.db, waY)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}

		return
	}

	podY, err := as.k8sClient.CoreV1().Pods(args.Y.Namespace).Get(context.TODO(), args.Y.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("error getting pod X: %v\n", err)

		waX.ActionStatus = db.ActionStatusEnumFailed
		endTime := time.Now()
		waX.ActionEndTime = &endTime
		waX, err = updateActionLog(as.db, waX)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}

		waY.ActionStatus = db.ActionStatusEnumFailed
		waY.ActionEndTime = &endTime
		waY, err = updateActionLog(as.db, waY)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}

		return
	}

	deleteArgsX := args.X.toDeleteArgs()
	deleteArgsY := args.Y.toDeleteArgs()

	createArgsX, err := args.X.toCreateArgs(as.k8sClient, podY.Spec.NodeName)
	if err != nil {
		log.Printf("error creating create args: %s", err.Error())

		waX.ActionStatus = db.ActionStatusEnumFailed
		endTime := time.Now()
		waX.ActionEndTime = &endTime
		waX, err = updateActionLog(as.db, waX)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}

		waY.ActionStatus = db.ActionStatusEnumFailed
		waY.ActionEndTime = &endTime
		waY, err = updateActionLog(as.db, waY)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}

		return
	}
	createArgsY, err := args.Y.toCreateArgs(as.k8sClient, podX.Spec.NodeName)
	if err != nil {
		log.Printf("error creating create args: %s", err.Error())

		waX.ActionStatus = db.ActionStatusEnumFailed
		endTime := time.Now()
		waX.ActionEndTime = &endTime
		waX, err = updateActionLog(as.db, waX)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}

		waY.ActionStatus = db.ActionStatusEnumFailed
		waY.ActionEndTime = &endTime
		waY, err = updateActionLog(as.db, waY)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}

		return
	}

	as.DeleteHandler(idX, db.ActionTypeEnumSwapX, deleteArgsX)
	as.DeleteHandler(idY, db.ActionTypeEnumSwapY, deleteArgsY)

	log.Printf("all deletes have completed")

	// Delete might take a while, so the created pods might be in pending state for some time
	// until they're finally scheduled where they're supposed to go. That's ok.

	log.Printf("continuing with creates")
	_, err = as.CreateHandler(idX, db.ActionTypeEnumSwapX, createArgsX)
	if err != nil {
		waX.ActionStatus = db.ActionStatusEnumFailed
		endTime := time.Now()
		waX.ActionEndTime = &endTime
		waX, err = updateActionLog(as.db, waX)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}

		waY.ActionStatus = db.ActionStatusEnumFailed
		waY.ActionEndTime = &endTime
		waY, err = updateActionLog(as.db, waY)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}
	}
	_, err = as.CreateHandler(idY, db.ActionTypeEnumSwapY, createArgsY)
	if err != nil {
		waX.ActionStatus = db.ActionStatusEnumFailed
		endTime := time.Now()
		waX.ActionEndTime = &endTime
		waX, err = updateActionLog(as.db, waX)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}

		waY.ActionStatus = db.ActionStatusEnumFailed
		waY.ActionEndTime = &endTime
		waY, err = updateActionLog(as.db, waY)
		if err != nil {
			log.Printf("error updating action in db: %v\n", err)
		}
	}

	log.Printf("all creates have completed")

	log.Println("swap action successful")
}

type SwapArgs struct {
	X Pod `json:"x"`
	Y Pod `json:"y"`
}

type SwapReply struct {
	Message string `json:"message"`
}

func validateSwapReq(args *SwapArgs) error {
	if args.X.Namespace == "" {
		args.X.Namespace = "default"
	}

	if args.X.Name == "" {
		return fmt.Errorf("x pod's name must be specified")
	}

	if args.Y.Name == "" {
		return fmt.Errorf("y pod's name must be specified")
	}

	return nil
}
