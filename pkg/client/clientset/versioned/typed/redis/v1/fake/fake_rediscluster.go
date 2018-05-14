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
package fake

import (
	redis_v1 "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeRedisClusters implements RedisClusterInterface
type FakeRedisClusters struct {
	Fake *FakeRedisoperatorV1
	ns   string
}

var redisclustersResource = schema.GroupVersionResource{Group: "redisoperator.k8s.io", Version: "v1", Resource: "redisclusters"}

var redisclustersKind = schema.GroupVersionKind{Group: "redisoperator.k8s.io", Version: "v1", Kind: "RedisCluster"}

// Get takes name of the redisCluster, and returns the corresponding redisCluster object, and an error if there is any.
func (c *FakeRedisClusters) Get(name string, options v1.GetOptions) (result *redis_v1.RedisCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(redisclustersResource, c.ns, name), &redis_v1.RedisCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*redis_v1.RedisCluster), err
}

// List takes label and field selectors, and returns the list of RedisClusters that match those selectors.
func (c *FakeRedisClusters) List(opts v1.ListOptions) (result *redis_v1.RedisClusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(redisclustersResource, redisclustersKind, c.ns, opts), &redis_v1.RedisClusterList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &redis_v1.RedisClusterList{}
	for _, item := range obj.(*redis_v1.RedisClusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested redisClusters.
func (c *FakeRedisClusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(redisclustersResource, c.ns, opts))

}

// Create takes the representation of a redisCluster and creates it.  Returns the server's representation of the redisCluster, and an error, if there is any.
func (c *FakeRedisClusters) Create(redisCluster *redis_v1.RedisCluster) (result *redis_v1.RedisCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(redisclustersResource, c.ns, redisCluster), &redis_v1.RedisCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*redis_v1.RedisCluster), err
}

// Update takes the representation of a redisCluster and updates it. Returns the server's representation of the redisCluster, and an error, if there is any.
func (c *FakeRedisClusters) Update(redisCluster *redis_v1.RedisCluster) (result *redis_v1.RedisCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(redisclustersResource, c.ns, redisCluster), &redis_v1.RedisCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*redis_v1.RedisCluster), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeRedisClusters) UpdateStatus(redisCluster *redis_v1.RedisCluster) (*redis_v1.RedisCluster, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(redisclustersResource, "status", c.ns, redisCluster), &redis_v1.RedisCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*redis_v1.RedisCluster), err
}

// Delete takes name of the redisCluster and deletes it. Returns an error if one occurs.
func (c *FakeRedisClusters) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(redisclustersResource, c.ns, name), &redis_v1.RedisCluster{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeRedisClusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(redisclustersResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &redis_v1.RedisClusterList{})
	return err
}

// Patch applies the patch and returns the patched redisCluster.
func (c *FakeRedisClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *redis_v1.RedisCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(redisclustersResource, c.ns, name, data, subresources...), &redis_v1.RedisCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*redis_v1.RedisCluster), err
}
