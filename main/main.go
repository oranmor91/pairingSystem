package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	fmt.Println("=== Lava Network Provider Pairing System ===")
	fmt.Println("🚀 Starting continuous pairing system...")
	fmt.Println("   Press Ctrl+C to stop gracefully")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Initialize fake data generator
	generator := NewFakeDataGenerator()

	// Generate initial realistic workload
	fmt.Println("\n1. Generating initial workload...")
	providerStorage, initialPolicies := generator.GenerateRealisticWorkload(200, 0) // Start with 0 policies

	fmt.Printf("✓ Generated %d providers across %d locations\n",
		providerStorage.GetProviderCount(),
		len(providerStorage.GetLocationStats()))

	// Display initial provider distribution
	fmt.Println("\n2. Initial Provider Distribution by Location:")
	displayProviderDistribution(providerStorage)

	// Initialize pairing system
	fmt.Println("\n3. Initializing Pairing System...")
	pairingSystem := NewDefaultPairingSystem(providerStorage, 100)

	// Initialize queue-based processing system
	fmt.Println("\n4. Starting Continuous Queue-Based Processing...")
	queueSystem := NewConcurrentPairingSystem(providerStorage, 5, 100) // 5 workers, 100 queue capacity

	// Start the queue system
	err := queueSystem.Start()
	if err != nil {
		log.Fatalf("❌ Failed to start queue system: %v", err)
	}
	defer queueSystem.Stop()

	fmt.Println("✓ Queue system started with 5 workers")

	// Start continuous processing
	fmt.Println("\n5. Starting Continuous Processing...")

	// Channel to coordinate shutdown
	shutdownChan := make(chan struct{})

	// Start continuous policy generation
	go continuousPolicyGeneration(generator, queueSystem, shutdownChan)

	// Start continuous provider generation
	go continuousProviderGeneration(generator, providerStorage, shutdownChan)

	// Start periodic statistics display
	go periodicStatsDisplay(queueSystem, pairingSystem, providerStorage, shutdownChan)

	// Start periodic results display
	go periodicResultsDisplay(queueSystem, shutdownChan)

	// Initial policies to get started
	if len(initialPolicies) > 0 {
		err = queueSystem.EnqueuePolicies(initialPolicies[:min(10, len(initialPolicies))])
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to enqueue initial policies: %v\n", err)
		}
	}

	fmt.Println("\n🔄 System running continuously...")
	fmt.Println("   - Generating policies every 2 seconds")
	fmt.Println("   - Adding providers every 10 seconds")
	fmt.Println("   - Displaying stats every 30 seconds")
	fmt.Println("   - Displaying results every 15 seconds")

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n\n🛑 Shutdown signal received, stopping gracefully...")

	// Signal all goroutines to stop
	close(shutdownChan)

	// Give some time for cleanup
	time.Sleep(2 * time.Second)

	// Display final statistics
	fmt.Println("\n📊 Final Statistics:")
	displaySystemStats(pairingSystem)

	finalResults := queueSystem.GetResults()
	fmt.Printf("\n📋 Final Results Count: %d\n", len(finalResults))

	fmt.Println("\n✅ System stopped gracefully")
}

// continuousPolicyGeneration generates policies continuously
func continuousPolicyGeneration(generator *FakeDataGenerator, queueSystem *ConcurrentPairingSystem, shutdown <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	policyCount := 0

	for {
		select {
		case <-shutdown:
			fmt.Println("🔄 Policy generation stopped")
			return
		case <-ticker.C:
			// Generate 1-3 policies
			numPolicies := generator.rand.Intn(3) + 1
			policies := make([]*ConsumerPolicy, numPolicies)

			for i := 0; i < numPolicies; i++ {
				// 40% relaxed, 30% strict, 30% random
				rand := generator.rand.Float32()
				if rand < 0.4 {
					policies[i] = generator.GenerateRelaxedPolicy()
				} else if rand < 0.7 {
					policies[i] = generator.GenerateStrictPolicy()
				} else {
					policies[i] = generator.GenerateRandomPolicy()
				}
			}

			err := queueSystem.EnqueuePolicies(policies)
			if err != nil {
				fmt.Printf("⚠️  Warning: Failed to enqueue policies: %v\n", err)
			} else {
				policyCount += numPolicies
				if policyCount%10 == 0 {
					fmt.Printf("🔄 Generated %d policies so far...\n", policyCount)
				}
			}
		}
	}
}

