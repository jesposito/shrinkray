package ffmpeg

import (
	"fmt"
	"strings"
)

// Preset defines a transcoding preset with its FFmpeg parameters
type Preset struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Encoder     HWAccel `json:"encoder"`    // Which encoder to use
	Codec       Codec   `json:"codec"`      // Target codec (HEVC or AV1)
	MaxHeight   int     `json:"max_height"` // 0 = no scaling, 1080, 720, etc.
}

// encoderSettings defines FFmpeg settings for each encoder
type encoderSettings struct {
	encoder     string   // FFmpeg encoder name
	qualityFlag string   // -crf, -b:v, -global_quality, etc.
	quality     string   // Quality value (CRF or bitrate modifier)
	extraArgs   []string // Additional encoder-specific args
	usesBitrate bool     // If true, quality value is a bitrate modifier (0.0-1.0)
	hwaccelArgs []string // Args to prepend before -i for hardware decoding
	scaleFilter string   // Hardware-specific scale filter (e.g., "scale_qsv", "scale_cuda")
}

// SubtitleHandling determines how to handle subtitle streams when transcoding to MKV.
type SubtitleHandling string

const (
	SubtitleHandlingConvert SubtitleHandling = "convert"
	SubtitleHandlingDrop    SubtitleHandling = "drop"
)

func normalizeSubtitleHandling(value string) SubtitleHandling {
	switch strings.ToLower(value) {
	case string(SubtitleHandlingDrop):
		return SubtitleHandlingDrop
	default:
		return SubtitleHandlingConvert
	}
}

// Bitrate constraints for dynamic bitrate calculation (VideoToolbox)
const (
	minBitrateKbps = 500   // Minimum target bitrate in kbps
	maxBitrateKbps = 15000 // Maximum target bitrate in kbps
)

var encoderConfigs = map[EncoderKey]encoderSettings{
	// HEVC encoders
	{HWAccelNone, CodecHEVC}: {
		encoder:     "libx265",
		qualityFlag: "-crf",
		quality:     "26",
		extraArgs:   []string{"-preset", "medium"},
		scaleFilter: "scale",
	},
	{HWAccelVideoToolbox, CodecHEVC}: {
		// VideoToolbox uses bitrate control (-b:v) with dynamic calculation
		// Target bitrate = source bitrate * modifier
		encoder:     "hevc_videotoolbox",
		qualityFlag: "-b:v",
		quality:     "0.35", // 35% of source bitrate (~50-60% smaller files)
		extraArgs:   []string{"-allow_sw", "1"},
		usesBitrate: true,
		hwaccelArgs: []string{"-hwaccel", "videotoolbox"},
		scaleFilter: "scale", // VideoToolbox doesn't have a HW scaler, use CPU
	},
	{HWAccelNVENC, CodecHEVC}: {
		encoder:     "hevc_nvenc",
		qualityFlag: "-cq",
		quality:     "28",
		extraArgs:   []string{"-preset", "p4", "-tune", "hq", "-rc", "vbr"},
		hwaccelArgs: []string{"-hwaccel", "cuda", "-hwaccel_output_format", "cuda"},
		scaleFilter: "scale_cuda",
	},
	{HWAccelQSV, CodecHEVC}: {
		encoder:     "hevc_qsv",
		qualityFlag: "-global_quality",
		quality:     "27",
		extraArgs:   []string{"-preset", "medium"},
		// VAAPI decode with CPU frame transfer to QSV encoder
		// Some CPU overhead but reliable - full GPU pipeline didn't work
		hwaccelArgs: []string{"-hwaccel", "vaapi", "-hwaccel_device", ""},
		scaleFilter: "scale",
	},
	{HWAccelVAAPI, CodecHEVC}: {
		encoder:     "hevc_vaapi",
		qualityFlag: "-qp",
		quality:     "27",
		extraArgs:   []string{},
		hwaccelArgs: []string{"-vaapi_device", "", "-hwaccel", "vaapi", "-hwaccel_output_format", "vaapi"}, // Device path filled dynamically
		scaleFilter: "scale_vaapi",
	},

	// AV1 encoders
	// More aggressive compression than HEVC - AV1 handles lower bitrates better
	{HWAccelNone, CodecAV1}: {
		encoder:     "libsvtav1",
		qualityFlag: "-crf",
		quality:     "35",
		extraArgs:   []string{"-preset", "6"},
		scaleFilter: "scale",
	},
	{HWAccelVideoToolbox, CodecAV1}: {
		// VideoToolbox AV1 (M3+ chips) uses bitrate control
		encoder:     "av1_videotoolbox",
		qualityFlag: "-b:v",
		quality:     "0.25", // 25% of source bitrate
		extraArgs:   []string{"-allow_sw", "1"},
		usesBitrate: true,
		hwaccelArgs: []string{"-hwaccel", "videotoolbox"},
		scaleFilter: "scale", // VideoToolbox doesn't have a HW scaler, use CPU
	},
	{HWAccelNVENC, CodecAV1}: {
		encoder:     "av1_nvenc",
		qualityFlag: "-cq",
		quality:     "32",
		extraArgs:   []string{"-preset", "p4", "-tune", "hq", "-rc", "vbr"},
		hwaccelArgs: []string{"-hwaccel", "cuda", "-hwaccel_output_format", "cuda"},
		scaleFilter: "scale_cuda",
	},
	{HWAccelQSV, CodecAV1}: {
		encoder:     "av1_qsv",
		qualityFlag: "-global_quality",
		quality:     "32",
		extraArgs:   []string{"-preset", "medium"},
		// VAAPI decode with CPU frame transfer to QSV encoder
		// Some CPU overhead but reliable - full GPU pipeline didn't work
		hwaccelArgs: []string{"-hwaccel", "vaapi", "-hwaccel_device", ""},
		scaleFilter: "scale",
	},
	{HWAccelVAAPI, CodecAV1}: {
		encoder:     "av1_vaapi",
		qualityFlag: "-qp",
		quality:     "32",
		extraArgs:   []string{},
		hwaccelArgs: []string{"-vaapi_device", "", "-hwaccel", "vaapi", "-hwaccel_output_format", "vaapi"}, // Device path filled dynamically
		scaleFilter: "scale_vaapi",
	},
}

