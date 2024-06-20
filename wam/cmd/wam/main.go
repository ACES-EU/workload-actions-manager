package main

import (
	"github.com/ACES-EU/workload-actions-manager/wam/pkg/actions"
	"github.com/gorilla/rpc"
	jsoncodec "github.com/gorilla/rpc/json"
	"github.com/redis/go-redis/v9"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"net/http"
	"path/filepath"
)

func main() {
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

	log.Println("Configured k8s client")

	rdb := redis.NewClient(&redis.Options{
		Addr:     "wam-redis-master.default.svc.cluster.local:6379",
		Password: "redis_test_password",
		DB:       0,
	})

	log.Println("Configured Redis client")

	s := rpc.NewServer()
	s.RegisterCodec(jsoncodec.NewCodec(), "application/json")
	s.RegisterService(actions.NewActionService(k8sClient, rdb), "action")
	http.Handle("/rpc", s)

	log.Println("Listening...")
	http.ListenAndServe(":3000", nil)
}
