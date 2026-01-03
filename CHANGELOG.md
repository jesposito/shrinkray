# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Fixed
- VAAPI AV1/HEVC transcode failure with "Impossible to convert between the formats
  supported by the filter 'Parsed_null_0' and the filter 'auto_scale_0'" error.
  This is a common issue for Unraid users with Intel Arc (A380, A770, B580) or AMD
  RDNA GPUs using VAAPI hardware encoding. Root cause: FFmpeg auto-inserted software
  scaling filters when using VAAPI hardware decode, which cannot handle VAAPI memory
  surfaces. Fix adds explicit `-vf scale_vaapi=format=nv12` filter to keep frames
  on GPU and ensure proper NV12 colorspace conversion for Intel QuickSync/AMD hardware.
- VAAPI mid-encode failures (exit code 218) when transcoding 10-bit/HDR content.
  Added automatic bit depth detection and pixel format selection: uses `nv12` for
  8-bit content and `p010` for 10-bit/HDR content. This prevents format mismatch
  errors that previously caused transcodes to fail mid-stream.
- "Multiple -codec/-c/-acodec/-vcodec options specified for stream 0" warning when
  using VAAPI encoders. Changed stream mapping from `-map 0 -c:v copy -c:v:0 encoder`
  to explicit stream selectors (`-map 0:v:0 -map 0:v:1? -map 0:a? -map 0:s?`).
- Ensured `-qp` quality parameter is always set for VAAPI encoders to avoid
  "No quality level set; using default (25)" warning.

### Added
- Bit depth detection during file probing (8-bit, 10-bit, 12-bit)
- VAAPI health check diagnostics for troubleshooting GPU access issues
- Intel Arc GPU support in Docker image (`intel-media-va-driver-non-free` package)
- Environment variable `LIBVA_DRIVER_NAME=iHD` set by default for Intel Arc

### Changed
- Docker image now includes Intel media driver for Arc GPU support out of the box
- Improved error messages for VAAPI failures with specific troubleshooting hints

## [1.1.0] - 2025-12-28

### Added
- Skip files already encoded in target codec (HEVC/AV1) to prevent unnecessary transcoding
- Skip files already at target resolution when using downscale presets (1080p/720p)
- Version number displayed in Settings panel

## [1.0.0] - 2025-12-25

### Added
- Initial public release
- Hardware-accelerated transcoding (VideoToolbox, NVENC, QSV, VAAPI)
- HEVC and AV1 compression presets
- 1080p and 720p downscale presets
- Batch folder selection for entire TV series
- Async job creation to prevent UI freezes
- Pushover notifications when queue completes
- Retry button for failed jobs
- Mobile-responsive stats bar
- Queue persistence across restarts
