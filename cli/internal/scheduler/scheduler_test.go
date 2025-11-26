package scheduler

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNewScheduler(t *testing.T) {
	s := NewScheduler()
	if s == nil {
		t.Fatal("scheduler is nil")
	}
	if s.IsRunning() {
		t.Error("scheduler should not be running initially")
	}
	if s.JobCount() != 0 {
		t.Error("scheduler should have no jobs initially")
	}
}

func TestScheduleJob(t *testing.T) {
	s := NewScheduler()

	action := func() {}

	if err := s.ScheduleJob("test-job", 1, action); err != nil {
		t.Fatalf("failed to schedule job: %v", err)
	}

	if s.JobCount() != 1 {
		t.Errorf("expected 1 job, got %d", s.JobCount())
	}
}

func TestRemoveJob(t *testing.T) {
	s := NewScheduler()

	action := func() {}

	s.ScheduleJob("test-job", 1, action)
	if s.JobCount() != 1 {
		t.Fatalf("expected 1 job after scheduling")
	}

	s.RemoveJob("test-job")
	if s.JobCount() != 0 {
		t.Errorf("expected 0 jobs after removal, got %d", s.JobCount())
	}
}

func TestStartStop(t *testing.T) {
	s := NewScheduler()

	if s.IsRunning() {
		t.Error("should not be running initially")
	}

	s.Start()
	if !s.IsRunning() {
		t.Error("should be running after Start")
	}

	// Start again should be no-op
	s.Start()
	if !s.IsRunning() {
		t.Error("should still be running after second Start")
	}

	s.Stop()
	if s.IsRunning() {
		t.Error("should not be running after Stop")
	}
}

func TestGetNextRun(t *testing.T) {
	s := NewScheduler()

	// No job scheduled
	if next := s.GetNextRun("nonexistent"); next != nil {
		t.Error("expected nil for nonexistent job")
	}

	action := func() {}
	s.ScheduleJob("test-job", 1, action)
	s.Start()
	defer s.Stop()

	next := s.GetNextRun("test-job")
	if next == nil {
		t.Error("expected next run time for scheduled job")
	} else if next.Before(time.Now()) {
		t.Error("next run time should be in the future")
	}
}

func TestRunJobNow(t *testing.T) {
	s := NewScheduler()

	var counter int32 = 0
	action := func() { atomic.AddInt32(&counter, 1) }

	s.RunJobNow("immediate-job", action)

	// Small delay to ensure action executed
	time.Sleep(10 * time.Millisecond)

	if atomic.LoadInt32(&counter) != 1 {
		t.Errorf("expected counter to be 1, got %d", counter)
	}
}

func TestScheduleJobWithMinutes(t *testing.T) {
	s := NewScheduler()

	action := func() {}

	if err := s.ScheduleJobWithMinutes("minute-job", 30, action); err != nil {
		t.Fatalf("failed to schedule job: %v", err)
	}

	if s.JobCount() != 1 {
		t.Errorf("expected 1 job, got %d", s.JobCount())
	}
}

func TestReplaceExistingJob(t *testing.T) {
	s := NewScheduler()

	var counter int32 = 0
	action1 := func() { atomic.AddInt32(&counter, 1) }
	action2 := func() { atomic.AddInt32(&counter, 10) }

	// Schedule first job
	s.ScheduleJob("replace-job", 1, action1)
	if s.JobCount() != 1 {
		t.Error("expected 1 job")
	}

	// Schedule second job with same ID (should replace)
	s.ScheduleJob("replace-job", 2, action2)
	if s.JobCount() != 1 {
		t.Error("still expected 1 job after replacement")
	}
}
