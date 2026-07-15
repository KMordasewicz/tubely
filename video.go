package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

type FFprobeOutput struct {
	Streams []struct {
		Index              int    `json:"index,omitempty"`
		CodecName          string `json:"codec_name,omitempty"`
		CodecLongName      string `json:"codec_long_name,omitempty"`
		Profile            string `json:"profile,omitempty"`
		CodecType          string `json:"codec_type,omitempty"`
		CodecTagString     string `json:"codec_tag_string,omitempty"`
		CodecTag           string `json:"codec_tag,omitempty"`
		Width              int    `json:"width,omitempty"`
		Height             int    `json:"height,omitempty"`
		CodedWidth         int    `json:"coded_width,omitempty"`
		CodedHeight        int    `json:"coded_height,omitempty"`
		ClosedCaptions     int    `json:"closed_captions,omitempty"`
		FilmGrain          int    `json:"film_grain,omitempty"`
		HasBFrames         int    `json:"has_b_frames,omitempty"`
		SampleAspectRatio  string `json:"sample_aspect_ratio,omitempty"`
		DisplayAspectRatio string `json:"display_aspect_ratio,omitempty"`
		PixFmt             string `json:"pix_fmt,omitempty"`
		Level              int    `json:"level,omitempty"`
		ColorRange         string `json:"color_range,omitempty"`
		ColorSpace         string `json:"color_space,omitempty"`
		ColorTransfer      string `json:"color_transfer,omitempty"`
		ColorPrimaries     string `json:"color_primaries,omitempty"`
		ChromaLocation     string `json:"chroma_location,omitempty"`
		FieldOrder         string `json:"field_order,omitempty"`
		Refs               int    `json:"refs,omitempty"`
		IsAvc              string `json:"is_avc,omitempty"`
		NalLengthSize      string `json:"nal_length_size,omitempty"`
		ID                 string `json:"id,omitempty"`
		RFrameRate         string `json:"r_frame_rate,omitempty"`
		AvgFrameRate       string `json:"avg_frame_rate,omitempty"`
		TimeBase           string `json:"time_base,omitempty"`
		StartPts           int    `json:"start_pts,omitempty"`
		StartTime          string `json:"start_time,omitempty"`
		DurationTs         int    `json:"duration_ts,omitempty"`
		Duration           string `json:"duration,omitempty"`
		BitRate            string `json:"bit_rate,omitempty"`
		BitsPerRawSample   string `json:"bits_per_raw_sample,omitempty"`
		NbFrames           string `json:"nb_frames,omitempty"`
		ExtradataSize      int    `json:"extradata_size,omitempty"`
		SampleFmt          string `json:"sample_fmt,omitempty"`
		SampleRate         string `json:"sample_rate,omitempty"`
		Channels           int    `json:"channels,omitempty"`
		ChannelLayout      string `json:"channel_layout,omitempty"`
		BitsPerSample      int    `json:"bits_per_sample,omitempty"`
		InitialPadding     int    `json:"initial_padding,omitempty"`
		Disposition        struct {
			Default         int `json:"default,omitempty"`
			Dub             int `json:"dub,omitempty"`
			Original        int `json:"original,omitempty"`
			Comment         int `json:"comment,omitempty"`
			Lyrics          int `json:"lyrics,omitempty"`
			Karaoke         int `json:"karaoke,omitempty"`
			Forced          int `json:"forced,omitempty"`
			HearingImpaired int `json:"hearing_impaired,omitempty"`
			VisualImpaired  int `json:"visual_impaired,omitempty"`
			CleanEffects    int `json:"clean_effects,omitempty"`
			AttachedPic     int `json:"attached_pic,omitempty"`
			TimedThumbnails int `json:"timed_thumbnails,omitempty"`
			NonDiegetic     int `json:"non_diegetic,omitempty"`
			Captions        int `json:"captions,omitempty"`
			Descriptions    int `json:"descriptions,omitempty"`
			Metadata        int `json:"metadata,omitempty"`
			Dependent       int `json:"dependent,omitempty"`
			StillImage      int `json:"still_image,omitempty"`
		} `json:"disposition,omitempty"`
		Tags struct {
			Language    string `json:"language,omitempty"`
			HandlerName string `json:"handler_name,omitempty"`
			VendorID    string `json:"vendor_id,omitempty"`
			Encoder     string `json:"encoder,omitempty"`
			Timecode    string `json:"timecode,omitempty"`
		} `json:"tags,omitempty"`
	} `json:"streams,omitempty"`
}

func isEqualRatioWithError(ratio, targetRatio, tolerance float64) bool {
	return math.Abs(ratio-targetRatio) < (targetRatio * tolerance)
}

func calcAspectRatio(width, hight int) string {
	const (
		horizontalRatio = 16.0 / 9.0
		verticalRatio   = 9.0 / 16.0
		errorTolerance  = 0.1
	)
	ratio := float64(width) / float64(hight)

	if isEqualRatioWithError(ratio, horizontalRatio, errorTolerance) {
		return "16:9"
	} else if isEqualRatioWithError(ratio, verticalRatio, errorTolerance) {
		return "9:16"
	} else {
		return "other"
	}
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v",
		"error",
		"-print_format",
		"json",
		"-show_streams",
		filePath,
	)

	var resultsBuffer bytes.Buffer
	cmd.Stdout = &resultsBuffer

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running ffprobe command: %v", err)
	}

	var ffprobeResult FFprobeOutput
	if err = json.Unmarshal(resultsBuffer.Bytes(), &ffprobeResult); err != nil {
		return "", fmt.Errorf("error unmarshaling ffprobe output: %v", err)
	}

	if len(ffprobeResult.Streams) == 0 {
		return "", errors.New("no video streams found")
	}

	return calcAspectRatio(ffprobeResult.Streams[0].Width, ffprobeResult.Streams[0].Height), nil
}

func getVideoPrefix(aspectRatio string) string {
	switch aspectRatio {
	case "16:9":
		return "landscape"
	case "9:16":
		return "portrait"
	default:
		return "other"
	}
}

func processVideoForFastStart(filePath string) (string, error) {
	newFilePath := filePath + ".processing"

	cmd := exec.Command(
		"ffmpeg",
		"-i",
		filePath,
		"-c",
		"copy",
		"-movflags",
		"faststart",
		"-f",
		"mp4",
		newFilePath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error running ffmpeg command: %s %v", stderr.String(), err)
	}

	fileInfo, err := os.Stat(newFilePath)
	if err != nil {
		return "", fmt.Errorf("could not stat processed file: %v", err)
	}
	if fileInfo.Size() == 0 {
		return "", fmt.Errorf("processed file is empty")
	}

	return newFilePath, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		log.Printf("No url to signed for video id: %v\n", video.ID)
		return video, nil
	}

	urlContent := strings.Split(*video.VideoURL, ",")
	if len(urlContent) != 2 {
		return video, fmt.Errorf("incorrect video url, should be bucket,key format, instead %s", *video.VideoURL)
	}
	bucket := urlContent[0]
	key := urlContent[1]

	presignURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Hour)
	if err != nil {
		return video, err
	}
	video.VideoURL = &presignURL

	return video, nil
}
