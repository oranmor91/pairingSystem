package main

import (
	"fmt"
	"sync"
	"time"
)

// CacheItem represents an item in the LFU cache
type CacheItem struct {
	Key       string
	Value     []*Provider
	Frequency int
	Timestamp time.Time
}

// LFUCache implements a thread-safe Least Frequently Used cache
type LFUCache struct {
	capacity     int
	items        map[string]*CacheItem
	frequencies  map[int][]*CacheItem
	minFrequency int
	mu           sync.RWMutex
	hits         int64
	misses       int64
}

// NewLFUCache creates a new LFU cache with specified capacity
func NewLFUCache(capacity int) *LFUCache {
	if capacity <= 0 {
		capacity = 100 // Default capacity
	}

	return &LFUCache{
		capacity:     capacity,
		items:        make(map[string]*CacheItem),
		frequencies:  make(map[int][]*CacheItem),
		minFrequency: 1,
	}
}

// generateKey generates a cache key from policy and provider list
func (cache *LFUCache) generateKey(policy *ConsumerPolicy) string {
	return fmt.Sprintf("policy_%s_%d_%v",
		policy.RequiredLocation,
		policy.MinStake,
		policy.RequiredFeatures)
}

// updateFrequency updates the frequency of an item
func (cache *LFUCache) updateFrequency(item *CacheItem) {
	oldFreq := item.Frequency

	// Remove from old frequency list
	if freqList, exists := cache.frequencies[oldFreq]; exists {
		for i, cachedItem := range freqList {
			if cachedItem.Key == item.Key {
				cache.frequencies[oldFreq] = append(freqList[:i], freqList[i+1:]...)
				break
			}
		}

		// Update minimum frequency if necessary
		if oldFreq == cache.minFrequency && len(cache.frequencies[oldFreq]) == 0 {
			cache.minFrequency++
		}
	}

	// Increment frequency and update timestamp
	item.Frequency++
	item.Timestamp = time.Now()

	// Add to new frequency list
	cache.frequencies[item.Frequency] = append(cache.frequencies[item.Frequency], item)
}

// evict removes the least frequently used item
func (cache *LFUCache) evict() {
	// Find the least frequently used item
	if freqList, exists := cache.frequencies[cache.minFrequency]; exists && len(freqList) > 0 {
		// Remove the oldest item among items with minimum frequency
		var oldestItem *CacheItem
		var oldestIndex int

		for i, item := range freqList {
			if oldestItem == nil || item.Timestamp.Before(oldestItem.Timestamp) {
				oldestItem = item
				oldestIndex = i
			}
		}

		if oldestItem != nil {
			// Remove from frequency list
			cache.frequencies[cache.minFrequency] = append(
				freqList[:oldestIndex],
				freqList[oldestIndex+1:]...,
			)

			// Remove from main cache
			delete(cache.items, oldestItem.Key)

			// Update minimum frequency if necessary
			if len(cache.frequencies[cache.minFrequency]) == 0 {
				cache.minFrequency++
			}
		}
	}
}

// Get retrieves providers from cache for a given policy
func (cache *LFUCache) Get(policy *ConsumerPolicy) ([]*Provider, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	key := cache.generateKey(policy)

	if item, exists := cache.items[key]; exists {
		cache.hits++
		cache.updateFrequency(item)
		return item.Value, true
	}

	cache.misses++
	return nil, false
}

// Put stores providers in cache for a given policy
func (cache *LFUCache) Put(policy *ConsumerPolicy, providers []*Provider) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	key := cache.generateKey(policy)

	// Check if item already exists
	if existingItem, exists := cache.items[key]; exists {
		// Update existing item
		existingItem.Value = providers
		cache.updateFrequency(existingItem)
		return
	}

	// Check if we need to evict
	if len(cache.items) >= cache.capacity {
		cache.evict()
	}

	// Create new item
	newItem := &CacheItem{
		Key:       key,
		Value:     providers,
		Frequency: 1,
		Timestamp: time.Now(),
	}

	// Add to cache
	cache.items[key] = newItem
	cache.frequencies[1] = append(cache.frequencies[1], newItem)
	cache.minFrequency = 1
}

// Contains checks if a policy exists in the cache
func (cache *LFUCache) Contains(policy *ConsumerPolicy) bool {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	key := cache.generateKey(policy)
	_, exists := cache.items[key]
	return exists
}

// Remove removes a policy from the cache
func (cache *LFUCache) Remove(policy *ConsumerPolicy) bool {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	key := cache.generateKey(policy)

	if item, exists := cache.items[key]; exists {
		// Remove from frequency list
		freq := item.Frequency
		if freqList, freqExists := cache.frequencies[freq]; freqExists {
			for i, cachedItem := range freqList {
				if cachedItem.Key == key {
					cache.frequencies[freq] = append(freqList[:i], freqList[i+1:]...)
					break
				}
			}

			// Update minimum frequency if necessary
			if freq == cache.minFrequency && len(cache.frequencies[freq]) == 0 {
				cache.minFrequency++
			}
		}

		// Remove from main cache
		delete(cache.items, key)
		return true
	}

	return false
}

// Clear removes all items from the cache
func (cache *LFUCache) Clear() {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.items = make(map[string]*CacheItem)
	cache.frequencies = make(map[int][]*CacheItem)
	cache.minFrequency = 1
	cache.hits = 0
	cache.misses = 0
}

// Size returns the current number of items in the cache
func (cache *LFUCache) Size() int {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return len(cache.items)
}

// Capacity returns the maximum capacity of the cache
func (cache *LFUCache) Capacity() int {
	return cache.capacity
}

// GetStats returns cache statistics
func (cache *LFUCache) GetStats() map[string]interface{} {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	hitRate := float64(0)
	total := cache.hits + cache.misses
	if total > 0 {
		hitRate = float64(cache.hits) / float64(total)
	}

	return map[string]interface{}{
		"size":                   len(cache.items),
		"capacity":               cache.capacity,
		"hits":                   cache.hits,
		"misses":                 cache.misses,
		"hit_rate":               hitRate,
		"min_frequency":          cache.minFrequency,
		"frequency_distribution": cache.getFrequencyDistribution(),
	}
}

// getFrequencyDistribution returns distribution of frequencies
func (cache *LFUCache) getFrequencyDistribution() map[int]int {
	distribution := make(map[int]int)
	for freq, items := range cache.frequencies {
		distribution[freq] = len(items)
	}
	return distribution
}

// GetAllKeys returns all keys currently in the cache
func (cache *LFUCache) GetAllKeys() []string {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	keys := make([]string, 0, len(cache.items))
	for key := range cache.items {
		keys = append(keys, key)
	}
	return keys
}

// GetItemsByFrequency returns items with a specific frequency
func (cache *LFUCache) GetItemsByFrequency(frequency int) []*CacheItem {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	if items, exists := cache.frequencies[frequency]; exists {
		// Return a copy to avoid race conditions
		result := make([]*CacheItem, len(items))
		copy(result, items)
		return result
	}
	return []*CacheItem{}
}

// SetCapacity updates the cache capacity (may trigger evictions)
func (cache *LFUCache) SetCapacity(newCapacity int) {
	if newCapacity <= 0 {
		return
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.capacity = newCapacity

	// Evict items if necessary
	for len(cache.items) > cache.capacity {
		cache.evict()
	}
}
