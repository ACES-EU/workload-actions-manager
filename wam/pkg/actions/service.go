package actions

import (
	"context"
	"log"
	"net/http"
	"time"

	walog "github.com/ACES-EU/workload-actions-manager/logger"
	"github.com/redis/go-redis/v9"
	clientset "k8s.io/client-go/kubernetes"
)

type ActionService struct {
	k8sClient *clientset.Clientset
	rdb       *redis.Client
	log       *walog.WALogger
}

func NewActionService(k8sClient *clientset.Clientset, rdb *redis.Client, logger *walog.WALogger) *ActionService {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	return &ActionService{
		k8sClient,
		rdb,
		logger,
	}
}

func (as *ActionService) Bind(r *http.Request, args *BindArgs, reply *BindReply) error {
	err := validateBindReq(args)
	if err != nil {
		return err
	}

	log.Println("bind action called")

	now := time.Now()
	wa, _, err := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
		ActionType:      walog.WorkloadActionTypeEnumBind,
		ActionStatus:    walog.WorkloadActionStatusEnumPending,
		ActionStartTime: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		log.Printf("error creating action log: %v\n", err)
		return err
	}

	reply.Message = "ok"
	err = as.BindHandler(wa.ID, args)
	if err != nil {
		return err
	}

	return nil
}

func (as *ActionService) Create(r *http.Request, args *CreateArgs, reply *CreateReply) error {
	err := validateCreateReq(args)
	if err != nil {
		return err
	}

	log.Println("create action called")

	now := time.Now()
	wa, _, err := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
		ActionType:      walog.WorkloadActionTypeEnumCreate,
		ActionStatus:    walog.WorkloadActionStatusEnumPending,
		ActionStartTime: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		log.Printf("error creating action log: %v\n", err)
		return err
	}

	reply.Message = "ok"

	// todo: Think about a worker pool here
	go func() {
		_, _ = as.CreateHandler(wa.ID, walog.WorkloadActionTypeEnumCreate, args)
	}()
	log.Println("spawning a handler")

	log.Println("returning to the caller that the request has been accepted")
	return nil
}

func (as *ActionService) Delete(r *http.Request, args *DeleteArgs, reply *DeleteReply) error {
	err := validateDeleteReq(args)
	if err != nil {
		return err
	}

	log.Println("delete action called")

	now := time.Now()
	wa, _, err := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
		ActionType:      walog.WorkloadActionTypeEnumDelete,
		ActionStatus:    walog.WorkloadActionStatusEnumPending,
		ActionStartTime: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		log.Printf("error creating action log: %v\n", err)
		return err
	}

	reply.Message = "ok"

	// todo: Think about a worker pool here
	go as.DeleteHandler(wa.ID, walog.WorkloadActionTypeEnumDelete, args)
	log.Println("spawning a handler")

	log.Println("returning to the caller that the request has been accepted")
	return nil
}

func (as *ActionService) Move(r *http.Request, args *MoveArgs, reply *MoveReply) error {
	err := validateMoveReq(args)
	if err != nil {
		return err
	}

	log.Println("move action called")

	now := time.Now()
	wa, _, err := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
		ActionType:      walog.WorkloadActionTypeEnumMove,
		ActionStatus:    walog.WorkloadActionStatusEnumPending,
		ActionStartTime: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		log.Printf("error creating action log: %v\n", err)
		return err
	}

	reply.Message = "ok"

	// todo: Think about a worker pool here
	go as.MoveHandler(wa.ID, walog.WorkloadActionTypeEnumMove, args)
	log.Println("spawning a handler")

	log.Println("returning to the caller that the request has been accepted")
	return nil
}

func (as *ActionService) Swap(r *http.Request, args *SwapArgs, reply *SwapReply) error {
	err := validateSwapReq(args)
	if err != nil {
		return err
	}

	log.Println("swap action called")

	now := time.Now()
	waX, _, err := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
		ActionType:      walog.WorkloadActionTypeEnumSwapX,
		ActionStatus:    walog.WorkloadActionStatusEnumPending,
		ActionStartTime: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		log.Printf("error creating action log: %v\n", err)
		return err
	}

	waY, _, err := as.log.CreateWorkloadAction(context.TODO(), walog.WorkloadActionCreate{
		ActionType:      walog.WorkloadActionTypeEnumSwapY,
		ActionStatus:    walog.WorkloadActionStatusEnumPending,
		ActionStartTime: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		log.Printf("error creating action log: %v\n", err)
		return err
	}

	reply.Message = "ok"

	// todo: Think about a worker pool here
	// ensure that no other actions related to the workloads accessed by the swap action run in parallel
	// since they might affect the wait part of the action or even prevent the action to succeed
	go as.SwapHandler(waX.ID, waY.ID, args)
	log.Println("spawning a handler")

	log.Println("returning to the caller that the request has been accepted")
	return nil
}