// BasePresets defines the core presets
var BasePresets = []struct {
	ID          string
	Name        string
	Description string
	Codec       Codec
	MaxHeight   int
}{
	{"compress-hevc", "Smaller files — HEVC", "Widely compatible, works almost everywhere", CodecHEVC, 0},
	{"compress-av1", "Smaller files — AV1", "Best quality per MB, newer devices", CodecAV1, 0},
	{"1080p", "Reduce to 1080p — HEVC", "Downscale to Full HD for big savings", CodecHEVC, 1080},
	{"720p", "Reduce to 720p — HEVC", "Maximum compatibility, smallest files", CodecHEVC, 720},
}

// hasVAAPIOutputFormat checks if hwaccelArgs specify -hwaccel_output_format vaapi,
// meaning decoded frames are in VAAPI GPU memory (not downloaded to CPU).
// This is used to determine whether frames need hwupload or are already on GPU.
func hasVAAPIOutputFormat(hwaccelArgs []string) bool {
	for i, arg := range hwaccelArgs {
		if arg == "-hwaccel_output_format" && i+1 < len(hwaccelArgs) && hwaccelArgs[i+1] == "vaapi" {
			return true
		}
	}
	return false
}

// isVAAPIIncompatiblePixFmt returns true if the pixel format cannot be hardware decoded by VAAPI.
// These formats require software decode + hwupload to VAAPI for encoding.
func isVAAPIIncompatiblePixFmt(pixFmt string) bool {
	// VAAPI typically only supports 4:2:0 formats (yuv420p, nv12, p010)
	// 4:4:4 formats like yuv444p are not supported by VAAPI decode
	incompatible := []string{
		"yuv444p", "yuv444p10", "yuv444p10le", "yuv444p10be",
		"yuv444p12", "yuv444p12le", "yuv444p12be",
		"yuvj444p", // JPEG full-range 4:4:4
		"gbrp", "gbrp10", "gbrp12", // Planar RGB
	}
	for _, fmt := range incompatible {
		if pixFmt == fmt {
			return true
		}
	}
	return false
}

