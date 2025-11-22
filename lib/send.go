package lib

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

type MediaType string

const (
	MediaText     MediaType = "text"
	MediaImage    MediaType = "image"
	MediaVideo    MediaType = "video"
	MediaAudio    MediaType = "audio"
	MediaSticker  MediaType = "sticker"
	MediaDocument MediaType = "document"
)

type SendOptions struct {
	Caption  string
	FileName string
	Mimetype string
	Quoted   bool
}

func (m *Message) Send(mediaType interface{}, content ...interface{}) (*Message, error) {
	var opts SendOptions
	var data []byte
	var err error

	if len(content) == 0 {
		return nil, fmt.Errorf("no content provided")
	}

	mType := MediaText
	switch v := mediaType.(type) {
	case MediaType:
		mType = v
	case string:
		mType = MediaType(strings.ToLower(v))
	}

	contentData := content[0]
	if len(content) > 1 {
		if o, ok := content[1].(SendOptions); ok {
			opts = o
		} else if caption, ok := content[1].(string); ok {
			opts.Caption = caption
		}
	}

	if mType == MediaText {
		text := ""
		switch v := contentData.(type) {
		case string:
			text = v
		case []byte:
			text = string(v)
		default:
			return nil, fmt.Errorf("invalid text content type")
		}

		if opts.Quoted && m.Data != nil {
			return m.Reply(text)
		}
		return m.sendText(text)
	}

	switch v := contentData.(type) {
	case string:
		if isURL(v) {
			data, err = getBuffer(v)
		} else {
			data, err = os.ReadFile(v)
			if opts.FileName == "" {
				opts.FileName = filepath.Base(v)
			}
		}
	case []byte:
		data = v
	default:
		return nil, fmt.Errorf("unsupported content type")
	}

	if err != nil {
		return nil, err
	}

	if opts.Mimetype == "" {
		opts.Mimetype = detectMimetype(mType, data)
	}

	switch mType {
	case MediaImage:
		return m.sendImage(data, opts)
	case MediaVideo:
		return m.sendVideo(data, opts)
	case MediaAudio:
		return m.sendAudio(data, opts)
	case MediaSticker:
		return m.sendSticker(data, opts)
	case MediaDocument:
		return m.sendDocument(data, opts)
	default:
		return nil, fmt.Errorf("unsupported media type: %s", mType)
	}
}

func (m *Message) sendText(text string) (*Message, error) {
	response, err := m.Client.SendMessage(context.Background(), m.Chat, &waE2E.Message{
		Conversation: proto.String(text),
	})
	if err != nil {
		return nil, err
	}

	return &Message{
		Client: m.Client,
		ID:     response.ID,
		Chat:   m.Chat,
		FromMe: true,
	}, nil
}

func (m *Message) sendImage(data []byte, opts SendOptions) (*Message, error) {
	uploaded, err := m.Client.Upload(context.Background(), data, whatsmeow.MediaImage)
	if err != nil {
		return nil, err
	}

	tmpFile, err := os.CreateTemp("", "wa_img_*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		return nil, err
	}
	tmpFile.Close()

	thumbnail, width, height, err := GenerateThumbnail(context.Background(), tmpFile.Name(), "image")
	if err != nil {
		thumbnail = ""
		width = 0
		height = 0
	}

	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(opts.Mimetype),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		},
	}

	if thumbnail != "" {
		thumbData, err := base64.StdEncoding.DecodeString(thumbnail)
		if err == nil {
			msg.ImageMessage.JPEGThumbnail = thumbData
			msg.ImageMessage.Width = proto.Uint32(uint32(width))
			msg.ImageMessage.Height = proto.Uint32(uint32(height))
		}
	}

	if opts.Caption != "" {
		msg.ImageMessage.Caption = proto.String(opts.Caption)
	}

	if opts.Quoted && m.Data != nil {
		msg.ImageMessage.ContextInfo = &waE2E.ContextInfo{
			StanzaID:      proto.String(m.ID),
			Participant:   proto.String(m.Sender.String()),
			QuotedMessage: m.Data.Message,
		}
	}

	response, err := m.Client.SendMessage(context.Background(), m.Chat, msg)
	if err != nil {
		return nil, err
	}

	return &Message{
		Client: m.Client,
		ID:     response.ID,
		Chat:   m.Chat,
		FromMe: true,
	}, nil
}

func (m *Message) sendVideo(data []byte, opts SendOptions) (*Message, error) {
	uploaded, err := m.Client.Upload(context.Background(), data, whatsmeow.MediaVideo)
	if err != nil {
		return nil, err
	}

	tmpFile, err := os.CreateTemp("", "wa_video_*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		return nil, err
	}
	tmpFile.Close()

	thumbnail, _, _, err := GenerateThumbnail(context.Background(), tmpFile.Name(), "video")
	if err != nil {
		thumbnail = ""
	}

	msg := &waE2E.Message{
		VideoMessage: &waE2E.VideoMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(opts.Mimetype),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		},
	}

	if thumbnail != "" {
		thumbData, err := base64.StdEncoding.DecodeString(thumbnail)
		if err == nil {
			msg.VideoMessage.JPEGThumbnail = thumbData
		}
	}

	if opts.Caption != "" {
		msg.VideoMessage.Caption = proto.String(opts.Caption)
	}

	if opts.Quoted && m.Data != nil {
		msg.VideoMessage.ContextInfo = &waE2E.ContextInfo{
			StanzaID:      proto.String(m.ID),
			Participant:   proto.String(m.Sender.String()),
			QuotedMessage: m.Data.Message,
		}
	}

	response, err := m.Client.SendMessage(context.Background(), m.Chat, msg)
	if err != nil {
		return nil, err
	}

	return &Message{
		Client: m.Client,
		ID:     response.ID,
		Chat:   m.Chat,
		FromMe: true,
	}, nil
}

