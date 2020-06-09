/*
Copyright 2019 Rancher Labs, Inc.

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

// Code generated by codegen. DO NOT EDIT.

package v1

import (
	"context"
	"time"

	v1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	clientset "github.com/rancher/system-upgrade-controller/pkg/generated/clientset/versioned/typed/upgrade.cattle.io/v1"
	informers "github.com/rancher/system-upgrade-controller/pkg/generated/informers/externalversions/upgrade.cattle.io/v1"
	listers "github.com/rancher/system-upgrade-controller/pkg/generated/listers/upgrade.cattle.io/v1"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/condition"
	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/kv"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type PlanHandler func(string, *v1.Plan) (*v1.Plan, error)

type PlanController interface {
	generic.ControllerMeta
	PlanClient

	OnChange(ctx context.Context, name string, sync PlanHandler)
	OnRemove(ctx context.Context, name string, sync PlanHandler)
	Enqueue(namespace, name string)
	EnqueueAfter(namespace, name string, duration time.Duration)

	Cache() PlanCache
}

type PlanClient interface {
	Create(*v1.Plan) (*v1.Plan, error)
	Update(*v1.Plan) (*v1.Plan, error)
	UpdateStatus(*v1.Plan) (*v1.Plan, error)
	Delete(namespace, name string, options *metav1.DeleteOptions) error
	Get(namespace, name string, options metav1.GetOptions) (*v1.Plan, error)
	List(namespace string, opts metav1.ListOptions) (*v1.PlanList, error)
	Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error)
	Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Plan, err error)
}

type PlanCache interface {
	Get(namespace, name string) (*v1.Plan, error)
	List(namespace string, selector labels.Selector) ([]*v1.Plan, error)

	AddIndexer(indexName string, indexer PlanIndexer)
	GetByIndex(indexName, key string) ([]*v1.Plan, error)
}

type PlanIndexer func(obj *v1.Plan) ([]string, error)

type planController struct {
	controllerManager *generic.ControllerManager
	clientGetter      clientset.PlansGetter
	informer          informers.PlanInformer
	gvk               schema.GroupVersionKind
}

func NewPlanController(gvk schema.GroupVersionKind, controllerManager *generic.ControllerManager, clientGetter clientset.PlansGetter, informer informers.PlanInformer) PlanController {
	return &planController{
		controllerManager: controllerManager,
		clientGetter:      clientGetter,
		informer:          informer,
		gvk:               gvk,
	}
}

func FromPlanHandlerToHandler(sync PlanHandler) generic.Handler {
	return func(key string, obj runtime.Object) (ret runtime.Object, err error) {
		var v *v1.Plan
		if obj == nil {
			v, err = sync(key, nil)
		} else {
			v, err = sync(key, obj.(*v1.Plan))
		}
		if v == nil {
			return nil, err
		}
		return v, err
	}
}

func (c *planController) Updater() generic.Updater {
	return func(obj runtime.Object) (runtime.Object, error) {
		newObj, err := c.Update(obj.(*v1.Plan))
		if newObj == nil {
			return nil, err
		}
		return newObj, err
	}
}

func UpdatePlanDeepCopyOnChange(client PlanClient, obj *v1.Plan, handler func(obj *v1.Plan) (*v1.Plan, error)) (*v1.Plan, error) {
	if obj == nil {
		return obj, nil
	}

	copyObj := obj.DeepCopy()
	newObj, err := handler(copyObj)
	if newObj != nil {
		copyObj = newObj
	}
	if obj.ResourceVersion == copyObj.ResourceVersion && !equality.Semantic.DeepEqual(obj, copyObj) {
		return client.Update(copyObj)
	}

	return copyObj, err
}

func (c *planController) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	c.controllerManager.AddHandler(ctx, c.gvk, c.informer.Informer(), name, handler)
}

func (c *planController) AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler) {
	removeHandler := generic.NewRemoveHandler(name, c.Updater(), handler)
	c.controllerManager.AddHandler(ctx, c.gvk, c.informer.Informer(), name, removeHandler)
}

func (c *planController) OnChange(ctx context.Context, name string, sync PlanHandler) {
	c.AddGenericHandler(ctx, name, FromPlanHandlerToHandler(sync))
}

func (c *planController) OnRemove(ctx context.Context, name string, sync PlanHandler) {
	removeHandler := generic.NewRemoveHandler(name, c.Updater(), FromPlanHandlerToHandler(sync))
	c.AddGenericHandler(ctx, name, removeHandler)
}

func (c *planController) Enqueue(namespace, name string) {
	c.controllerManager.Enqueue(c.gvk, c.informer.Informer(), namespace, name)
}

func (c *planController) EnqueueAfter(namespace, name string, duration time.Duration) {
	c.controllerManager.EnqueueAfter(c.gvk, c.informer.Informer(), namespace, name, duration)
}

func (c *planController) Informer() cache.SharedIndexInformer {
	return c.informer.Informer()
}

func (c *planController) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *planController) Cache() PlanCache {
	return &planCache{
		lister:  c.informer.Lister(),
		indexer: c.informer.Informer().GetIndexer(),
	}
}

func (c *planController) Create(obj *v1.Plan) (*v1.Plan, error) {
	return c.clientGetter.Plans(obj.Namespace).Create(context.TODO(), obj, metav1.CreateOptions{})
}

func (c *planController) Update(obj *v1.Plan) (*v1.Plan, error) {
	return c.clientGetter.Plans(obj.Namespace).Update(context.TODO(), obj, metav1.UpdateOptions{})
}

func (c *planController) UpdateStatus(obj *v1.Plan) (*v1.Plan, error) {
	return c.clientGetter.Plans(obj.Namespace).UpdateStatus(context.TODO(), obj, metav1.UpdateOptions{})
}

func (c *planController) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	return c.clientGetter.Plans(namespace).Delete(context.TODO(), name, *options)
}

func (c *planController) Get(namespace, name string, options metav1.GetOptions) (*v1.Plan, error) {
	return c.clientGetter.Plans(namespace).Get(context.TODO(), name, options)
}

func (c *planController) List(namespace string, opts metav1.ListOptions) (*v1.PlanList, error) {
	return c.clientGetter.Plans(namespace).List(context.TODO(), opts)
}

func (c *planController) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return c.clientGetter.Plans(namespace).Watch(context.TODO(), opts)
}

func (c *planController) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Plan, err error) {
	return c.clientGetter.Plans(namespace).Patch(context.TODO(), name, pt, data, metav1.PatchOptions{}, subresources...)
}

type planCache struct {
	lister  listers.PlanLister
	indexer cache.Indexer
}

func (c *planCache) Get(namespace, name string) (*v1.Plan, error) {
	return c.lister.Plans(namespace).Get(name)
}

func (c *planCache) List(namespace string, selector labels.Selector) ([]*v1.Plan, error) {
	return c.lister.Plans(namespace).List(selector)
}

func (c *planCache) AddIndexer(indexName string, indexer PlanIndexer) {
	utilruntime.Must(c.indexer.AddIndexers(map[string]cache.IndexFunc{
		indexName: func(obj interface{}) (strings []string, e error) {
			return indexer(obj.(*v1.Plan))
		},
	}))
}

func (c *planCache) GetByIndex(indexName, key string) (result []*v1.Plan, err error) {
	objs, err := c.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	result = make([]*v1.Plan, 0, len(objs))
	for _, obj := range objs {
		result = append(result, obj.(*v1.Plan))
	}
	return result, nil
}

type PlanStatusHandler func(obj *v1.Plan, status v1.PlanStatus) (v1.PlanStatus, error)

type PlanGeneratingHandler func(obj *v1.Plan, status v1.PlanStatus) ([]runtime.Object, v1.PlanStatus, error)

func RegisterPlanStatusHandler(ctx context.Context, controller PlanController, condition condition.Cond, name string, handler PlanStatusHandler) {
	statusHandler := &planStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, FromPlanHandlerToHandler(statusHandler.sync))
}

func RegisterPlanGeneratingHandler(ctx context.Context, controller PlanController, apply apply.Apply,
	condition condition.Cond, name string, handler PlanGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &planGeneratingHandler{
		PlanGeneratingHandler: handler,
		apply:                 apply,
		name:                  name,
		gvk:                   controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterPlanStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type planStatusHandler struct {
	client    PlanClient
	condition condition.Cond
	handler   PlanStatusHandler
}

func (a *planStatusHandler) sync(key string, obj *v1.Plan) (*v1.Plan, error) {
	if obj == nil {
		return obj, nil
	}

	origStatus := obj.Status.DeepCopy()
	obj = obj.DeepCopy()
	newStatus, err := a.handler(obj, obj.Status)
	if err != nil {
		// Revert to old status on error
		newStatus = *origStatus.DeepCopy()
	}

	if a.condition != "" {
		if errors.IsConflict(err) {
			a.condition.SetError(&newStatus, "", nil)
		} else {
			a.condition.SetError(&newStatus, "", err)
		}
	}
	if !equality.Semantic.DeepEqual(origStatus, &newStatus) {
		var newErr error
		obj.Status = newStatus
		obj, newErr = a.client.UpdateStatus(obj)
		if err == nil {
			err = newErr
		}
	}
	return obj, err
}

type planGeneratingHandler struct {
	PlanGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
}

func (a *planGeneratingHandler) Remove(key string, obj *v1.Plan) (*v1.Plan, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v1.Plan{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(a.gvk)

	return nil, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects()
}

func (a *planGeneratingHandler) Handle(obj *v1.Plan, status v1.PlanStatus) (v1.PlanStatus, error) {
	objs, newStatus, err := a.PlanGeneratingHandler(obj, status)
	if err != nil {
		return newStatus, err
	}

	return newStatus, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
}
