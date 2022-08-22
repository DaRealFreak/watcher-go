package vimeo

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/raven"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

type Media struct {
	ID            string      `json:"id"`
	BaseURL       string      `json:"base_url"`
	Format        string      `json:"format"`
	MimeType      string      `json:"mime_type"`
	Bitrate       json.Number `json:"bitrate"`
	AvgBitrate    json.Number `json:"avg_bitrate"`
	Width         json.Number `json:"width"`
	Height        json.Number `json:"height"`
	InitSegment   string      `json:"init_segment"`
	IndexSegments string      `json:"index_segment"`
	Segments      []*struct {
		Start json.Number `json:"start"`
		End   json.Number `json:"end"`
		URL   string      `json:"url"`
		Size  json.Number `json:"size"`
	} `json:"segments"`
}

type MasterJsonContent struct {
	ClipID  string   `json:"clip_id"`
	BaseURL string   `json:"base_url"`
	Video   []*Media `json:"video"`
	Audio   []*Media `json:"audio"`
}

func (c *MasterJsonContent) GetBestVideo() (bestMedia *Media) {
	bestBitrate := int64(-1)

	for _, media := range c.Video {
		bitrate, _ := media.Bitrate.Int64()
		if bitrate > bestBitrate {
			bestBitrate = bitrate
			bestMedia = media
		}
	}

	return bestMedia
}

func (c *MasterJsonContent) GetBestAudio() (bestMedia *Media) {
	bestBitrate := int64(-1)

	for _, media := range c.Audio {
		bitrate, _ := media.Bitrate.Int64()
		if bitrate > bestBitrate {
			bestBitrate = bitrate
			bestMedia = media
		}
	}

	return bestMedia
}

func (m *vimeo) parseVideo(item *models.TrackedItem, masterJSONUrl string, videoTitle string) error {
	baseURL, err := m.getBaseURL(masterJSONUrl)
	if err != nil {
		return err
	}

	res, masterErr := m.Session.Get(masterJSONUrl)
	if masterErr != nil {
		return masterErr
	}

	out, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return readErr
	}

	var masterJsonContent MasterJsonContent
	if marshalErr := json.Unmarshal(out, &masterJsonContent); err != nil {
		return marshalErr
	}

	ref, urlErr := url.Parse(masterJsonContent.BaseURL)
	if urlErr != nil {
		return urlErr
	}

	baseURL = baseURL.ResolveReference(ref)

	if err = m.downloadVideo(masterJsonContent, baseURL, videoTitle); err != nil {
		return err
	}

	m.DbIO.ChangeTrackedItemCompleteStatus(item, true)

	return nil
}

func (m *vimeo) downloadVideo(content MasterJsonContent, baseURL *url.URL, videoTitle string) error {
	finalFilePath := path.Join(
		viper.GetString("download.directory"),
		m.Key,
		fp.TruncateMaxLength(fp.SanitizePath(videoTitle, false))+".mp4",
	)
	m.Session.EnsureDownloadDirectory(finalFilePath)

	videoFile, err := ioutil.TempFile("", ".*.mp4")
	if err != nil {
		return err
	}

	defer raven.CheckPathRemoval(videoFile.Name())

	var audioFile *os.File
	if err = m.downloadMedia(videoFile, content.GetBestVideo(), *baseURL); err != nil {
		return err
	}

	if err = videoFile.Close(); err != nil {
		return err
	}

	if content.GetBestAudio() != nil {
		audioFile, err = ioutil.TempFile("", ".*.mp3")
		if err != nil {
			return err
		}

		defer raven.CheckPathRemoval(audioFile.Name())

		if err = m.downloadMedia(audioFile, content.GetBestAudio(), *baseURL); err != nil {
			return err
		}

		if err = audioFile.Close(); err != nil {
			return err
		}

		return m.mergeVideoWithAudio(videoFile.Name(), audioFile.Name(), finalFilePath)
	} else {
		if _, err = os.Stat(finalFilePath); err == nil {
			if err = os.Remove(finalFilePath); err != nil {
				return err
			}
		}

		if err = fp.MoveFile(videoFile.Name(), finalFilePath); err != nil {
			return err
		}
	}

	return nil
}

func (m *vimeo) downloadMedia(file *os.File, media *Media, baseURL url.URL) error {
	mediaBaseURL := &baseURL
	partUrl, err := url.Parse(media.BaseURL)
	if err != nil {
		return err
	}

	mediaBaseURL = mediaBaseURL.ResolveReference(partUrl)
	initSegment, decodeErr := base64.StdEncoding.DecodeString(media.InitSegment)
	if decodeErr != nil {
		return decodeErr
	}

	if _, err = file.Write(initSegment); err != nil {
		return err
	}

	for _, segment := range media.Segments {
		singleSegmentUrl := &*mediaBaseURL
		ref, urlParseErr := url.Parse(segment.URL)
		if urlParseErr != nil {
			return urlParseErr
		}

		singleSegmentUrl = singleSegmentUrl.ResolveReference(ref)
		res, downloadErr := m.Session.Get(singleSegmentUrl.String())
		if downloadErr != nil {
			return downloadErr
		}

		content, readErr := ioutil.ReadAll(res.Body)
		if readErr != nil {
			return readErr
		}

		if _, err = file.Write(content); err != nil {
			return err
		}
	}

	return nil
}

func (m *vimeo) getBaseURL(masterJSONUrl string) (*url.URL, error) {
	masterJSON, err := url.Parse(masterJSONUrl)
	if err != nil {
		return nil, err
	}

	ref, _ := url.Parse("./")

	// remove master.json part
	return masterJSON.ResolveReference(ref), nil
}

func (m *vimeo) mergeVideoWithAudio(videoFilePath string, audioFilePath string, finalFilePath string) error {
	executable := "ffmpeg"
	args := []string{
		"-i",
		videoFilePath,
		"-i", audioFilePath,
		"-y", "-c:v", "copy", "-c:a", "aac", "-map", "0:v:0", "-map", "1:a:0", finalFilePath,
	}

	log.Debugf("running command: %s %s", executable, strings.Join(args, " "))

	cmd := exec.Command(executable, args...)
	err := cmd.Start()
	if err == nil {
		err = cmd.Wait()
	}

	return err
}
