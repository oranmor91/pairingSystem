package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"pairingSystem/internal/cache"
	"pairingSystem/internal/filters"
	"pairingSystem/internal/models"
	"pairingSystem/internal/storage"
)

// ProcessorResult represents the result of processing a policy
type ProcessorResult struct {
	Policy         *models.ConsumerPolicy
	FilteredCount  int
	RankedCount    int
	TopProviders   []*models.Provider
	ProcessingTime time.Duration
	Error          error
}

// Processor handles the filtering and ranking of providers
type Processor struct {
	providerStorage *storage.ProviderStorage
	filterChain     *filters.FilterChain
	scoringSystem   *ScoringSystem
	cache           *cache.LFUCache

	// Processing statistics
	mu                 sync.RWMutex
	totalProcessed     int64
	totalFiltered      int64
	totalRanked        int64
	totalCacheHits     int64
	totalCacheMisses   int64
	averageProcessTime time.Duration

	// Configuration
	topN          int
	enableCaching bool
	enableStats   bool
}

// NewProcessor creates a new processor instance
func NewProcessor(storage *storage.ProviderStorage, cacheCapacity int) *Processor {
	return &Processor{
		providerStorage: storage,
		filterChain:     filters.NewFilterChain(),
		scoringSystem:   NewScoringSystem(),
		cache:           cache.NewLFUCache(cacheCapacity),
		topN:            5,
		enableCaching:   true,
		enableStats:     true,
	}
}

// SetTopN sets the number of top providers to return
func (p *Processor) SetTopN(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if n > 0 {
		p.topN = n
	}
}

// SetCachingEnabled enables or disables caching
func (p *Processor) SetCachingEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.enableCaching = enabled
}

// SetStatsEnabled enables or disables statistics collection
func (p *Processor) SetStatsEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.enableStats = enabled
}

// FilterProviders filters providers based on policy requirements
func (p *Processor) FilterProviders(providers []*models.Provider, policy *models.ConsumerPolicy) []*models.Provider {
	if len(providers) == 0 {
		return providers
	}

	// Validate policy
	if err := filters.ValidatePolicy(policy); err != nil {
		return []*models.Provider{}
	}

	// Apply filters
	filtered := p.filterChain.ApplyFilters(providers, policy)

	// Update statistics
	if p.enableStats {
		p.mu.Lock()
		p.totalFiltered += int64(len(filtered))
		p.mu.Unlock()
	}

	return filtered
}

// RankProviders ranks providers based on policy preferences
func (p *Processor) RankProviders(providers []*models.Provider, policy *models.ConsumerPolicy) []*models.PairingScore {
	if len(providers) == 0 {
		return []*models.PairingScore{}
	}

	// Validate policy
	if err := filters.ValidatePolicy(policy); err != nil {
		return []*models.PairingScore{}
	}

	// Rank providers
	scores := p.scoringSystem.RankProviders(providers, policy)

	// Update statistics
	if p.enableStats {
		p.mu.Lock()
		p.totalRanked += int64(len(scores))
		p.mu.Unlock()
	}

	return scores
}

// ProcessPolicy processes a single policy through the complete pipeline
func (p *Processor) ProcessPolicy(policy *models.ConsumerPolicy) *ProcessorResult {
	startTime := time.Now()

	result := &ProcessorResult{
		Policy: policy,
	}

	// Check cache first
	if p.enableCaching {
		if cachedProviders, found := p.cache.Get(policy); found {
			result.TopProviders = cachedProviders
			result.ProcessingTime = time.Since(startTime)

			if p.enableStats {
				p.mu.Lock()
				p.totalCacheHits++
				p.mu.Unlock()
			}

			return result
		}

		if p.enableStats {
			p.mu.Lock()
			p.totalCacheMisses++
			p.mu.Unlock()
		}
	}

	// Get all providers
	allProviders := p.providerStorage.GetAllProviders()

	// Step 1: Filter providers
	filteredProviders := p.FilterProviders(allProviders, policy)
	result.FilteredCount = len(filteredProviders)

	if len(filteredProviders) == 0 {
		result.ProcessingTime = time.Since(startTime)
		return result
	}

	// Step 2: Rank providers
	rankedScores := p.RankProviders(filteredProviders, policy)
	result.RankedCount = len(rankedScores)

	// Step 3: Get top N providers
	topN := p.topN
	if topN > len(rankedScores) {
		topN = len(rankedScores)
	}

	result.TopProviders = make([]*models.Provider, topN)
	for i := 0; i < topN; i++ {
		result.TopProviders[i] = rankedScores[i].Provider
	}

	// Cache the result
	if p.enableCaching {
		p.cache.Put(policy, result.TopProviders)
	}

	result.ProcessingTime = time.Since(startTime)

	// Update statistics
	if p.enableStats {
		p.mu.Lock()
		p.totalProcessed++
		p.updateAverageProcessTime(result.ProcessingTime)
		p.mu.Unlock()
	}

	return result
}

