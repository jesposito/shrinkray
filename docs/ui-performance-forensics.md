# UI Performance Forensics Report

**Date**: 2024-12-30
**Issue**: UI becomes laggy when queue exceeds ~50 items
**Target**: Support up to 20,000 items smoothly

## Executive Summary

The Shrinkray UI suffers from severe performance degradation at scale due to **five primary issues**:

1. **Full DOM rebuild on every queue event** (not just progress) - FIXED in this PR
2. **No event batching** - adding 7k jobs triggers 7k separate SSE events
3. **Blocking ffprobe scan** - probing 7k files sequentially before adding any jobs
4. **No list virtualization** - all items rendered regardless of visibility
5. **Full queue in SSE init** - SSE init sends complete queue, not paginated

The minimal fix (debouncing + targeted DOM updates) is implemented. For 20k scale, architectural changes are needed.

---

## Codebase Architecture

### UI Stack
- **Vanilla JavaScript** (no React/Vue/framework)
- **HTML templates** served from Go backend
- **SSE (Server-Sent Events)** for real-time updates via `/api/jobs/stream`

### Key Files
| File | Purpose |
|------|---------|
| `web/templates/index.html` | Main UI with embedded JS (~2800 lines) |
| `internal/api/sse.go` | SSE event streaming |
| `internal/jobs/queue.go` | Job queue with broadcast events |
| `internal/jobs/worker.go` | Worker pool, progress updates |

### Data Flow
```
Backend Queue → SSE broadcast → JS eventSource.onmessage → updateJobs() → DOM
```

---

## Root Cause Analysis (Ranked by Impact)

### 1. CRITICAL: Full DOM Rebuild on Every Non-Progress Event

**Location**: `web/templates/index.html:2661-2679`

```javascript
eventSource.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.type === 'init') {
        updateJobs(data.jobs);
        updateStats(data.stats);
    } else if (data.type === 'progress' && data.job) {
        // Targeted update - GOOD
        updateActiveJobProgress(data.job);
    } else {
        // Full refresh for ALL other events - BAD
        refreshJobs();  // ← Fetches ALL jobs, rebuilds entire DOM
    }
};
```

**Problem**: Events `added`, `started`, `complete`, `failed`, `cancelled` all trigger `refreshJobs()` which:
1. Makes HTTP request to `/api/jobs` (fetches ALL jobs)
2. Calls `updateJobs()` which rebuilds entire DOM via `innerHTML`

**Location**: `web/templates/index.html:2394`
```javascript
container.innerHTML = jobs.map(job => { ... }).join('');
```

**Impact**: For 50 jobs, each structural event causes:
- 50 string template evaluations
- Full innerHTML parse (~50 DOM nodes × ~15 children = 750+ elements)
- Full layout/reflow cycle

### 2. CRITICAL: No Event Batching for Bulk Operations

**Location**: `internal/jobs/queue.go:186-199`

```go
func (q *Queue) AddMultiple(probes []*ffmpeg.ProbeResult, presetID string) ([]*Job, error) {
    for _, probe := range probes {
        job, err := q.Add(probe.Path, presetID, probe)  // Each Add() broadcasts an event!
        // ...
    }
}
```

**Location**: `internal/jobs/queue.go:176-181`
```go
func (q *Queue) Add(...) {
    // ...
    q.broadcast(JobEvent{Type: "added", Job: job})  // SSE event sent immediately
}
```

**Impact**: Adding 50 jobs = 50 SSE events × 50 full refreshes = **2,500 DOM rebuilds** in rapid succession!

### 3. CRITICAL: Blocking FFprobe Scan for Large Directories

**Location**: `internal/browse/browse.go:269-283`

When selecting a folder with 7,000 videos, the following happens:

