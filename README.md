# go-dashcam

A Go learning project for a dashcam. Captures camera frames through direct Linux V4L2 ioctl calls and pipes MJPEG to FFmpeg for H.264 recording.

## Run

Requires Linux, Go matching `go.mod`, FFmpeg with `libx264`, and access to a V4L2 camera supporting MJPEG at 1280×720 and 30 fps.

```sh
go run ./cmd/dashcam -video_device /dev/video0 -output recording.mp4
```

Currently stops after 300 frames or Ctrl+C. Existing output files are not overwritten. Device paths may change between connections.

## Tests

```sh
go test ./...
```

These tests require no camera. Recording and stalled-FFmpeg shutdown have been checked manually; automated hardware tests are not implemented.

## Ideal layout

```
dashcam/
├── cmd/
│   └── dashcam/
│       └── main.go
│
├── camera/
│   ├── camera.go
│   └── v4l2/
│
├── video/
│   ├── frame.go
│   ├── overlay.go
│   └── encoder.go
│
├── gps/
│   └── gps.go
│
├── recording/
│   └── recording.go
│
└── platform/
    └── capabilities.go
```

## Milestones

- [x] Capture frames in Go and deliver them to FFmpeg for MP4 recording.
- [ ] Add a configurable title overlay.
- [ ] Add configurable recording segments or continuous recording without splitting, with oldest completed recordings deleted when storage runs low.
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

Development currently uses Linux. Raspberry Pi 4 and 5 are the first deployment targets; the complete pipeline still needs benchmarking on both.

Long-term goals include Linux and FreeBSD on ARM64, AMD64, and RISC-V. These are intended targets, not verified support.
