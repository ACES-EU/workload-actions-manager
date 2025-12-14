package wam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	walog "github.com/ACES-EU/workload-actions-manager/logger"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type WAM struct {
	handle    framework.Handle
	k8sClient *kubernetes.Clientset
	rdb       *redis.Client
	log       *walog.WALogger
}

type SchedulingSuggestion struct {
	ID         uuid.UUID                    `json:"id"`
	NodeName   string                       `json:"node_name"`
	ActionType walog.WorkloadActionTypeEnum `json:"action_type"`
}

func (sg *SchedulingSuggestion) Clone() framework.StateData {
	return &SchedulingSuggestion{
		ID:         sg.ID,
		NodeName:   sg.NodeName,
		ActionType: sg.ActionType,
	}
}

var _ = framework.PreFilterPlugin(&WAM{})
var _ = framework.FilterPlugin(&WAM{})
var _ = framework.PostBindPlugin(&WAM{})

// Name is the name of the plugin used in the Registry and configurations.
const Name = "WAM"
const schedulingSuggestionKey = "scheduling-suggestion"

func (w *WAM) Name() string {
	return Name
}

func queueName(or *metav1.OwnerReference, namespace string) string {
	return fmt.Sprintf("%s:%s:%s:%s", namespace, or.APIVersion, or.Kind, or.Name)
}

