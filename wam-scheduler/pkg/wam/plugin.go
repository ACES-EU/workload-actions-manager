package wam

import (
	"context"
	"fmt"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	apiconfig "sigs.k8s.io/scheduler-plugins/apis/config"
	"sigs.k8s.io/scheduler-plugins/apis/config/validation"
)

type WAM struct {
	handle framework.Handle
}

var _ = framework.FilterPlugin(&WAM{})

// Name is the name of the plugin used in the Registry and configurations.
const Name = "WAM"

func (ps *WAM) Name() string {
	return Name
}

func (w *WAM) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	if nodeInfo.Node().Name == "k3d-aces-agent-7" {
		return framework.NewStatus(framework.Success, "k3d-aces-agent-7 is hardcoded")
	}

	return framework.NewStatus(framework.Unschedulable)
}

// New initializes a new plugin and returns it.
func New(ctx context.Context, args runtime.Object, h framework.Handle) (framework.Plugin, error) {
	lh := klog.FromContext(ctx)

	lh.V(5).Info("creating new WAM plugin")
	wamArgs, ok := args.(*apiconfig.WAMArgs)
	if !ok {
		return nil, fmt.Errorf("want args to be of type WAMArgs, got %T", args)
	}

	if err := validation.ValidateWAMPluginArgs(wamArgs); err != nil {
		return nil, err
	}

	return &WAM{handle: h}, nil
}
