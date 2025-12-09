package ffmpeg

import (
	"time"
)

// Estimate contains size and time estimates for transcoding
type Estimate struct {
	CurrentSize      int64         `json:"current_size"`
	EstimatedSize    int64         `json:"estimated_size"`
	EstimatedSizeMin int64         `json:"estimated_size_min"` // Lower bound
	EstimatedSizeMax int64         `json:"estimated_size_max"` // Upper bound
	SpaceSaved       int64         `json:"space_saved"`
	SavingsPercent   float64       `json:"savings_percent"`
	EstimatedTime    time.Duration `json:"estimated_time"`
	Warning          string        `json:"warning,omitempty"`
}

// Bitrate modifiers matching presets.go
const (
	bitrateModifierStandard = 0.50 // 50% of source bitrate for "standard" quality
	bitrateModifierSmaller  = 0.35 // 35% of source bitrate for "smaller" quality

	// HEVC is ~40% more efficient than H.264 at same quality
	// So targeting 50% of H.264 bitrate with HEVC gives similar quality
	hevcEfficiencyGain = 0.40

	// Uncertainty range for estimates
	// Hardware encoders are less predictable than software
	estimateUncertaintySoftware = 0.20 // ±20% for software
	estimateUncertaintyHardware = 0.35 // ±35% for hardware (more variance)

	// Audio/subtitle streams are copied unchanged
	// Typical audio is 128-640 kbps, estimate ~15% of file is non-video
	nonVideoOverheadRatio = 0.15
)

// EstimateTranscode estimates the output size and time for transcoding
func EstimateTranscode(probe *ProbeResult, preset *Preset) *Estimate {
	est := &Estimate{
		CurrentSize: probe.Size,
	}

	// Get base bitrate modifier from preset quality
	bitrateModifier := bitrateModifierStandard
	if preset.Quality == "smaller" {
		bitrateModifier = bitrateModifierSmaller
	}

	// Calculate target bitrate
	var targetBitrate int64
	var compressionRatio float64

	if probe.Bitrate > 0 {
		// Use bitrate-based estimation (more accurate)
		targetBitrate = int64(float64(probe.Bitrate) * bitrateModifier)

		// Apply resolution scaling if downscaling
		if preset.MaxHeight > 0 && probe.Height > preset.MaxHeight {
			// Bitrate scales roughly with pixel count
			// (target_height / source_height)^2 gives pixel ratio
			heightRatio := float64(preset.MaxHeight) / float64(probe.Height)
			pixelRatio := heightRatio * heightRatio
			targetBitrate = int64(float64(targetBitrate) * pixelRatio)
		}

		// Apply min/max constraints (matching presets.go)
		targetBitrateKbps := targetBitrate / 1000
		if targetBitrateKbps < minBitrateKbps {
			targetBitrateKbps = minBitrateKbps
			targetBitrate = targetBitrateKbps * 1000
		}
		if targetBitrateKbps > maxBitrateKbps {
			targetBitrateKbps = maxBitrateKbps
			targetBitrate = targetBitrateKbps * 1000
		}

		// Compression ratio = target / source
		compressionRatio = float64(targetBitrate) / float64(probe.Bitrate)

		// Adjust for source codec
		if probe.IsHEVC {
			// Already HEVC - re-encoding to HEVC gives minimal gains
			// The bitrate modifier assumes H.264→HEVC efficiency gain
			// For HEVC→HEVC, we lose that gain, so adjust upward
			compressionRatio = compressionRatio / (1 - hevcEfficiencyGain)
			if compressionRatio > 0.95 {
				compressionRatio = 0.95 // Cap at 5% savings
			}
			est.Warning = "Already encoded in x265/HEVC. Re-encoding will save minimal space and may reduce quality."
		}
	} else {
		// Fallback to file-size based estimation (less accurate)
		compressionRatio = bitrateModifier

		// Adjust for resolution scaling
		if preset.MaxHeight > 0 && probe.Height > preset.MaxHeight {
			heightRatio := float64(preset.MaxHeight) / float64(probe.Height)
			pixelRatio := heightRatio * heightRatio
			compressionRatio = compressionRatio * pixelRatio
		}

		// Adjust for source codec
		if probe.IsHEVC {
			compressionRatio = 0.95
			est.Warning = "Already encoded in x265/HEVC. Re-encoding will save minimal space and may reduce quality."
		}
	}

	// Clamp compression ratio to reasonable bounds
	if compressionRatio < 0.10 {
		compressionRatio = 0.10 // At least 10% of original
	}
	if compressionRatio > 1.05 {
		compressionRatio = 1.05 // Could be slightly larger due to overhead
	}

	// Account for non-video streams (audio, subtitles) which are copied unchanged
	// The compression ratio only applies to the video portion
	// Final size = (video_size * compression_ratio) + non_video_size
	// Simplified: final_ratio = compression_ratio * (1 - overhead) + overhead
	adjustedRatio := compressionRatio*(1-nonVideoOverheadRatio) + nonVideoOverheadRatio

	// Calculate estimated sizes with uncertainty range
	// Use larger uncertainty for hardware encoders (less predictable)
	uncertainty := estimateUncertaintySoftware
	if preset.Encoder != HWAccelNone {
		uncertainty = estimateUncertaintyHardware
	}

	est.EstimatedSize = int64(float64(probe.Size) * adjustedRatio)
	est.EstimatedSizeMin = int64(float64(est.EstimatedSize) * (1 - uncertainty))
	est.EstimatedSizeMax = int64(float64(est.EstimatedSize) * (1 + uncertainty))

	// Don't estimate larger than original for max (unless already HEVC)
	if est.EstimatedSizeMax > probe.Size && !probe.IsHEVC {
		est.EstimatedSizeMax = probe.Size
	}

	// Calculate savings
	est.SpaceSaved = probe.Size - est.EstimatedSize
	if probe.Size > 0 {
		est.SavingsPercent = float64(est.SpaceSaved) / float64(probe.Size) * 100
	}

	// Add warning for low bitrate sources
	if probe.Bitrate > 0 && probe.Bitrate < 2_000_000 && est.Warning == "" {
		est.Warning = "Source has low bitrate. Transcoding may not save much space."
	}

	// Check for low savings warning
	if est.SavingsPercent < 20 && est.Warning == "" {
		est.Warning = "Estimated savings are less than 20%. This content may already be well-compressed."
	}

	// Estimate encoding time based on encoder type
	est.EstimatedTime = estimateEncodeTime(probe.Duration, preset.Encoder)

	return est
}

