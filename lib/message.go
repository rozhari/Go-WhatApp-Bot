package lib

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

type Message struct {
	Client        *whatsmeow.Client
	Data          *events.Message
	ID            string
	Sender        types.JID
	FromMe        bool
	Chat          types.JID
	Type          string
	Text          string
	IsGroup       bool
	IsPm          bool
	IsBot         bool
	IsSudo        bool
	PushName      string
	MentionedJid  []types.JID
	Quoted        *ReplyMessage
}

func NewMessage(client *whatsmeow.Client, evt *events.Message) *Message {
	msg := &Message{
		Client:   client,
		Data:     evt,
		ID:       evt.Info.ID,
		Sender:   evt.Info.Sender,
		FromMe:   evt.Info.IsFromMe,
		Chat:     evt.Info.Chat,
		IsGroup:  evt.Info.IsGroup,
		IsPm:     !evt.Info.IsGroup,
		PushName: evt.Info.PushName,
	}

	msg.Type = getContentType(evt.Message)
	msg.Text = getMessageText(evt.Message)
	msg.IsBot = strings.HasPrefix(evt.Info.ID, "BAE5") && len(evt.Info.ID) == 16

	sudos := strings.Split(Config.SUDO, ",")
	sudoJids := []string{client.Store.ID.User}
	sudoJids = append(sudoJids, sudos...)
	
	for _, sudo := range sudoJids {
		if sudo != "" && sudo+"@s.whatsapp.net" == msg.Sender.String() {
			msg.IsSudo = true
			break
		}
	}

	if evt.Message.ExtendedTextMessage != nil && evt.Message.ExtendedTextMessage.ContextInfo != nil {
		if evt.Message.ExtendedTextMessage.ContextInfo.MentionedJID != nil {
			for _, jid := range evt.Message.ExtendedTextMessage.ContextInfo.MentionedJID {
				msg.MentionedJid = append(msg.MentionedJid, types.NewJID(jid, types.DefaultUserServer))
			}
		}
		if evt.Message.ExtendedTextMessage.ContextInfo.QuotedMessage != nil {
			msg.Quoted = NewReplyMessage(client, evt)
		}
	}

	return msg
}

func (m *Message) Reply(text string) (*Message, error) {
	response, err := m.Client.SendMessage(context.Background(), m.Chat, &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      proto.String(m.ID),
				Participant:   proto.String(m.Sender.String()),
				QuotedMessage: m.Data.Message,
			},
		},
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

func (m *Message) Delete() error {
	_, err := m.Client.SendMessage(context.Background(), m.Chat, m.Client.BuildRevoke(m.Chat, types.EmptyJID, m.ID))
	return err
}

func getContentType(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if msg.Conversation != nil {
		return "conversation"
	}
	if msg.ExtendedTextMessage != nil {
		return "extendedTextMessage"
	}
	if msg.ImageMessage != nil {
		return "imageMessage"
	}
	if msg.VideoMessage != nil {
		return "videoMessage"
	}
	if msg.AudioMessage != nil {
		return "audioMessage"
	}
	if msg.DocumentMessage != nil {
		return "documentMessage"
	}
	if msg.StickerMessage != nil {
		return "stickerMessage"
	}
	return ""
}

func getMessageText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if msg.Conversation != nil {
		return *msg.Conversation
	}
	if msg.ExtendedTextMessage != nil && msg.ExtendedTextMessage.Text != nil {
		return *msg.ExtendedTextMessage.Text
	}
	if msg.ImageMessage != nil && msg.ImageMessage.Caption != nil {
		return *msg.ImageMessage.Caption
	}
	if msg.VideoMessage != nil && msg.VideoMessage.Caption != nil {
		return *msg.VideoMessage.Caption
	}
	if msg.DocumentMessage != nil && msg.DocumentMessage.Caption != nil {
		return *msg.DocumentMessage.Caption
	}
	return ""
}