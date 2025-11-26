// Package scheduler provides periodic task scheduling for the Drata Agent CLI.
package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler manages periodic tasks.
type Scheduler struct {
	cron    *cron.Cron
	jobs    map[string]cron.EntryID
	mu      sync.RWMutex
	running bool
}

// NewScheduler creates a new scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		cron: cron.New(cron.WithSeconds()),
		jobs: make(map[string]cron.EntryID),
	}
}

// ScheduleJob schedules a job to run at a specified interval.
func (s *Scheduler) ScheduleJob(id string, intervalHours int, action func()) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing job with same ID if it exists
	if entryID, exists := s.jobs[id]; exists {
		s.cron.Remove(entryID)
	}

	// Create cron expression for hourly interval
	// Run every N hours at the start of the hour
	cronExpr := fmt.Sprintf("0 0 */%d * * *", intervalHours)

	entryID, err := s.cron.AddFunc(cronExpr, action)
	if err != nil {
		return fmt.Errorf("failed to schedule job %s: %w", id, err)
	}

	s.jobs[id] = entryID
	log.Printf("Scheduled job '%s' to run every %d hours", id, intervalHours)

	return nil
}

// ScheduleJobWithMinutes schedules a job to run at a specified interval in minutes.
func (s *Scheduler) ScheduleJobWithMinutes(id string, intervalMinutes int, action func()) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing job with same ID if it exists
	if entryID, exists := s.jobs[id]; exists {
		s.cron.Remove(entryID)
	}

	// Create cron expression for minute interval
	cronExpr := fmt.Sprintf("0 */%d * * * *", intervalMinutes)

	entryID, err := s.cron.AddFunc(cronExpr, action)
	if err != nil {
		return fmt.Errorf("failed to schedule job %s: %w", id, err)
	}

	s.jobs[id] = entryID
	log.Printf("Scheduled job '%s' to run every %d minutes", id, intervalMinutes)

	return nil
}

// RemoveJob removes a scheduled job.
func (s *Scheduler) RemoveJob(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.jobs[id]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, id)
		log.Printf("Removed job '%s'", id)
	}
}

// Start starts the scheduler.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		s.cron.Start()
		s.running = true
		log.Println("Scheduler started")
	}
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		ctx := s.cron.Stop()
		s.running = false
		log.Println("Scheduler stopped")
		return ctx
	}

	return context.Background()
}

// RunJobNow runs a job immediately in addition to its scheduled runs.
func (s *Scheduler) RunJobNow(id string, action func()) {
	log.Printf("Running job '%s' immediately", id)
	action()
}

// GetNextRun returns the next scheduled run time for a job.
func (s *Scheduler) GetNextRun(id string) *time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if entryID, exists := s.jobs[id]; exists {
		entry := s.cron.Entry(entryID)
		if entry.ID != 0 {
			return &entry.Next
		}
	}

	return nil
}

// IsRunning returns true if the scheduler is running.
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// JobCount returns the number of scheduled jobs.
func (s *Scheduler) JobCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.jobs)
}
