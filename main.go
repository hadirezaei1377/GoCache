package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// todo :
// add new features
// test
// review line by line in future

// Compression: Introducing compression techniques can be helpful for reducing the amount of memory consumed by the cache.

type item struct {
	val    string
	expiry int64
}

type Cache struct {
	mu            *sync.RWMutex
	items         map[string]*item
	defaultExpiry time.Duration
	readOnly      int32
}

func NewCache(ed time.Duration) *Cache {
	return &Cache{
		mu:            &sync.RWMutex{},
		items:         make(map[string]*item),
		defaultExpiry: ed,
	}
}

func NewCacheWithJanitor(ed time.Duration, maxItems int) *Cache {
	c := &Cache{
		mu:            &sync.RWMutex{},
		items:         make(map[string]*item),
		defaultExpiry: ed,
	}

	go c.janitor(maxItems)

	return c
}

func (c *Cache) Set(k, v string, maxItems int, expiry time.Duration) {
	if atomic.LoadInt32(&c.readOnly) == 1 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if the number of items in the cache exceeds the maximum limit.
	if len(c.items) >= maxItems {
		c.cleanup()
	}

	c.items[k] = &item{
		val:    v,
		expiry: time.Now().Add(expiry).UnixNano(),
	}
}

func (c *Cache) GetOrDelete(k string) (string, bool) {
	c.mu.RLock()
	v, ok := c.items[k]
	if !ok {
		c.mu.RUnlock()
		return "", false
	}
	if time.Now().UnixNano() > v.expiry {
		c.mu.RUnlock()
		c.delete(k)
		return "", false
	}

	c.mu.RUnlock()
	return v.val, true
}

func (c *Cache) Get(k string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.items[k]
	if !ok {
		return "", false
	}

	if time.Now().UnixNano() > v.expiry {
		return "", false
	}

	return v.val, true
}

func (c *Cache) delete(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, k)
}

func (c *Cache) SaveAndExit(k string) {
	atomic.AddInt32(&c.readOnly, 1)
}

func (c *Cache) cleanup() {
	c.mu.RLock()
	keys := []string{}

	for k, item := range c.items {
		if time.Now().UnixNano() > item.expiry {
			keys = append(keys, k)
		}
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range keys {
		delete(c.items, k)
	}
}

func (c *Cache) janitor(maxItems int) {
	for {
		<-time.After(c.defaultExpiry * 2)
		c.cleanup()
	}
}

func main() {
	maxItems := 10000 // Set the maximum number of items
	c := NewCacheWithJanitor(time.Millisecond*20, maxItems)
	start := time.Now()
	ch := make(chan bool)
	fmt.Println("Start writing to the cache")
	go writeRand(c, ch, maxItems)
	<-time.After(time.Millisecond)
	fmt.Println("Start reading from the cache")
	go readRand(c, ch)

	<-ch
	<-ch

	fmt.Printf("%d items remained in the cache. \n", len(c.items))
	fmt.Printf("Total exec time: %d milisecond. \n", time.Since(start).Milliseconds())

}

func writeRand(c *Cache, ch chan<- bool, maxItems int) {
	wg := new(sync.WaitGroup)
	seed := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(seed)
	mu := &sync.RWMutex{}
	n := 1000 * 1000

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			mu.RLock()
			r := fmt.Sprintf("%d", rnd.Intn(20*1000))
			m := time.Duration(rnd.Int63n(int64(5 * time.Minute)))
			mu.RUnlock()
			c.Set(r, r, maxItems, m)
			wg.Done()
		}()
	}

	wg.Wait()
	fmt.Println("Finished writing")
	ch <- true
}

func readRand(c *Cache, ch chan<- bool) {

	wg := new(sync.WaitGroup)

	seed := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(seed)
	mu := &sync.RWMutex{}

	n := 3 * 1000 * 1000
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			mu.RLock()
			r := fmt.Sprintf("%d", rnd.Intn(20*1000))
			mu.RUnlock()
			c.Get(r)
			wg.Done()
		}()
	}

	wg.Wait()
	fmt.Println("Finished reading")
	ch <- true
}
