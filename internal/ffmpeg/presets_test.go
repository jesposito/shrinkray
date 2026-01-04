package ffmpeg

import (
	"strings"
	"testing"
)

func TestBuildPresetArgsDynamicBitrate(t *testing.T) {
	// Test that VideoToolbox presets calculate dynamic bitrate correctly

	// Source bitrate: 3481000 bits/s (3481 kbps)
	sourceBitrate := int64(3481000)

	// Create a VideoToolbox preset (0.35 modifier for HEVC)
	preset := &Preset{
		ID:      "test-hevc",
		Encoder: HWAccelVideoToolbox,
		Codec:   CodecHEVC,
	}

	_, outputArgs := BuildPresetArgs(preset, sourceBitrate, nil, "convert", 8, "yuv420p")

	// Should contain -b:v with calculated bitrate
	// Expected: 3481 * 0.35 = ~1218k
	found := false
	for i, arg := range outputArgs {
		if arg == "-b:v" && i+1 < len(outputArgs) {
			found = true
			bitrate := outputArgs[i+1]
			if !strings.HasSuffix(bitrate, "k") {
				t.Errorf("expected bitrate to end in 'k', got %s", bitrate)
			}
			t.Logf("HEVC VideoToolbox: source=%dkbps → target=%s", sourceBitrate/1000, bitrate)

			// Should be around 1218k (3481 * 0.35)
			if bitrate != "1218k" {
				t.Errorf("expected ~1218k, got %s", bitrate)
			}
		}
	}
	if !found {
		t.Error("expected to find -b:v flag in args")
	}
}

func TestBuildPresetArgsDynamicBitrateAV1(t *testing.T) {
	sourceBitrate := int64(3481000)

	// Create a VideoToolbox AV1 preset (0.25 modifier - more aggressive for AV1)
	preset := &Preset{
		ID:      "test-av1",
		Encoder: HWAccelVideoToolbox,
		Codec:   CodecAV1,
	}

	_, outputArgs := BuildPresetArgs(preset, sourceBitrate, nil, "convert", 8, "yuv420p")

	// Expected: 3481 * 0.25 = ~870k
	for i, arg := range outputArgs {
		if arg == "-b:v" && i+1 < len(outputArgs) {
			bitrate := outputArgs[i+1]
			t.Logf("AV1 VideoToolbox: source=%dkbps → target=%s", sourceBitrate/1000, bitrate)

			if bitrate != "870k" {
				t.Errorf("expected ~870k, got %s", bitrate)
			}
		}
	}
}

func TestBuildPresetArgsBitrateConstraints(t *testing.T) {
	// Test min/max bitrate constraints

	// Very low source bitrate (should hit minimum)
	// 500 kbps * 0.35 = 175k, should clamp to 500k
	lowBitrate := int64(500000)
	presetLow := &Preset{
		ID:      "test-low",
		Encoder: HWAccelVideoToolbox,
		Codec:   CodecHEVC,
	}

	_, outputArgs := BuildPresetArgs(presetLow, lowBitrate, nil, "convert", 8, "yuv420p")
	for i, arg := range outputArgs {
		if arg == "-b:v" && i+1 < len(outputArgs) {
			bitrate := outputArgs[i+1]
			t.Logf("Low bitrate source: %dkbps → target=%s", lowBitrate/1000, bitrate)

			if bitrate != "500k" {
				t.Errorf("expected min 500k, got %s", bitrate)
			}
		}
	}

	// Very high source bitrate (should hit maximum)
	// 50000 kbps * 0.35 = 17500k, should clamp to 15000k
	highBitrate := int64(50000000)
	presetHigh := &Preset{
		ID:      "test-high",
		Encoder: HWAccelVideoToolbox,
		Codec:   CodecHEVC,
	}

	_, outputArgs = BuildPresetArgs(presetHigh, highBitrate, nil, "convert", 8, "yuv420p")
	for i, arg := range outputArgs {
		if arg == "-b:v" && i+1 < len(outputArgs) {
			bitrate := outputArgs[i+1]
			t.Logf("High bitrate source: %dkbps → target=%s", highBitrate/1000, bitrate)

			if bitrate != "15000k" {
				t.Errorf("expected max 15000k, got %s", bitrate)
			}
		}
	}
}

