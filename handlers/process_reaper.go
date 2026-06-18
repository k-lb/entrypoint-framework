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
	"context"
	"log/slog"
	"sync"
	"syscall"
	"time"
)

const reapingPeriod = time.Second

// ChildrenProcessReaperHandler handles reaping of zombie child processes.
// It runs as a background goroutine that waits for child processes to complete
// and collects their exit status to prevent zombie process accumulation.
type ChildrenProcessReaperHandler struct {
	log       *slog.Logger
	stats     ProcessReaperStats
	statsLock sync.Locker
	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
}

func newProcessReaperHandler(log *slog.Logger) *ChildrenProcessReaperHandler {
	ctx, cancel := context.WithCancel(context.Background())
	h := &ChildrenProcessReaperHandler{
		log:       log,
		statsLock: &sync.Mutex{},
		ctx:       ctx,
		cancel:    cancel,
		done:      make(chan struct{}),
	}
	return h
}

func (h *ChildrenProcessReaperHandler) Start() {
	go h.waitForChildren()
}

func (h *ChildrenProcessReaperHandler) Stop() {
	h.cancel()
	<-h.done
}

// waitForChildren runs an infinite loop that waits for child processes to complete.
// It reaps zombie processes and logs information about finished processes.
// In case no subprocesses are present (ECHILD error), it sleeps for a short period
// before retrying (reapingPeriod).
func (h *ChildrenProcessReaperHandler) waitForChildren() {
	h.log.Info("process reaper started")

	defer close(h.done)

	var stat syscall.WaitStatus
	for {
		select {
		case <-h.ctx.Done():
			h.log.Info("process reaper stopped", slog.Any("error", h.ctx.Err()))
			return
		default:
		}

		pid, err := syscall.Wait4(-1, &stat, 0, nil)
		if err != nil {
			switch err {
			case syscall.EINTR: // interrupted by signal
				continue
			case syscall.ECHILD: // no children in container
			default:
				h.log.Error("error waiting for child process", slog.Any("error", err))
			}
			time.Sleep(reapingPeriod)
			continue
		}

		// pid <= 0 should not happen with -1 (wait for any child), but handle it just in case
		if pid <= 0 {
			h.log.Debug("invalid pid received from Wait4", slog.Int("pid", pid))
			continue
		}

		// Add specific exit information based on how the process ended
		if stat.Exited() {
			h.log.Debug("process exit status",
				slog.Int("pid", pid),
				slog.Int("exit_code", stat.ExitStatus()))
			h.statsLock.Lock()
			h.stats.Exited++
			h.statsLock.Unlock()
		}

		if stat.Signaled() {
			h.log.Debug("process terminated by signal",
				slog.Int("pid", pid),
				slog.String("signal", syscall.Signal(stat.Signal()).String()),
				slog.Bool("core_dumped", stat.CoreDump()))
			h.statsLock.Lock()
			h.stats.Signaled++
			h.statsLock.Unlock()
		}
	}
}

func (h *ChildrenProcessReaperHandler) Stats() ProcessReaperStats {
	h.statsLock.Lock()
	defer h.statsLock.Unlock()
	return h.stats
}
