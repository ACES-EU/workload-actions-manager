package actions

import (
	"context"
	"encoding/json"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"log"
)

func (w Workload) QueueName() string {
	return fmt.Sprintf("%s:%s:%s:%s", w.Namespace, w.APIVersion, w.Kind, w.Name)
}

func validateCreateReq(args *CreateArgs) error {
	if args.Workload.APIVersion != "apps/v1" {
		return fmt.Errorf("only apps/v1 workload API version is supported")
	}

	if args.Workload.Kind != "Deployment" {
		return fmt.Errorf("only Deployment kind is supported")
	}

	if args.Workload.Namespace == "" {
		args.Workload.Namespace = "default"
	}

	if args.Workload.Name == "" {
		return fmt.Errorf("workload name is required")
	}

	if args.Node.Name == "" {
		return fmt.Errorf("node name is required")
	}

	return nil
}

func (as *ActionService) addSchedulingSuggestion(queue string, nodeName string) (*SchedulingSuggestion, error) {
	sug := &SchedulingSuggestion{
		ID:       uuid.NewUUID(),
		NodeName: nodeName,
	}

	log.Printf("created scheduling suggestion %+v\n", sug)

	sugEncoded, err := json.Marshal(sug)
	if err != nil {
		return sug, err
	}

	_, err = as.rdb.LPush(context.TODO(), queue, sugEncoded).Result()
	if err != nil {
		return sug, err
	}

	log.Printf("pushed suggestion to the queue %s\n", queue)

	return sug, nil
}

func (as *ActionService) removeSchedulingSuggestion(queue string, sug *SchedulingSuggestion) error {
	log.Printf("removing suggestion %+v\n", sug)

	sugEncoded, err := json.Marshal(sug)
	if err != nil {
		return err
	}

	_, err = as.rdb.LRem(context.TODO(), queue, 1, sugEncoded).Result()
	if err != nil {
		return err
	}

	log.Println("suggestion removed")

	return nil
}

func (as *ActionService) CreateHandler(args *CreateArgs) (*SchedulingSuggestion, error) {
	queue := args.Workload.QueueName()

	log.Printf("using queue %s\n", queue)

	suggestion, err := as.addSchedulingSuggestion(queue, args.Node.Name)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// send WAM scheduling suggestion to the queue
	// call API and increment replication by 1
	// exponential backoff retry
	// if it fails, remove suggestion from the queue and abort action

	// todo: race conditions here, think about a distributed lock
	scale, err := as.k8sClient.AppsV1().
		Deployments(args.Workload.Namespace).
		GetScale(context.TODO(), args.Workload.Name, metav1.GetOptions{})
	if err != nil {
		log.Println(err)

		err = as.removeSchedulingSuggestion(queue, suggestion)
		if err != nil {
			log.Println(err)
		}

		return nil, err
	}

	log.Printf("got current scale for %s: %d\n", args.Workload.Name, scale.Spec.Replicas)

	s := *scale
	s.Spec.Replicas += 1

	_, err = as.k8sClient.AppsV1().
		Deployments(args.Workload.Namespace).UpdateScale(context.TODO(),
		args.Workload.Name, &s, metav1.UpdateOptions{})
	if err != nil {
		log.Println(err)

		err = as.removeSchedulingSuggestion(queue, suggestion)
		if err != nil {
			log.Println(err)
		}

		return nil, err
	}

	log.Printf("updated new scale of %s to: %d\n", args.Workload.Name, s.Spec.Replicas)

	log.Println("create action successful")

	return suggestion, nil
}

type CreateArgs struct {
	Workload `json:"workload"`
	Node     `json:"node"`
}

type CreateReply struct {
	Message string `json:"message"`
}

type SchedulingSuggestion struct {
	ID       types.UID `json:"id"`
	NodeName string    `json:"node_name"`
}
