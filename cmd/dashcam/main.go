package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
	"github.com/dinapappor/go-dashcam/camera/v4l2"
)

func main() {
	// Flags
	videoDevice := flag.String("video_device", "/dev/video0", "Which video device to use")

	output := flag.String("output", "video.mp4", "Which filename to output to")

	//debug := flag.Bool("debug", false, "Whether or not to print debug flags")


	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Look for ffmpeg
	ffmpeg, err :=  exec.LookPath("ffmpeg")
	if err != nil {
		log.Printf("Could not find ffmpeg in path")
		return
	}

	log.Printf("FFmpeg path: %s", ffmpeg)

	cmd := exec.Command(ffmpeg,
		"-hide_banner",
		"-f", "mjpeg",
		"-framerate", "30",
		"-i", "pipe:0",
		"-n",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		*output,
	)
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}


	pipeReader, ffmpegInput, err := os.Pipe()
	if err != nil {
		log.Printf("create FFmpeg pipe: %v", err)
		return
	}

	defer pipeReader.Close()
	defer ffmpegInput.Close()

	cmd.Stdin = pipeReader


	stopPipeClose := context.AfterFunc(ctx, func() {
		ffmpegInput.Close()
	})
	defer stopPipeClose()


	// open the video device
	device, err := v4l2.Open(*videoDevice)

	if err != nil {
		log.Printf("%s: %s", *videoDevice, err)
		return
	}

	defer func() {
		if err := device.Close(); err != nil {
			log.Printf("Unable to close device: %v", err)
		}
	}()

	deviceCapability, err := v4l2.QueryCapability(device.Fd())

	if err != nil {
		log.Printf("Unable to query for device Capabilities, err:%s", err)
		return
	}


	log.Printf("Device Capability struct:%+v", deviceCapability)

	driverName, err := v4l2.ConvertByteToString(deviceCapability.Driver[:])

	if err != nil {
		log.Printf("Could not get driver name, err: %s", err)
		return
	}


	cameraName, err := v4l2.ConvertByteToString(deviceCapability.Card[:])

	if err != nil {
		log.Printf("Could not get camera name, err: %s", err)
		return
	}

	busInfo, err := v4l2.ConvertByteToString(deviceCapability.BusInfo[:])

	if err != nil {
		log.Printf("Could not get bus info, err: %s", err)
		return
	}

	fmt.Printf("Video device: %s (%s), using %s driver on %s\n", cameraName, *videoDevice, driverName, busInfo)

	capabilities := v4l2.SelectCapabilities(deviceCapability)

	// Check for video capture flag
	if v4l2.HasCapability(capabilities, v4l2.VideoCapture) {
		log.Printf("We have video capture capabilities")

	} else {
		log.Printf("We do not have video capture capabilities")
		return
	}

	// Check for streaming capabilities.
	// This means buffers that we queue and requeue later
	if v4l2.HasCapability(capabilities, v4l2.StreamingFlag) {
		log.Printf("We have streaming capabilities")
	} else {
		log.Printf("We do not have streaming capabilities")
		return
	}

	deviceFormat, err := v4l2.QueryVideoCaptureFormat(device.Fd())

	if err != nil {
		log.Printf("Unable to query video format: %s", err)
		return
	}

	log.Printf("%+v", deviceFormat.PixFormat)

	// Get the four CC
	fourCC := v4l2.DecodeFourCC(deviceFormat.PixFormat.PixelFormat)

	log.Printf("Pixformat: %s", fourCC)

	requestedFormat, err := v4l2.SetVideoCaptureFormat(device.Fd(), 1280, 720, v4l2.PixFmtMJPEG)

	if err != nil {
		log.Printf("Unable to set format: %s", err)
		return
	}

	log.Printf("We got format: %s, width:%d, height: %d",
		v4l2.DecodeFourCC(requestedFormat.PixFormat.PixelFormat),
		requestedFormat.PixFormat.Width,
		requestedFormat.PixFormat.Height)

	streamParams, err := v4l2.QueryVideoCaptureStreamParams(device.Fd())
	if err != nil {
		log.Printf("Unable to query for stream params, got: %s", err)
		return
	}

	log.Printf("%v", streamParams.Capture.TimePerFrame)

	if v4l2.HasCapability(streamParams.Capture.Capability, v4l2.CapTimePerFrame) {
		log.Printf("We have TimePerFrame capability")
	} else {
		log.Printf("We do not have TimePerFrame capability")
		return
	}

	requestedParams, err := v4l2.SetVideoCaptureStreamParams(device.Fd(), 1, 30)

	if err != nil {
		log.Printf("Unable to set stream params, got %s", err)
		return
	}


	log.Printf("%+v", requestedParams.Capture.TimePerFrame)

	err = device.SetupBuffers(4)

	if err != nil {
		log.Printf("Unable to setup buffers: %v", err)
		return
	}
	err = cmd.Start()

	if err != nil {
		log.Printf("Could not start FFmpeg: %v", err)
		return
	}

	pipeReader.Close()

	defer func() {
		ffmpegInput.Close()

		timer := time.AfterFunc(5*time.Second, func() {
			log.Printf("FFmpeg shutdown timed out; stopping it")
			if err := cmd.Process.Kill(); err != nil {
				log.Printf("Could not kill FFmpeg: %v", err)
			}
		})
		defer timer.Stop()

		if waitErr := cmd.Wait(); waitErr != nil {
			log.Printf("FFmpeg failed: %v", waitErr)
		}
	}()

	err = device.StreamOn()
	if err != nil {
		log.Printf("Unable to start stream: %v", err)
		return
	}

	for frameNumber := 0; frameNumber < 300 && ctx.Err() == nil; {
		// Get frames
		frame, err := device.NextFrame(ctx)

		if errors.Is(err, context.Canceled) {
			log.Printf("Shutting down")
			return
		}

		if err != nil {
			log.Printf("Unable to get frame. Got %+v", err)
			return
		}

		err = ffmpegInput.SetWriteDeadline(time.Now().Add(2 * time.Second))

		if err != nil {
			log.Printf("Could not set deadline for ffmpeg: %v", err)
			return
		}

		written, err := ffmpegInput.Write(frame.Data)

		if err != nil {
			log.Printf("Could not write data to ffmpeg: %v", err)
			return
		}

		if written != len(frame.Data) {
			log.Printf("Incomplete frame write: %d of %d bytes", written, len(frame.Data))
			return
		}

		err = device.Requeue(frame)
		if err != nil {
			log.Printf("Could not requeue the buffer: %v", err)
			return
		}
		frameNumber++
	}
}
