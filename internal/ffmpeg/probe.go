package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ProbeResult contains metadata about a video file
type ProbeResult struct {
	Path           string        `json:"path"`
	Size           int64         `json:"size"`
	Duration       time.Duration `json:"duration"`
	Format         string        `json:"format"`
	VideoCodec     string        `json:"video_codec"`
	AudioCodec     string        `json:"audio_codec"`
	SubtitleCodecs []string      `json:"subtitle_codecs"`
	Width          int           `json:"width"`
	Height         int           `json:"height"`
	Bitrate        int64         `json:"bitrate"` // bits per second
	FrameRate      float64       `json:"frame_rate"`
	IsHEVC         bool          `json:"is_hevc"`    // true if already x265/HEVC
	IsAV1          bool          `json:"is_av1"`     // true if already AV1
	PixFmt         string        `json:"pix_fmt"`    // pixel format (e.g., yuv420p, yuv420p10le)
	BitDepth       int           `json:"bit_depth"`  // color bit depth (8, 10, 12)
	ColorRange     string        `json:"color_range"` // tv (limited) or pc (full)
	Streams        []ProbeStream `json:"streams,omitempty"`
}

// ProbeStream contains metadata about a media stream.
type ProbeStream struct {
	Type      string  `json:"type"`
	Codec     string  `json:"codec"`
	Width     int     `json:"width,omitempty"`
	Height    int     `json:"height,omitempty"`
	FrameRate float64 `json:"frame_rate,omitempty"`
}

// ffprobeOutput represents the JSON output from ffprobe
type ffprobeOutput struct {
	Format  ffprobeFormat   `json:"format"`
	Streams []ffprobeStream `json:"streams"`
}

type ffprobeFormat struct {
	Filename   string `json:"filename"`
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
	Size       string `json:"size"`
	BitRate    string `json:"bit_rate"`
}

type ffprobeStream struct {
	CodecType        string            `json:"codec_type"`
	CodecName        string            `json:"codec_name"`
	Width            int               `json:"width"`
	Height           int               `json:"height"`
	PixFmt           string            `json:"pix_fmt"`
	BitsPerRawSample string            `json:"bits_per_raw_sample"`
	ColorRange       string            `json:"color_range"`
	RFrameRate       string            `json:"r_frame_rate"`
	AvgFrameRate     string            `json:"avg_frame_rate"`
	Duration         string            `json:"duration"`
	Tags             map[string]string `json:"tags"`
}

// Prober wraps ffprobe functionality
type Prober struct {
	ffprobePath string
}

// NewProber creates a new Prober with the given ffprobe path
func NewProber(ffprobePath string) *Prober {
	return &Prober{ffprobePath: ffprobePath}
}

