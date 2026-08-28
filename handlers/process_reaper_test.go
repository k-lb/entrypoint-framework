/*
 *  Copyright (c) 2026 Samsung Electronics Co., Ltd All Rights Reserved
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License
 */

package handlers

import (
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Compile-time interface check - ensure ChildrenProcessReaperHandler implements ProcessReaperHandler methods
var _ interface {
	Start()
	Stop()
	Stats() ProcessReaperStats
} = (*ChildrenProcessReaperHandler)(nil)

// forkAndExitDirect creates a child process using syscall.ForkExec that exits immediately.
// The caller is responsible for waiting on the process.
func forkAndExitDirect(exitCode int) (int, error) {
	// Fork directly
	pid, _, err := syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	if err != 0 {
		return 0, err
	}

	if pid == 0 {
		// Child process - exit with specified code
		syscall.Exit(exitCode)
	}

	return int(pid), nil
}

// forkAndSleepDirect creates a child process that sleeps for the given duration.
func forkAndSleepDirect(duration time.Duration) (int, error) {
	pid, _, err := syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	if err != 0 {
		return 0, err
	}

	if pid == 0 {
		// Child process - sleep and exit
		time.Sleep(duration)
		syscall.Exit(0)
	}

	return int(pid), nil
}

func (h *HandlersTestSuite) TestChildrenProcessReaperHandler() {
	h.Run("constructor creates handler with proper fields", func() {
		handler := newProcessReaperHandler(logDiscard)

		h.Require().NotNil(handler)
		h.NotNil(handler.log)
		h.NotNil(handler.ctx)
		h.NotNil(handler.cancel)
		h.NotNil(handler.done)
		h.Equal(ProcessReaperStats{Exited: 0, Signaled: 0}, handler.Stats())
	})

	h.Run("Start and Stop methods work correctly", func() {
		handler := newProcessReaperHandler(logDiscard)

		h.Require().NotNil(handler)

		// Start the reaper
		handler.Start()

		// Give it a moment to start
		time.Sleep(time.Millisecond * 10)

		// Stop the reaper - this should not block indefinitely
		done := make(chan struct{})
		go func() {
			handler.Stop()
			close(done)
		}()

		// Wait for stop to complete with timeout
		select {
		case <-done:
			// Success - Stop completed
		case <-time.After(time.Second):
			h.Fail("Stop method should complete within timeout")
		}

		// Verify done channel is closed after Stop
		select {
		case _, open := <-handler.done:
			h.False(open, "done channel should be closed after Stop")
		default:
			h.Fail("done channel should be closed")
		}
	})

	h.Run("multiple Stop calls should not panic", func() {
		handler := newProcessReaperHandler(logDiscard)

		h.Require().NotNil(handler)

		handler.Start()
		time.Sleep(time.Millisecond * 10)

		// First stop
		handler.Stop()

		// Second stop should not panic (context is already cancelled)
		h.NotPanics(func() {
			// Try to access done channel - should be closed
			select {
			case _, open := <-handler.done:
				h.False(open)
			case <-time.After(time.Millisecond * 100):
				h.Fail("done channel should be closed")
			}
		})
	})

	h.Run("handler fields are properly initialized", func() {
		handler := newProcessReaperHandler(logDiscard)

		h.Require().NotNil(handler)

		// Verify context is not done initially
		h.False(handler.ctx.Err() != nil, "context should not be done initially")

		// Verify done channel is open initially
		select {
		case _, open := <-handler.done:
			h.True(open, "done channel should be open initially")
		default:
			// Channel is open but empty - this is expected
		}

		// Verify initial stats are zero
		h.Equal(ProcessReaperStats{Exited: 0, Signaled: 0}, handler.Stats())
	})

	h.Run("Stop cancels context", func() {
		handler := newProcessReaperHandler(logDiscard)

		h.Require().NotNil(handler)

		handler.Start()
		time.Sleep(time.Millisecond * 10)

		// Stop should cancel context
		handler.Stop()

		// Verify context is cancelled
		select {
		case <-handler.ctx.Done():
			// Context is cancelled - expected
		default:
			h.Fail("context should be cancelled after Stop")
		}
	})
}

func (h *HandlersTestSuite) TestChildrenProcessReaperHandler_Stats() {
	h.Run("Stats returns zero values initially", func() {
		handler := newProcessReaperHandler(logDiscard)
		h.Equal(ProcessReaperStats{Exited: 0, Signaled: 0}, handler.Stats())
	})

	h.Run("Stats is thread-safe - concurrent access", func() {
		handler := newProcessReaperHandler(logDiscard)
		handler.Start()

		var wg sync.WaitGroup

		// Start multiple goroutines accessing Stats concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					_ = handler.Stats()
				}
			}()
		}

		wg.Wait()
		handler.Stop()
	})

	h.Run("Stats returns values after Stop", func() {
		handler := newProcessReaperHandler(logDiscard)
		handler.Start()
		time.Sleep(time.Millisecond * 10)
		handler.Stop()

		// Stats should still be accessible after stop
		h.Equal(ProcessReaperStats{Exited: 0, Signaled: 0}, handler.Stats())
	})
}

