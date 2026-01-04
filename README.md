# Shrinkray

> **This is a community fork of [gwlsn/shrinkray](https://github.com/gwlsn/shrinkray).**
>
> The original project focuses on UI polish, scheduling, and quality controls.
> This fork focuses on hardware encoding reliability and auth.
>
> **Choose the original if you want:** A simpler click and go experience.
>
> **Choose this fork if you need:** VAAPI fixes for Intel Arc/AMD, authentication (OIDC/password), ntfy notifications

A simple video transcoding tool for Unraid. Select a folder, pick a preset, and shrink your media library.

## Fork Differences

If you're running Intel Arc, AMD VAAPI, or need authentication—use this fork.
Otherwise use [the original](https://github.com/gwlsn/shrinkray).

## Quick Start (Unraid)

1. Install from Community Applications (search "Shrinkray") or add manually:
   - **Repository**: `ghcr.io/jesposito/shrinkray:latest`
   - **WebUI**: `8080`
   - **Volumes**: `/config` → appdata, `/media` → your media library
   - **Optional**: `/temp` → fast storage for temp file

2. Open the WebUI, browse to a folder, select files, and click **Start Transcode**

## Quick Start (Docker)

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

**Optional**: For better performance, mount fast storage for temp files:

```bash
  -v /path/to/fast/storage:/temp
```

For hardware acceleration, add the appropriate device:

```bash
# Intel QSV / AMD VAAPI
--device /dev/dri:/dev/dri

# NVIDIA (requires Nvidia-Driver plugin on Unraid)
--runtime=nvidia --gpus all
```

## Presets

| Preset | Description |
|--------|-------------|
| **Compress (HEVC)** | Re-encode to H.265 (HEVC) |
| **Compress (AV1)** | Re-encode to AV1 |
| **1080p** | Downscale to 1080p + HEVC |
| **720p** | Downscale to 720p + HEVC |

All presets copy audio, and subtitle streams are copied unless a `mov_text` subtitle needs conversion for MKV output.

## Hardware Acceleration

Automatically detected and used when available:

| Platform | Encoder |
|----------|---------|
| Intel | Quick Sync (QSV) |
| Intel Arc | VAAPI |
| NVIDIA | NVENC |
| AMD (Linux) | VAAPI |
| macOS | VideoToolbox |

Falls back to software encoding if no hardware is available.

### Intel Arc GPU Setup (Docker/Unraid)

For Intel Arc A380, A770, B580 and similar GPUs:

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
- `--device /dev/dri:/dev/dri` - Pass GPU device to container
- `--group-add render` - Add render group permissions (may need GID like `105`)
- `-e LIBVA_DRIVER_NAME=iHD` - Intel Arc requires the iHD driver

**Verify GPU access:**
```bash
docker exec -it shrinkray vainfo
```

### Troubleshooting VAAPI

| Error | Cause | Fix |
|-------|-------|-----|
| Exit code 218 mid-encode | 10-bit/HDR format mismatch | Update to latest version (auto-detects bit depth) |
| "auto_scale_0" filter error | Missing VAAPI filter chain | Update to latest version (fixed) |
| "Cannot open DRM render node" | No GPU access | Add `--device /dev/dri:/dev/dri` |
| "vaInitialize failed" | Wrong driver | Set `LIBVA_DRIVER_NAME=iHD` for Intel Arc |
| "Permission denied" | Render group missing | Add `--group-add render` or GID |

## Settings

Access via the gear icon in the WebUI:

- **Original files**: Delete after transcode, or keep as `.old`
- **Concurrent jobs**: 1-6 simultaneous transcodes
- **Allow CPU encode fallback**: Retry failed GPU encodes with CPU (off by default)
- **Pushover notifications**: Get notified when all jobs complete
- **ntfy notifications**: Send notifications to an ntfy topic

### CPU Encode Fallback

By default, if a GPU encode fails, the job fails with a clear error message. This is intentional—on systems with working VAAPI (Intel Arc, AMD), GPU encodes should succeed. A failure usually indicates a configuration problem that should be fixed.

Enable **"Allow CPU encode fallback"** only if:
- A small number of files fail due to unusual codecs or formats
- You want those files transcoded anyway, even if slower
- You've verified your GPU is working correctly for other files

When enabled, failed GPU encodes automatically retry with CPU encoding (slower but more compatible). The job status will show "Retrying with CPU encode" so you know what happened.

**Unraid + Intel Arc users**: Keep this OFF. If encodes are failing, check your VAAPI setup first (see Troubleshooting VAAPI section).

## Pushover Notifications

1. Create an app at [pushover.net](https://pushover.net)
2. Enter your **User Key** and **App Token** in Settings
3. Check **"Notify when done"** in the queue header before starting jobs

You'll receive a notification with job counts and total space saved when the queue empties.

## ntfy Notifications

1. Pick an ntfy server (default: `https://ntfy.sh`) and topic
2. Enter the server, topic, and optional token in Settings
3. Check **"Notify when done"** in the queue header before starting jobs

You'll receive a notification with job counts and total space saved when the queue empties.

## Configuration

Config is stored in `/config/shrinkray.yaml`. Most settings are available in the WebUI, but you can also edit the file directly:

| Setting | Default | Description |
|---------|---------|-------------|
| `media_path` | `/media` | Root directory to browse for media files |
| `temp_path` | *(empty)* | Directory for temp files during transcode. If empty, uses same directory as source file |
| `original_handling` | `replace` | What to do with originals: `replace` (delete) or `keep` (rename to `.old`) |
| `subtitle_handling` | `convert` | How to handle unsupported subtitles: `convert` (convert `mov_text` to SRT) or `drop` (remove unsupported subtitles) |
| `workers` | `1` | Number of concurrent transcode jobs (1-6) |
| `ffmpeg_path` | `ffmpeg` | Path to ffmpeg binary |
| `ffprobe_path` | `ffprobe` | Path to ffprobe binary |
| `pushover_user_key` | *(empty)* | Pushover user key for notifications |
| `pushover_app_token` | *(empty)* | Pushover application token for notifications |
| `ntfy_server` | `https://ntfy.sh` | ntfy server URL for notifications |
| `ntfy_topic` | *(empty)* | ntfy topic for notifications |
| `ntfy_token` | *(empty)* | ntfy access token (optional) |
| `allow_software_fallback` | `false` | Retry failed GPU encodes with CPU (see CPU Encode Fallback) |
| `auth.enabled` | `false` | Require authentication |
| `auth.provider` | `noop` | Auth provider name (`noop`, `password`) |
| `auth.secret` | *(empty)* | HMAC secret used to sign session cookies |
| `auth.bypass_paths` | *(empty)* | Extra unauthenticated paths (comma-separated in env) |
| `auth.password.hash_algo` | `auto` | Password hash algorithm (`auto`, `bcrypt`, `argon2id`) |
| `auth.password.users` | *(empty)* | Map of usernames to password hashes |
| `auth.oidc.issuer` | *(empty)* | OIDC issuer URL |
| `auth.oidc.client_id` | *(empty)* | OIDC client ID |
| `auth.oidc.client_secret` | *(empty)* | OIDC client secret |
| `auth.oidc.redirect_url` | *(empty)* | Callback URL registered with the IdP |
| `auth.oidc.scopes` | `["openid","profile","email"]` | OIDC scopes (openid is always enforced) |
| `auth.oidc.group_claim` | *(empty)* | Claim containing group membership |
| `auth.oidc.allowed_groups` | *(empty)* | Allowed groups (any match grants access) |

Example:

```yaml
media_path: /media
temp_path: /tmp/shrinkray
original_handling: replace
subtitle_handling: convert
workers: 2
```

Auth example:

```yaml
auth:
  enabled: true
  provider: password
  secret: "change-me"
  password:
    hash_algo: auto
    users:
      admin: "$2b$12$abcdefghijklmnopqrstuv1234567890abcdefghijklmnopqrstuv"
```

OIDC example:

```yaml
auth:
  enabled: true
  provider: oidc
  secret: "change-me"
  oidc:
    issuer: "https://accounts.example.com"
    client_id: "shrinkray"
    client_secret: "super-secret"
    redirect_url: "https://shrinkray.example.com/auth/callback"
    scopes: ["openid", "profile", "email", "groups"]
    group_claim: "groups"
    allowed_groups: ["media-admins", "ops"]
```

Environment overrides:

```bash
# Enable CPU fallback for edge-case files
SHRINKRAY_ALLOW_SOFTWARE_FALLBACK=true

# Authentication
SHRINKRAY_AUTH_ENABLED=1
SHRINKRAY_AUTH_PROVIDER=password
SHRINKRAY_AUTH_SECRET=change-me
SHRINKRAY_AUTH_USERS='admin:$2b$12$abcdefghijklmnopqrstuv1234567890abcdefghijklmnopqrstuv'
```

OIDC overrides:

```bash
SHRINKRAY_AUTH_ENABLED=1
SHRINKRAY_AUTH_PROVIDER=oidc
SHRINKRAY_AUTH_SECRET=change-me
SHRINKRAY_AUTH_OIDC_ISSUER=https://accounts.example.com
SHRINKRAY_AUTH_OIDC_CLIENT_ID=shrinkray
SHRINKRAY_AUTH_OIDC_CLIENT_SECRET=super-secret
SHRINKRAY_AUTH_OIDC_REDIRECT_URL=https://shrinkray.example.com/auth/callback
SHRINKRAY_AUTH_OIDC_SCOPES="openid,profile,email,groups"
SHRINKRAY_AUTH_OIDC_GROUP_CLAIM=groups
SHRINKRAY_AUTH_OIDC_ALLOWED_GROUPS="media-admins,ops"
```

### OIDC IdP Settings

Configure your identity provider with:

- **Redirect/Callback URL:** `https://<your-host>/auth/callback`
- **Grant type:** Authorization Code (OIDC)
- **Scopes:** `openid` plus any additional scopes (e.g., `profile`, `email`, `groups`)
- **Group claim:** Ensure your IdP includes the configured `group_claim` (string or array)

## Building from Source

```bash
go build -o shrinkray ./cmd/shrinkray
./shrinkray -media /path/to/media
```

Requires Go 1.22+ and FFmpeg with HEVC/AV1 support.

## Testing

### Go Unit Tests

```bash
go test ./...
```

### E2E Tests (Playwright)

End-to-end tests verify the web UI works correctly across browsers.

```bash
# Install dependencies
npm install
npx playwright install

# Run tests (requires running server)
./shrinkray -media /tmp/test-media &
npm test

# Run with UI (interactive)
npm run test:ui

# Run headed (see browser)
npm run test:headed

# View test report
npm run test:report
```

**Test suites:**
- `navigation.spec.ts` - Layout, navigation, keyboard access
- `file-browser.spec.ts` - File/folder browsing
- `presets.spec.ts` - Preset selection, help modal
- `job-queue.spec.ts` - Job display, progress, cancellation
- `settings.spec.ts` - Settings panel, toggles
- `sse-streaming.spec.ts` - Real-time updates
- `accessibility.spec.ts` - A11y checks

## Docker Image Publishing

The Docker image is automatically built and published to GitHub Container Registry (GHCR) when changes are pushed to the `main` branch.

**Image URL**: `ghcr.io/jesposito/shrinkray:latest`

**Available Tags**:
- `latest` - Always points to the most recent build from main
- `<sha>` - Short git commit SHA for specific versions (e.g., `ghcr.io/jesposito/shrinkray:abc1234`)

### Making the Package Public (Required for Unraid)

By default, GHCR packages are private. To allow Unraid to pull the image without authentication:

1. Go to your GitHub profile → **Packages**
2. Click on the `shrinkray` package
3. Click **Package settings** (right sidebar)
4. Scroll to **Danger Zone** → Click **Change visibility**
5. Select **Public** and confirm

### Unraid Setup

Use this exact image name in Unraid:

```
ghcr.io/jesposito/shrinkray:latest
```

Add a new container in Unraid with:
- **Repository**: `ghcr.io/jesposito/shrinkray:latest`
- **WebUI**: `http://[IP]:[PORT:8080]`
- **Port**: `8080` → `8080`
- **Path**: `/config` → `/mnt/user/appdata/shrinkray`
- **Path**: `/media` → `/mnt/user/` (or your media location)

For hardware acceleration, add the appropriate device mapping:
- Intel/AMD: `/dev/dri` → `/dev/dri`
- NVIDIA: Enable `--runtime=nvidia` with the Nvidia-Driver plugin
