package wam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"log"
	"os"
	"path/filepath"
)

type WAM struct {
	handle    framework.Handle
	k8sClient *kubernetes.Clientset
	rdb       *redis.Client
}

type SchedulingSuggestion struct {
	ID       types.UID `json:"id"`
	NodeName string    `json:"node_name"`
}

func (sg *SchedulingSuggestion) Clone() framework.StateData {
	return &SchedulingSuggestion{
		ID:       sg.ID,
		NodeName: sg.NodeName,
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

func (w *WAM) getDeploymentName(pod *v1.Pod) (*metav1.OwnerReference, error) {
	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Kind == "ReplicaSet" {
			rs, err := w.k8sClient.AppsV1().ReplicaSets(pod.Namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}

			for _, rsOwnerRef := range rs.OwnerReferences {
				if rsOwnerRef.Kind == "Deployment" {
					return &rsOwnerRef, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("deployment not found for pod %s", pod.Name)
}

func (w *WAM) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) (*framework.PreFilterResult, *framework.Status) {
	lh := klog.FromContext(ctx)

	deployment, err := w.getDeploymentName(pod)
	if err != nil {
		lh.V(3).Error(err, "pod's deployment not found")
		return nil, nil
	}

	lh.V(5).Info(fmt.Sprintf("found pod's deployment %+v", deployment))

	queue := queueName(deployment, pod.Namespace)

	sugEncoded, err := w.rdb.LPop(context.TODO(), queue).Result()
	if errors.Is(err, redis.Nil) {
		lh.V(3).Info(fmt.Sprintf("no suggestion found for %s: scheduling without a scheduling suggestion", pod.Name))
		return nil, framework.NewStatus(framework.Success, "")
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
		ID:       suggestion.ID,
		NodeName: suggestion.NodeName,
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
		// no suggestion has been found, fallback to the rest of the plugins' filters
		return framework.NewStatus(framework.Success)
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
				"example.com/scheduling-suggestion-id": string(suggestion.ID),
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

	lh.V(5).Info("creating a new WAM plugin")

	return &WAM{
		handle:    h,
		rdb:       rdb,
		k8sClient: k8sClient,
	}, nil
}
