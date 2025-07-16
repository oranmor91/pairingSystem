package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	fmt.Println("=== Lava Network Provider Pairing System ===")
	fmt.Println("Initializing system components...")

	// Initialize fake data generator
	generator := NewFakeDataGenerator()

	// Generate realistic workload
	fmt.Println("\n1. Generating realistic workload...")
	providerStorage, policies := generator.GenerateRealisticWorkload(200, 50)

	fmt.Printf("✓ Generated %d providers across %d locations\n",
		providerStorage.GetProviderCount(),
		len(providerStorage.GetLocationStats()))
	fmt.Printf("✓ Generated %d consumer policies\n", len(policies))

	// Display provider distribution
	fmt.Println("\n2. Provider Distribution by Location:")
	locationStats := providerStorage.GetLocationStats()
	for location, count := range locationStats {
		fmt.Printf("   %s: %d providers\n", location, count)
	}

	// Initialize pairing system
	fmt.Println("\n3. Initializing Pairing System...")
	pairingSystem := NewDefaultPairingSystem(providerStorage, 50)

	// Test basic pairing functionality
	fmt.Println("\n4. Testing Basic Pairing Functionality...")
	testBasicPairing(pairingSystem, providerStorage, policies[:5])

	//Test concurrent processing
	fmt.Println("\n5. Testing Concurrent Processing...")
	testConcurrentProcessing(providerStorage, policies)

	//	Test queue-based processing
	fmt.Println("\n6. Testing Queue-Based Processing...")
	testQueueBasedProcessing(providerStorage, policies)

	// Display comprehensive statistics
	fmt.Println("\n7. System Statistics:")
	displaySystemStats(pairingSystem)

	fmt.Println("\n=== System Demonstration Complete ===")
}

func testBasicPairing(pairingSystem *DefaultPairingSystem, storage *ProviderStorage, policies []*ConsumerPolicy) {
	allProviders := storage.GetAllProviders()

	for i, policy := range policies {
		fmt.Printf("\n   Policy %d Requirements:\n", i+1)
		fmt.Printf("     Location: %s\n", policy.RequiredLocation)
		fmt.Printf("     Min Stake: %d\n", policy.MinStake)
		fmt.Printf("     Required Features: %v\n", policy.RequiredFeatures)

		// Get pairing list
		startTime := time.Now()
		pairingList, err := pairingSystem.GetPairingList(allProviders, policy)
		processingTime := time.Since(startTime)

		if err != nil {
			fmt.Printf("     ❌ Error: %v\n", err)
			continue
		}

		fmt.Printf("     ✓ Found %d matching providers (processed in %v)\n",
			len(pairingList), processingTime)

		// Display top providers
		for j, provider := range pairingList {
			fmt.Printf("       %d. %s (Stake: %d, Location: %s, Features: %d)\n",
				j+1, provider.Address, provider.Stake, provider.Location, len(provider.Features))
		}

		// Get detailed results
		result, err := pairingSystem.GetPairingListWithDetails(allProviders, policy)
		if err == nil {
			fmt.Printf("     📊 Filtered: %d → Ranked: %d → Top: %d\n",
				result.FilteredCount, result.RankedCount, len(result.TopProviders))
		}
	}
}

func testConcurrentProcessing(storage *ProviderStorage, policies []*ConsumerPolicy) {
	fmt.Println("   Creating concurrent pairing system...")

	// Create concurrent system
	concurrentSystem := NewConcurrentPairingSystem(storage, 5, 100)

	// Start the system
	err := concurrentSystem.Start()
	if err != nil {
		fmt.Printf("   ❌ Failed to start concurrent system: %v\n", err)
		return
	}
	defer concurrentSystem.Stop()

	fmt.Println("   ✓ Concurrent system started with 5 workers")

	// Process policies directly
	fmt.Println("   Processing policies directly...")
	startTime := time.Now()

	results, err := concurrentSystem.ProcessPoliciesDirectly(policies[:10])
	processingTime := time.Since(startTime)

	if err != nil {
		fmt.Printf("   ❌ Error processing policies: %v\n", err)
		return
	}

	fmt.Printf("   ✓ Processed %d policies in %v (avg: %v per policy)\n",
		len(results), processingTime, processingTime/time.Duration(len(results)))

	// Display results summary
	successCount := 0
	totalProviders := 0

	for _, result := range results {
		if result.Error == nil {
			successCount++
			totalProviders += len(result.TopProviders)
		}
	}

	fmt.Printf("   📊 Success rate: %d/%d (%.1f%%)\n",
		successCount, len(results), float64(successCount)/float64(len(results))*100)
	fmt.Printf("   📊 Average providers per policy: %.1f\n",
		float64(totalProviders)/float64(successCount))

	// Display system stats
	stats := concurrentSystem.GetStats()
	fmt.Printf("   📊 System stats: %d processed, %d errors\n",
		stats["total_processed"], stats["total_errors"])
}

