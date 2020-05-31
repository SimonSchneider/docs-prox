package kubernetes

import (
	// Import to initialize client auth plugins.
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/SimonSchneider/docs-prox/openapi"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	template = "%-32s%-8s\n"
)

// Config for kubernetes
type Config struct {
}

// Build the repository
func (c *Config) Build() openapi.Repsitory {
	kubeconfig := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	repo := &kubernetesRepo{resources: sync.Map{}}
	fmt.Println("setting up kube")
	go repo.watchServices(clientset)
	return repo
}

type kubernetesRepo struct {
	resources sync.Map
}

func (r *kubernetesRepo) Keys() []string {
	keys := make([]string, 0)
	r.resources.Range(func(key interface{}, val interface{}) bool {
		keys = append(keys, key.(string))
		return true
	})
	return keys
}

func (r *kubernetesRepo) Spec(key string) (openapi.Spec, error) {
	if url, ok := r.resources.Load(key); ok {
		return openapi.NewRemoteSpec(url.(string)), nil
	}
	return nil, fmt.Errorf("no such key")
}

func (r *kubernetesRepo) watchServices(clientset *kubernetes.Clientset) {
	api := clientset.CoreV1()
	listOptions := metav1.ListOptions{
		LabelSelector: "swagger",
		FieldSelector: "",
	}
	watcher, err := api.Services("").Watch(context.TODO(), listOptions)
	if err != nil {
		log.Fatal(err)
	}
	ch := watcher.ResultChan()
	for event := range ch {
		svc, ok := event.Object.(*v1.Service)
		if !ok {
			log.Fatal("unexpected type")
		}
		fmt.Println("handling event")
		switch event.Type {
		case watch.Deleted:
			r.deleteService(svc)
		default:
			r.addService(svc)
		}
	}
}

func (r *kubernetesRepo) addService(svc *v1.Service) {
	var ok bool
	var path, port string
	if path, ok = svc.Annotations["swagger"]; !ok {
		fmt.Println("path cant be empty")
		r.deleteService(svc)
		return
	}
	if port, ok = svc.Labels["swagger-port"]; !ok {
		fmt.Println("no port finding default")
		ports := svc.Spec.Ports
		if len(ports) == 1 {
			port = fmt.Sprint(ports[0].Port)
		} else {
			fmt.Println("no default port found")
			r.deleteService(svc)
			return
		}
	}
	url := "http://" + svc.Name + ":" + port + path
	fmt.Printf("storing %s - %s\n", svc.Name, url)
	r.resources.Store(svc.Name, url)
}

func (r *kubernetesRepo) deleteService(svc *v1.Service) {
	fmt.Printf("service deleted %s\n", svc.Name)
	r.resources.Delete(svc.Name)
}