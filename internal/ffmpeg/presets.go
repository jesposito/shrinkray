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
	{"compress-hevc", "Compress (HEVC)", "Reduce size with HEVC encoding", CodecHEVC, 0},
	{"compress-av1", "Compress (AV1)", "Maximum compression with AV1 encoding", CodecAV1, 0},
	{"1080p", "Downscale to 1080p", "Downscale to 1080p max (HEVC)", CodecHEVC, 1080},
	{"720p", "Downscale to 720p", "Downscale to 720p (big savings)", CodecHEVC, 720},
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

// BuildPresetArgs builds FFmpeg arguments for a preset with the specified encoder
// sourceBitrate is the source video bitrate in bits/second (used for dynamic bitrate calculation)
// bitDepth is the source video bit depth (8, 10, 12) - used for VAAPI format selection
// Returns (inputArgs, outputArgs) - inputArgs go before -i, outputArgs go after
func BuildPresetArgs(preset *Preset, sourceBitrate int64, subtitleCodecs []string, subtitleHandling string, bitDepth int) (inputArgs []string, outputArgs []string) {
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
		framesOnGPU := hasVAAPIOutputFormat(config.hwaccelArgs)

		// Select output pixel format based on source bit depth:
		// - 8-bit content: nv12 (standard 8-bit YUV 4:2:0)
		// - 10-bit content: p010 (10-bit YUV 4:2:0, required for HDR)
		// Using wrong format causes mid-encode failures (exit 218) or quality loss.
		vaapiFormat := "nv12"
		swFormat := "nv12" // format filter for software frames before hwupload
		if bitDepth >= 10 {
			vaapiFormat = "p010"
			swFormat = "p010le" // little-endian for hwupload compatibility
		}

		if preset.MaxHeight > 0 {
			// Scaling needed
			if framesOnGPU {
				// HW decode → frames already on GPU → HW scale
				outputArgs = append(outputArgs,
					"-vf", fmt.Sprintf("scale_vaapi=w=-2:h='min(ih,%d)':format=%s", preset.MaxHeight, vaapiFormat),
				)
			} else {
				// SW/CPU decode → upload to GPU → HW scale
				outputArgs = append(outputArgs,
					"-vf", fmt.Sprintf("format=%s,hwupload,scale_vaapi=w=-2:h='min(ih,%d)':format=%s", swFormat, preset.MaxHeight, vaapiFormat),
				)
			}
		} else {
			// No scaling needed
			if framesOnGPU {
				// HW decode → ensure format compatibility for encoder
				outputArgs = append(outputArgs,
					"-vf", fmt.Sprintf("scale_vaapi=format=%s", vaapiFormat),
				)
			} else {
				// SW/CPU decode → upload and format for encoder
				outputArgs = append(outputArgs,
					"-vf", fmt.Sprintf("format=%s,hwupload,scale_vaapi=format=%s", swFormat, vaapiFormat),
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
	outputArgs = append(outputArgs,
		"-map", "0:v:0",          // First video stream (for transcoding)
		"-map", "0:v:1?",         // Second video stream if exists (cover art) - ? means optional
		"-map", "0:a?",           // All audio streams
		"-map", "0:s?",           // All subtitle streams
		"-c:v:0", config.encoder, // Transcode first video stream
		"-c:v:1", "copy",         // Copy second video stream (cover art) if present
	)

	// Add quality and encoder-specific args (apply to -c:v:0)
	outputArgs = append(outputArgs, config.qualityFlag, qualityStr)
	outputArgs = append(outputArgs, config.extraArgs...)

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