func testQueueBasedProcessing(storage *ProviderStorage, policies []*ConsumerPolicy) {
	fmt.Println("   Creating queue-based processing system...")

	// Create concurrent system
	queueSystem := NewConcurrentPairingSystem(storage, 3, 50)

	// Start the system
	err := queueSystem.Start()
	if err != nil {
		fmt.Printf("   ❌ Failed to start queue system: %v\n", err)
		return
	}
	defer queueSystem.Stop()

	fmt.Println("   ✓ Queue system started with 3 workers")

	// Enqueue policies
	fmt.Println("   Enqueuing policies for processing...")
	startTime := time.Now()

	err = queueSystem.EnqueuePolicies(policies[:15])
	if err != nil {
		fmt.Printf("   ❌ Error enqueuing policies: %v\n", err)
		return
	}

	fmt.Printf("   ✓ Enqueued %d policies\n", 15)

	// Wait for processing
	fmt.Println("   Waiting for processing to complete...")
	time.Sleep(2 * time.Second)

	// Get results
	results := queueSystem.GetResults()
	processingTime := time.Since(startTime)

	fmt.Printf("   ✓ Retrieved %d results in %v\n", len(results), processingTime)

	// Display queue statistics
	queueStats := queueSystem.GetQueueStats()
	fmt.Printf("   📊 Queue stats: %v\n", queueStats)
}

func displaySystemStats(pairingSystem *DefaultPairingSystem) {
	stats := pairingSystem.GetStats()

	fmt.Printf("   Total Processed: %v\n", stats["total_processed"])
	fmt.Printf("   Total Filtered: %v\n", stats["total_filtered"])
	fmt.Printf("   Total Ranked: %v\n", stats["total_ranked"])
	fmt.Printf("   Cache Hits: %v\n", stats["total_cache_hits"])
	fmt.Printf("   Cache Misses: %v\n", stats["total_cache_misses"])
	fmt.Printf("   Average Process Time: %v\n", stats["average_process_time"])
	fmt.Printf("   Provider Count: %v\n", stats["provider_count"])
	fmt.Printf("   Caching Enabled: %v\n", stats["caching_enabled"])

	if cacheStats, exists := stats["cache_stats"]; exists {
		if cacheMap, ok := cacheStats.(map[string]interface{}); ok {
			fmt.Printf("   Cache Hit Rate: %.2f%%\n", cacheMap["hit_rate"].(float64)*100)
			fmt.Printf("   Cache Size: %v/%v\n", cacheMap["size"], cacheMap["capacity"])
		}
	}
}

// Additional demonstration functions

func demonstrateFiltering() {
	fmt.Println("\n=== Filtering Demonstration ===")

	generator := NewFakeDataGenerator()

	// Generate test data
	providers := generator.GenerateRandomProviders(20)
	policy := &ConsumerPolicy{
		RequiredLocation: "US-East-1",
		RequiredFeatures: []string{"HTTP", "HTTPS"},
		MinStake:         10000,
	}

	// Create filter chain
	filterChain := NewFilterChain()

	fmt.Printf("Initial providers: %d\n", len(providers))

	// Apply filters
	filtered := filterChain.ApplyFilters(providers, policy)

	fmt.Printf("After filtering: %d\n", len(filtered))

	// Show filter stats
	stats := filterChain.GetFilterStats(providers, policy)
	fmt.Printf("Filter stats: %v\n", stats)
}

func demonstrateScoring() {
	fmt.Println("\n=== Scoring Demonstration ===")

	generator := NewFakeDataGenerator()

	// Generate test data
	providers := generator.GenerateRandomProviders(10)
	policy := generator.GenerateRandomPolicy()

	// Create scoring system
	scoringSystem := NewScoringSystem()

	// Score providers
	scores := scoringSystem.RankProviders(providers, policy)

	fmt.Printf("Policy: Location=%s, MinStake=%d, Features=%v\n",
		policy.RequiredLocation, policy.MinStake, policy.RequiredFeatures)

	fmt.Println("Ranked providers:")
	for i, score := range scores {
		fmt.Printf("%d. %s (Score: %.3f) [Stake: %.3f, Location: %.3f, Features: %.3f]\n",
			i+1, score.Provider.Address, score.Score,
			score.Components["stake"], score.Components["location"], score.Components["features"])
	}
}

func demonstrateCaching() {
	fmt.Println("\n=== Caching Demonstration ===")

	cache := NewLFUCache(5)
	generator := NewFakeDataGenerator()

	// Generate test policies
	policies := generator.GenerateRandomPolicies(10)

	// Test cache operations
	for i, policy := range policies {
		providers := generator.GenerateRandomProviders(5)

		// Cache miss
		if cached, found := cache.Get(policy); found {
			fmt.Printf("Policy %d: Cache HIT (%d providers)\n", i+1, len(cached))
		} else {
			fmt.Printf("Policy %d: Cache MISS\n", i+1)
			cache.Put(policy, providers)
		}
	}

	// Display cache stats
	stats := cache.GetStats()
	fmt.Printf("Cache stats: %v\n", stats)
}

func init() {
	// Initialize logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