// continuousProviderGeneration adds providers continuously
func continuousProviderGeneration(generator *FakeDataGenerator, storage *ProviderStorage, shutdown <-chan struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-shutdown:
			fmt.Println("🔄 Provider generation stopped")
			return
		case <-ticker.C:
			// Add 5-15 new providers
			numProviders := generator.rand.Intn(11) + 5

			for i := 0; i < numProviders; i++ {
				// 50% random, 30% high stake, 20% feature rich
				rand := generator.rand.Float32()
				var provider *Provider

				if rand < 0.5 {
					provider = generator.GenerateRandomProvider()
				} else if rand < 0.8 {
					provider = generator.GenerateRandomProvider()
					provider.Stake = int64(generator.rand.Intn(500000) + 500000) // High stake
				} else {
					provider = generator.GenerateRandomProvider()
					// Add more features
					allFeatures := generator.GetAvailableFeatures()
					numFeatures := generator.rand.Intn(10) + 10 // 10-20 features
					selectedFeatures := make([]string, 0, numFeatures)
					usedFeatures := make(map[string]bool)

					for _, existing := range provider.Features {
						usedFeatures[existing] = true
					}

					for len(selectedFeatures) < numFeatures && len(selectedFeatures) < len(allFeatures) {
						feature := allFeatures[generator.rand.Intn(len(allFeatures))]
						if !usedFeatures[feature] {
							selectedFeatures = append(selectedFeatures, feature)
							usedFeatures[feature] = true
						}
					}
					provider.Features = selectedFeatures
				}

				storage.AddProvider(provider)
			}

			fmt.Printf("🔄 Added %d new providers (Total: %d)\n", numProviders, storage.GetProviderCount())
		}
	}
}

// periodicStatsDisplay displays system statistics periodically
func periodicStatsDisplay(queueSystem *ConcurrentPairingSystem, pairingSystem *DefaultPairingSystem, storage *ProviderStorage, shutdown <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-shutdown:
			fmt.Println("🔄 Statistics display stopped")
			return
		case <-ticker.C:
			fmt.Println("\n" + strings.Repeat("=", 60))
			fmt.Printf("📊 PERIODIC STATISTICS - %s\n", time.Now().Format("15:04:05"))
			fmt.Println(strings.Repeat("=", 60))

			// Display provider distribution
			fmt.Println("\n🌍 Provider Distribution:")
			displayProviderDistribution(storage)

			// Display system stats
			fmt.Println("\n⚙️  System Statistics:")
			displaySystemStats(pairingSystem)

			// Display queue stats
			queueStats := queueSystem.GetQueueStats()
			fmt.Printf("\n📋 Queue Statistics: %v\n", queueStats)

			fmt.Println(strings.Repeat("=", 60))
		}
	}
}

// periodicResultsDisplay displays processing results periodically
func periodicResultsDisplay(queueSystem *ConcurrentPairingSystem, shutdown <-chan struct{}) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-shutdown:
			fmt.Println("🔄 Results display stopped")
			return
		case <-ticker.C:
			results := queueSystem.GetResults()
			if len(results) > 0 {
				fmt.Printf("\n📋 Processing Results (%d new results):\n", len(results))

				// Display only the first 3 results to avoid spam
				maxDisplay := min(3, len(results))
				for i := 0; i < maxDisplay; i++ {
					result := results[i]
					fmt.Printf("\n   📋 Result %d:\n", i+1)

					// Display Policy Information
					fmt.Printf("      Policy: %s, Features: %v, MinStake: %d\n",
						result.Policy.RequiredLocation,
						result.Policy.RequiredFeatures,
						result.Policy.MinStake)

					// Display Processing Statistics
					fmt.Printf("      Stats: %d filtered, %d ranked, %v processing time\n",
						result.FilteredCount,
						result.RankedCount,
						result.ProcessingTime)

					// Display Top Providers
					if len(result.TopProviders) > 0 {
						fmt.Printf("      Top Providers (%d found):\n", len(result.TopProviders))
						for j, provider := range result.TopProviders {
							fmt.Printf("        %d. %s (Stake: %d, Location: %s)\n",
								j+1, provider.Address, provider.Stake, provider.Location)
							if j >= 2 { // Show max 3 providers
								break
							}
						}
					} else {
						fmt.Printf("      ❌ No providers found\n")
					}

					// Display any errors
					if result.Error != nil {
						fmt.Printf("      ❌ Error: %v\n", result.Error)
					}
				}

				if len(results) > maxDisplay {
					fmt.Printf("   ... and %d more results\n", len(results)-maxDisplay)
				}
			}
		}
	}
}

// displayProviderDistribution displays provider distribution by location
func displayProviderDistribution(storage *ProviderStorage) {
	locationStats := storage.GetLocationStats()
	for location, count := range locationStats {
		fmt.Printf("   %s: %d providers\n", location, count)
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
