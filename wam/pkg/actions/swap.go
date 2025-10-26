package actions

import (
	"context"
	"fmt"
	"log"
	"time"

	walog "github.com/ACES-EU/workload-actions-manager/logger"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
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
	podX, err := as.k8sClient.CoreV1().Pods(args.X.Namespace).Get(context.TODO(), args.X.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("error getting pod X: %v\n", err)

		status := walog.WorkloadActionStatusEnumFailed
		logTime := time.Now()

		_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), idX, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}

		_, _, logErr = as.log.UpdateWorkloadAction(context.TODO(), idY, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}

		return
	}

	podY, err := as.k8sClient.CoreV1().Pods(args.Y.Namespace).Get(context.TODO(), args.Y.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("error getting pod X: %v\n", err)

		status := walog.WorkloadActionStatusEnumFailed
		logTime := time.Now()

		_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), idX, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}

		_, _, logErr = as.log.UpdateWorkloadAction(context.TODO(), idY, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}

		return
	}

	deleteArgsX := args.X.toDeleteArgs()
	deleteArgsY := args.Y.toDeleteArgs()

	createArgsX, err := args.X.toCreateArgs(as.k8sClient, podY.Spec.NodeName)
	if err != nil {
		log.Printf("error creating create args: %s", err.Error())

		status := walog.WorkloadActionStatusEnumFailed
		logTime := time.Now()

		_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), idX, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}

		_, _, logErr = as.log.UpdateWorkloadAction(context.TODO(), idY, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}

		return
	}
	createArgsY, err := args.Y.toCreateArgs(as.k8sClient, podX.Spec.NodeName)
	if err != nil {
		log.Printf("error creating create args: %s", err.Error())

		status := walog.WorkloadActionStatusEnumFailed
		logTime := time.Now()

		_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), idX, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}

		_, _, logErr = as.log.UpdateWorkloadAction(context.TODO(), idY, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}

		return
	}

	as.DeleteHandler(idX, walog.WorkloadActionTypeEnumSwapX, deleteArgsX)
	as.DeleteHandler(idY, walog.WorkloadActionTypeEnumSwapY, deleteArgsY)

	time.Sleep(5 * time.Second) // hack to wait for deletes (prober way would be to watch to reach desired replicas)

	log.Printf("all deletes have completed")

	// Delete might take a while, so the created pods might be in pending state for some time
	// until they're finally scheduled where they're supposed to go. That's ok.

	log.Printf("continuing with creates")
	_, err = as.CreateHandler(idX, walog.WorkloadActionTypeEnumSwapX, createArgsX)
	if err != nil {
		status := walog.WorkloadActionStatusEnumFailed
		logTime := time.Now()

		_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), idX, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}
	}
	_, err = as.CreateHandler(idY, walog.WorkloadActionTypeEnumSwapY, createArgsY)
	if err != nil {
		status := walog.WorkloadActionStatusEnumFailed
		logTime := time.Now()

		_, _, logErr := as.log.UpdateWorkloadAction(context.TODO(), idY, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
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