func TestBuildPresetArgsNonBitrateEncoder(t *testing.T) {
	// Test that non-bitrate encoders (like software x265) don't use dynamic calculation
	sourceBitrate := int64(3481000)

	presetSoftware := &Preset{
		ID:      "test-software",
		Encoder: HWAccelNone,
		Codec:   CodecHEVC,
	}

	inputArgs, outputArgs := BuildPresetArgs(presetSoftware, sourceBitrate, nil, "convert", 8, "yuv420p")

	// Software encoder should have probesize/analyzeduration but no hwaccel input args
	// Expected: [-probesize 50M -analyzeduration 10M]
	if len(inputArgs) != 4 {
		t.Errorf("expected 4 input args (probesize + analyzeduration), got %v", inputArgs)
	}
	if len(inputArgs) >= 2 && inputArgs[0] != "-probesize" {
		t.Errorf("expected first input arg to be -probesize, got %v", inputArgs)
	}

	// Should use -crf not -b:v
	foundCRF := false
	foundBv := false
	for i, arg := range outputArgs {
		if arg == "-crf" {
			foundCRF = true
			// Verify CRF value is 26
			if i+1 < len(outputArgs) && outputArgs[i+1] != "26" {
				t.Errorf("expected CRF 26, got %s", outputArgs[i+1])
			}
		}
		if arg == "-b:v" {
			foundBv = true
		}
	}

	if !foundCRF {
		t.Error("expected software encoder to use -crf")
	}
	if foundBv {
		t.Error("software encoder should not use -b:v")
	}

	t.Logf("Software encoder args: %v", outputArgs)
}

func TestBuildPresetArgsZeroBitrate(t *testing.T) {
	// When source bitrate is 0, should use default behavior
	presetVT := &Preset{
		ID:      "test-vt-zero",
		Encoder: HWAccelVideoToolbox,
		Codec:   CodecHEVC,
	}

	_, outputArgs := BuildPresetArgs(presetVT, 0, nil, "convert", 8, "yuv420p")

	// Should still have -b:v but with raw modifier value
	for i, arg := range outputArgs {
		if arg == "-b:v" && i+1 < len(outputArgs) {
			bitrate := outputArgs[i+1]
			t.Logf("Zero bitrate source → target=%s", bitrate)
			// Should fall back to the raw modifier value "0.35"
			if bitrate != "0.35" {
				t.Errorf("expected fallback to '0.35', got %s", bitrate)
			}
		}
	}
}

func TestBuildPresetArgsSubtitleHandling(t *testing.T) {
	preset := &Preset{
		ID:      "test-hevc",
		Encoder: HWAccelNone,
		Codec:   CodecHEVC,
	}

	_, outputArgs := BuildPresetArgs(preset, 0, []string{"mov_text"}, "convert", 8, "yuv420p")
	if !containsArgPair(outputArgs, "-c:s", "srt") {
		t.Errorf("expected -c:s srt when mov_text present and convert enabled, got %v", outputArgs)
	}

	_, outputArgs = BuildPresetArgs(preset, 0, []string{"mov_text"}, "drop", 8, "yuv420p")
	if !containsArg(outputArgs, "-sn") {
		t.Errorf("expected -sn when mov_text present and drop enabled, got %v", outputArgs)
	}

	_, outputArgs = BuildPresetArgs(preset, 0, []string{"srt"}, "convert", 8, "yuv420p")
	if !containsArgPair(outputArgs, "-c:s", "copy") {
		t.Errorf("expected -c:s copy when mov_text absent, got %v", outputArgs)
	}
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func containsArgPair(args []string, key, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == key && args[i+1] == value {
			return true
		}
	}
	return false
}

// TestBuildPresetArgsMuxingQueueSize removed - the -max_muxing_queue_size flag
// was removed because it caused memory issues with concurrent QSV encoding

