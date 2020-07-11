package kube

import (
	"context"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/watch"

	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v12 "k8s.io/client-go/kubernetes/typed/core/v1"

	// Import to initialize client auth plugins.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// EventType that was seen
type EventType int32

// EventTypes for different events
const (
	Added EventType = iota
	Modified
	Deleted
	Bookmark
	Error
)

// ListOptions filters watchers
type ListOptions struct {
	Namespace     string
	LabelSelector string
}

// Client is a wrapper of the kubernetes API
type Client struct {
	api v12.CoreV1Interface
}

// NewKubeClient creates a new Client and tries to authenticate with kubernetes
func NewKubeClient() (*Client, error) {
	kubeConfig := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	api := clientSet.CoreV1()
	return &Client{api: api}, nil
}

// Service represents a kubernetes service
type Service struct {
	Name   string
	Labels map[string]string
	Host   string
	Ports  map[string]int32
}

func toService(svc *v1.Service) *Service {
	ports := make(map[string]int32, len(svc.Spec.Ports))
	for _, port := range svc.Spec.Ports {
		ports[port.Name] = port.Port
	}
	host := svc.Name
	return &Service{Name: svc.Name, Labels: merge(svc.Labels, svc.Annotations), Ports: ports, Host: host}
}

// WatchService watch changes of services
func (k *Client) WatchService(ctx context.Context, opts ListOptions, watcherFunc func(*Service, EventType)) error {
	return watchAny(ctx, k.api.Services(opts.Namespace), opts, func(object runtime.Object, eventType EventType) {
		if svc, ok := object.(*v1.Service); ok {
			watcherFunc(toService(svc), eventType)
		}
	})
}

// ConfigMap represents a kubernetes configMap
type ConfigMap struct {
	Name string
	Data map[string]string
}

func toConfigMap(cm *v1.ConfigMap) *ConfigMap {
	return &ConfigMap{
		Name: cm.Name,
		Data: merge(cm.Data),
	}
}

// WatchConfigMap watch changes of ConfigMaps
func (k *Client) WatchConfigMap(ctx context.Context, opts ListOptions, watcherFunc func(*ConfigMap, EventType)) error {
	return watchAny(ctx, k.api.ConfigMaps(opts.Namespace), opts, func(object runtime.Object, eventType EventType) {
		if cm, ok := object.(*v1.ConfigMap); ok {
			watcherFunc(toConfigMap(cm), eventType)
		}
	})
}

type watchable interface {
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

func watchAny(ctx context.Context, watchable watchable, opts ListOptions, watcherFunc func(object runtime.Object, eventType EventType)) error {
	watcher, err := watchable.Watch(ctx, toListOpts(opts))
	if err != nil {
		return err
	}
	go func() {
		for event := range watcher.ResultChan() {
			watcherFunc(event.Object, toEventType(event.Type))
		}
	}()
	return nil
}

func merge(others ...map[string]string) map[string]string {
	length := 0
	for _, other := range others {
		length += len(other)
	}
	labels := make(map[string]string, length)
	for _, other := range others {
		for k, v := range other {
			labels[k] = v
		}
	}
	return labels
}

func toListOpts(opts ListOptions) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
	}
}

func toEventType(eventType watch.EventType) EventType {
	switch eventType {
	case watch.Added:
		return Added
	case watch.Modified:
		return Modified
	case watch.Deleted:
		return Deleted
	case watch.Bookmark:
		return Bookmark
	case watch.Error:
		return Error
	}
	return Error
}