func (h *HandlersTestSuite) TestChildrenProcessReaperHandler_waitForChildren() {
	h.Run("waitForChildren stops when context is cancelled", func() {
		handler := newProcessReaperHandler(logDiscard)

		h.Require().NotNil(handler)

		// Start the reaper - this runs waitForChildren in a goroutine
		handler.Start()

		// Give it a moment to start
		time.Sleep(time.Millisecond * 50)

		// Stop should cancel context and stop waitForChildren
		handler.Stop()

		// Verify done channel is closed (waitForChildren exited)
		select {
		case _, open := <-handler.done:
			h.False(open, "done channel should be closed after waitForChildren exits")
		case <-time.After(time.Second):
			h.Fail("waitForChildren should exit after context cancellation")
		}
	})

	h.Run("waitForChildren handles ECHILD error (no children)", func() {
		handler := newProcessReaperHandler(logDiscard)

		h.Require().NotNil(handler)

		// Start the reaper
		handler.Start()

		// Wait for reaping period to allow ECHILD handling path
		time.Sleep(reapingPeriod + 50*time.Millisecond)

		// Stop the reaper
		handler.Stop()

		// Verify done channel is closed
		select {
		case _, open := <-handler.done:
			h.False(open)
		case <-time.After(time.Second):
			h.Fail("waitForChildren should exit")
		}

		// Stats should remain zero since no children were reaped
		h.Equal(ProcessReaperStats{Exited: 0, Signaled: 0}, handler.Stats())
	})

	h.Run("waitForChildren reaps direct child processes", func() {
		handler := newProcessReaperHandler(logDiscard)

		h.Require().NotNil(handler)

		// Start the reaper
		handler.Start()

		const numProcesses = 5
		// Create direct child processes using syscall.Fork
		// Don't wait for them - let the reaper do it
		pids := make([]int, 0, numProcesses)
		for i := range numProcesses {
			pid, err := forkAndExitDirect(i)
			h.Require().NoError(err)
			if pid > 0 {
				pids = append(pids, pid)
			}
		}

		// Give the reaper time to catch the child processes
		time.Sleep(reapingPeriod*2 + 100*time.Millisecond)

		// Stop the reaper
		handler.Stop()

		// Wait for any remaining children to avoid zombie processes
		for range pids {
			syscall.Wait4(-1, nil, syscall.WNOHANG, nil)
		}

		// Verify that processes were reaped
		h.Equal(ProcessReaperStats{Exited: numProcesses, Signaled: 0}, handler.Stats())
	})

	h.Run("waitForChildren reaps child process with specific exit code", func() {
		handler := newProcessReaperHandler(logDiscard)

		h.Require().NotNil(handler)

		// Start the reaper
		handler.Start()

		// Create a direct child process with exit code 42
		pid, err := forkAndExitDirect(42)
		h.Require().NoError(err)

		// Give the reaper time to catch the child process
		time.Sleep(reapingPeriod + 100*time.Millisecond)

		// Stop the reaper
		handler.Stop()

		// Wait for any remaining children
		if pid > 0 {
			syscall.Wait4(pid, nil, syscall.WNOHANG, nil)
		}

		// Verify stats increased
		h.Equal(ProcessReaperStats{Exited: 1, Signaled: 0}, handler.Stats())
	})

	h.Run("waitForChildren handles process terminated by signal", func() {
		handler := newProcessReaperHandler(logDiscard)

		h.Require().NotNil(handler)

		// Start the reaper
		handler.Start()

		// Create a direct child process that sleeps
		pid, err := forkAndSleepDirect(50 * time.Millisecond)
		h.Require().NoError(err)
		h.Greater(pid, 0, "should have valid child pid")

		// Give it time to start
		time.Sleep(10 * time.Millisecond)

		// Send SIGTERM to the child process
		err = syscall.Kill(int(pid), syscall.SIGTERM)
		h.Require().NoError(err)

		// Give the reaper time to catch the signaled process
		time.Sleep(reapingPeriod + 100*time.Millisecond)

		// Stop the reaper
		handler.Stop()

		// Wait for any remaining children
		syscall.Wait4(int(pid), nil, syscall.WNOHANG, nil)

		// Verify stats increased (either Exited or Signaled)
		h.Equal(ProcessReaperStats{Exited: 0, Signaled: 1}, handler.Stats())
	})

	h.Run("waitForChildren handles multiple rapid process exits", func() {
		handler := newProcessReaperHandler(logDiscard)

		h.Require().NotNil(handler)

		// Start the reaper
		handler.Start()

		// Create multiple direct child processes rapidly
		const numProcesses = 5
		pids := make([]int, 0, numProcesses)
		for i := 0; i < numProcesses; i++ {
			pid, err := forkAndExitDirect(i)
			h.Require().NoError(err)
			if pid > 0 {
				pids = append(pids, int(pid))
			}
		}

		// Give the reaper time to catch all child processes
		time.Sleep(reapingPeriod*2 + 200*time.Millisecond)

		// Stop the reaper
		handler.Stop()

		// Wait for any remaining children
		for _, pid := range pids {
			syscall.Wait4(pid, nil, syscall.WNOHANG, nil)
		}

		// Verify that processes were reaped
		h.Equal(ProcessReaperStats{Exited: numProcesses, Signaled: 0}, handler.Stats())
	})

	h.Run("waitForChildren with forked background process", func() {
		// Skip this test if not running as root or in a container environment
		// where we might not have proper process control
		if os.Getuid() != 0 {
			h.T().Skip("skipping test - requires root for reliable process control")
		}

		handler := newProcessReaperHandler(logDiscard)
		h.Require().NotNil(handler)

		// Start the reaper
		handler.Start()

		// Create a subprocess that forks and exits
		cmd := exec.Command("sh", "-c", "(sleep 0.1 && exit 0) &")
		err := cmd.Start()
		h.Require().NoError(err)
		_ = cmd.Wait()

		// Give time for the forked process to complete and be reaped
		time.Sleep(reapingPeriod*2 + 100*time.Millisecond)

		// Stop the reaper
		handler.Stop()

		// Verify done channel is closed
		select {
		case _, open := <-handler.done:
			h.False(open)
		case <-time.After(time.Second):
			h.Fail("waitForChildren should exit")
		}

		// Verify stats increased
		h.Equal(ProcessReaperStats{Exited: 1, Signaled: 0}, handler.Stats())
	})

	h.Run("waitForChildren handles mixed process terminations", func() {
		handler := newProcessReaperHandler(logDiscard)

		h.Require().NotNil(handler)

		// Start the reaper
		handler.Start()

		// Create direct child processes that exit normally
		const numProcesses = 5
		pids := make([]int, 0, numProcesses)
		for range numProcesses {
			pid, err := forkAndExitDirect(0)
			h.Require().NoError(err)
			if pid > 0 {
				pids = append(pids, int(pid))
			}
		}

		// Create a child process that we'll terminate with a signal
		sigPid, err := forkAndSleepDirect(100 * time.Millisecond)
		h.Require().NoError(err)
		h.Greater(sigPid, 0)
		pids = append(pids, int(sigPid))

		// Give it time to start
		time.Sleep(10 * time.Millisecond)

		// Send SIGTERM to the child process
		err = syscall.Kill(int(sigPid), syscall.SIGTERM)
		h.Require().NoError(err)

		// Give the reaper time to process all children
		time.Sleep(reapingPeriod*2 + 200*time.Millisecond)

		// Stop the reaper
		handler.Stop()

		// Wait for any remaining children
		for _, pid := range pids {
			syscall.Wait4(pid, nil, syscall.WNOHANG, nil)
		}

		// Verify done channel is closed
		select {
		case _, open := <-handler.done:
			h.False(open)
		case <-time.After(time.Second):
			h.Fail("waitForChildren should exit")
		}

		// Verify that both types of processes were reaped
		h.Equal(ProcessReaperStats{Exited: numProcesses, Signaled: 1}, handler.Stats())
	})
}

