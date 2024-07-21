package actions

import (
	"github.com/redis/go-redis/v9"
	clientset "k8s.io/client-go/kubernetes"
	"log"
	"net/http"
)

type ActionService struct {
	k8sClient *clientset.Clientset
	rdb       *redis.Client
}

func NewActionService(k8sClient *clientset.Clientset, rdb *redis.Client) *ActionService {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	return &ActionService{
		k8sClient,
		rdb,
	}
}

func (as *ActionService) Create(r *http.Request, args *CreateArgs, reply *CreateReply) error {
	err := validateCreateReq(args)
	if err != nil {
		return err
	}

	log.Println("create action called")

	reply.Message = "ok"

	// todo: Think about a worker pool here
	go as.CreateHandler(args)
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

	// todo: Think about a worker pool here
	go as.DeleteHandler(args)
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

	reply.Message = "ok"

	// todo: Think about a worker pool here
	go as.MoveHandler(args)
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

	// todo: Think about a worker pool here
	// ensure that no other actions related to the workloads accessed by the swap action run in parallel
	// since they might affect the wait part of the action or even prevent the action to succeed
	go as.SwapHandler(args)
	log.Println("spawning a handler")

	log.Println("returning to the caller that the request has been accepted")
	return nil
}
