package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// item is value or our imported data in cache
type item struct {
	val string
	// Each value in the cache has an expiration date
	expiry int64
}

// map for caching data
type Cache struct {
	// mu = mutex
	mu *sync.RWMutex
	// map of strings
	items map[string]*item
	// clinet can change expiry time
	defaultExpiry time.Duration
	readOnly      int32
	// readonly is a flag
}

// a function for changing expiry time by client
func NewCache(ed time.Duration) *Cache {
	return &Cache{
		mu:            &sync.RWMutex{},
		items:         make(map[string]*item),
		defaultExpiry: ed,
	}
}

func NewCacheWithJanitor(ed time.Duration) *Cache {
	c := &Cache{
		mu:            &sync.RWMutex{},
		items:         make(map[string]*item),
		defaultExpiry: ed,
	}

	go c.janitor()

	return c
}

// set method

/*
for readonly part :
when struct of cache receives a signal, it should make a decision saves all caching data in hard disk as a file.
and in this time that it should save it , cache should be on ReadOnly mode that doesnt allow adding new data in cache.
*/

// importing data to cache = set
// for this work always we must check that ReadOnly = True or not.
/*
if it was in true mode means we are in readonly of cache and we cant import data to it.
for this work always we must check that ReadOnly = True or not.
if it was in true mode means we are in readonly of cache and we cant import data to it.
*/
func (c *Cache) Set(k, v string) {
	/*
			why do we use atomic ? a subset of sync package because we are in a mode
			that multi Goroutine want to access to memory concurrently , some of them just set data in it and one of them read it.
		     synchronizing must be abled.
			 load function make it safe(Safe means that if there are several goroutines reading the same part of the memory at the same time, there will be no interference.)

	*/
	if atomic.LoadInt32(&c.readOnly) == 1 {
		return
	}

	// when we set data in memory , lock it safty(read section and set section)

	// Item is real place for storage cached data. we use mutex for providing errors(related to concurrency) while multi goroutines are setting data in cache...lock mutex , save data and then unlock mutex
	c.mu.Lock()
	defer c.mu.Unlock()

	// expire data after a time
	c.items[k] = &item{
		val:    v,
		expiry: time.Now().Add(time.Duration(c.defaultExpiry)).UnixNano(),
	}
}

// delete expired cached data
func (c *Cache) GetOrDelete(k string) (string, bool) {
	// it need be locked
	c.mu.RLock()
	// value is available?
	v, ok := c.items[k]
	if !ok {
		c.mu.RUnlock()
		return "", false
	}
	// compare now time to expiry time , if it is further unlock(we are not in set data in memory!) and delete
	if time.Now().UnixNano() > v.expiry {
		c.mu.RUnlock()
		c.delete(k)
		return "", false
	}

	// return value for client if it not expired
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
	// Copy cache items on disk
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

func (c *Cache) janitor() {
	for {
		<-time.After(c.defaultExpiry * 2)
		c.cleanup()
	}
}

func main() {

	c := NewCacheWithJanitor(time.Millisecond * 20)

	start := time.Now()

	ch := make(chan bool)

	fmt.Println("Start writing to the cache")
	go writeRand(c, ch)
	<-time.After(time.Millisecond)
	fmt.Println("Start reading from the cache")
	go readRand(c, ch)

	<-ch
	<-ch

	fmt.Printf("%d items remained in the cache. \n", len(c.items))
	fmt.Printf("Total exec time: %d milisecond. \n", time.Now().Sub(start).Milliseconds())
}

func writeRand(c *Cache, ch chan<- bool) {

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
			mu.RUnlock()
			c.Set(r, r)
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
