package plugins

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"gobot/lib"
)

type YTSearchItem struct {
	Type        string `json:"type"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	Thumbnail   string `json:"thumbnail"`
	Views       int64  `json:"views"`
	Author      struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"author"`
}

type YTAudioResponse struct {
	Status bool `json:"status"`
	Result struct {
		Download  string `json:"download"`
		Thumbnail string `json:"thumbnail"`
		Title     string `json:"title"`
	} `json:"result"`
}

type YTVideoResponse struct {
	Status bool `json:"status"`
	Result struct {
		Download string `json:"download"`
		Title    string `json:"title"`
	} `json:"result"`
}

func getJSON(apiURL string, target interface{}) error {
	resp, err := http.Get(apiURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %s", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func downloadBuffer(fileURL string) ([]byte, error) {
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func init() {
	lib.Function(map[string]interface{}{
		"pattern": "song ?(.*)",
		"fromMe":  lib.Mode(),
		"desc":    "Download audio from YouTube",
		"type":    "download",
	}, func(message *lib.Message, match string) {
		if match == "" && message.Quoted != nil {
			match = message.Quoted.Text
		}
		if match == "" {
			message.Reply("_Need URL or song name!_\n*Example: .song URL/song name*")
			return
		}

		var videoURL string
		if strings.Contains(match, "youtu") {
			videoURL = match
		} else {
			var searchResults []YTSearchItem
			err := getJSON(fmt.Sprintf("https://api-25ca.onrender.com/api/yts?q=%s", url.QueryEscape(match)), &searchResults)
			if err != nil || len(searchResults) == 0 {
				message.Reply("_No results found_")
				return
			}
			videoURL = searchResults[0].URL
		}

		message.Reply("_Downloading audio..._")

		var audio YTAudioResponse
		err := getJSON(fmt.Sprintf("https://api-25ca.onrender.com/api/yta?url=%s&format=mp3", url.QueryEscape(videoURL)), &audio)
		if err != nil || !audio.Status {
			message.Reply("_Failed to download audio_")
			return
		}

		data, err := downloadBuffer(audio.Result.Download)
		if err != nil {
			message.Reply(fmt.Sprintf("_Error downloading audio: %v_", err))
			return
		}

		message.Send("audio", data, lib.SendOptions{
			Caption:  audio.Result.Title,
			Mimetype: "audio/mpeg",
			Quoted:   true,
		})
	})

	lib.Function(map[string]interface{}{
		"pattern": "video ?(.*)",
		"fromMe":  lib.Mode(),
		"desc":    "Download video from YouTube",
		"type":    "download",
	}, func(message *lib.Message, match string) {
		if match == "" && message.Quoted != nil {
			match = message.Quoted.Text
		}
		if match == "" {
			message.Reply("_Need URL or video name!_\n*Example: .video URL/video name*")
			return
		}

		var videoURL string
		if strings.Contains(match, "youtu") {
			videoURL = match
		} else {
			var searchResults []YTSearchItem
			err := getJSON(fmt.Sprintf("https://api-25ca.onrender.com/api/yts?q=%s", url.QueryEscape(match)), &searchResults)
			if err != nil || len(searchResults) == 0 {
				message.Reply("_No results found_")
				return
			}
			videoURL = searchResults[0].URL
		}

		message.Reply("_Downloading video..._")

		var video YTVideoResponse
		err := getJSON(fmt.Sprintf("https://api-25ca.onrender.com/api/ytv?url=%s&format=360", url.QueryEscape(videoURL)), &video)
		if err != nil || !video.Status {
			message.Reply("_Failed to download video_")
			return
		}

		data, err := downloadBuffer(video.Result.Download)
		if err != nil {
			message.Reply(fmt.Sprintf("_Error downloading video: %v_", err))
			return
		}

		message.Send("video", data, lib.SendOptions{
			Caption: video.Result.Title,
			Quoted:  true,
		})
	})
}
