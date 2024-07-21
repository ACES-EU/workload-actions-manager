package actions

import (
	"context"
	"fmt"
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

func (as *ActionService) SwapHandler(args *SwapArgs) {
	pods := make([]Pod, len(args.Y)+1)
	pods[0] = args.X
	for i := 0; i < len(args.Y); i++ {
		pods[i+1] = args.Y[i]
	}

	targetScales := make(map[TargetScaleKey]int32)
	selectors := make(map[TargetScaleKey]map[string]string)
	nodeX := ""
	nodeY := ""

	for i, pod := range pods {
		podObj, err := as.k8sClient.CoreV1().Pods(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})
		if err != nil {
			log.Printf("error getting pod: %v\n", err)
			return
		}

		nodeName := podObj.Spec.NodeName
		if i == 0 {
			nodeX = nodeName
		} else if nodeY == "" {
			nodeY = nodeName
		} else if nodeY != nodeName {
			// verify all Y pod nodes are the same
			log.Printf("aborting move: all Y pods must be running on the same node: %s != %s\n", nodeY, nodeName)
			return
		}

		deployment, err := getPodsDeployment(podObj, as.k8sClient)
		if err != nil {
			log.Printf("error getting pod's owner reference: %v\n", err)
			return
		}

		deploymentObj, _ := as.k8sClient.AppsV1().Deployments(pod.Namespace).Get(context.TODO(), deployment.Name, metav1.GetOptions{})
		if err != nil {
			log.Printf("error getting pod's deployment: %v\n", err)
			return
		}

		scale := deploymentObj.Status.Replicas
		if err != nil {
			log.Printf("error getting pod's deployment current scale: %v\n", err)
			return
		}

		key := TargetScaleKey{pod.Namespace, deployment.Name}
		_, ok := targetScales[key]
		if !ok {
			targetScales[key] = scale
			selectors[key] = deploymentObj.Spec.Selector.MatchLabels
		}

		targetScales[key] -= 1
	}

	// this will be used later on, but we need to prepare it here
	createArgs := make([]*CreateArgs, len(pods))
	for i, pod := range pods {
		if i == 0 {
			ca, err := pod.toCreateArgs(as.k8sClient, nodeY)
			if err != nil {
				log.Printf("error creating create args: %s", err.Error())
				return
			}
			createArgs[i] = ca
		} else {
			ca, err := pod.toCreateArgs(as.k8sClient, nodeX)
			if err != nil {
				log.Printf("error creating create args: %s", err.Error())
				return
			}
			createArgs[i] = ca
		}
	}

	log.Printf("deleting x and y pods\n")
	// here, calling create for pods of the same workload is UNSAFE! since there is no locking mechanism (yet)
	// for prototype purposes we hence call creates sequentially
	for _, pod := range pods {
		as.DeleteHandler(pod.toDeleteArgs())
	}

	log.Printf("waiting for 1 X pod to be deleted and %d Y pods to be deleted\n", len(args.Y))

	timeout := time.Minute * 5
	timer := time.NewTimer(timeout)

	// once the delete target is met, it is removed from the map
	// wait until all targets have been met or timeout is reached (deleting a pod could take a long time...)
	for len(targetScales) > 0 {
		select {
		case <-timer.C:
			log.Println("waiting for deletes exceeded timeout")
			return
		default:
			log.Printf("waiting for deletes: %d left\n", len(targetScales))
			time.Sleep(30 * time.Second)

			for key := range targetScales {
				targetScale := targetScales[key]
				selector := selectors[key]

				// does not include pods that are terminating
				//currentScale, err := as.k8sClient.AppsV1().
				//	Deployments(key.Namespace).
				//	GetScale(context.TODO(), key.DeploymentName, metav1.GetOptions{})
				//if err != nil {
				//	log.Printf("error getting deployment's current scale: %v\n", err)
				//	return
				//}

				podList, err := as.k8sClient.CoreV1().Pods(key.Namespace).List(context.TODO(), metav1.ListOptions{
					LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: selector}),
				})
				if err != nil {
					panic(err.Error())
				}
				currentScale := len(podList.Items) // including terminating pods

				if int32(currentScale) <= targetScale {
					delete(targetScales, key)
					log.Printf("deployment %s in namespace %s reached target scale %d\n", key.DeploymentName, key.Namespace, targetScale)
				}
			}
		}
	}

	log.Printf("all deletes have completed")
	log.Printf("continuing with creates")

	// here, calling create for pods of the same workload is UNSAFE! since there is no locking mechanism (yet)
	// for prototype purposes we hence call creates sequentially
	for _, createArg := range createArgs {
		_, _ = as.CreateHandler(createArg)
	}

	log.Println("swap action successful")
}

type SwapArgs struct {
	X Pod   `json:"x"`
	Y []Pod `json:"y"`
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

	for i := range args.Y {
		pod := args.Y[i]
		if pod.Namespace == "" {
			pod.Namespace = "default"
		}

		if pod.Name == "" {
			return fmt.Errorf("y pod's, at index %d, name must be specified", i)
		}
	}

	return nil
}