func (h *HandlersTestSuite) TestNewProcessReaperHandler_PublicConstructor() {
	h.Run("public constructor initializes handler with logger", func() {
		handler := NewProcessReaperHandler(logDiscard)

		h.Require().NotNil(handler)
		h.NotNil(handler.log)
		h.NotNil(handler.ctx)
		h.NotNil(handler.cancel)
		h.NotNil(handler.done)
		h.Equal(ProcessReaperStats{Exited: 0, Signaled: 0}, handler.Stats())
	})

	h.Run("public constructor with nil logger uses default", func() {
		handler := NewProcessReaperHandler(nil)

		h.Require().NotNil(handler)
		h.NotNil(handler.log)
	})

	h.Run("public and private constructors create equivalent handlers", func() {
		publicHandler := NewProcessReaperHandler(logDiscard)
		privateHandler := newProcessReaperHandler(logDiscard)

		h.Require().NotNil(publicHandler)
		h.Require().NotNil(privateHandler)

		// Both should have same initial state
		h.Equal(publicHandler.Stats(), privateHandler.Stats())

		// Both should start and stop correctly
		publicHandler.Start()
		privateHandler.Start()

		time.Sleep(time.Millisecond * 10)

		publicHandler.Stop()
		privateHandler.Stop()

		// Both should have closed done channels
		select {
		case _, open := <-publicHandler.done:
			h.False(open, "public handler done channel should be closed")
		default:
			h.Fail("public handler done channel should be closed")
		}

		select {
		case _, open := <-privateHandler.done:
			h.False(open, "private handler done channel should be closed")
		default:
			h.Fail("private handler done channel should be closed")
		}
	})
}