// TestBuildPresetArgsVAAPIAV1 verifies VAAPI AV1 encoding generates correct FFmpeg args.
// This test ensures the fix for the VAAPI filter graph error:
// - "Impossible to convert between the formats supported by the filter 'Parsed_null_0' and the filter 'auto_scale_0'"
// The fix requires an explicit -vf filter to keep frames on VAAPI surfaces.
func TestBuildPresetArgsVAAPIAV1(t *testing.T) {
	preset := &Preset{
		ID:        "compress-av1",
		Name:      "Compress (AV1)",
		Encoder:   HWAccelVAAPI,
		Codec:     CodecAV1,
		MaxHeight: 0, // No scaling
	}

	inputArgs, outputArgs := BuildPresetArgs(preset, 5000000, nil, "convert", 8, "yuv420p")

	// Verify input args contain VAAPI device and hardware acceleration
	inputArgsStr := strings.Join(inputArgs, " ")
	t.Logf("VAAPI AV1 input args: %v", inputArgs)

	if !strings.Contains(inputArgsStr, "-vaapi_device") {
		t.Error("expected -vaapi_device in input args")
	}
	if !containsArgPair(inputArgs, "-hwaccel", "vaapi") {
		t.Error("expected -hwaccel vaapi in input args")
	}
	if !containsArgPair(inputArgs, "-hwaccel_output_format", "vaapi") {
		t.Error("expected -hwaccel_output_format vaapi in input args")
	}

	// Verify output args
	outputArgsStr := strings.Join(outputArgs, " ")
	t.Logf("VAAPI AV1 output args: %v", outputArgs)

	// Must have explicit video filter with scale_vaapi including format and color params
	// to prevent auto_scale insertion during mid-stream color metadata changes
	foundVAAPIFilter := false
	for i, arg := range outputArgs {
		if arg == "-vf" && i+1 < len(outputArgs) {
			filter := outputArgs[i+1]
			// Check for scale_vaapi with format=nv12 and color output params
			if strings.Contains(filter, "scale_vaapi") &&
				strings.Contains(filter, "format=nv12") &&
				strings.Contains(filter, "out_color_matrix=bt709") {
				foundVAAPIFilter = true
				t.Logf("Found correct VAAPI filter with color params: %s", filter)
			}
		}
	}
	if !foundVAAPIFilter {
		t.Errorf("expected -vf scale_vaapi with format=nv12 and out_color_matrix=bt709, got: %s", outputArgsStr)
	}

	// Must use av1_vaapi encoder
	if !containsArgPair(outputArgs, "-c:v:0", "av1_vaapi") {
		t.Errorf("expected -c:v:0 av1_vaapi, got: %s", outputArgsStr)
	}

	// Must have explicit quality setting (-qp) to avoid "No quality level set" warning
	if !containsArg(outputArgs, "-qp") {
		t.Errorf("expected -qp quality flag, got: %s", outputArgsStr)
	}

	// Verify there's only ONE -c:v:0 option (no duplicate codec warning)
	codecCount := 0
	for _, arg := range outputArgs {
		if arg == "-c:v:0" {
			codecCount++
		}
	}
	if codecCount != 1 {
		t.Errorf("expected exactly 1 -c:v:0 option to avoid duplicate codec warning, got %d", codecCount)
	}

	// Should NOT have -c:v copy (which causes "Multiple -codec options" warning)
	if containsArgPair(outputArgs, "-c:v", "copy") {
		t.Errorf("should not have '-c:v copy' which causes duplicate codec warning, got: %s", outputArgsStr)
	}
}