// BuildPresetArgs builds FFmpeg arguments for a preset with the specified encoder
// sourceBitrate is the source video bitrate in bits/second (used for dynamic bitrate calculation)
// bitDepth is the source video bit depth (8, 10, 12) - used for VAAPI format selection
// pixFmt is the source pixel format - used to detect formats requiring software decode
// Returns (inputArgs, outputArgs) - inputArgs go before -i, outputArgs go after
func BuildPresetArgs(preset *Preset, sourceBitrate int64, subtitleCodecs []string, subtitleHandling string, bitDepth int, pixFmt string) (inputArgs []string, outputArgs []string) {
	key := EncoderKey{preset.Encoder, preset.Codec}
	config, ok := encoderConfigs[key]
	if !ok {
		// Fallback to software encoder for the target codec
		config = encoderConfigs[EncoderKey{HWAccelNone, preset.Codec}]
	}

	// Input args: Add probesize and analyzeduration to speed up analysis
	// of files with many streams (especially PGS subtitles)
	// 50MB probesize and 10 seconds analyzeduration handles most files well
	inputArgs = append(inputArgs,
		"-probesize", "50M",
		"-analyzeduration", "10M", // 10 seconds in microseconds
	)

	// Hardware acceleration for decoding
	// Skip hwaccel for pixel formats that VAAPI can't decode (e.g., yuv444p from AI upscales)
	// These require software decode → format conversion → hwupload → VAAPI encode
	useHWAccelDecode := !isVAAPIIncompatiblePixFmt(pixFmt) || preset.Encoder != HWAccelVAAPI
	if useHWAccelDecode {
		for _, arg := range config.hwaccelArgs {
			// Fill in VAAPI device path dynamically
			if arg == "" && len(inputArgs) > 0 {
				lastArg := inputArgs[len(inputArgs)-1]
				if lastArg == "-vaapi_device" || lastArg == "-hwaccel_device" {
					arg = GetVAAPIDevice()
				}
			}
			inputArgs = append(inputArgs, arg)
		}
	} else {
		// For VAAPI with incompatible pixel format, we still need -vaapi_device for encoding
		// but NOT -hwaccel vaapi (which would fail for yuv444p)
		inputArgs = append(inputArgs, "-vaapi_device", GetVAAPIDevice())
	}

	// VAAPI: Add -reinit_filter 0 to INPUT args to prevent mid-stream filter reconfiguration.
	// This MUST be an input option (before -i), not an output option.
	// When input color metadata changes (e.g., SEI messages updating color from untagged to bt709),
	// FFmpeg reconfigures the filter graph and may insert auto_scale between VAAPI filters,
	// causing "Impossible to convert between formats" error after 40+ minutes.
	//
	// Also force input colorspace interpretation to prevent "hwaccel changed" reconfiguration.
	// When input has untagged/unknown colorspace and FFmpeg discovers bt709 mid-stream via SEI,
	// it triggers filter graph reconfiguration with "hwaccel changed" even with -reinit_filter 0.
	// Forcing colorspace tells FFmpeg what to interpret the input as from the start.
	if preset.Encoder == HWAccelVAAPI {
		inputArgs = append(inputArgs, "-reinit_filter", "0")
		// Force input colorspace and range based on bit depth to match output expectations.
		// This prevents "hwaccel changed" reconfiguration when FFmpeg discovers bt709 mid-stream.
		// Without this, FFmpeg starts with "csp: unknown range: unknown" and when it detects
		// bt709/tv from the stream, it triggers filter graph reconfiguration.
		if bitDepth >= 10 {
			// 10-bit HDR: bt2020 with PQ transfer, limited range
			inputArgs = append(inputArgs, "-color_primaries", "bt2020", "-color_trc", "smpte2084", "-colorspace", "bt2020nc", "-color_range", "tv")
		} else {
			// 8-bit SDR: bt709, limited range (tv)
			inputArgs = append(inputArgs, "-color_primaries", "bt709", "-color_trc", "bt709", "-colorspace", "bt709", "-color_range", "tv")
		}
	}

	// Output args
	outputArgs = []string{}

	// Build video filter based on encoder type and decode mode.
	// VAAPI encoder requires explicit filter chain to keep frames on GPU and ensure
	// format compatibility. Without this, FFmpeg auto-inserts software filters that
	// fail with: "Impossible to convert between the formats supported by the filter
	// 'Parsed_null_0' and the filter 'auto_scale_0'"
	if preset.Encoder == HWAccelVAAPI {
		// Check if decode outputs frames directly to VAAPI GPU memory.
		// QSV uses VAAPI decode but without -hwaccel_output_format, so frames
		// download to CPU. This check ensures correct filter chain selection.
		// Also force CPU path for incompatible pixel formats (yuv444p, etc.)
		framesOnGPU := hasVAAPIOutputFormat(config.hwaccelArgs) && !isVAAPIIncompatiblePixFmt(pixFmt)

		// Select output pixel format and color parameters based on source bit depth:
		// - 8-bit content: nv12 with bt709 color (standard SDR)
		// - 10-bit content: p010 with bt2020 color (required for HDR)
		// Using wrong format causes mid-encode failures (exit 218) or quality loss.
		// Explicit color params prevent filter reconfiguration on metadata changes.
		vaapiFormat := "nv12"
		swFormat := "nv12" // format filter for software frames before hwupload
		// Color output params for scale_vaapi to prevent reconfiguration
		// out_range=tv (limited range), out_color_matrix, out_color_primaries, out_color_transfer
		colorParams := "out_range=tv:out_color_matrix=bt709:out_color_primaries=bt709:out_color_transfer=bt709"
		if bitDepth >= 10 {
			vaapiFormat = "p010"
			swFormat = "p010le" // little-endian for hwupload compatibility
			// For 10-bit/HDR content, use bt2020 color space with PQ transfer (HDR10)
			colorParams = "out_range=tv:out_color_matrix=bt2020nc:out_color_primaries=bt2020:out_color_transfer=smpte2084"
		}

		if preset.MaxHeight > 0 {
			// Scaling needed
			if framesOnGPU {
				// HW decode → frames already on GPU → HW scale with explicit color handling
				outputArgs = append(outputArgs,
					"-vf", fmt.Sprintf("scale_vaapi=w=-2:h='min(ih,%d)':format=%s:%s", preset.MaxHeight, vaapiFormat, colorParams),
				)
			} else {
				// SW/CPU decode → upload to GPU → HW scale with explicit color handling
				outputArgs = append(outputArgs,
					"-vf", fmt.Sprintf("format=%s,hwupload,scale_vaapi=w=-2:h='min(ih,%d)':format=%s:%s", swFormat, preset.MaxHeight, vaapiFormat, colorParams),
				)
			}
		} else {
			// No scaling needed
			if framesOnGPU {
				// HW decode → ensure format and color compatibility for encoder
				outputArgs = append(outputArgs,
					"-vf", fmt.Sprintf("scale_vaapi=format=%s:%s", vaapiFormat, colorParams),
				)
			} else {
				// SW/CPU decode → upload and format for encoder with color handling
				outputArgs = append(outputArgs,
					"-vf", fmt.Sprintf("format=%s,hwupload,scale_vaapi=format=%s:%s", swFormat, vaapiFormat, colorParams),
				)
			}
		}
	} else if preset.MaxHeight > 0 {
		// Non-VAAPI paths: QSV, NVENC, Software, VideoToolbox - unchanged
		scaleFilter := config.scaleFilter
		if scaleFilter == "" {
			scaleFilter = "scale"
		}
		outputArgs = append(outputArgs,
			"-vf", fmt.Sprintf("%s=-2:'min(ih,%d)'", scaleFilter, preset.MaxHeight),
		)
	}
	// No filter for non-VAAPI paths without scaling (correct)

	// Get quality setting
	qualityStr := config.quality

	// For encoders that use dynamic bitrate calculation
	if config.usesBitrate && sourceBitrate > 0 {
		// Parse modifier (e.g., "0.5" = 50% of source bitrate)
		modifier := 0.5 // default
		fmt.Sscanf(qualityStr, "%f", &modifier)

		// Calculate target bitrate in kbps
		targetKbps := int64(float64(sourceBitrate) * modifier / 1000)

		// Apply min/max constraints
		if targetKbps < minBitrateKbps {
			targetKbps = minBitrateKbps
		}
		if targetKbps > maxBitrateKbps {
			targetKbps = maxBitrateKbps
		}

		qualityStr = fmt.Sprintf("%dk", targetKbps)
	}

	// Stream mapping: Use explicit stream selectors to avoid "Multiple -codec/-c... options"
	// warning. Map first video for transcoding, additional video streams (cover art) with copy,
	// and all audio/subtitle streams.
	//
	// IMPORTANT: Encoder-specific options (quality, extra args) MUST come immediately after
	// -c:v:0 and BEFORE -c:v:1 copy. FFmpeg associates options with streams based on position,
	// so options after -c:v:1 would try to apply to the copy stream (which ignores them).
	// See: https://ffmpeg.org/ffmpeg.html (stream specifiers section)
	outputArgs = append(outputArgs,
		"-map", "0:v:0",          // First video stream (for transcoding)
		"-map", "0:v:1?",         // Second video stream if exists (cover art) - ? means optional
		"-map", "0:a?",           // All audio streams
		"-map", "0:s?",           // All subtitle streams
		"-c:v:0", config.encoder, // Transcode first video stream
	)

	// Add quality and encoder-specific args immediately after -c:v:0 encoder selection
	// These must come before -c:v:1 to be associated with stream v:0
	outputArgs = append(outputArgs, config.qualityFlag, qualityStr)
	outputArgs = append(outputArgs, config.extraArgs...)

	// Now add the copy codec for cover art (second video stream if present)
	outputArgs = append(outputArgs, "-c:v:1", "copy")

	// Copy audio and handle subtitle codecs (convert mov_text if present).
	outputArgs = append(outputArgs, "-c:a", "copy")

	if containsSubtitleCodec(subtitleCodecs, "mov_text") {
		switch normalizeSubtitleHandling(subtitleHandling) {
		case SubtitleHandlingDrop:
			outputArgs = append(outputArgs, "-sn")
		default:
			outputArgs = append(outputArgs, "-c:s", "srt")
		}
	} else {
		outputArgs = append(outputArgs, "-c:s", "copy")
	}

	return inputArgs, outputArgs
}