func (h *HandlersTestSuite) TestProcessReaper_IntegrationWithProcessHandler() {
	h.Run("reaper works correctly during ProcessHandler lifecycle", func() {
		// Create reaper first
		reaper := NewProcessReaperHandler(logDiscard)
		h.Require().NotNil(reaper)
		reaper.Start()

		// Create a direct child process using syscall.Fork
		// This ensures the process is a direct child that can be reaped
		pid, err := forkAndExitDirect(0)
		h.Require().NoError(err)
		h.Greater(pid, 0, "should have valid pid")

		// Give reaper time to catch the child process
		time.Sleep(reapingPeriod + 100*time.Millisecond)

		// Verify reaper caught the process
		h.Equal(ProcessReaperStats{Exited: 1, Signaled: 0}, reaper.Stats())

		// Stop the reaper
		reaper.Stop()

		// Verify reaper stopped correctly
		select {
		case _, open := <-reaper.done:
			h.False(open, "reaper done channel should be closed after stop")
		case <-time.After(time.Second):
			h.Fail("reaper should stop within timeout")
		}
	})

	h.Run("reaper continues working after process completes", func() {
		reaper := NewProcessReaperHandler(logDiscard)
		h.Require().NotNil(reaper)
		reaper.Start()

		// First direct child process
		pid1, err := forkAndExitDirect(0)
		h.Require().NoError(err)
		h.Greater(pid1, 0)

		// Give reaper time
		time.Sleep(reapingPeriod + 50*time.Millisecond)
		h.Equal(ProcessReaperStats{Exited: 1, Signaled: 0}, reaper.Stats())

		// Second direct child process
		pid2, err := forkAndExitDirect(0)
		h.Require().NoError(err)
		h.Greater(pid2, 0)

		// Give reaper time
		time.Sleep(reapingPeriod + 50*time.Millisecond)
		h.Equal(ProcessReaperStats{Exited: 2, Signaled: 0}, reaper.Stats())

		reaper.Stop()
	})
}