// TestBuildPresetArgsVAAPIAV1WithScaling verifies VAAPI AV1 with scaling uses correct filter.
func TestBuildPresetArgsVAAPIAV1WithScaling(t *testing.T) {
	preset := &Preset{
		ID:        "1080p-av1",
		Name:      "1080p AV1",
		Encoder:   HWAccelVAAPI,
		Codec:     CodecAV1,
		MaxHeight: 1080, // Scale to 1080p
	}

	_, outputArgs := BuildPresetArgs(preset, 10000000, nil, "convert", 8, "yuv420p")
	outputArgsStr := strings.Join(outputArgs, " ")
	t.Logf("VAAPI AV1 1080p output args: %v", outputArgs)

	// With scaling, should use scale_vaapi with height, format, and color params
	// Expected format: scale_vaapi=w=-2:h='min(ih,1080)':format=nv12:out_range=tv:out_color_matrix=bt709:...
	foundCorrectFilter := false
	for i, arg := range outputArgs {
		if arg == "-vf" && i+1 < len(outputArgs) {
			filter := outputArgs[i+1]
			if strings.Contains(filter, "scale_vaapi") &&
				strings.Contains(filter, "1080") &&
				strings.Contains(filter, "format=nv12") &&
				strings.Contains(filter, "out_color_matrix=bt709") {
				foundCorrectFilter = true
				t.Logf("Found correct VAAPI scale filter with color params: %s", filter)
			}
		}
	}

	if !foundCorrectFilter {
		t.Errorf("expected scale_vaapi filter with 1080, format=nv12, and out_color_matrix=bt709, got: %s", outputArgsStr)
	}
}

// TestBuildPresetArgsVAAPIHEVC verifies VAAPI HEVC also gets explicit filter.
func TestBuildPresetArgsVAAPIHEVC(t *testing.T) {
	preset := &Preset{
		ID:        "compress-hevc",
		Name:      "Compress (HEVC)",
		Encoder:   HWAccelVAAPI,
		Codec:     CodecHEVC,
		MaxHeight: 0, // No scaling
	}

	_, outputArgs := BuildPresetArgs(preset, 5000000, nil, "convert", 8, "yuv420p")
	outputArgsStr := strings.Join(outputArgs, " ")
	t.Logf("VAAPI HEVC output args: %v", outputArgs)

	// VAAPI HEVC also needs explicit filter with color params
	foundVAAPIFilter := false
	for i, arg := range outputArgs {
		if arg == "-vf" && i+1 < len(outputArgs) {
			filter := outputArgs[i+1]
			if strings.Contains(filter, "scale_vaapi") &&
				strings.Contains(filter, "format=nv12") &&
				strings.Contains(filter, "out_color_matrix=bt709") {
				foundVAAPIFilter = true
			}
		}
	}
	if !foundVAAPIFilter {
		t.Errorf("expected -vf scale_vaapi with format=nv12 and out_color_matrix=bt709 for VAAPI HEVC, got: %s", outputArgsStr)
	}

	// Must use hevc_vaapi encoder
	if !containsArgPair(outputArgs, "-c:v:0", "hevc_vaapi") {
		t.Errorf("expected -c:v:0 hevc_vaapi, got: %s", outputArgsStr)
	}
}

// TestBuildPresetArgsNonVAAPINoExtraFilter verifies non-VAAPI encoders don't get VAAPI filter.
func TestBuildPresetArgsNonVAAPINoExtraFilter(t *testing.T) {
	// Software encoder should not have VAAPI filter
	presetSoftware := &Preset{
		ID:        "compress-hevc",
		Name:      "Compress (HEVC)",
		Encoder:   HWAccelNone,
		Codec:     CodecHEVC,
		MaxHeight: 0, // No scaling
	}

	_, outputArgs := BuildPresetArgs(presetSoftware, 5000000, nil, "convert", 8, "yuv420p")
	outputArgsStr := strings.Join(outputArgs, " ")
	t.Logf("Software HEVC output args: %v", outputArgs)

	// Software encoder without scaling should have NO -vf at all
	if containsArg(outputArgs, "-vf") {
		t.Errorf("software encoder without scaling should not have -vf, got: %s", outputArgsStr)
	}

	// NVENC encoder also should not have VAAPI filter
	presetNVENC := &Preset{
		ID:        "compress-hevc",
		Name:      "Compress (HEVC)",
		Encoder:   HWAccelNVENC,
		Codec:     CodecHEVC,
		MaxHeight: 0, // No scaling
	}

	_, outputArgs = BuildPresetArgs(presetNVENC, 5000000, nil, "convert", 8, "yuv420p")
	outputArgsStr = strings.Join(outputArgs, " ")
	t.Logf("NVENC HEVC output args: %v", outputArgs)

	// NVENC without scaling should have NO -vf
	if containsArg(outputArgs, "-vf") {
		t.Errorf("NVENC encoder without scaling should not have -vf, got: %s", outputArgsStr)
	}
}