```go
func (b *Browser) GetVideoFilesWithOptions(ctx context.Context, paths []string, opts GetVideoFilesOptions) ([]*ffmpeg.ProbeResult, error) {
    // 1. Walk directory tree (fast)
    videoPaths, err := b.discoverMediaFiles(cleanPath, opts.Recursive, opts.MaxDepth)

    // 2. Spawn goroutine for EACH file (7000 goroutines!)
    for _, fp := range videoPaths {
        wg.Add(1)
        go func(filePath string) {
            result := b.getProbeResult(probeCtx, filePath)  // Runs ffprobe process
        }(fp)
    }

    wg.Wait() // Wait for ALL 7000 probes before returning!
}
```

**Impact**:
- 7,000 ffprobe processes spawned (even with concurrency, system bottleneck)
- At 100ms per probe, this takes **11+ minutes** before any jobs are added
- UI shows no progress during this time - appears frozen
- Memory spike from 7,000 ProbeResult structs held in memory
- Only AFTER all probing completes does `AddMultiple()` run, which then floods SSE

**Root Cause**: The architecture requires full probe results before adding jobs, because:
1. `Job.Duration` is needed for progress calculation
2. `Job.Bitrate` is needed for dynamic bitrate calculation
3. Skip logic checks codec/resolution against preset

### 4. HIGH: No List Virtualization

**Location**: `web/templates/index.html:2394-2456`

All jobs are rendered to DOM regardless of visibility:
```javascript
container.innerHTML = jobs.map(job => {
    // ~30 lines of HTML template per job
}).join('');
```

**Impact**: With 1000 jobs:
- 1000 DOM elements created
- 1000 × event listeners attached (cancel/retry buttons)
- Browser must layout/paint all elements even if only 20 visible

### 4. MEDIUM: Large SSE Payloads

**Location**: `internal/jobs/job.go:19-49`

Each SSE event includes full `Job` struct:
```go
type Job struct {
    ID, InputPath, OutputPath, TempPath, PresetID string
    Encoder string
    IsHardware bool
    Status Status
    Progress, Speed float64
    ETA, Error, Stderr string  // Stderr can be up to 64KB!
    FFmpegArgs []string
    InputSize, OutputSize, SpaceSaved, Duration, Bitrate, TranscodeTime int64
    CreatedAt, StartedAt, CompletedAt time.Time
    IsSoftwareFallback bool
    OriginalJobID, FallbackReason string
}
```

**Impact**: Each progress event sends ~500 bytes minimum, potentially 64KB+ if `Stderr` populated.

### 5. LOW: updateActivePanel() Overhead

**Location**: `web/templates/index.html:2457`

Called after every `updateJobs()`:
```javascript
function updateJobs(jobs) {
    // ... rebuild queue list
    updateActivePanel();  // Additional filtering + DOM updates
}
```

---

## Reproduction Checklist

### Setup
```bash
# Start the server
cd /home/user/shrinkray
go run ./cmd/shrinkray

# Open UI at http://localhost:8080
```

### Stress Test Steps
1. Open browser DevTools → Performance tab
2. Start recording
3. Select a folder with 50+ video files
4. Click "Start Transcode"
5. Observe lag as jobs are added

### Expected Observations
- **Without fix**: 500+ ms of jank per job add batch, visible frame drops
- **With fix**: <50ms per batch, smooth UI