// ProcessPoliciesConcurrently processes multiple policies concurrently
func (p *Processor) ProcessPoliciesConcurrently(policies []*models.ConsumerPolicy, maxConcurrency int) []*ProcessorResult {
	if len(policies) == 0 {
		return []*ProcessorResult{}
	}

	if maxConcurrency <= 0 {
		maxConcurrency = 10 // Default concurrency
	}

	// Create channels for work distribution
	policyChannel := make(chan *models.ConsumerPolicy, len(policies))
	resultChannel := make(chan *ProcessorResult, len(policies))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for policy := range policyChannel {
				result := p.ProcessPolicy(policy)
				resultChannel <- result
			}
		}()
	}

	// Send policies to workers
	for _, policy := range policies {
		policyChannel <- policy
	}
	close(policyChannel)

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultChannel)
	}()

	// Collect results
	results := make([]*ProcessorResult, 0, len(policies))
	for result := range resultChannel {
		results = append(results, result)
	}

	return results
}

// ProcessPoliciesWithContext processes policies with context for cancellation
func (p *Processor) ProcessPoliciesWithContext(ctx context.Context, policies []*models.ConsumerPolicy, maxConcurrency int) []*ProcessorResult {
	if len(policies) == 0 {
		return []*ProcessorResult{}
	}

	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}

	// Create channels
	policyChannel := make(chan *models.ConsumerPolicy, len(policies))
	resultChannel := make(chan *ProcessorResult, len(policies))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case policy, ok := <-policyChannel:
					if !ok {
						return
					}
					result := p.ProcessPolicy(policy)
					select {
					case resultChannel <- result:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	// Send policies to workers
	go func() {
		defer close(policyChannel)
		for _, policy := range policies {
			select {
			case policyChannel <- policy:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for completion or cancellation
	go func() {
		wg.Wait()
		close(resultChannel)
	}()

	// Collect results
	results := make([]*ProcessorResult, 0)
	for {
		select {
		case <-ctx.Done():
			return results
		case result, ok := <-resultChannel:
			if !ok {
				return results
			}
			results = append(results, result)
		}
	}
}

// updateAverageProcessTime updates the running average of processing time
func (p *Processor) updateAverageProcessTime(newTime time.Duration) {
	if p.totalProcessed == 1 {
		p.averageProcessTime = newTime
	} else {
		// Calculate running average
		p.averageProcessTime = time.Duration(
			(int64(p.averageProcessTime)*(p.totalProcessed-1) + int64(newTime)) / p.totalProcessed,
		)
	}
}

// GetStats returns comprehensive processor statistics
func (p *Processor) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := map[string]interface{}{
		"total_processed":      p.totalProcessed,
		"total_filtered":       p.totalFiltered,
		"total_ranked":         p.totalRanked,
		"total_cache_hits":     p.totalCacheHits,
		"total_cache_misses":   p.totalCacheMisses,
		"average_process_time": p.averageProcessTime.String(),
		"top_n":                p.topN,
		"caching_enabled":      p.enableCaching,
		"stats_enabled":        p.enableStats,
	}

	// Add cache stats
	if p.enableCaching {
		stats["cache_stats"] = p.cache.GetStats()
	}

	// Add provider storage stats
	stats["provider_count"] = p.providerStorage.GetProviderCount()
	stats["location_distribution"] = p.providerStorage.GetLocationStats()

	return stats
}

// GetFilterStats returns detailed filtering statistics
func (p *Processor) GetFilterStats(policy *models.ConsumerPolicy) map[string]interface{} {
	allProviders := p.providerStorage.GetAllProviders()
	stats := p.filterChain.GetFilterStats(allProviders, policy)

	// Convert map[string]int to map[string]interface{}
	result := make(map[string]interface{})
	for k, v := range stats {
		result[k] = v
	}
	return result
}

// GetScoringStats returns detailed scoring statistics
func (p *Processor) GetScoringStats(policy *models.ConsumerPolicy) map[string]interface{} {
	allProviders := p.providerStorage.GetAllProviders()
	filteredProviders := p.FilterProviders(allProviders, policy)
	return p.scoringSystem.GetScoringStats(filteredProviders, policy)
}

// ResetStats resets all statistics
func (p *Processor) ResetStats() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.totalProcessed = 0
	p.totalFiltered = 0
	p.totalRanked = 0
	p.totalCacheHits = 0
	p.totalCacheMisses = 0
	p.averageProcessTime = 0

	if p.enableCaching {
		p.cache.Clear()
	}

	p.scoringSystem.ResetNormalization()
}

// ValidateConfiguration validates the processor configuration
func (p *Processor) ValidateConfiguration() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.providerStorage == nil {
		return fmt.Errorf("provider storage is nil")
	}

	if p.filterChain == nil {
		return fmt.Errorf("filter chain is nil")
	}

	if p.scoringSystem == nil {
		return fmt.Errorf("scoring system is nil")
	}

	if p.enableCaching && p.cache == nil {
		return fmt.Errorf("cache is nil but caching is enabled")
	}

	if p.topN <= 0 {
		return fmt.Errorf("topN must be positive")
	}

	return nil
}

// UpdateScoringWeights updates the scoring system weights
func (p *Processor) UpdateScoringWeights(stake, location, feature float64) error {
	return p.scoringSystem.SetWeights(stake, location, feature)
}

// GetCache returns the processor's cache (for testing/debugging)
func (p *Processor) GetCache() *cache.LFUCache {
	return p.cache
}

// GetProviderStorage returns the processor's provider storage (for testing/debugging)
func (p *Processor) GetProviderStorage() *storage.ProviderStorage {
	return p.providerStorage
}