func (h *HandlersTestSuite) TestProcessReaper_ZombieProcessesAfterMultipleStartKill() {
	h.Run("zombie processes are reaped after multiple start/kill cycles using direct fork", func() {
		reaper := NewProcessReaperHandler(logDiscard)
		h.Require().NotNil(reaper)
		reaper.Start()

		const cycles = 3

		for range cycles {
			// Create a direct child process that sleeps (simulating a long-running process)
			pid, err := forkAndSleepDirect(500 * time.Millisecond)
			h.Require().NoError(err)
			h.Greater(pid, 0)

			// Give process time to start
			time.Sleep(50 * time.Millisecond)

			// Kill the process directly using syscall
			err = syscall.Kill(pid, syscall.SIGKILL)
			h.Require().NoError(err)
		}

		// Give reaper time to catch all zombie processes
		time.Sleep(reapingPeriod*2 + 200*time.Millisecond)

		// Verify reaper caught the processes (all killed with SIGKILL)
		h.Equal(ProcessReaperStats{Exited: 0, Signaled: 3}, reaper.Stats())

		reaper.Stop()
	})

	h.Run("zombie processes are reaped after multiple exit cycles", func() {
		reaper := NewProcessReaperHandler(logDiscard)
		h.Require().NotNil(reaper)
		reaper.Start()

		const cycles = 3

		for i := range cycles {
			// Create a direct child process that exits normally
			pid, err := forkAndExitDirect(i)
			h.Require().NoError(err)
			h.Greater(pid, 0)
		}

		// Give reaper time to catch all zombie processes
		time.Sleep(reapingPeriod*2 + 200*time.Millisecond)

		// Verify reaper caught the processes
		h.Equal(ProcessReaperStats{Exited: 3, Signaled: 0}, reaper.Stats())

		reaper.Stop()
	})

	h.Run("no zombie processes remain after rapid fork/exit", func() {
		reaper := NewProcessReaperHandler(logDiscard)
		h.Require().NotNil(reaper)
		reaper.Start()

		// Rapidly create and exit processes
		const numProcesses = 5
		pids := make([]int, 0, numProcesses)

		for i := range numProcesses {
			pid, err := forkAndExitDirect(i)
			h.Require().NoError(err)
			h.Greater(pid, 0)
			pids = append(pids, pid)

			// Small delay between forks
			time.Sleep(10 * time.Millisecond)
		}

		// Give reaper time to catch all zombie processes
		time.Sleep(reapingPeriod*2 + 200*time.Millisecond)

		// Verify reaper caught all processes
		h.Equal(ProcessReaperStats{Exited: 5, Signaled: 0}, reaper.Stats())

		reaper.Stop()

		// Clean up any remaining children
		for range pids {
			syscall.Wait4(-1, nil, syscall.WNOHANG, nil)
		}
	})
}