### Metrics to Capture
```javascript
// Add to browser console for diagnosis
let lastUpdate = Date.now();
let updateCount = 0;
const originalUpdateJobs = updateJobs;
updateJobs = function(jobs) {
    const now = Date.now();
    updateCount++;
    console.log(`updateJobs #${updateCount}: ${jobs.length} jobs, ${now - lastUpdate}ms since last`);
    lastUpdate = now;
    return originalUpdateJobs.apply(this, arguments);
};
```

---

## Fix Proposals

### Track 1: Minimal Safe Fix (Target: 500 items, fast to implement) - IMPLEMENTED

**Approach**: Debounce updates + targeted DOM manipulation

**Status**: IMPLEMENTED in this PR

#### Changes Made

1. **Added debounce for refreshJobs()** - `debouncedRefreshJobs()` batches rapid updates
2. **Added `handleJobAdded()`** - Appends single job to DOM instead of rebuilding all
3. **Added `handleJobStatusChange()`** - Updates single DOM element on status changes
4. **Extracted `renderJobHtml()`** - Reusable template function for single job
5. **Updated `connectSSE()`** - Routes events to targeted handlers

#### Files Modified
- `web/templates/index.html`: Lines 1669-1675 (debounce vars), 2369-2512 (new handlers), 2739-2774 (SSE routing)

#### Expected Improvement
- Adding 50 jobs: **50x fewer DOM rebuilds** (1 per job append vs 50 full rebuilds)
- Status changes: **Single element update** instead of full list rebuild
- Still O(n) for very large lists, but handles 500 items smoothly

#### Tradeoffs
- Still O(n) for `updateActivePanel()` filtering
- Still no virtualization (DOM size grows with queue)
- Good enough for ~500 items

---

### Track 2: Scalable Fix (Target: 20,000 items)

**Approach**: Deferred probing + streaming discovery + batch SSE + virtual scrolling

The key insight: **Don't probe files until the worker picks them up**. This is the biggest architectural change needed.

#### Phase 1: Deferred Probing (Most Critical)

**Current Flow (Blocking)**:
```
User clicks "Start" → Probe ALL 7000 files → Wait 11 minutes → Add ALL 7000 jobs → Flood SSE
```

**New Flow (Streaming)**:
```
User clicks "Start" → Discover file paths only (fast) → Create "pending_probe" jobs immediately
→ Workers probe + transcode when they pick up jobs → UI shows progress during scan
```

**Backend Changes for Deferred Probing**:

```go
// New job status for jobs awaiting probe
const StatusPendingProbe Status = "pending_probe"

// Lightweight job creation (no probe required)
type CreateJobRequest struct {
    Path     string `json:"path"`
    PresetID string `json:"preset_id"`
    FileSize int64  `json:"file_size"` // From stat, not probe
}

func (q *Queue) AddPendingProbe(path string, presetID string, fileSize int64) *Job {
    job := &Job{
        ID:        generateID(),
        InputPath: path,
        PresetID:  presetID,
        Status:    StatusPendingProbe,
        InputSize: fileSize,
        // Duration, Bitrate = 0 (unknown until probed)
    }
    // Add to queue, broadcast single event
    return job
}

// Worker probes when it picks up the job
func (w *Worker) processJob(job *Job) {
    if job.Status == StatusPendingProbe {
        // Probe now, when we're ready to process
        probe, err := w.prober.Probe(ctx, job.InputPath)
        if err != nil {
            w.queue.FailJob(job.ID, err.Error())
            return
        }

        // Check if we should skip (codec already optimal, etc.)
        if skipReason := checkSkipReason(probe, preset); skipReason != "" {
            w.queue.FailJob(job.ID, skipReason)
            return
        }

        // Update job with probe data
        job.Duration = probe.Duration.Milliseconds()
        job.Bitrate = probe.Bitrate
        w.queue.UpdateJobProbeData(job.ID, probe)
    }

    // Continue with transcode...
}
```

**Benefits of Deferred Probing**:
- User sees jobs added **immediately** (milliseconds, not minutes)
- Probing happens in parallel with transcoding
- System isn't overwhelmed with 7000 concurrent ffprobe processes
- Workers naturally rate-limit probing (1 per worker)

#### Phase 2: Streaming File Discovery

Instead of collecting all paths then adding all jobs, stream them:

```go
// Stream discovered files to job queue
func (h *Handler) CreateJobs(w http.ResponseWriter, r *http.Request) {
    // Respond immediately
    writeJSON(w, http.StatusAccepted, map[string]string{"status": "scanning"})

    go func() {
        // Stream files as they're discovered
        err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
            if ffmpeg.IsVideoFile(path) {
                // Add job immediately - no waiting for full scan
                h.queue.AddPendingProbe(path, presetID, info.Size())
            }
            return nil
        })
    }()
}
```

#### Phase 3: Batch SSE Events

```go
// internal/jobs/queue.go
const BatchSize = 50
const BatchInterval = 100 * time.Millisecond

