package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ConcurrentPairingSystem manages the complete concurrent pairing system
type ConcurrentPairingSystem struct {
	// Core components
	providerStorage *ProviderStorage
	pairingSystem   *DefaultPairingSystem
	queueManager    *QueueManager

	// Concurrency control
	maxWorkers int
	workers    []*PairingWorker
	workerPool chan *PairingWorker

	// System state
	mu      sync.RWMutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc

	// Statistics
	totalProcessed int64
	totalErrors    int64
	startTime      time.Time

	// Results channel
	resultsChan     chan *ProcessorResult
	resultsBuffer   []*ProcessorResult
	resultsBufferMu sync.Mutex
}

// PairingWorker represents a worker that processes policies
type PairingWorker struct {
	id            int
	pairingSystem *DefaultPairingSystem
	processed     int64
	errors        int64
	mu            sync.RWMutex
}

// NewConcurrentPairingSystem creates a new concurrent pairing system
func NewConcurrentPairingSystem(providerStorage *ProviderStorage, maxWorkers int, queueCapacity int) *ConcurrentPairingSystem {
	if maxWorkers <= 0 {
		maxWorkers = 10 // Default number of workers
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ConcurrentPairingSystem{
		providerStorage: providerStorage,
		pairingSystem:   NewDefaultPairingSystem(providerStorage, 100),
		queueManager:    NewQueueManager(queueCapacity),
		maxWorkers:      maxWorkers,
		workers:         make([]*PairingWorker, 0, maxWorkers),
		workerPool:      make(chan *PairingWorker, maxWorkers),
		ctx:             ctx,
		cancel:          cancel,
		resultsChan:     make(chan *ProcessorResult, 1000),
		resultsBuffer:   make([]*ProcessorResult, 0),
	}
}

// NewPairingWorker creates a new pairing worker
func NewPairingWorker(id int, pairingSystem *DefaultPairingSystem) *PairingWorker {
	return &PairingWorker{
		id:            id,
		pairingSystem: pairingSystem,
	}
}

// ProcessPolicy processes a single policy
func (pw *PairingWorker) ProcessPolicy(policy *ConsumerPolicy) *ProcessorResult {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	// Validate policy before processing
	if policy == nil {
		pw.errors++
		return &ProcessorResult{
			Policy: nil,
			Error:  fmt.Errorf("policy cannot be nil"),
		}
	}

	result, err := pw.pairingSystem.GetPairingListWithDetails(nil, policy)
	if err != nil {
		pw.errors++
		return &ProcessorResult{
			Policy: policy,
			Error:  fmt.Errorf("failed to process policy: %w", err),
		}
	}

	pw.processed++
	return result
}

// GetStats returns worker statistics
func (pw *PairingWorker) GetStats() map[string]interface{} {
	pw.mu.RLock()
	defer pw.mu.RUnlock()

	return map[string]interface{}{
		"id":        pw.id,
		"processed": pw.processed,
		"errors":    pw.errors,
	}
}

// Start starts the concurrent pairing system
func (cps *ConcurrentPairingSystem) Start() error {
	cps.mu.Lock()
	defer cps.mu.Unlock()

	if cps.running {
		return fmt.Errorf("system is already running")
	}

	// Initialize workers
	for i := 0; i < cps.maxWorkers; i++ {
		worker := NewPairingWorker(i, cps.pairingSystem)
		cps.workers = append(cps.workers, worker)
		cps.workerPool <- worker
	}

	// Start processing goroutine
	go cps.processLoop()

	// Start results collector
	go cps.collectResults()

	// Add consumer with processing function
	cps.queueManager.AddConsumer(cps.processPolicyFromQueue)

	// Start queue system
	cps.queueManager.StartAll()

	cps.running = true
	cps.startTime = time.Now()

	return nil
}

// Stop stops the concurrent pairing system
func (cps *ConcurrentPairingSystem) Stop() {
	cps.mu.Lock()
	defer cps.mu.Unlock()

	if !cps.running {
		return
	}

	// Stop queue system
	cps.queueManager.StopAll()

	// Cancel context to stop all goroutines
	cps.cancel()

	// Close results channel
	close(cps.resultsChan)

	cps.running = false
}

// processLoop is the main processing loop
func (cps *ConcurrentPairingSystem) processLoop() {
	for {
		select {
		case <-cps.ctx.Done():
			return
		default:
			// This loop handles the worker pool and processing
			// The actual processing is handled by the queue consumer
		}
	}
}

// processPolicyFromQueue processes a policy from the queue
func (cps *ConcurrentPairingSystem) processPolicyFromQueue(policy *ConsumerPolicy) error {
	select {
	case <-cps.ctx.Done():
		return fmt.Errorf("system is shutting down")
	case worker := <-cps.workerPool:
		// Process policy in a goroutine
		go func() {
			defer func() {
				// Return worker to pool
				select {
				case cps.workerPool <- worker:
				case <-cps.ctx.Done():
					// System is shutting down, don't block
				}
			}()

			result := worker.ProcessPolicy(policy)

			// Send result to results channel
			select {
			case cps.resultsChan <- result:
			case <-cps.ctx.Done():
				return
			}

			// Update statistics
			cps.mu.Lock()
			cps.totalProcessed++
			if result.Error != nil {
				cps.totalErrors++
			}
			cps.mu.Unlock()
		}()

		return nil
	default:
		// If no worker is immediately available, wait with timeout
		select {
		case <-cps.ctx.Done():
			return fmt.Errorf("system is shutting down")
		case worker := <-cps.workerPool:
			// Process policy in a goroutine
			go func() {
				defer func() {
					// Return worker to pool
					select {
					case cps.workerPool <- worker:
					case <-cps.ctx.Done():
						// System is shutting down, don't block
					}
				}()

				result := worker.ProcessPolicy(policy)

				// Send result to results channel
				select {
				case cps.resultsChan <- result:
				case <-cps.ctx.Done():
					return
				}

				// Update statistics
				cps.mu.Lock()
				cps.totalProcessed++
				if result.Error != nil {
					cps.totalErrors++
				}
				cps.mu.Unlock()
			}()

			return nil
		case <-time.After(100 * time.Millisecond):
			return fmt.Errorf("no workers available after timeout")
		}
	}
}

// collectResults collects processing results
func (cps *ConcurrentPairingSystem) collectResults() {
	for {
		select {
		case <-cps.ctx.Done():
			return
		case result, ok := <-cps.resultsChan:
			if !ok {
				return
			}

			// Add result to buffer
			cps.resultsBufferMu.Lock()
			cps.resultsBuffer = append(cps.resultsBuffer, result)
			cps.resultsBufferMu.Unlock()
		}
	}
}

// EnqueuePolicy adds a policy to the processing queue
func (cps *ConcurrentPairingSystem) EnqueuePolicy(policy *ConsumerPolicy) error {
	if !cps.IsRunning() {
		return fmt.Errorf("system is not running")
	}

	producer := cps.queueManager.AddProducer()
	producer.Start()
	defer producer.Stop()

	return producer.ProducePolicy(policy)
}

// EnqueuePolicies adds multiple policies to the processing queue
func (cps *ConcurrentPairingSystem) EnqueuePolicies(policies []*ConsumerPolicy) error {
	if !cps.IsRunning() {
		return fmt.Errorf("system is not running")
	}

	producer := cps.queueManager.AddProducer()
	producer.Start()
	defer producer.Stop()

	return producer.ProducePoliciesBatch(policies)
}

// GetResults returns and clears the results buffer
func (cps *ConcurrentPairingSystem) GetResults() []*ProcessorResult {
	cps.resultsBufferMu.Lock()
	defer cps.resultsBufferMu.Unlock()

	results := cps.resultsBuffer
	cps.resultsBuffer = make([]*ProcessorResult, 0)
	return results
}

// GetResultsCount returns the current number of results in the buffer
func (cps *ConcurrentPairingSystem) GetResultsCount() int {
	cps.resultsBufferMu.Lock()
	defer cps.resultsBufferMu.Unlock()

	return len(cps.resultsBuffer)
}

// IsRunning returns whether the system is running
func (cps *ConcurrentPairingSystem) IsRunning() bool {
	cps.mu.RLock()
	defer cps.mu.RUnlock()

	return cps.running
}

// GetStats returns comprehensive system statistics
func (cps *ConcurrentPairingSystem) GetStats() map[string]interface{} {
	cps.mu.RLock()
	defer cps.mu.RUnlock()

	var uptime time.Duration
	if cps.running {
		uptime = time.Since(cps.startTime)
	}

	// Get worker statistics
	workerStats := make([]map[string]interface{}, len(cps.workers))
	for i, worker := range cps.workers {
		workerStats[i] = worker.GetStats()
	}

	return map[string]interface{}{
		"running":              cps.running,
		"max_workers":          cps.maxWorkers,
		"active_workers":       len(cps.workers),
		"total_processed":      cps.totalProcessed,
		"total_errors":         cps.totalErrors,
		"uptime":               uptime.String(),
		"results_buffer_size":  cps.GetResultsCount(),
		"worker_stats":         workerStats,
		"queue_stats":          cps.queueManager.GetStats(),
		"pairing_system_stats": cps.pairingSystem.GetStats(),
		"provider_count":       cps.providerStorage.GetProviderCount(),
	}
}

// GetQueueStats returns queue system statistics
func (cps *ConcurrentPairingSystem) GetQueueStats() map[string]interface{} {
	return cps.queueManager.GetStats()
}

// GetPairingSystemStats returns pairing system statistics
func (cps *ConcurrentPairingSystem) GetPairingSystemStats() map[string]interface{} {
	return cps.pairingSystem.GetStats()
}

// GetProviderCount returns the number of providers in storage
func (cps *ConcurrentPairingSystem) GetProviderCount() int {
	return cps.providerStorage.GetProviderCount()
}

// AddProvider adds a provider to the storage
func (cps *ConcurrentPairingSystem) AddProvider(provider *Provider) {
	cps.providerStorage.AddProvider(provider)
}

// GetProviderStorage returns the provider storage (for testing/debugging)
func (cps *ConcurrentPairingSystem) GetProviderStorage() *ProviderStorage {
	return cps.providerStorage
}

// GetPairingSystem returns the pairing system (for testing/debugging)
func (cps *ConcurrentPairingSystem) GetPairingSystem() *DefaultPairingSystem {
	return cps.pairingSystem
}

// GetQueueManager returns the queue manager (for testing/debugging)
func (cps *ConcurrentPairingSystem) GetQueueManager() *QueueManager {
	return cps.queueManager
}

// ProcessPolicyDirectly processes a policy directly without using the queue
func (cps *ConcurrentPairingSystem) ProcessPolicyDirectly(policy *ConsumerPolicy) (*ProcessorResult, error) {
	if !cps.IsRunning() {
		return nil, fmt.Errorf("system is not running")
	}

	return cps.pairingSystem.GetPairingListWithDetails(nil, policy)
}

// ProcessPoliciesDirectly processes multiple policies directly without using the queue
func (cps *ConcurrentPairingSystem) ProcessPoliciesDirectly(policies []*ConsumerPolicy) ([]*ProcessorResult, error) {
	if !cps.IsRunning() {
		return nil, fmt.Errorf("system is not running")
	}

	return cps.pairingSystem.GetPairingListConcurrently(nil, policies, cps.maxWorkers)
}

// ResetStats resets all statistics
func (cps *ConcurrentPairingSystem) ResetStats() {
	cps.mu.Lock()
	defer cps.mu.Unlock()

	cps.totalProcessed = 0
	cps.totalErrors = 0
	cps.startTime = time.Now()

	// Reset worker stats
	for _, worker := range cps.workers {
		worker.mu.Lock()
		worker.processed = 0
		worker.errors = 0
		worker.mu.Unlock()
	}

	// Reset pairing system stats
	cps.pairingSystem.ResetStats()

	// Clear results buffer
	cps.resultsBufferMu.Lock()
	cps.resultsBuffer = make([]*ProcessorResult, 0)
	cps.resultsBufferMu.Unlock()
}
