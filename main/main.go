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

	//	Test queue-based processing
	fmt.Println("\n4. Testing Queue-Based Processing...")
	testQueueBasedProcessing(providerStorage, policies)

	// Display comprehensive statistics
	fmt.Println("\n5. System Statistics:")
	displaySystemStats(pairingSystem)

	fmt.Println("\n=== System Demonstration Complete ===")
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

	// Print each result with clear formatting
	fmt.Println("   Results:")
	for i, result := range results {
		fmt.Printf("\n   📋 Result %d:\n", i+1)

		// Display Policy Information
		fmt.Printf("      Policy Details:\n")
		fmt.Printf("        Required Location: %s\n", result.Policy.RequiredLocation)
		fmt.Printf("        Required Features: %v\n", result.Policy.RequiredFeatures)
		fmt.Printf("        Min Stake: %d\n", result.Policy.MinStake)

		// Display Processing Statistics
		fmt.Printf("      Processing Stats:\n")
		fmt.Printf("        Filtered Count: %d\n", result.FilteredCount)
		fmt.Printf("        Ranked Count: %d\n", result.RankedCount)
		fmt.Printf("        Processing Time: %v\n", result.ProcessingTime)

		// Display Top Providers
		fmt.Printf("      Top Providers (%d found):\n", len(result.TopProviders))
		if len(result.TopProviders) == 0 {
			fmt.Printf("        ❌ No providers found matching criteria\n")
		} else {
			for j, provider := range result.TopProviders {
				fmt.Printf("        %d. Address: %s\n", j+1, provider.Address)
				fmt.Printf("           Stake: %d\n", provider.Stake)
				fmt.Printf("           Location: %s\n", provider.Location)
				fmt.Printf("           Features: %v\n", provider.Features)
				if j < len(result.TopProviders)-1 {
					fmt.Printf("           ---\n")
				}
			}
		}

		// Display any errors
		if result.Error != nil {
			fmt.Printf("      ❌ Error: %v\n", result.Error)
		}
	}

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

func init() {
	// Initialize logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