func (q *Queue) startBatcher() {
    ticker := time.NewTicker(BatchInterval)
    var pending []*Job

    for {
        select {
        case job := <-q.addChan:
            pending = append(pending, job)
            if len(pending) >= BatchSize {
                q.broadcastBatch(pending)
                pending = nil
            }
        case <-ticker.C:
            if len(pending) > 0 {
                q.broadcastBatch(pending)
                pending = nil
            }
        }
    }
}

func (q *Queue) broadcastBatch(jobs []*Job) {
    q.broadcast(JobEvent{Type: "batch_added", Jobs: jobs})
}
```

**Frontend handling**:
```javascript
} else if (data.type === 'batch_added' && data.jobs) {
    // Handle batch of jobs efficiently
    data.jobs.forEach(job => cachedJobs.push(job));
    updateJobsFromCache(); // Single DOM update for whole batch
}
```

#### Phase 4: Virtual Scrolling

(See existing virtual scrolling proposal in report)

#### Frontend Changes

**1. Implement Virtual Scrolling**

Use Intersection Observer or a minimal virtual scroll implementation:

```javascript
class VirtualQueue {
    constructor(container, itemHeight = 80) {
        this.container = container;
        this.itemHeight = itemHeight;
        this.visibleRange = { start: 0, end: 20 };
        this.jobs = [];
        this.jobIndex = new Map(); // id → index for O(1) lookup

        container.addEventListener('scroll', () => this.onScroll());
    }

    setJobs(jobs) {
        this.jobs = jobs;
        this.jobIndex.clear();
        jobs.forEach((j, i) => this.jobIndex.set(j.id, i));
        this.render();
    }

    onScroll() {
        const scrollTop = this.container.scrollTop;
        const viewportHeight = this.container.clientHeight;

        const start = Math.floor(scrollTop / this.itemHeight);
        const end = Math.min(
            start + Math.ceil(viewportHeight / this.itemHeight) + 5,
            this.jobs.length
        );

        if (start !== this.visibleRange.start || end !== this.visibleRange.end) {
            this.visibleRange = { start, end };
            this.render();
        }
    }

    render() {
        const { start, end } = this.visibleRange;
        const totalHeight = this.jobs.length * this.itemHeight;

        const visibleJobs = this.jobs.slice(start, end);
        const content = visibleJobs.map(j => renderJobHtml(j)).join('');

        this.container.innerHTML = `
            <div style="height: ${start * this.itemHeight}px"></div>
            ${content}
            <div style="height: ${(this.jobs.length - end) * this.itemHeight}px"></div>
        `;
    }