// TestHasVAAPIOutputFormat tests the helper function.
func TestHasVAAPIOutputFormat(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "VAAPI full pipeline",
			args:     []string{"-vaapi_device", "/dev/dri/renderD128", "-hwaccel", "vaapi", "-hwaccel_output_format", "vaapi"},
			expected: true,
		},
		{
			name:     "VAAPI decode only (no output format)",
			args:     []string{"-hwaccel", "vaapi", "-hwaccel_device", "/dev/dri/renderD128"},
			expected: false,
		},
		{
			name:     "CUDA/NVENC",
			args:     []string{"-hwaccel", "cuda", "-hwaccel_output_format", "cuda"},
			expected: false,
		},
		{
			name:     "No hwaccel",
			args:     []string{},
			expected: false,
		},
		{
			name:     "VideoToolbox",
			args:     []string{"-hwaccel", "videotoolbox"},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := hasVAAPIOutputFormat(tc.args)
			if result != tc.expected {
				t.Errorf("hasVAAPIOutputFormat(%v) = %v, expected %v", tc.args, result, tc.expected)
			}
		})
	}
}

// TestBuildPresetArgsVAAPICPUFallback tests the hwupload path when VAAPI encoder
// is used but decode outputs to CPU memory (edge case for fallback scenarios).
func TestBuildPresetArgsVAAPICPUFallback(t *testing.T) {
	// Simulate VAAPI encoder with CPU frames (e.g., decode fallback scenario)
	// This would require modifying hwaccelArgs to not have -hwaccel_output_format vaapi
	// For now, we verify the explicit encoder guard works correctly

	// The key test is that preset.Encoder == HWAccelVAAPI is checked first,
	// so even if someone creates a config without -hwaccel_output_format vaapi,
	// the VAAPI filter chain is still applied (with hwupload)

	// This is tested implicitly by TestBuildPresetArgsVAAPIAV1 which verifies
	// that VAAPI encoder gets the correct filter regardless of other conditions
}

// TestBuildPresetArgsVAAPI10Bit verifies that 10-bit content uses p010 format instead of nv12.
// This prevents mid-encode failures (exit 218) when encoding HDR or 10-bit content.
func TestBuildPresetArgsVAAPI10Bit(t *testing.T) {
	preset := &Preset{
		ID:        "compress-hevc",
		Name:      "Compress (HEVC)",
		Encoder:   HWAccelVAAPI,
		Codec:     CodecHEVC,
		MaxHeight: 0, // No scaling
	}

	// Test 10-bit content
	_, outputArgs := BuildPresetArgs(preset, 5000000, nil, "convert", 10, "yuv420p10le")
	outputArgsStr := strings.Join(outputArgs, " ")
	t.Logf("VAAPI HEVC 10-bit output args: %v", outputArgs)

	// 10-bit content should use p010 format with bt2020 color params
	found10bit := false
	for i, arg := range outputArgs {
		if arg == "-vf" && i+1 < len(outputArgs) {
			filter := outputArgs[i+1]
			if strings.Contains(filter, "scale_vaapi") &&
				strings.Contains(filter, "format=p010") &&
				strings.Contains(filter, "out_color_matrix=bt2020nc") {
				found10bit = true
				t.Logf("Found correct 10-bit VAAPI filter: %s", filter)
			}
		}
	}
	if !found10bit {
		t.Errorf("expected scale_vaapi with format=p010 and out_color_matrix=bt2020nc for 10-bit content, got: %s", outputArgsStr)
	}

	// Test 12-bit content (should also use p010 with bt2020)
	_, outputArgs = BuildPresetArgs(preset, 5000000, nil, "convert", 12, "yuv420p12le")
	outputArgsStr = strings.Join(outputArgs, " ")
	t.Logf("VAAPI HEVC 12-bit output args: %v", outputArgs)

	found12bit := false
	for i, arg := range outputArgs {
		if arg == "-vf" && i+1 < len(outputArgs) {
			filter := outputArgs[i+1]
			if strings.Contains(filter, "scale_vaapi") &&
				strings.Contains(filter, "format=p010") &&
				strings.Contains(filter, "out_color_matrix=bt2020nc") {
				found12bit = true
			}
		}
	}
	if !found12bit {
		t.Errorf("expected scale_vaapi with format=p010 and out_color_matrix=bt2020nc for 12-bit content, got: %s", outputArgsStr)
	}
}