func (m *Message) sendAudio(data []byte, opts SendOptions) (*Message, error) {
	uploaded, err := m.Client.Upload(context.Background(), data, whatsmeow.MediaAudio)
	if err != nil {
		return nil, err
	}

	msg := &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(opts.Mimetype),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		},
	}

	if opts.Quoted && m.Data != nil {
		msg.AudioMessage.ContextInfo = &waE2E.ContextInfo{
			StanzaID:      proto.String(m.ID),
			Participant:   proto.String(m.Sender.String()),
			QuotedMessage: m.Data.Message,
		}
	}

	response, err := m.Client.SendMessage(context.Background(), m.Chat, msg)
	if err != nil {
		return nil, err
	}

	return &Message{
		Client: m.Client,
		ID:     response.ID,
		Chat:   m.Chat,
		FromMe: true,
	}, nil
}

func (m *Message) sendSticker(data []byte, opts SendOptions) (*Message, error) {
	uploaded, err := m.Client.Upload(context.Background(), data, whatsmeow.MediaImage)
	if err != nil {
		return nil, err
	}

	msg := &waE2E.Message{
		StickerMessage: &waE2E.StickerMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(opts.Mimetype),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		},
	}

	if opts.Quoted && m.Data != nil {
		msg.StickerMessage.ContextInfo = &waE2E.ContextInfo{
			StanzaID:      proto.String(m.ID),
			Participant:   proto.String(m.Sender.String()),
			QuotedMessage: m.Data.Message,
		}
	}

	response, err := m.Client.SendMessage(context.Background(), m.Chat, msg)
	if err != nil {
		return nil, err
	}

	return &Message{
		Client: m.Client,
		ID:     response.ID,
		Chat:   m.Chat,
		FromMe: true,
	}, nil
}

func (m *Message) sendDocument(data []byte, opts SendOptions) (*Message, error) {
	uploaded, err := m.Client.Upload(context.Background(), data, whatsmeow.MediaDocument)
	if err != nil {
		return nil, err
	}

	if opts.FileName == "" {
		opts.FileName = "document"
	}

	msg := &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(opts.Mimetype),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			FileName:      proto.String(opts.FileName),
		},
	}

	if opts.Caption != "" {
		msg.DocumentMessage.Caption = proto.String(opts.Caption)
	}

	if opts.Quoted && m.Data != nil {
		msg.DocumentMessage.ContextInfo = &waE2E.ContextInfo{
			StanzaID:      proto.String(m.ID),
			Participant:   proto.String(m.Sender.String()),
			QuotedMessage: m.Data.Message,
		}
	}

	response, err := m.Client.SendMessage(context.Background(), m.Chat, msg)
	if err != nil {
		return nil, err
	}

	return &Message{
		Client: m.Client,
		ID:     response.ID,
		Chat:   m.Chat,
		FromMe: true,
	}, nil
}

func isURL(str string) bool {
	return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")
}

func getBuffer(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download: status code %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func detectMimetype(mediaType MediaType, data []byte) string {
	switch mediaType {
	case MediaImage:
		return http.DetectContentType(data)
	case MediaVideo:
		return "video/mp4"
	case MediaAudio:
		return "audio/ogg; codecs=opus"
	case MediaSticker:
		return "image/webp"
	case MediaDocument:
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
}

func GenerateThumbnail(ctx context.Context, filePath string, mediaType string) (string, int, int, error) {
	switch mediaType {
	case "image":
		img, err := imaging.Open(filePath)
		if err != nil {
			return "", 0, 0, err
		}
		originalBounds := img.Bounds()
		resized := imaging.Resize(img, 32, 0, imaging.Lanczos)
		buf := new(bytes.Buffer)
		err = imaging.Encode(buf, resized, imaging.JPEG, imaging.JPEGQuality(50))
		if err != nil {
			return "", 0, 0, err
		}
		return base64.StdEncoding.EncodeToString(buf.Bytes()), originalBounds.Dx(), originalBounds.Dy(), nil
	case "video":
		tmpFile := fmt.Sprintf("%s_thumb.jpg", filePath)
		cmd := exec.CommandContext(ctx, "ffmpeg", "-ss", "00:00:00", "-i", filePath, "-y", "-vf", "scale=32:-1", "-vframes", "1", "-f", "image2", tmpFile)
		err := cmd.Run()
		if err != nil {
			return "", 0, 0, err
		}
		data, err := os.ReadFile(tmpFile)
		if err != nil {
			return "", 0, 0, err
		}
		_ = os.Remove(tmpFile)
		return base64.StdEncoding.EncodeToString(data), 0, 0, nil
	}
	return "", 0, 0, fmt.Errorf("unsupported media type")
}
