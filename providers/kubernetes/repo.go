package kubernetes

import (
	// Import to initialize client auth plugins.
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	v12 "k8s.io/client-go/kubernetes/typed/core/v1"

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

// Build the repository
func Configure(ctx context.Context, store openapi.ApiStore) error {
	kubeconfig := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	api := clientSet.CoreV1()
	repo := &kubernetesRepo{api: api, watchers: make([]watch.Interface, 0), store: store}
	return repo.Start(ctx)
}

type kubernetesRepo struct {
	api      v12.CoreV1Interface
	watchers []watch.Interface
	store    openapi.ApiStore
}

type WatcherBuilder func(ctx context.Context) error

func (r *kubernetesRepo) Start(ctx context.Context) error {
	for _, builder := range []WatcherBuilder{r.startSvcWatcher, r.startRemoteCMWatcher} {
		err := builder(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *kubernetesRepo) startSvcWatcher(ctx context.Context) error {
	listOptions := metav1.ListOptions{
		LabelSelector: "swagger",
	}
	watcher, err := r.api.Services("").Watch(ctx, listOptions)
	if err != nil {
		return err
	}
	go func() {
		for event := range watcher.ResultChan() {
			svc, ok := event.Object.(*v1.Service)
			if !ok {
				log.Printf("ignoring unknown event type %v\n", event)
				continue
			}
			switch event.Type {
			case watch.Deleted:
				r.deleteSvc(svc)
			case watch.Added, watch.Modified:
				r.addSvc(svc)
			}
		}
	}()
	return nil
}

func (r *kubernetesRepo) addSvc(svc *v1.Service) {
	var ok bool
	var path, port string
	if path, ok = svc.Annotations["swagger"]; !ok {
		fmt.Println("path cant be empty")
		r.deleteSvc(svc)
		return
	}
	if port, ok = svc.Labels["swagger-port"]; !ok {
		fmt.Println("no port finding default")
		ports := svc.Spec.Ports
		if len(ports) == 1 {
			port = fmt.Sprint(ports[0].Port)
		} else {
			fmt.Println("no default port found")
			r.deleteSvc(svc)
			return
		}
	}
	url := "http://" + svc.Name + ":" + port + path
	fmt.Printf("storing %s - %s\n", svc.Name, url)
	r.store.Put("kubesvc", svc.Name, openapi.NewRemoteSpec(url))
}

func (r *kubernetesRepo) deleteSvc(svc *v1.Service) {
	r.store.Remove("kubesvc", svc.Name)
	fmt.Printf("service deleted %s\n", svc.Name)
}

func (r *kubernetesRepo) startRemoteCMWatcher(ctx context.Context) error {
	listOptions := metav1.ListOptions{
		LabelSelector: "remote-swagger",
	}
	watcher, err := r.api.ConfigMaps("").Watch(ctx, listOptions)
	if err != nil {
		return err
	}
	go func() {
		for event := range watcher.ResultChan() {
			svc, ok := event.Object.(*v1.ConfigMap)
			if !ok {
				log.Printf("ignoring unknown event type %v\n", event)
				continue
			}
			switch event.Type {
			case watch.Deleted:
				r.deleteRemoteCM(svc)
			default:
				r.addRemoteCM(svc)
			}
		}
	}()
	return nil
}

func (r *kubernetesRepo) addRemoteCM(cm *v1.ConfigMap) {
	data := make(map[string]openapi.Spec)
	source := sourceOfCM(cm)
	for key, val := range cm.Data {
		data[key] = openapi.NewRemoteSpec(val)
	}
	r.store.ReplaceAllOf(source, data)
}

func (r *kubernetesRepo) deleteRemoteCM(cm *v1.ConfigMap) {
	r.store.RemoveAllOf(sourceOfCM(cm))
	fmt.Printf("remoteConfigMap deleted %s\n", cm.Name)
}

func sourceOfCM(c *v1.ConfigMap) string {
	return fmt.Sprintf("kubec:cm:%s", c.Name)
}