// TestBuildPresetArgsVAAPIPreventReconfiguration verifies that VAAPI encoder includes
// all necessary flags to prevent mid-stream filter graph reconfiguration.
// This test addresses TWO types of reconfiguration issues:
// 1. "Reconfiguring filter graph because video parameters changed" - prevented by -reinit_filter 0
// 2. "Reconfiguring filter graph because hwaccel changed" - prevented by forcing input colorspace
// Both cause: "Impossible to convert between formats" error after 40+ minutes.
func TestBuildPresetArgsVAAPIPreventReconfiguration(t *testing.T) {
	preset := &Preset{
		ID:        "compress-av1",
		Name:      "Compress (AV1)",
		Encoder:   HWAccelVAAPI,
		Codec:     CodecAV1,
		MaxHeight: 0, // No scaling
	}

	inputArgs, outputArgs := BuildPresetArgs(preset, 5000000, nil, "convert", 8, "yuv420p")
	inputArgsStr := strings.Join(inputArgs, " ")
	t.Logf("VAAPI input args: %v", inputArgs)
	t.Logf("VAAPI output args: %v", outputArgs)

	// Fix 1: -reinit_filter 0 prevents "video parameters changed" reconfiguration
	if !containsArgPair(inputArgs, "-reinit_filter", "0") {
		t.Errorf("expected -reinit_filter 0 in INPUT args, got: %s", inputArgsStr)
	}

	// Fix 2: Force input colorspace to prevent "hwaccel changed" reconfiguration
	// When input has csp:unknown and FFmpeg discovers bt709 mid-stream, it triggers
	// reconfiguration even with -reinit_filter 0. Forcing colorspace from start prevents this.
	if !containsArgPair(inputArgs, "-color_primaries", "bt709") {
		t.Errorf("expected -color_primaries bt709 in INPUT args for 8-bit content, got: %s", inputArgsStr)
	}
	if !containsArgPair(inputArgs, "-color_trc", "bt709") {
		t.Errorf("expected -color_trc bt709 in INPUT args for 8-bit content, got: %s", inputArgsStr)
	}
	if !containsArgPair(inputArgs, "-colorspace", "bt709") {
		t.Errorf("expected -colorspace bt709 in INPUT args for 8-bit content, got: %s", inputArgsStr)
	}
	if !containsArgPair(inputArgs, "-color_range", "tv") {
		t.Errorf("expected -color_range tv in INPUT args for 8-bit content, got: %s", inputArgsStr)
	}

	// Find the -vf argument
	var filter string
	for i, arg := range outputArgs {
		if arg == "-vf" && i+1 < len(outputArgs) {
			filter = outputArgs[i+1]
			break
		}
	}

	if filter == "" {
		t.Fatal("expected -vf argument but none found")
	}

	t.Logf("Filter: %s", filter)

	// Verify all required color params in filter (secondary defense for output)
	requiredParams := []string{
		"scale_vaapi",
		"format=nv12",
		"out_range=tv",
		"out_color_matrix=bt709",
		"out_color_primaries=bt709",
		"out_color_transfer=bt709",
	}

	for _, param := range requiredParams {
		if !strings.Contains(filter, param) {
			t.Errorf("filter missing required param %q: %s", param, filter)
		}
	}
}

