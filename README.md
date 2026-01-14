<div align="center">
  <img src="web/templates/logo.png" alt="Shrinkray" width="128" height="128">
  <h1>Shrinkray</h1>
  <p><strong>Simple video transcoding for Unraid</strong></p>
  <p>Select a folder. Pick a preset. Shrink your media library.</p>

  ![Go](https://img.shields.io/badge/go-1.22+-00ADD8?style=flat-square&logo=go)
  ![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)
  ![Docker](https://img.shields.io/badge/docker-ghcr.io-2496ED?style=flat-square&logo=docker)
</div>

---

## About This Fork

This is a community fork of [gwlsn/shrinkray](https://github.com/gwlsn/shrinkray), the excellent video transcoding tool created by [@gwlsn](https://github.com/gwlsn).

Both projects share the same core goal: make video transcoding simple and accessible. This fork extends the original with additional features for users who need authentication or have specific hardware configurations.

### Which Should You Use?

| Use Case | Recommendation |
|----------|----------------|
| Simple home setup, no auth needed | [Original](https://github.com/gwlsn/shrinkray) — cleaner, more focused |
| Intel Arc GPU (A380, A770, B580) | **This fork** — extensive VAAPI fixes |
| AMD GPU on Linux | **This fork** — improved VAAPI support |
| Need authentication (OIDC/password) | **This fork** — full auth support |
| Want ntfy notifications | **This fork** — ntfy + Pushover |
| Prefer mature, stable release | [Original](https://github.com/gwlsn/shrinkray) |

Both versions include: GPU acceleration, scheduling, quality controls, batch selection, Pushover notifications, and smart skipping.

---

## Features

- **Smart Presets** — HEVC compress, AV1 compress, 1080p downscale, 720p downscale
- **Full GPU Pipeline** — Hardware decoding AND encoding (frames stay on GPU)
- **Batch Selection** — Select entire folders to transcode whole seasons at once
- **Scheduling** — Restrict transcoding to specific hours (e.g., overnight only)
- **Quality Control** — Adjustable CRF for fine-tuned compression
- **Smart Skipping** — Automatically skips files already in target codec/resolution
- **Authentication** — Password or OIDC (Authentik, Keycloak, etc.)
- **Notifications** — Pushover and ntfy support
- **CPU Fallback** — Optional automatic retry with software encoding

---

## Quick Start

### Unraid (Community Applications)

1. Search **"Shrinkray"** in Community Applications, or add manually:
   - **Repository**: `ghcr.io/jesposito/shrinkray:latest`
   - **WebUI**: `8080`
   - **Volumes**: `/config` → appdata, `/media` → your media library
2. For GPU acceleration, pass through your GPU device (see [Hardware Acceleration](#hardware-acceleration))
3. Open the WebUI at port **8080**

### Docker Compose

```yaml
services:
  shrinkray:
    image: ghcr.io/jesposito/shrinkray:latest
    container_name: shrinkray
    ports:
      - 8080:8080
    volumes:
      - /path/to/config:/config
      - /path/to/media:/media
      - /path/to/fast/storage:/temp  # Optional: SSD for temp files
    environment:
      - PUID=99
      - PGID=100
    restart: unless-stopped
```

### Docker CLI

```bash
docker run -d \
  --name shrinkray \
  -p 8080:8080 \
  -e PUID=99 \
  -e PGID=100 \
  -v /path/to/config:/config \
  -v /path/to/media:/media \
  ghcr.io/jesposito/shrinkray:latest
```

---

## Presets

| Preset | Codec | Description | Typical Savings |
|--------|-------|-------------|-----------------|
| **Compress (HEVC)** | H.265 | Re-encode to HEVC | 40–60% smaller |
| **Compress (AV1)** | AV1 | Re-encode to AV1 | 50–70% smaller |
| **1080p** | HEVC | Downscale 4K → 1080p | 60–80% smaller |
| **720p** | HEVC | Downscale to 720p | 70–85% smaller |

All presets copy audio and subtitles unchanged (stream copy).

---

## Hardware Acceleration

Shrinkray automatically detects and uses the best available hardware encoder—no configuration required, just pass through your GPU.

### Supported Hardware

| Platform | Requirements | Docker Flags |
|----------|--------------|--------------|
| **NVIDIA (NVENC)** | GTX 1050+ / RTX series | `--runtime=nvidia --gpus all` |
| **Intel Quick Sync** | 6th gen+ CPU | `--device /dev/dri:/dev/dri` |
| **Intel Arc** | Arc A-series / B-series | `--device /dev/dri:/dev/dri` + see below |
| **AMD (VAAPI)** | Polaris+ GPU on Linux | `--device /dev/dri:/dev/dri` |
| **Apple (VideoToolbox)** | Any Mac (M1/M2/M3 or Intel) | Native (no Docker) |

### Intel Arc GPU Setup

Intel Arc GPUs (A380, A770, B580, etc.) require specific configuration:

```bash
docker run -d \
  --name shrinkray \
  --device /dev/dri:/dev/dri \
  --group-add render \
  -e LIBVA_DRIVER_NAME=iHD \
  -e PUID=99 \
  -e PGID=100 \
  -p 8080:8080 \
  -v /path/to/config:/config \
  -v /path/to/media:/media \
  ghcr.io/jesposito/shrinkray:latest
```

Key settings:
- `--device /dev/dri:/dev/dri` — Pass GPU device to container
- `--group-add render` — Add render group permissions (may need GID like `105`)
- `-e LIBVA_DRIVER_NAME=iHD` — Intel Arc requires the iHD driver

Verify GPU access:
```bash
docker exec -it shrinkray vainfo
```

### AV1 Hardware Requirements

AV1 hardware encoding requires newer GPUs:

| Platform | Minimum Hardware |
|----------|------------------|
| **NVIDIA** | RTX 40 series (Ada Lovelace) |
| **Intel** | Arc GPUs, 14th gen+ iGPUs |
| **Apple** | M3 chip or newer |
| **AMD** | RX 7000 series (RDNA 3) |

### VAAPI Troubleshooting

| Error | Cause | Fix |
|-------|-------|-----|
| Exit code 218 mid-encode | 10-bit/HDR format mismatch | Update to latest version (auto-detects bit depth) |
| "auto_scale_0" filter error | Missing VAAPI filter chain | Update to latest version (fixed) |
| "Cannot open DRM render node" | No GPU access | Add `--device /dev/dri:/dev/dri` |
| "vaInitialize failed" | Wrong driver | Set `LIBVA_DRIVER_NAME=iHD` for Intel Arc |
| "Permission denied" | Render group missing | Add `--group-add render` or GID |
| MPEG4/XVID decode failures | Legacy codec VAAPI issue | Update to latest version (fixed) |

---

## Scheduling

Restrict transcoding to specific hours to reduce system load during the day.

1. Open **Settings** (gear icon)
2. Enable **Schedule transcoding**
3. Set start and end hours (e.g., 22:00 – 06:00 for overnight)

**Behavior:**
- Jobs can always be added to the queue
- Transcoding only runs during the allowed window
- Running jobs complete even if the window closes
- Jobs automatically resume when the window reopens

---

## Authentication

This fork supports optional authentication to protect your Shrinkray instance.

### Password Authentication

```yaml
auth:
  enabled: true
  provider: password
  secret: "your-random-secret-here"
  password:
    hash_algo: auto
    users:
      admin: "$2b$12$..." # bcrypt hash
```

Generate a password hash:
```bash
htpasswd -nbB admin yourpassword | cut -d: -f2
```

### OIDC Authentication

Works with Authentik, Keycloak, Authelia, and other OIDC providers:

```yaml
auth:
  enabled: true
  provider: oidc
  secret: "your-random-secret-here"
  oidc:
    issuer: "https://auth.example.com"
    client_id: "shrinkray"
    client_secret: "your-client-secret"
    redirect_url: "https://shrinkray.example.com/auth/callback"
    scopes: ["openid", "profile", "email", "groups"]
    group_claim: "groups"
    allowed_groups: ["media-admins"]
```

Configure your IdP with:
- **Redirect URL:** `https://<your-host>/auth/callback`
- **Grant type:** Authorization Code

Environment variable overrides are also supported—see [Configuration](#configuration).

---

## Notifications

### Pushover

1. Create an app at [pushover.net](https://pushover.net)
2. Enter your **User Key** and **App Token** in Settings
3. Check **"Notify when done"** before starting jobs

### ntfy

1. Pick a server (default: `https://ntfy.sh`) and topic
2. Enter the server, topic, and optional token in Settings
3. Check **"Notify when done"** before starting jobs

Notifications include job counts and total space saved when the queue empties.

---

## Configuration

Config is stored in `/config/shrinkray.yaml`. Most settings are available in the WebUI.

| Setting | Default | Description |
|---------|---------|-------------|
| `media_path` | `/media` | Root directory to browse |
| `temp_path` | *(empty)* | Fast storage for temp files (SSD recommended) |
| `original_handling` | `replace` | `replace` = delete original, `keep` = rename to `.old` |
| `subtitle_handling` | `convert` | `convert` or `drop` unsupported subtitles |
| `workers` | `1` | Concurrent transcode jobs (1–6) |
| `quality_hevc` | `0` | CRF override for HEVC (0 = default, 15–40) |
| `quality_av1` | `0` | CRF override for AV1 (0 = default, 20–50) |
| `schedule_enabled` | `false` | Enable time-based scheduling |
| `schedule_start_hour` | `22` | Hour transcoding may start (0–23) |
| `schedule_end_hour` | `6` | Hour transcoding must stop (0–23) |
| `allow_software_fallback` | `false` | Retry failed GPU encodes with CPU |
| `pushover_user_key` | *(empty)* | Pushover user key |
| `pushover_app_token` | *(empty)* | Pushover app token |
| `ntfy_server` | `https://ntfy.sh` | ntfy server URL |
| `ntfy_topic` | *(empty)* | ntfy topic |
| `ntfy_token` | *(empty)* | ntfy access token (optional) |

### Environment Variables

All settings can be overridden with environment variables using the `SHRINKRAY_` prefix:

```bash
SHRINKRAY_WORKERS=2
SHRINKRAY_ALLOW_SOFTWARE_FALLBACK=true
SHRINKRAY_AUTH_ENABLED=1
SHRINKRAY_AUTH_PROVIDER=password
SHRINKRAY_AUTH_SECRET=change-me
```

---

## CPU Fallback

By default, if a GPU encode fails, the job fails with a clear error message. This is intentional—GPU encodes should succeed on properly configured systems.

Enable **"Allow CPU encode fallback"** only if:
- A small number of files fail due to unusual codecs
- You want those files transcoded anyway, even if slower
- You've verified your GPU is working correctly for other files

When enabled, failed GPU encodes automatically retry with software encoding.

---

## Building from Source

```bash
git clone https://github.com/jesposito/shrinkray.git
cd shrinkray

go build -o shrinkray ./cmd/shrinkray
./shrinkray -media /path/to/media
```

**Requirements:** Go 1.22+, FFmpeg with HEVC/AV1 support

### Running Tests

```bash
# Go unit tests
go test ./...

# E2E tests (Playwright)
npm install && npx playwright install
./shrinkray -media /tmp/test-media &
npm test
```

---

## Acknowledgments

This project is built on the excellent work of [@gwlsn](https://github.com/gwlsn) and the original [shrinkray](https://github.com/gwlsn/shrinkray) project. Thank you for creating such a useful tool and making it open source.

Additional contributions from [@akaBilih](https://github.com/akaBilih):
- **Authentication system** — Complete OIDC and password authentication implementation 
- **Tabbed layout** — Optional tabbed UI with activity badges for queue and active jobs
- **Queue management** — Manual reordering, processed history tracking, smart duplicate detection
- **UI enhancements** — Collapsible panels, virtual scroll performance, directory history navigation, sorting and filtering controls
- **Job features** — Elapsed time display with clock skew handling, processed indicators, bulk selection with exclusion of already processed files
- **Progress streaming** — Improved SSE reliability and ffmpeg progress parsing with fallback support
- **File management** — Video info modal with probe details, tmp file hiding, processed file tracking
- **Configuration** — Config file watcher for auto-reload
- **Notifications** — ntfy support
- **Developer experience** — Docker build updates, error handling improvements, numerous bug fixes

---

## License

MIT License — see [LICENSE](LICENSE) for details.
