# go-dashcam

A Go learning project for a dashcam. Captures camera frames through direct Linux V4L2 ioctl calls and pipes frames to ffmpeg.

## Current config file format

Subject to change - for instance if we have multiple cameras in the future.
```json
{
    "dashcam_title": "Andreas Dashcam",
    "output_directory": "/path/to/storage",
    "video_config": {
        "video_source": "/dev/video4",
        "width": 1280,
        "height": 720,
        "fps": 30,
        "segmented": true,
        "segment_duration": "10s",
        "pixel_format": "MJPG",
        "encoder": "software"
    }
}
```

Raspberry Pi with Camera Module 3 and the hardware H.264 encoder:
```json
{
    "dashcam_title": "Andreas Dashcam",
    "output_directory": "/path/to/storage",
    "video_config": {
        "video_source": "/dev/video0",
        "width": 1920,
        "height": 1080,
        "fps": 30,
        "segmented": true,
        "segment_duration": "10s",
        "pixel_format": "YU12",
        "encoder": "v4l2m2m",
        "encoder_device": "/dev/video11",
        "bitrate": 4000000
    }
}
```

| Field | Meaning |
|---|---|
| `dashcam_title` | The title of the dashcam in the top left corner. |
| `output_directory` | Where recordings are saved. Defaults to `videos`; created if it does not exist. |
| `segmented` | `true` splits recordings into `segment_duration` pieces; `false` records one continuous file. |
| `segment_duration` | Go duration such as `10s`, `1.5s` or `2m`. Required when `segmented` is `true`. |
| `pixel_format` | `MJPG` (default, USB webcams) or `YU12` (raw frames, e.g. libcamera on a Pi). |
| `encoder` | `software` (default, FFmpeg libx264) or `v4l2m2m` (hardware H.264, Pi 4 / Zero 2 W). |
| `encoder_device` | Hardware encoder device. Required with `v4l2m2m`. |
| `bitrate` | Bits per second for the hardware encoder. Required with `v4l2m2m`. |

## Building it

Normally
```sh
go build  ./cmd/dashcam
```

On Raspberry pi's (with libcamera) do
```sh
go build -tags libcamera ./cmd/dashcam
```

The libcamera tag makes the program open and control the camera through libc, so the `LD_PRELOAD` compat library can intercept those calls.

## Run

Requires Linux, Go matching `go.mod`, FFmpeg with `libx264`, and access to a V4L2 camera that supports configured format, size and framerate.

On Raspberry PI you also need `libcamera-v4l2` which provides `/usr/libexec/aarch64-linux-gnu/libcamera/v4l2-compat.so`

```sh
go run ./cmd/dashcam -config /path/to/config_file.json
```

Or with Raspberry pi's (pi zero 2w is tested, raspberry pi 3 and 4 likely works but are untested)
```sh
LD_PRELOAD=/usr/libexec/aarch64-linux-gnu/libcamera/v4l2-compat.so ./dashcam -config /path/to/config_file.json
```

Should `-config` be omitted it will default to try to load `dashcam_config.json` from the same PWD as the dashcam is running from

Running with `-cpuprofile cpu.pprof` will record CPU profile for performance debugging.

## Tests

```sh
go test ./...
```

These tests require no camera. Recording and stalled-FFmpeg shutdown have been checked manually; automated hardware tests are not implemented.

## Ideal layout

Current packages and planned additions are shown below. Add future packages when
their milestone starts; the camera interface and platform package wait until a
second backend or OS needs them.

```
go-dashcam/
├── cmd/
│   └── dashcam/
│       ├── main.go
│       ├── camera.go
│       ├── hardware.go
│       ├── software.go
│       └── profile.go
│
├── config/
│   └── config.go
│
├── camera/
│   ├── camera.go
│   └── v4l2/
│
├── overlay/
│   ├── overlay.go
│   └── blend.go
│
├── video/
│   ├── ffmpeg.go
│   ├── input.go
│   └── v4l2m2m/
│
├── recording/
│   ├── recording.go
│   ├── cleanup.go
│   ├── monitor.go
│   └── diskspace_linux.go
│
├── gps/
│
├── stream/
│
└── platform/
```

`main` connects the packages. Lower-level packages do not import `main` or the
application-wide `config` package. Overlay drawing stays independent of the
encoder so it can support both decoded MJPEG and packed YU12 frames.

## Milestones

- [x] Capture frames in Go and deliver them to FFmpeg for MP4 recording.
- [x] Add a configurable title overlay.
- [x] Add configurable recording segments or continuous recording without splitting, with oldest completed recordings deleted when storage runs low.
- [ ] Read and validate GPS location and speed.
- [ ] Add GPS overlays.
- [ ] Stream to YouTube and configurable destinations, with reconnection after outages.
- [ ] Add optional microphone audio with video synchronization.

## Version goals

- **v0.1:** Title overlay drawn in Go, configurable recording segments or continuous recording, and oldest-file cleanup.
- **v0.2:** GPS input and location/speed overlay.
- **v0.3:** YouTube and other FFmpeg-compatible streaming destinations, including reconnection. Supply silent audio for YouTube when no microphone is available.
- **v0.4:** Optional microphone audio synchronized with video.
- **v0.5 (tentative):** Web UI and/or API for configuring settings.

## Targets

Development currently uses Linux. Raspberry Pi 2W and Raspberry Pi 4 are the first targets. Raspberry Pi5 needs software mode (no h264 encoder block)

Long-term goals include Linux and FreeBSD on ARM64, AMD64, and RISC-V. These are intended targets, not verified support.

## Supported platforms

- Raspberry Pi zero 2w - at 1080p with 30 fps (~7% CPU usage in total). Pi 3 expected to work as well since it has the same CPU and even more ram.
- amd64/x86_64 with usb webcamera.