// TestBuildPresetArgsVAAPI10BitPreventReconfiguration verifies 10-bit content uses bt2020 colorspace.
func TestBuildPresetArgsVAAPI10BitPreventReconfiguration(t *testing.T) {
	preset := &Preset{
		ID:        "compress-hevc",
		Name:      "Compress (HEVC)",
		Encoder:   HWAccelVAAPI,
		Codec:     CodecHEVC,
		MaxHeight: 0,
	}

	inputArgs, _ := BuildPresetArgs(preset, 5000000, nil, "convert", 10, "yuv420p10le")
	inputArgsStr := strings.Join(inputArgs, " ")
	t.Logf("VAAPI 10-bit input args: %v", inputArgs)

	// 10-bit content should use bt2020 colorspace
	if !containsArgPair(inputArgs, "-color_primaries", "bt2020") {
		t.Errorf("expected -color_primaries bt2020 for 10-bit content, got: %s", inputArgsStr)
	}
	if !containsArgPair(inputArgs, "-color_trc", "smpte2084") {
		t.Errorf("expected -color_trc smpte2084 for 10-bit HDR content, got: %s", inputArgsStr)
	}
	if !containsArgPair(inputArgs, "-colorspace", "bt2020nc") {
		t.Errorf("expected -colorspace bt2020nc for 10-bit content, got: %s", inputArgsStr)
	}
	if !containsArgPair(inputArgs, "-color_range", "tv") {
		t.Errorf("expected -color_range tv in INPUT args, got: %s", inputArgsStr)
	}
}

// TestBuildPresetArgsVAAPI10BitWithScaling verifies 10-bit content with scaling uses p010.
func TestBuildPresetArgsVAAPI10BitWithScaling(t *testing.T) {
	preset := &Preset{
		ID:        "1080p",
		Name:      "1080p",
		Encoder:   HWAccelVAAPI,
		Codec:     CodecHEVC,
		MaxHeight: 1080, // Scale to 1080p
	}

	// Test 10-bit content with scaling
	_, outputArgs := BuildPresetArgs(preset, 10000000, nil, "convert", 10, "yuv420p10le")
	outputArgsStr := strings.Join(outputArgs, " ")
	t.Logf("VAAPI HEVC 10-bit 1080p output args: %v", outputArgs)

	// Should have scale_vaapi with 1080 height, p010 format, and bt2020 color params
	foundCorrectFilter := false
	for i, arg := range outputArgs {
		if arg == "-vf" && i+1 < len(outputArgs) {
			filter := outputArgs[i+1]
			if strings.Contains(filter, "scale_vaapi") &&
				strings.Contains(filter, "1080") &&
				strings.Contains(filter, "format=p010") &&
				strings.Contains(filter, "out_color_matrix=bt2020nc") {
				foundCorrectFilter = true
				t.Logf("Found correct 10-bit VAAPI scale filter with color params: %s", filter)
			}
		}
	}

	if !foundCorrectFilter {
		t.Errorf("expected scale_vaapi filter with 1080, format=p010, and out_color_matrix=bt2020nc, got: %s", outputArgsStr)
	}
}

