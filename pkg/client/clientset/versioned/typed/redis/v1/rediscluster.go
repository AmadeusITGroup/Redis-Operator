/*
Copyright 2017 Redis-Operator

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
package v1

import (
	v1 "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	scheme "github.com/amadeusitgroup/redis-operator/pkg/client/clientset/versioned/scheme"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// RedisClustersGetter has a method to return a RedisClusterInterface.
// A group's client should implement this interface.
type RedisClustersGetter interface {
	RedisClusters(namespace string) RedisClusterInterface
}

// RedisClusterInterface has methods to work with RedisCluster resources.
type RedisClusterInterface interface {
	Create(*v1.RedisCluster) (*v1.RedisCluster, error)
	Update(*v1.RedisCluster) (*v1.RedisCluster, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.RedisCluster, error)
	List(opts meta_v1.ListOptions) (*v1.RedisClusterList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.RedisCluster, err error)
	RedisClusterExpansion
}

// redisClusters implements RedisClusterInterface
type redisClusters struct {
	client rest.Interface
	ns     string
}

// newRedisClusters returns a RedisClusters
func newRedisClusters(c *RedisV1Client, namespace string) *redisClusters {
	return &redisClusters{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the redisCluster, and returns the corresponding redisCluster object, and an error if there is any.
func (c *redisClusters) Get(name string, options meta_v1.GetOptions) (result *v1.RedisCluster, err error) {
	result = &v1.RedisCluster{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("redisclusters").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of RedisClusters that match those selectors.
func (c *redisClusters) List(opts meta_v1.ListOptions) (result *v1.RedisClusterList, err error) {
	result = &v1.RedisClusterList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("redisclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested redisClusters.
func (c *redisClusters) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("redisclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a redisCluster and creates it.  Returns the server's representation of the redisCluster, and an error, if there is any.
func (c *redisClusters) Create(redisCluster *v1.RedisCluster) (result *v1.RedisCluster, err error) {
	result = &v1.RedisCluster{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("redisclusters").
		Body(redisCluster).
		Do().
		Into(result)
	return
}

// Update takes the representation of a redisCluster and updates it. Returns the server's representation of the redisCluster, and an error, if there is any.
func (c *redisClusters) Update(redisCluster *v1.RedisCluster) (result *v1.RedisCluster, err error) {
	result = &v1.RedisCluster{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("redisclusters").
		Name(redisCluster.Name).
		Body(redisCluster).
		Do().
		Into(result)
	return
}

// Delete takes name of the redisCluster and deletes it. Returns an error if one occurs.
func (c *redisClusters) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("redisclusters").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *redisClusters) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("redisclusters").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched redisCluster.
func (c *redisClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.RedisCluster, err error) {
	result = &v1.RedisCluster{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("redisclusters").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