func containsSubtitleCodec(codecs []string, target string) bool {
	for _, codec := range codecs {
		if strings.EqualFold(codec, target) {
			return true
		}
	}
	return false
}

// GeneratePresets creates presets using the best available encoder for each codec
func GeneratePresets() map[string]*Preset {
	presets := make(map[string]*Preset)

	for _, base := range BasePresets {
		// Get the best available encoder for this preset's target codec
		bestEncoder := GetBestEncoderForCodec(base.Codec)

		presets[base.ID] = &Preset{
			ID:          base.ID,
			Name:        base.Name,
			Description: base.Description,
			Encoder:     bestEncoder.Accel,
			Codec:       base.Codec,
			MaxHeight:   base.MaxHeight,
		}
	}

	return presets
}

// Presets cache - populated after encoder detection
var generatedPresets map[string]*Preset
var presetsInitialized bool

// InitPresets initializes presets based on available encoders
// Must be called after DetectEncoders
func InitPresets() {
	generatedPresets = GeneratePresets()
	presetsInitialized = true
}

// GetPreset returns a preset by ID
func GetPreset(id string) *Preset {
	if !presetsInitialized {
		// Fallback to software-only presets
		return getSoftwarePreset(id)
	}
	return generatedPresets[id]
}

// getSoftwarePreset returns a software-only preset (fallback)
func getSoftwarePreset(id string) *Preset {
	for _, base := range BasePresets {
		if base.ID == id {
			return &Preset{
				ID:          base.ID,
				Name:        base.Name,
				Description: base.Description,
				Encoder:     HWAccelNone,
				Codec:       base.Codec,
				MaxHeight:   base.MaxHeight,
			}
		}
	}
	return nil
}