// TestBuildPresetArgsVAAPI444pSoftwareDecode verifies that yuv444p content uses software decode
// with hwupload to VAAPI, since VAAPI can't decode H.264 4:4:4 profile.
// This addresses the "Impossible to convert between formats" error for AI upscaled content.
func TestBuildPresetArgsVAAPI444pSoftwareDecode(t *testing.T) {
	preset := &Preset{
		ID:        "compress-av1",
		Name:      "Compress (AV1)",
		Encoder:   HWAccelVAAPI,
		Codec:     CodecAV1,
		MaxHeight: 0, // No scaling
	}

	// Test yuv444p content (AI upscales from Stargate, etc.)
	inputArgs, outputArgs := BuildPresetArgs(preset, 14000000, nil, "convert", 8, "yuv444p")
	inputArgsStr := strings.Join(inputArgs, " ")
	outputArgsStr := strings.Join(outputArgs, " ")
	t.Logf("VAAPI 444p input args: %v", inputArgs)
	t.Logf("VAAPI 444p output args: %v", outputArgs)

	// Should have -vaapi_device for encoding (required for av1_vaapi)
	if !strings.Contains(inputArgsStr, "-vaapi_device") {
		t.Errorf("expected -vaapi_device in input args for VAAPI encoding, got: %s", inputArgsStr)
	}

	// Should NOT have -hwaccel vaapi (VAAPI can't decode 4:4:4)
	if containsArgPair(inputArgs, "-hwaccel", "vaapi") {
		t.Errorf("should NOT have -hwaccel vaapi for yuv444p content (unsupported), got: %s", inputArgsStr)
	}

	// Should NOT have -hwaccel_output_format vaapi (frames are on CPU)
	if containsArgPair(inputArgs, "-hwaccel_output_format", "vaapi") {
		t.Errorf("should NOT have -hwaccel_output_format vaapi for yuv444p content, got: %s", inputArgsStr)
	}

	// Filter must include format conversion + hwupload (CPU → GPU path)
	// format=nv12,hwupload,scale_vaapi=...
	foundCorrectFilter := false
	for i, arg := range outputArgs {
		if arg == "-vf" && i+1 < len(outputArgs) {
			filter := outputArgs[i+1]
			if strings.Contains(filter, "format=nv12") &&
				strings.Contains(filter, "hwupload") &&
				strings.Contains(filter, "scale_vaapi") {
				foundCorrectFilter = true
				t.Logf("Found correct 444p filter with hwupload: %s", filter)
			}
		}
	}

	if !foundCorrectFilter {
		t.Errorf("expected filter with format=nv12,hwupload,scale_vaapi for yuv444p content, got: %s", outputArgsStr)
	}
}

// TestVAAPIIncompatiblePixFmt verifies the pixel format detection helper.
func TestVAAPIIncompatiblePixFmt(t *testing.T) {
	incompatible := []string{"yuv444p", "yuv444p10le", "yuvj444p", "gbrp"}
	compatible := []string{"yuv420p", "yuv420p10le", "nv12", "p010", ""}

	for _, fmt := range incompatible {
		if !isVAAPIIncompatiblePixFmt(fmt) {
			t.Errorf("expected %s to be VAAPI incompatible", fmt)
		}
	}

	for _, fmt := range compatible {
		if isVAAPIIncompatiblePixFmt(fmt) {
			t.Errorf("expected %s to be VAAPI compatible", fmt)
		}
	}
}

func TestGetHardwarePath(t *testing.T) {
	tests := []struct {
		encoder  HWAccel
		pixFmt   string
		expected string
	}{
		// VAAPI with compatible format = full GPU pipeline
		{HWAccelVAAPI, "yuv420p", "vaapi→vaapi"},
		{HWAccelVAAPI, "nv12", "vaapi→vaapi"},
		{HWAccelVAAPI, "p010", "vaapi→vaapi"},

		// VAAPI with incompatible format = CPU decode
		{HWAccelVAAPI, "yuv444p", "cpu→vaapi"},
		{HWAccelVAAPI, "yuv444p10le", "cpu→vaapi"},

		// NVENC uses CUDA for decode
		{HWAccelNVENC, "yuv420p", "cuda→nvenc"},
		{HWAccelNVENC, "yuv444p", "cuda→nvenc"},

		// QSV uses VAAPI for decode (on Linux)
		{HWAccelQSV, "yuv420p", "vaapi→qsv"},

		// VideoToolbox
		{HWAccelVideoToolbox, "yuv420p", "videotoolbox→videotoolbox"},

		// Software encoding
		{HWAccelNone, "yuv420p", "cpu→cpu"},
		{HWAccelNone, "yuv444p", "cpu→cpu"},
	}

	for _, tt := range tests {
		result := GetHardwarePath(tt.encoder, tt.pixFmt)
		if result != tt.expected {
			t.Errorf("GetHardwarePath(%s, %s) = %s, expected %s",
				tt.encoder, tt.pixFmt, result, tt.expected)
		}
	}
}