// estimateEncodeTime estimates how long encoding will take
func estimateEncodeTime(duration time.Duration, encoder HWAccel) time.Duration {
	// Encode speeds vary significantly by encoder
	// These are conservative estimates (actual may be faster)
	var encodeSpeed float64

	switch encoder {
	case HWAccelVideoToolbox:
		encodeSpeed = 3.0 // ~3x realtime on Apple Silicon
	case HWAccelNVENC:
		encodeSpeed = 4.0 // ~4x realtime typical
	case HWAccelQSV:
		encodeSpeed = 3.0 // ~3x realtime typical
	case HWAccelVAAPI:
		encodeSpeed = 2.5 // ~2.5x realtime typical
	default:
		encodeSpeed = 0.5 // Software encoding: ~0.5x realtime (conservative)
	}

	return time.Duration(float64(duration) / encodeSpeed)
}

// EstimateMultiple estimates totals for multiple files
func EstimateMultiple(probes []*ProbeResult, preset *Preset) *Estimate {
	total := &Estimate{}

	for _, probe := range probes {
		est := EstimateTranscode(probe, preset)
		total.CurrentSize += est.CurrentSize
		total.EstimatedSize += est.EstimatedSize
		total.EstimatedSizeMin += est.EstimatedSizeMin
		total.EstimatedSizeMax += est.EstimatedSizeMax
		total.EstimatedTime += est.EstimatedTime
	}

	total.SpaceSaved = total.CurrentSize - total.EstimatedSize
	if total.CurrentSize > 0 {
		total.SavingsPercent = float64(total.SpaceSaved) / float64(total.CurrentSize) * 100
	}

	// Set warning if overall savings are low
	if total.SavingsPercent < 20 {
		total.Warning = "Estimated savings are less than 20%. This content may already be well-compressed."
	}

	return total
}
