/*
Copyright 2017 The Kubernetes Authors.

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

package warmimage

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	daemonsetinformer "knative.dev/pkg/client/injection/kube/informers/apps/v1/daemonset"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"

	extlisters "k8s.io/client-go/listers/apps/v1"

	warmimagev2 "github.com/mattmoor/warm-image/pkg/apis/warmimage/v2"
	clientset "github.com/mattmoor/warm-image/pkg/client/clientset/versioned"
	warmimagescheme "github.com/mattmoor/warm-image/pkg/client/clientset/versioned/scheme"
	clients "github.com/mattmoor/warm-image/pkg/client/injection/client"
	informers "github.com/mattmoor/warm-image/pkg/client/injection/informers/warmimage/v2/warmimage"
	listers "github.com/mattmoor/warm-image/pkg/client/listers/warmimage/v2"
	"github.com/mattmoor/warm-image/pkg/reconciler/warmimage/resources"
)

const controllerAgentName = "warmimage-controller"

// Reconciler is the controller implementation for WarmImage resources
type Reconciler struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// warmimageclientset is a clientset for our own API group
	warmimageclientset clientset.Interface

	daemonsetsLister extlisters.DaemonSetLister
	warmimagesLister listers.WarmImageLister

	sleeperImage string

	// Sugared logger is easier to use but is not as performant as the
	// raw logger. In performance critical paths, call logger.Desugar()
	// and use the returned raw logger instead. In addition to the
	// performance benefits, raw logger also preserves type-safety at
	// the expense of slightly greater verbosity.
	Logger *zap.SugaredLogger
}

// Check that we implement the controller.Reconciler interface.
var _ controller.Reconciler = (*Reconciler)(nil)

func init() {
	// Add warmimage-controller types to the default Kubernetes Scheme so Events can be
	// logged for warmimage-controller types.
	warmimagescheme.AddToScheme(scheme.Scheme)
}

// NewController returns a new warmimage controller
func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {

	// Enrich the logs with controller name
	logger := logging.FromContext(ctx)
	warmImageInformer := informers.Get(ctx)
	daemonsetInformer := daemonsetinformer.Get(ctx)

	r := &Reconciler{
		kubeclientset:      kubeclient.Get(ctx),
		warmimageclientset: clients.Get(ctx),
		daemonsetsLister:   daemonsetInformer.Lister(),
		warmimagesLister:   warmImageInformer.Lister(),
		sleeperImage:       os.Getenv("SLEEPER_IMAGE"),
		Logger:             logger,
	}
	impl := controller.NewImpl(r, logger, "WarmImages")

	logger.Info("Setting up event handlers")
	// Set up an event handler for when WarmImage resources change
	warmImageInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
	})

	return impl
}

// Reconcile implements controller.Reconciler
func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the WarmImage resource with this namespace/name
	warmimage, err := c.warmimagesLister.WarmImages(namespace).Get(name)
	if errors.IsNotFound(err) {
		// The WarmImage resource may no longer exist, in which case we stop processing.
		runtime.HandleError(fmt.Errorf("warmimage '%s' in work queue no longer exists", key))
		return nil
	} else if err != nil {
		return err
	}

	if err := c.reconcileDaemonSet(ctx, warmimage); err != nil {
		return err
	}

	return nil
}

func (c *Reconciler) reconcileDaemonSet(ctx context.Context, wi *warmimagev2.WarmImage) error {
	// Make sure the desired image is warmed up ASAP.
	dss, err := c.daemonsetsLister.DaemonSets(wi.Namespace).List(resources.MakeLabelSelector(wi))
	if err != nil {
		return err
	}

	switch {
	// If none exist, create one.
	case len(dss) == 0:
		ds := resources.MakeDaemonSet(wi, c.sleeperImage)
		ds, err = c.kubeclientset.AppsV1().DaemonSets(wi.Namespace).Create(ctx, ds, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		c.Logger.Infof("Warming up: %q, with %q", wi.Spec.Image, ds.Name)

	// If multiple exist, delete all but one.
	case len(dss) > 1:
		c.Logger.Error("NYI: cleaning up multiple daemonsets for a single WarmImage.")
	}

	// Delete any older versions of this WarmImage.
	//propPolicy := metav1.DeletePropagationForeground
	err = c.kubeclientset.ExtensionsV1beta1().DaemonSets(wi.Namespace).DeleteCollection(
		ctx,
		metav1.DeleteOptions{},
		metav1.ListOptions{
			LabelSelector: resources.MakeOldVersionLabelSelector(wi).String()},
	)
	if err != nil {
		return err
	}
	return nil
}