func (w *WAM) getPodOwner(pod *v1.Pod) (*metav1.OwnerReference, error) {
	obj := metav1.Object(pod)
	owner := metav1.GetControllerOf(obj)

	if owner == nil {
		return &metav1.OwnerReference{}, fmt.Errorf("no owner of pod: %s found", pod.Name)
	}

	for {
		var parent metav1.Object
		var err error

		namespace := obj.GetNamespace()

		switch owner.Kind {
		case "ReplicaSet":
			parent, err = w.k8sClient.AppsV1().ReplicaSets(namespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
		case "StatefulSet":
			parent, err = w.k8sClient.AppsV1().StatefulSets(namespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
		case "DaemonSet":
			parent, err = w.k8sClient.AppsV1().DaemonSets(namespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
		case "Job":
			parent, err = w.k8sClient.BatchV1().Jobs(namespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
		case "Deployment":
			parent, err = w.k8sClient.AppsV1().Deployments(namespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
		case "CronJob":
			parent, err = w.k8sClient.BatchV1().CronJobs(namespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
		default:
			return owner, nil
		}

		if err != nil {
			return owner, fmt.Errorf("failed to get owner %s %s: %w", owner.Kind, owner.Name, err)
		}

		nextOwner := metav1.GetControllerOf(parent)
		if nextOwner == nil {
			return owner, nil
		}

		obj = parent
		owner = nextOwner
	}
}

func k8sKindToParentType(kind string) (walog.PodParentTypeEnum, error) {
	switch kind {
	case "ReplicaSet":
		return walog.PodParentTypeEnumReplicaset, nil
	case "StatefulSet":
		return walog.PodParentTypeEnumStatefulset, nil
	case "DaemonSet":
		return walog.PodParentTypeEnumDaemonset, nil
	case "Job":
		return walog.PodParentTypeEnumJob, nil
	case "CronJob":
		return walog.PodParentTypeEnumCronjob, nil
	case "Deployment":
		return walog.PodParentTypeEnumDeployment, nil
	default:
		return "", fmt.Errorf("unknown kind: %s", kind)
	}
}

func (w *WAM) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) (*framework.PreFilterResult, *framework.Status) {
	lh := klog.FromContext(ctx)

	owner, err := w.getPodOwner(pod)
	if err != nil {
		lh.V(3).Error(err, "pod's owner not found")
		return nil, nil
	}

	lh.V(5).Info(fmt.Sprintf("found pod's owner %+v", owner))

	queue := queueName(owner, pod.Namespace)

	sugEncoded, err := w.rdb.LPop(context.TODO(), queue).Result()
	if errors.Is(err, redis.Nil) {
		lh.V(3).Info(fmt.Sprintf("no suggestion found for %s: keeping pod in pending state", pod.Name))
		return nil, framework.NewStatus(framework.Unschedulable, "")
	} else if err != nil {
		lh.Error(err, "error connecting to Redis")
		return nil, framework.NewStatus(framework.Error, "")
	}

	var suggestion SchedulingSuggestion

	if err = json.Unmarshal([]byte(sugEncoded), &suggestion); err != nil {
		lh.Error(err, "")
		return nil, framework.NewStatus(framework.Error, "")
	}

	state.Write(schedulingSuggestionKey, &SchedulingSuggestion{
		ID:         suggestion.ID,
		NodeName:   suggestion.NodeName,
		ActionType: suggestion.ActionType,
	})

	lh.V(5).Info(fmt.Sprintf("adding suggestion %+v to cycle state", suggestion))

	return nil, framework.NewStatus(framework.Success, "")
}

func (w *WAM) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func (w *WAM) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	lh := klog.FromContext(ctx)

	data, err := state.Read(schedulingSuggestionKey)
	if err != nil {
		return nil
	}
	suggestion, ok := data.(*SchedulingSuggestion)
	if !ok || suggestion == nil {
		// no suggestion has been found, keep the pod pending
		return framework.NewStatus(framework.Unschedulable)
	}

	lh.V(5).Info(fmt.Sprintf("using suggestion %+v", suggestion))

	if nodeInfo.Node().Name == suggestion.NodeName {
		return framework.NewStatus(framework.Success, fmt.Sprintf("found suggested node %s", suggestion.NodeName))
	}

	return framework.NewStatus(framework.Unschedulable)
}

func (w *WAM) PostBind(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) {
	lh := klog.FromContext(ctx)

	data, err := state.Read(schedulingSuggestionKey)
	if err != nil {
		// todo
		return
	}
	suggestion, ok := data.(*SchedulingSuggestion)
	if !ok {
		// todo
		return
	}

	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				"example.com/scheduling-suggestion-id": suggestion.ID.String(),
			},
		},
	}

	// Convert the patch to JSON
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		panic(err.Error())
	}

	// Apply the patch
	_, err = w.k8sClient.CoreV1().Pods(pod.Namespace).Patch(context.TODO(), pod.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		// todo
		return
	}

	owner, err := w.getPodOwner(pod)
	if err == nil {
		depUid := string(owner.UID)
		parentType, _ := k8sKindToParentType(owner.Kind)
		logTime := time.Now()
		_, _, logErr := w.log.UpdateWorkloadAction(context.TODO(), suggestion.ID, walog.WorkloadActionUpdate{
			PodParentType: &parentType,
			PodParentName: &owner.Name,
			PodParentUID:  &depUid,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}
	}

	logTime := time.Now()
	wa, _, logErr := w.log.UpdateWorkloadAction(context.TODO(), suggestion.ID, walog.WorkloadActionUpdate{
		CreatedPodName:      &pod.Name,
		CreatedPodNamespace: &pod.Namespace,
		CreatedNodeName:     &nodeName,
		UpdatedAt:           &logTime,
	})
	if logErr != nil {
		log.Printf("error updating action log: %v\n", logErr)
	}

	if wa.ActionType == walog.WorkloadActionTypeEnumCreate || wa.ActionType == walog.WorkloadActionTypeEnumSwapX || wa.ActionType == walog.WorkloadActionTypeEnumSwapY {
		status := walog.WorkloadActionStatusEnumSucceeded
		logTime := time.Now()
		_, _, logErr := w.log.UpdateWorkloadAction(context.TODO(), suggestion.ID, walog.WorkloadActionUpdate{
			ActionStatus:  &status,
			ActionEndTime: &logTime,
			UpdatedAt:     &logTime,
		})
		if logErr != nil {
			log.Printf("error updating action log: %v\n", logErr)
		}
		h := walog.WorkloadDecisionStatusUpdate{
			PodName:        pod.Name,
			Namespace:      pod.Namespace,
			NodeName:       nodeName,
			ActionType:     suggestion.ActionType,
			DecisionStatus: status,
		}
		fmt.Printf("%+v\n", h)
		_, logErr = w.log.UpdateWorkloadDecisionStatus(context.TODO(), h)
		if logErr != nil {
			log.Printf("error updating decision status log: %v\n", logErr)
		}
	}

	lh.V(5).Info(fmt.Sprintf("added suggestion %+v as `example.com/scheduling-suggestion` annotation to %s", suggestion, pod.Name))
}

// New initializes a new plugin and returns it.
func New(ctx context.Context, args runtime.Object, h framework.Handle) (framework.Plugin, error) {
	lh := klog.FromContext(ctx)

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			log.Fatal(err)
		}
	}

	k8sClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatal(err)
	}

	redisHost := os.Getenv("WAM_REDIS_HOST")
	redisPort := os.Getenv("WAM_REDIS_PORT")
	redisPassword := os.Getenv("WAM_REDIS_PASSWORD")
	lh.V(5).Info(fmt.Sprintf("connecting to Redis on %s:%s", redisHost, redisPort))
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       0,
	})

	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatal("error connecting to Redis")
	}

	waLoggerScheme := os.Getenv("WALOGGER_SCHEME")
	waLoggerHost := os.Getenv("WALOGGER_HOST")

	waLogger := walog.NewWALogger(waLoggerScheme, waLoggerHost)

	lh.V(5).Info("creating a new WAM plugin")

	return &WAM{
		handle:    h,
		rdb:       rdb,
		log:       waLogger,
		k8sClient: k8sClient,
	}, nil
}
