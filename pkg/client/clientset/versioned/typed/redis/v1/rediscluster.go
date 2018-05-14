/*
MIT License

Copyright (c) 2018 Amadeus s.a.s.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
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
	UpdateStatus(*v1.RedisCluster) (*v1.RedisCluster, error)
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
func newRedisClusters(c *RedisoperatorV1Client, namespace string) *redisClusters {
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

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *redisClusters) UpdateStatus(redisCluster *v1.RedisCluster) (result *v1.RedisCluster, err error) {
	result = &v1.RedisCluster{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("redisclusters").
		Name(redisCluster.Name).
		SubResource("status").
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
