package main

import (
	"fmt"
	"github.com/ACES-EU/workload-actions-manager/wam/pkg/actions"
	wamconfig "github.com/ACES-EU/workload-actions-manager/wam/pkg/config"
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
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	config, err := wamconfig.New()
	if err != nil {
		log.Fatal(err)
	}

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

	log.Println("configured k8s client")

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.Redis.Host, config.Redis.Port),
		Password: config.Redis.Password,
		DB:       0,
	})

	log.Println("configured Redis client")

	s := rpc.NewServer()
	s.RegisterCodec(jsoncodec.NewCodec(), "application/json")
	s.RegisterService(actions.NewActionService(k8sClient, rdb), "action")
	http.Handle("/rpc", s)

	log.Printf("Listening on %s...\n", config.Server.Address)
	http.ListenAndServe(config.Server.Address, nil)
}