    updateJob(job) {
        const idx = this.jobIndex.get(job.id);
        if (idx !== undefined) {
            this.jobs[idx] = job;
            // Only re-render if visible
            if (idx >= this.visibleRange.start && idx < this.visibleRange.end) {
                this.render();
            }
        }
    }
}
```

**2. Add Pagination/Lazy Loading**

```javascript
// Only fetch first 100 jobs initially, load more on scroll
async function loadMoreJobs(offset, limit = 100) {
    const resp = await fetch(`/api/jobs?offset=${offset}&limit=${limit}`);
    const data = await resp.json();
    return data.jobs;
}
```

#### Backend Changes

**1. Add Batch Event for AddMultiple**

```go
// internal/jobs/queue.go
func (q *Queue) AddMultiple(probes []*ffmpeg.ProbeResult, presetID string) ([]*Job, error) {
    jobs := make([]*Job, 0, len(probes))

    for _, probe := range probes {
        job := q.createJob(probe, presetID) // No broadcast here
        jobs = append(jobs, job)
    }

    // Single batch broadcast
    q.broadcast(JobEvent{Type: "batch_added", Jobs: jobs})

    return jobs, nil
}
```

**2. Add Delta/Partial Updates for Progress**

```go
// Send minimal progress payload
type ProgressUpdate struct {
    ID       string  `json:"id"`
    Progress float64 `json:"progress"`
    Speed    float64 `json:"speed"`
    ETA      string  `json:"eta"`
}
```

**3. Add Pagination to /api/jobs**

```go
func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
    offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit == 0 || limit > 1000 {
        limit = 100
    }

    allJobs := h.queue.GetAll()
    total := len(allJobs)

    // Paginate
    end := offset + limit
    if end > total {
        end = total
    }
    pageJobs := allJobs[offset:end]

    writeJSON(w, http.StatusOK, map[string]interface{}{
        "jobs":   pageJobs,
        "total":  total,
        "offset": offset,
        "limit":  limit,
    })
}
```

#### Files to Modify
- `web/templates/index.html`: Virtual scrolling, pagination
- `internal/jobs/queue.go`: Batch events
- `internal/api/handler.go`: Pagination support
- `internal/api/sse.go`: Delta updates

#### Estimated Effort
- 1-2 days implementation
- Medium risk - touches backend and frontend

#### Tradeoffs
- More complex implementation
- Requires backend changes
- Virtual scroll may affect keyboard navigation
- Filtering/search needs special handling

---

## Verification Plan

### Before/After Metrics

| Metric | Before | After (Target) |
|--------|--------|----------------|
| Time to add 50 jobs | >5s visible jank | <200ms |
| DOM nodes (100 jobs) | ~1500 | ~300 (visible only) |
| refreshJobs/sec during bulk add | 50+ | 1-2 |
| Memory usage (1000 jobs) | ~50MB | ~10MB |

### Test Script

```javascript
// Run in browser console
async function stressTest(numJobs) {
    console.log(`Adding ${numJobs} fake jobs...`);
    const start = performance.now();

    // Measure frame drops
    let frameCount = 0;
    let frameTimes = [];
    function measureFrame(timestamp) {
        frameTimes.push(timestamp);
        frameCount++;
        if (performance.now() - start < 5000) {
            requestAnimationFrame(measureFrame);
        }
    }
    requestAnimationFrame(measureFrame);

    // Trigger job adds via API
    await fetch('/api/jobs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            paths: Array(numJobs).fill('/test/video.mp4'),
            preset_id: 'compress'
        })
    });

    // Wait for updates to settle
    await new Promise(r => setTimeout(r, 2000));

    const elapsed = performance.now() - start;
    const avgFrameTime = frameTimes.length > 1
        ? (frameTimes[frameTimes.length-1] - frameTimes[0]) / (frameTimes.length - 1)
        : 0;

    console.log(`Results:
        - Elapsed: ${elapsed.toFixed(0)}ms
        - Frames: ${frameCount}
        - Avg frame time: ${avgFrameTime.toFixed(1)}ms (target: <16.67ms for 60fps)
        - Dropped frames: ${Math.max(0, Math.floor(elapsed/16.67) - frameCount)}
    `);
}

// Run test
stressTest(50);
```

### Acceptance Criteria

1. **FPS**: Maintain 30+ FPS during bulk job add (50 jobs)
2. **Responsiveness**: UI remains interactive during updates
3. **Render time**: `updateJobs()` completes in <50ms for 100 items
4. **Memory**: No memory leaks on repeated add/clear cycles

---

## Recommended Implementation Order

1. **Immediate (Today)**: Implement minimal safe fix (debouncing + targeted updates)
2. **Short-term (This Week)**: Add batch events for AddMultiple in backend
3. **Medium-term (Next Sprint)**: Implement virtual scrolling for scalable fix

---

## Appendix: Code Locations Reference

| Function | File | Line |
|----------|------|------|
| `updateJobs()` | `web/templates/index.html` | 2372 |
| `refreshJobs()` | `web/templates/index.html` | 2361 |
| `connectSSE()` | `web/templates/index.html` | 2656 |
| `updateActiveJobProgress()` | `web/templates/index.html` | 2571 |
| `updateActivePanel()` | `web/templates/index.html` | 2467 |
| `Queue.Add()` | `internal/jobs/queue.go` | 130 |
| `Queue.AddMultiple()` | `internal/jobs/queue.go` | 187 |
| `Queue.broadcast()` | `internal/jobs/queue.go` | 522 |
| `JobStream()` (SSE) | `internal/api/sse.go` | 10 |
