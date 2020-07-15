package kubernetes

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/SimonSchneider/docs-prox/providers/kubernetes/kube"

	"github.com/SimonSchneider/docs-prox/openapi"
)

const (
	serviceSource = "kubeService"
)

// Configure the SpecStore
func Configure(ctx context.Context, store openapi.SpecStore) error {
	api, err := kube.NewKubeClient()
	if err != nil {
		return err
	}
	repo := &kubeWatcher{client: api, store: store}
	return repo.start(ctx)
}

type kubeWatcher struct {
	client *kube.Client
	store  openapi.SpecStore
}

func (r *kubeWatcher) start(ctx context.Context) error {
	builders := []func(context.Context) error{
		r.startSvcWatcher, r.startRemoteCMWatcher,
	}
	for _, builder := range builders {
		err := builder(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *kubeWatcher) startSvcWatcher(ctx context.Context) error {
	opts := kube.ListOptions{
		Namespace:     "",
		LabelSelector: "swagger",
	}
	return r.client.WatchService(ctx, opts, func(svc *kube.Service, eventType kube.EventType) {
		switch eventType {
		case kube.Added, kube.Modified:
			r.addSvc(svc)
		case kube.Deleted:
			r.deleteSvc(svc)
		}
	})
}

func (r *kubeWatcher) addSvc(svc *kube.Service) {
	var ok bool
	var path string
	var port int32
	if path, ok = svc.Labels["swagger-path"]; !ok {
		fmt.Println("path cant be empty")
		r.deleteSvc(svc)
		return
	}
	if len(svc.Ports) == 1 {
		for _, p := range svc.Ports {
			port = p
		}
	} else if portLabel, ok := svc.Labels["swagger-port"]; ok {
		if p, err := strconv.Atoi(portLabel); err == nil {
			for _, portNumber := range svc.Ports {
				if int32(p) == portNumber {
					port = int32(p)
					break
				}
			}
		} else {
			found := false
			for name, portNumber := range svc.Ports {
				if name == portLabel {
					port = portNumber
					found = true
					break
				}
			}
			if !found {
				fmt.Println("Wasn't able to find port")
				r.deleteSvc(svc)
				return
			}
		}
	} else {
		fmt.Println("Wasn't able to find port")
		r.deleteSvc(svc)
		return
	}
	url := "http://" + svc.Host + ":" + fmt.Sprintf("%d", port) + path
	fmt.Printf("storing %s - %s\n", svc.Name, url)
	r.store.Put(serviceSource, svc.Name, openapi.NewCachedRemoteSpec(url, 20*time.Second))
}

func (r *kubeWatcher) deleteSvc(svc *kube.Service) {
	r.store.Remove(serviceSource, svc.Name)
	fmt.Printf("service deleted %s\n", svc.Name)
}

func (r *kubeWatcher) startRemoteCMWatcher(ctx context.Context) error {
	opts := kube.ListOptions{
		Namespace:     "",
		LabelSelector: "remote-swagger",
	}
	return r.client.WatchConfigMap(ctx, opts, func(cm *kube.ConfigMap, eventType kube.EventType) {
		switch eventType {
		case kube.Deleted:
			r.deleteRemoteCM(cm)
		default:
			r.addRemoteCM(cm)
		}
	})
}

func (r *kubeWatcher) addRemoteCM(cm *kube.ConfigMap) {
	data := make(map[string]openapi.Spec)
	source := sourceOfCM(cm)
	for key, val := range cm.Data {
		data[key] = openapi.NewCachedRemoteSpec(val, 20*time.Second)
	}
	r.store.ReplaceAllOf(source, data)
}

func (r *kubeWatcher) deleteRemoteCM(cm *kube.ConfigMap) {
	r.store.RemoveAllOf(sourceOfCM(cm))
	fmt.Printf("remoteConfigMap deleted %s\n", cm.Name)
}

func sourceOfCM(c *kube.ConfigMap) string {
	return fmt.Sprintf("kubec:cm:%s", c.Name)
}
