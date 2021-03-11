/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v2

import (
	v2 "github.com/mattmoor/warm-image/pkg/apis/warmimage/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// WarmImageLister helps list WarmImages.
// All objects returned here must be treated as read-only.
type WarmImageLister interface {
	// List lists all WarmImages in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v2.WarmImage, err error)
	// WarmImages returns an object that can list and get WarmImages.
	WarmImages(namespace string) WarmImageNamespaceLister
	WarmImageListerExpansion
}

// warmImageLister implements the WarmImageLister interface.
type warmImageLister struct {
	indexer cache.Indexer
}

// NewWarmImageLister returns a new WarmImageLister.
func NewWarmImageLister(indexer cache.Indexer) WarmImageLister {
	return &warmImageLister{indexer: indexer}
}

// List lists all WarmImages in the indexer.
func (s *warmImageLister) List(selector labels.Selector) (ret []*v2.WarmImage, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v2.WarmImage))
	})
	return ret, err
}

// WarmImages returns an object that can list and get WarmImages.
func (s *warmImageLister) WarmImages(namespace string) WarmImageNamespaceLister {
	return warmImageNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// WarmImageNamespaceLister helps list and get WarmImages.
// All objects returned here must be treated as read-only.
type WarmImageNamespaceLister interface {
	// List lists all WarmImages in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v2.WarmImage, err error)
	// Get retrieves the WarmImage from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v2.WarmImage, error)
	WarmImageNamespaceListerExpansion
}

// warmImageNamespaceLister implements the WarmImageNamespaceLister
// interface.
type warmImageNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all WarmImages in the indexer for a given namespace.
func (s warmImageNamespaceLister) List(selector labels.Selector) (ret []*v2.WarmImage, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v2.WarmImage))
	})
	return ret, err
}

// Get retrieves the WarmImage from the indexer for a given namespace and name.
func (s warmImageNamespaceLister) Get(name string) (*v2.WarmImage, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v2.Resource("warmimage"), name)
	}
	return obj.(*v2.WarmImage), nil
}
