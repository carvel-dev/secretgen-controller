// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "carvel.dev/secretgen-controller/pkg/apis/secretgen/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// RSAKeyLister helps list RSAKeys.
// All objects returned here must be treated as read-only.
type RSAKeyLister interface {
	// List lists all RSAKeys in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.RSAKey, err error)
	// RSAKeys returns an object that can list and get RSAKeys.
	RSAKeys(namespace string) RSAKeyNamespaceLister
	RSAKeyListerExpansion
}

// rSAKeyLister implements the RSAKeyLister interface.
type rSAKeyLister struct {
	indexer cache.Indexer
}

// NewRSAKeyLister returns a new RSAKeyLister.
func NewRSAKeyLister(indexer cache.Indexer) RSAKeyLister {
	return &rSAKeyLister{indexer: indexer}
}

// List lists all RSAKeys in the indexer.
func (s *rSAKeyLister) List(selector labels.Selector) (ret []*v1alpha1.RSAKey, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.RSAKey))
	})
	return ret, err
}

// RSAKeys returns an object that can list and get RSAKeys.
func (s *rSAKeyLister) RSAKeys(namespace string) RSAKeyNamespaceLister {
	return rSAKeyNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// RSAKeyNamespaceLister helps list and get RSAKeys.
// All objects returned here must be treated as read-only.
type RSAKeyNamespaceLister interface {
	// List lists all RSAKeys in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.RSAKey, err error)
	// Get retrieves the RSAKey from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.RSAKey, error)
	RSAKeyNamespaceListerExpansion
}

// rSAKeyNamespaceLister implements the RSAKeyNamespaceLister
// interface.
type rSAKeyNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all RSAKeys in the indexer for a given namespace.
func (s rSAKeyNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.RSAKey, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.RSAKey))
	})
	return ret, err
}

// Get retrieves the RSAKey from the indexer for a given namespace and name.
func (s rSAKeyNamespaceLister) Get(name string) (*v1alpha1.RSAKey, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("rsakey"), name)
	}
	return obj.(*v1alpha1.RSAKey), nil
}
