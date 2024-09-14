package wam

import (
	"context"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	clientsetfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/defaultbinder"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/queuesort"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	tf "k8s.io/kubernetes/pkg/scheduler/testing/framework"
	"testing"
)

func newFake(ctx context.Context, args runtime.Object, h framework.Handle) (framework.Plugin, error) {
	return &WAM{
		handle:    h,
		rdb:       nil,
		k8sClient: nil,
	}, nil
}

func TestWAMPlugin(t *testing.T) {

	noResources := v1.PodSpec{
		Containers: []v1.Container{},
	}

	tests := []struct {
		name                 string
		pod                  *v1.Pod
		pods                 []*v1.Pod
		nodeInfos            []*framework.NodeInfo
		schedulingSuggestion *SchedulingSuggestion
		expected             []framework.Code
	}{
		{
			name:                 "no suggestion",
			pod:                  &v1.Pod{Spec: noResources},
			nodeInfos:            []*framework.NodeInfo{makeNodeInfo("node_1", 4000, 10000), makeNodeInfo("node_2", 4000, 10000)},
			schedulingSuggestion: nil,
			expected:             []framework.Code{framework.Success, framework.Success},
		},
		{
			name:      "suggestion for an non-existing node",
			pod:       &v1.Pod{Spec: noResources},
			nodeInfos: []*framework.NodeInfo{makeNodeInfo("node_1", 4000, 10000), makeNodeInfo("node_2", 4000, 10000)},
			schedulingSuggestion: &SchedulingSuggestion{
				ID:       "id",
				NodeName: "node_3",
			},
			expected: []framework.Code{framework.Unschedulable, framework.Unschedulable},
		},
		{
			name:      "suggestion for a node in cluster",
			pod:       &v1.Pod{Spec: noResources},
			nodeInfos: []*framework.NodeInfo{makeNodeInfo("node_1", 4000, 10000), makeNodeInfo("node_2", 4000, 10000)},
			schedulingSuggestion: &SchedulingSuggestion{
				ID:       "id",
				NodeName: "node_2",
			},
			expected: []framework.Code{framework.Unschedulable, framework.Success},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cs := clientsetfake.NewSimpleClientset()
			informerFactory := informers.NewSharedInformerFactory(cs, 0)
			podInformer := informerFactory.Core().V1().Pods().Informer()
			for _, p := range test.pods {
				podInformer.GetStore().Add(p)
			}
			registeredPlugins := []tf.RegisterPluginFunc{
				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
				tf.RegisterFilterPlugin(Name, newFake),
			}
			fakeSharedLister := &fakeSharedLister{nodes: test.nodeInfos}

			fh, err := tf.NewFramework(
				ctx,
				registeredPlugins,
				"default-scheduler",
				frameworkruntime.WithClientSet(cs),
				frameworkruntime.WithInformerFactory(informerFactory),
				frameworkruntime.WithSnapshotSharedLister(fakeSharedLister),
			)
			if err != nil {
				t.Fatalf("fail to create framework: %s", err)
			}

			wam, err := newFake(ctx, nil, fh)

			plugin := wam.(framework.FilterPlugin)
			var actual []framework.Code
			for i := range test.nodeInfos {
				state := framework.NewCycleState()
				state.Write(schedulingSuggestionKey, test.schedulingSuggestion)
				status := plugin.Filter(context.Background(), state, test.pod, test.nodeInfos[i])
				actual = append(actual, status.Code())
			}

			assert.Equal(t, test.expected, actual)
		})
	}
}

func makeNodeInfo(node string, milliCPU, memory int64) *framework.NodeInfo {
	ni := framework.NewNodeInfo()
	ni.SetNode(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: node},
		Status: v1.NodeStatus{
			Capacity: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewMilliQuantity(milliCPU, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(memory, resource.BinarySI),
			},
			Allocatable: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewMilliQuantity(milliCPU, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(memory, resource.BinarySI),
			},
		},
	})
	return ni
}

func makePod(name string, requests v1.ResourceList) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: name,
					Resources: v1.ResourceRequirements{
						Requests: requests,
					},
				},
			},
		},
	}
}

var _ framework.SharedLister = &fakeSharedLister{}

type fakeSharedLister struct {
	nodes []*framework.NodeInfo
}

func (f *fakeSharedLister) StorageInfos() framework.StorageInfoLister {
	return nil
}

func (f *fakeSharedLister) NodeInfos() framework.NodeInfoLister {
	return tf.NodeInfoLister(f.nodes)
}
