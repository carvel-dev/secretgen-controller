// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	secretgenv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	versioned "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client/clientset/versioned"
	internalinterfaces "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client/listers/secretgen/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// SSHKeyInformer provides access to a shared informer and lister for
// SSHKeys.
type SSHKeyInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.SSHKeyLister
}

type sSHKeyInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewSSHKeyInformer constructs a new informer for SSHKey type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewSSHKeyInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredSSHKeyInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredSSHKeyInformer constructs a new informer for SSHKey type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredSSHKeyInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.SecretgenV1alpha1().SSHKeys(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.SecretgenV1alpha1().SSHKeys(namespace).Watch(context.TODO(), options)
			},
		},
		&secretgenv1alpha1.SSHKey{},
		resyncPeriod,
		indexers,
	)
}

func (f *sSHKeyInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredSSHKeyInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *sSHKeyInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&secretgenv1alpha1.SSHKey{}, f.defaultInformer)
}

func (f *sSHKeyInformer) Lister() v1alpha1.SSHKeyLister {
	return v1alpha1.NewSSHKeyLister(f.Informer().GetIndexer())
}