// Probe returns metadata about a video file
func (p *Prober) Probe(ctx context.Context, path string) (*ProbeResult, error) {
	cmd := exec.CommandContext(ctx, p.ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("ffprobe failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probeOutput ffprobeOutput
	if err := json.Unmarshal(output, &probeOutput); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	result := &ProbeResult{
		Path:   path,
		Format: probeOutput.Format.FormatName,
	}

	// Parse format-level metadata
	if probeOutput.Format.Size != "" {
		result.Size, _ = strconv.ParseInt(probeOutput.Format.Size, 10, 64)
	}
	if probeOutput.Format.BitRate != "" {
		result.Bitrate, _ = strconv.ParseInt(probeOutput.Format.BitRate, 10, 64)
	}
	if probeOutput.Format.Duration != "" {
		durationSec, _ := strconv.ParseFloat(probeOutput.Format.Duration, 64)
		result.Duration = time.Duration(durationSec * float64(time.Second))
	}

	// Parse stream-level metadata
	var maxVideoDuration time.Duration
	var maxStreamDuration time.Duration
	for _, stream := range probeOutput.Streams {
		if streamDuration, ok := parseDurationValue(stream.Duration); ok && streamDuration > maxStreamDuration {
			maxStreamDuration = streamDuration
		}
		if tagDuration, ok := parseDurationValue(stream.Tags["DURATION"]); ok && tagDuration > maxStreamDuration {
			maxStreamDuration = tagDuration
		}

		probeStream := ProbeStream{
			Type:  stream.CodecType,
			Codec: stream.CodecName,
		}
		if stream.Width > 0 {
			probeStream.Width = stream.Width
		}
		if stream.Height > 0 {
			probeStream.Height = stream.Height
		}
		if stream.CodecType == "video" {
			frameRate := parseFrameRate(stream.RFrameRate)
			if frameRate == 0 {
				frameRate = parseFrameRate(stream.AvgFrameRate)
			}
			probeStream.FrameRate = frameRate
		}
		result.Streams = append(result.Streams, probeStream)

		switch stream.CodecType {
		case "video":
			if result.VideoCodec == "" { // Take first video stream
				result.VideoCodec = stream.CodecName
				result.Width = stream.Width
				result.Height = stream.Height
				result.IsHEVC = isHEVCCodec(stream.CodecName)
				result.IsAV1 = isAV1Codec(stream.CodecName)
				result.FrameRate = probeStream.FrameRate
				result.PixFmt = stream.PixFmt
				result.ColorRange = stream.ColorRange
				result.BitDepth = detectBitDepth(stream.PixFmt, stream.BitsPerRawSample)
			}
			if streamDuration, ok := parseDurationValue(stream.Duration); ok && streamDuration > maxVideoDuration {
				maxVideoDuration = streamDuration
			}
			if tagDuration, ok := parseDurationValue(stream.Tags["DURATION"]); ok && tagDuration > maxVideoDuration {
				maxVideoDuration = tagDuration
			}
		case "audio":
			if result.AudioCodec == "" { // Take first audio stream
				result.AudioCodec = stream.CodecName
			}
		case "subtitle":
			if stream.CodecName != "" {
				result.SubtitleCodecs = append(result.SubtitleCodecs, strings.ToLower(stream.CodecName))
			}
		}
	}

	if result.Duration == 0 {
		if maxVideoDuration > 0 {
			result.Duration = maxVideoDuration
		} else if maxStreamDuration > 0 {
			result.Duration = maxStreamDuration
		}
	}

	return result, nil
}

// isHEVCCodec returns true if the codec is HEVC/x265
func isHEVCCodec(codec string) bool {
	codec = strings.ToLower(codec)
	return codec == "hevc" || codec == "h265" || codec == "x265"
}

// isAV1Codec returns true if the codec is AV1
func isAV1Codec(codec string) bool {
	codec = strings.ToLower(codec)
	return codec == "av1" || codec == "libaom-av1" || codec == "libsvtav1"
}

// detectBitDepth determines the color bit depth from pixel format and bits_per_raw_sample.
// Returns 8, 10, or 12 (defaults to 8 if unknown).
func detectBitDepth(pixFmt, bitsPerRawSample string) int {
	// First try explicit bits_per_raw_sample
	if bitsPerRawSample != "" {
		if bits, err := strconv.Atoi(bitsPerRawSample); err == nil && bits > 0 {
			return bits
		}
	}

	// Infer from pixel format name
	pixFmt = strings.ToLower(pixFmt)

	// 12-bit formats
	if strings.Contains(pixFmt, "12le") || strings.Contains(pixFmt, "12be") ||
		strings.HasSuffix(pixFmt, "p12") {
		return 12
	}

	// 10-bit formats (common: yuv420p10le, p010, p010le)
	if strings.Contains(pixFmt, "10le") || strings.Contains(pixFmt, "10be") ||
		strings.HasSuffix(pixFmt, "p10") || pixFmt == "p010" || pixFmt == "p010le" {
		return 10
	}

	// Default to 8-bit
	return 8
}

// Is10Bit returns true if the probe result indicates 10-bit or higher content
func (p *ProbeResult) Is10Bit() bool {
	return p.BitDepth >= 10
}

// parseFrameRate parses a frame rate string like "30000/1001" or "30/1"
func parseFrameRate(s string) float64 {
	if s == "" || s == "0/0" {
		return 0
	}
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}
	num, _ := strconv.ParseFloat(parts[0], 64)
	den, _ := strconv.ParseFloat(parts[1], 64)
	if den == 0 {
		return 0
	}
	return num / den
}

func parseDurationValue(value string) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "N/A" {
		return 0, false
	}

	if strings.Contains(value, ":") {
		parts := strings.Split(value, ":")
		if len(parts) != 3 {
			return 0, false
		}
		hours, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, false
		}
		minutes, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, false
		}
		secondsPart := parts[2]
		secondsFragments := strings.SplitN(secondsPart, ".", 2)
		seconds, err := strconv.Atoi(secondsFragments[0])
		if err != nil {
			return 0, false
		}
		nanos := 0
		if len(secondsFragments) == 2 {
			frac := secondsFragments[1]
			if len(frac) > 9 {
				frac = frac[:9]
			}
			for len(frac) < 9 {
				frac += "0"
			}
			nanos, _ = strconv.Atoi(frac)
		}
		duration := time.Duration(hours)*time.Hour +
			time.Duration(minutes)*time.Minute +
			time.Duration(seconds)*time.Second +
			time.Duration(nanos)
		return duration, true
	}

	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return time.Duration(seconds * float64(time.Second)), true
}

// IsVideoFile returns true if the file extension suggests a video file
func IsVideoFile(path string) bool {
	ext := strings.ToLower(path)
	videoExtensions := []string{
		".mkv", ".mp4", ".avi", ".mov", ".wmv", ".flv",
		".webm", ".m4v", ".mpeg", ".mpg", ".m2ts", ".ts",
	}
	for _, ve := range videoExtensions {
		if strings.HasSuffix(ext, ve) {
			return true
		}
	}
	return false
}
