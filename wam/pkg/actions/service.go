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

	log.Println("Creation action called")

	reply.Message = "ok"

	// todo: Think about a worker pool here
	go as.CreateHandler(args)
	log.Println("Spawning a handler")

	log.Println("Returning to the caller that the request has been accepted")
	return nil
}

func (as *ActionService) Delete(r *http.Request, args *DeleteArgs, reply *DeleteReply) error {
	err := validateDeleteReq(args)
	if err != nil {
		return err
	}

	log.Println("Delete action called")

	reply.Message = "ok"

	// todo: Think about a worker pool here
	go as.DeleteHandler(args)
	log.Println("Spawning a handler")

	log.Println("Returning to the caller that the request has been accepted")
	return nil
}

func (as *ActionService) Move(r *http.Request, args *MoveArgs, reply *MoveReply) error {
	err := validateMoveReq(args)
	if err != nil {
		return err
	}

	log.Println("Move action called")

	reply.Message = "ok"

	// todo: Think about a worker pool here
	go as.MoveHandler(args)
	log.Println("Spawning a handler")

	log.Println("Returning to the caller that the request has been accepted")
	return nil
}

type Workload struct {
	Namespace  string `json:"namespace"`
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
}

type Pod struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type Node struct {
	Name string `json:"name"`
}
