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
	"path/filepath"
	apiconfig "sigs.k8s.io/scheduler-plugins/apis/config"
	"sigs.k8s.io/scheduler-plugins/apis/config/validation"
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

var _ = framework.FilterPlugin(&WAM{})
var _ = framework.PreFilterPlugin(&WAM{})

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
		lh.Error(err, "Pod's deployment not found")
		return nil, nil
	}

	lh.Info(fmt.Sprintf("Found pod's deployment %+v", deployment))

	queue := queueName(deployment, pod.Namespace)

	sugEncoded, err := w.rdb.LPop(context.TODO(), queue).Result()
	if errors.Is(err, redis.Nil) {
		lh.Info("no suggestion found: scheduling without a scheduling suggestion")
		return nil, framework.NewStatus(framework.Success, "")
	} else if err != nil {
		lh.Error(err, "")
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

	lh.Info(fmt.Sprintf("adding suggestion %+v to cycle state", suggestion))

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
	if !ok {
		// todo
	}

	lh.Info(fmt.Sprintf("using suggestion %+v", suggestion))

	if nodeInfo.Node().Name == suggestion.NodeName {
		return framework.NewStatus(framework.Success, fmt.Sprintf("found suggested node %s", suggestion.NodeName))
	}

	return framework.NewStatus(framework.Unschedulable)
}

// New initializes a new plugin and returns it.
func New(ctx context.Context, args runtime.Object, h framework.Handle) (framework.Plugin, error) {
	lh := klog.FromContext(ctx)

	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatal(err)
		}
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     "wam-redis-master.default.svc.cluster.local:6379",
		Password: "redis_test_password",
		DB:       0,
	})

	lh.V(5).Info("creating new WAM plugin")
	wamArgs, ok := args.(*apiconfig.WAMArgs)
	if !ok {
		return nil, fmt.Errorf("want args to be of type WAMArgs, got %T", args)
	}

	if err := validation.ValidateWAMPluginArgs(wamArgs); err != nil {
		return nil, err
	}

	return &WAM{
		handle:    h,
		rdb:       rdb,
		k8sClient: k8sClient,
	}, nil
}
