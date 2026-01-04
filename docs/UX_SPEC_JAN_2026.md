# Shrinkray UX Specification — January 2026

Based on product decisions made January 4, 2026.

---

## 1. Preset Dropdown Changes

### Current State
```
Compress (HEVC)
Compress (AV1)
Downscale to 1080p
Downscale to 720p
```

### New UI Labels (IDs unchanged)
| Preset ID | New UI Label | Description |
|-----------|--------------|-------------|
| `compress-hevc` | Smaller files — HEVC | Widely compatible, works almost everywhere |
| `compress-av1` | Smaller files — AV1 | Best quality per MB, newer devices |
| `1080p` | Reduce to 1080p — HEVC | Downscale to Full HD for big savings |
| `720p` | Reduce to 720p — HEVC | Maximum compatibility, smallest files |

### Implementation
- **File**: `internal/ffmpeg/presets.go` (lines 154-157)
- **Change**: Update `Name` field in `BasePresets` array
- Keep preset IDs (`compress-hevc`, `compress-av1`, `1080p`, `720p`) unchanged

---

## 2. "Help Me Choose" Modal

### Trigger
Link text `Help me choose` placed directly after the preset dropdown.

### Modal Structure

```
┌─────────────────────────────────────────────────────────────┐
│  Choose a preset                                        [X] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────┐  ┌─────────────────────┐          │
│  │ Smaller files,      │  │ Smaller files,      │          │
│  │ best quality        │  │ works almost        │          │
│  │                     │  │ everywhere          │          │
│  │ • Smallest files    │  │ • Excellent quality │          │
│  │ • Best quality/MB   │  │ • Widely compatible │          │
│  │ • Newer devices     │  │                     │          │
│  │                     │  │ Uses: HEVC (H.265)  │          │
│  │ Uses: AV1           │  │                     │          │
│  └─────────────────────┘  └─────────────────────┘          │
│                                                             │
│  ┌─────────────────────┐  ┌─────────────────────┐          │
│  │ Make it 1080p       │  │ Make it 720p        │          │
│  │                     │  │                     │          │
│  │ • Downscale to HD   │  │ • Maximum compat    │          │
│  │ • Big savings for   │  │ • Smallest files    │          │
│  │   4K content        │  │   overall           │          │
│  │                     │  │                     │          │
│  │ Uses: HEVC          │  │ Uses: HEVC          │          │
│  └─────────────────────┘  └─────────────────────┘          │
│                                                             │
│  ─────────────────────────────────────────────────────────  │
│  Not sure? "Smaller files, works almost everywhere" is     │
│  the safest choice.                                        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Card Content (Final Copy)

| Card | Title | Bullets | Footer |
|------|-------|---------|--------|
| 1 | Smaller files, best quality | Smallest files • Best quality per MB • Newer devices recommended | Uses: AV1 |
| 2 | Smaller files, works almost everywhere | Excellent quality • Widely compatible | Uses: HEVC (H.265) |
| 3 | Make it 1080p | Downscale to Full HD • Big space savings for 4K | Uses: HEVC |
| 4 | Make it 720p | Maximum compatibility • Smallest files overall | Uses: HEVC |

### Behavior
- Clicking a card: selects preset in dropdown, closes modal
- Clicking X or outside modal: closes without selection
- ESC key: closes modal

### Implementation
- **File**: `web/templates/index.html`
- Add modal HTML after preset dropdown (around line 2273)
- Add CSS for modal and cards
- Add JS for modal open/close and preset selection

---

## 3. Job Metadata Changes

### New Field: `hardware_path`

Added to `Job` struct to track decode → encode path.

| Value | Meaning |
|-------|---------|
| `vaapi→vaapi` | Full GPU pipeline (preferred) |
| `cpu→vaapi` | CPU decode, GPU encode (fallback) |
| `cpu→cpu` | Full software (only if enabled) |
| `nvenc→nvenc` | NVIDIA full pipeline |
| `cpu→nvenc` | CPU decode, NVIDIA encode |

### Implementation
- **File**: `internal/jobs/job.go`
- Add field: `HardwarePath string `json:"hardware_path,omitempty"``
- **File**: `internal/jobs/worker.go`
- Set `HardwarePath` when starting transcode based on detected path

---

## 4. Job Status Messaging

### Running Job Messages

| Preset | Hardware Path | Message |
|--------|---------------|---------|
| compress-av1 | vaapi→vaapi | "Compressing using AV1 for best quality and smallest size" |
| compress-av1 | cpu→vaapi | "Compressing using AV1 (GPU decode not supported for this file)" |
| compress-hevc | vaapi→vaapi | "Compressing using HEVC for wide compatibility" |
| compress-hevc | cpu→vaapi | "Compressing using HEVC (decode running on CPU)" |
| 1080p | vaapi→vaapi | "Downscaling to 1080p to reduce file size" |
| 720p | vaapi→vaapi | "Downscaling to 720p for maximum savings" |

### Completed Job Messages

| Status | Message |
|--------|---------|
| complete | "Saved X MB (Y% smaller)" |
| no_gain | "File already optimized (no space saved)" |
| failed | "Failed: {FallbackReason}" |
| skipped | "Skipped: already in target format" |

### Implementation
- **File**: `web/templates/index.html`
- Add function `getJobStatusMessage(job)` that returns human-readable message
- Update job card rendering to show message

---

## 5. Acceptance Criteria

### Dropdown Changes
- [ ] Preset names show user-friendly labels
- [ ] Preset IDs remain unchanged (API compatibility)
- [ ] No codec acronyms as primary text

### Help Me Choose Modal
- [ ] "Help me choose" link appears next to dropdown
- [ ] Modal opens on click
- [ ] Four cards displayed in 2x2 grid
- [ ] Clicking card selects preset and closes modal
- [ ] Guidance text appears at bottom
- [ ] Modal closes on X, outside click, or ESC

### Job Metadata
- [ ] `hardware_path` field populated on all jobs
- [ ] `fallback_reason` already exists, ensure populated

### Job Messaging
- [ ] Running jobs show descriptive message
- [ ] Messages describe outcome, not implementation
- [ ] CPU decode fallback is visible but not alarming

---

## 6. Code Changes Summary

| File | Change |
|------|--------|
| `internal/ffmpeg/presets.go:154-157` | Update preset names |
| `internal/jobs/job.go:50` | Add `HardwarePath` field |
| `internal/jobs/worker.go` | Set `HardwarePath` during job start |
| `web/templates/index.html` | Add modal HTML, CSS, JS |
| `web/templates/index.html` | Add `getJobStatusMessage()` function |
| `web/templates/index.html` | Update job card rendering |

---

## 7. Conflicts with Current Codebase

| Issue | Location | Resolution |
|-------|----------|------------|
| Technical preset names | `presets.go:154-157` | Update names as specified |
| No hardware path tracking | `job.go` | Add field |
| No modal exists | `index.html` | Create new component |
| Job cards show raw status | `index.html` | Add message generation |

None of these conflict with existing functionality—all are additive changes.

---

## 8. CPU Encode Fallback Setting

### Overview

A new setting controls whether GPU encode failures trigger automatic CPU retry.

**Default: OFF** — GPU failures fail the job with guidance to check VAAPI setup.

### UI Implementation

**Location**: Settings panel → Transcoding section

**Toggle Copy**:
- **Name**: "Allow CPU encode fallback"
- **Description**: "Useful if a few files won't encode on GPU"

### Behavior

| Setting | GPU Encode Fails | Result |
|---------|------------------|--------|
| OFF (default) | Job fails | Error message + guidance to enable setting |
| ON | Job retries | Automatic CPU encode retry |

### Rationale

For Unraid + Intel Arc VAAPI users, GPU encodes should work. Failures usually indicate:
- Missing render group permissions
- Wrong VAAPI driver
- Container misconfiguration

Automatic fallback masks these problems. By defaulting OFF, users are guided to fix their setup rather than silently degrading to slow CPU encodes.

### Implementation

| File | Change |
|------|--------|
| `internal/config/config.go` | Added `AllowSoftwareFallback` field |
| `internal/jobs/worker.go` | Check config before fallback, fail with guidance if disabled |
| `internal/jobs/queue.go` | `FailJobDetails.FallbackReason` for user guidance |
| `internal/api/handler.go` | API endpoints for get/set |
| `web/templates/index.html` | Settings toggle UI |

### Rate Limiting

When fallback IS enabled, software fallbacks are rate-limited to 5 per 5 minutes to prevent queue flooding if GPU has systemic issues.

---

*End of UX Spec*