// ListPresets returns all available presets
func ListPresets() []*Preset {
	if !presetsInitialized {
		// Return software-only presets as fallback
		var presets []*Preset
		for _, base := range BasePresets {
			presets = append(presets, &Preset{
				ID:          base.ID,
				Name:        base.Name,
				Description: base.Description,
				Encoder:     HWAccelNone,
				Codec:       base.Codec,
				MaxHeight:   base.MaxHeight,
			})
		}
		return presets
	}

	// Return presets in order
	var result []*Preset
	for _, base := range BasePresets {
		if preset, ok := generatedPresets[base.ID]; ok {
			result = append(result, preset)
		}
	}

	return result
}

// GetHardwarePath returns a human-readable string describing the decode→encode pipeline.
// Examples: "vaapi→vaapi", "cpu→vaapi", "cpu→cpu", "cuda→nvenc", "vaapi→qsv"
// This reflects the actual hwaccelArgs configuration in encoderConfigs.
func GetHardwarePath(encoder HWAccel, pixFmt string) string {
	// Determine decode method based on actual hwaccel configuration
	// See encoderConfigs for the hwaccelArgs used by each encoder
	var decode string
	switch encoder {
	case HWAccelVAAPI:
		// VAAPI uses -hwaccel vaapi, but falls back to CPU for incompatible formats
		if isVAAPIIncompatiblePixFmt(pixFmt) {
			decode = "cpu"
		} else {
			decode = "vaapi"
		}
	case HWAccelNVENC:
		// NVENC uses -hwaccel cuda for decode
		decode = "cuda"
	case HWAccelQSV:
		// QSV uses -hwaccel vaapi for decode (see comment: "VAAPI decode with CPU frame transfer")
		decode = "vaapi"
	case HWAccelVideoToolbox:
		// VideoToolbox uses -hwaccel videotoolbox for decode
		decode = "videotoolbox"
	default:
		decode = "cpu"
	}

	// Determine encode method
	var encode string
	switch encoder {
	case HWAccelVAAPI:
		encode = "vaapi"
	case HWAccelNVENC:
		encode = "nvenc"
	case HWAccelQSV:
		encode = "qsv"
	case HWAccelVideoToolbox:
		encode = "videotoolbox"
	default:
		encode = "cpu"
	}

	return decode + "→" + encode
}
