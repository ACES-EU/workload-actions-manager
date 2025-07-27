package actions

import (
	"github.com/ACES-EU/workload-actions-manager/db"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	clientset "k8s.io/client-go/kubernetes"
	"log"
	"net/http"
)

type ActionService struct {
	k8sClient *clientset.Clientset
	rdb       *redis.Client
	db        *db.Queries
}

func NewActionService(k8sClient *clientset.Clientset, rdb *redis.Client, db *db.Queries) *ActionService {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	return &ActionService{
		k8sClient,
		rdb,
		db,
	}
}

func (as *ActionService) Bind(r *http.Request, args *BindArgs, reply *BindReply) error {
	err := validateBindReq(args)
	if err != nil {
		return err
	}

	log.Println("bind action called")

	id := uuid.Must(uuid.NewUUID())

	reply.Message = "ok"
	err = as.BindHandler(id, args)
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

	id := uuid.Must(uuid.NewUUID())
	actionType := db.ActionTypeEnumCreate

	reply.Message = "ok"

	// todo: Think about a worker pool here
	go func() {
		_, _ = as.CreateHandler(id, actionType, args)
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

	reply.Message = "ok"

	id := uuid.Must(uuid.NewUUID())
	actionType := db.ActionTypeEnumDelete

	// todo: Think about a worker pool here
	go as.DeleteHandler(id, actionType, args)
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

	id := uuid.Must(uuid.NewUUID())
	actionType := db.ActionTypeEnumMove

	reply.Message = "ok"

	// todo: Think about a worker pool here
	go as.MoveHandler(id, actionType, args)
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

	reply.Message = "ok"

	idX := uuid.Must(uuid.NewUUID())
	idY := uuid.Must(uuid.NewUUID())

	// todo: Think about a worker pool here
	// ensure that no other actions related to the workloads accessed by the swap action run in parallel
	// since they might affect the wait part of the action or even prevent the action to succeed
	go as.SwapHandler(idX, idY, args)
	log.Println("spawning a handler")

	log.Println("returning to the caller that the request has been accepted")
	return nil
}
