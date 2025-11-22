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

type ReplyMessage struct {
	Client   *whatsmeow.Client
	ID       string
	Sender   types.JID
	FromMe   bool
	Chat     types.JID
	Type     string
	Text     string
	IsGroup  bool
	IsPm     bool
	IsBot    bool
	IsSudo   bool
	Message  *waE2E.Message
}

func NewReplyMessage(client *whatsmeow.Client, evt *events.Message) *ReplyMessage {
	if evt.Message.ExtendedTextMessage == nil || evt.Message.ExtendedTextMessage.ContextInfo == nil {
		return nil
	}

	ctx := evt.Message.ExtendedTextMessage.ContextInfo
	quotedMsg := ctx.QuotedMessage

	var sender types.JID
	if ctx.Participant != nil {
		sender = types.NewJID(strings.Split(*ctx.Participant, "@")[0], types.DefaultUserServer)
	} else {
		sender = evt.Info.Sender
	}

	reply := &ReplyMessage{
		Client:  client,
		ID:      *ctx.StanzaID,
		Sender:  sender,
		FromMe:  sender.User == client.Store.ID.User,
		Chat:    evt.Info.Chat,
		IsGroup: evt.Info.IsGroup,
		IsPm:    !evt.Info.IsGroup,
		Message: quotedMsg,
	}

	reply.Type = getContentType(quotedMsg)
	reply.Text = getMessageText(quotedMsg)
	reply.IsBot = strings.HasPrefix(reply.ID, "BAE5") && len(reply.ID) == 16

	sudos := strings.Split(Config.SUDO, ",")
	sudoJids := []string{client.Store.ID.User}
	sudoJids = append(sudoJids, sudos...)
	
	for _, sudo := range sudoJids {
		if sudo != "" && sudo+"@s.whatsapp.net" == reply.Sender.String() {
			reply.IsSudo = true
			break
		}
	}

	return reply
}

func (r *ReplyMessage) Reply(text string) (*Message, error) {
	response, err := r.Client.SendMessage(context.Background(), r.Chat, &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      proto.String(r.ID),
				Participant:   proto.String(r.Sender.String()),
				QuotedMessage: r.Message,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return &Message{
		Client: r.Client,
		ID:     response.ID,
		Chat:   r.Chat,
		FromMe: true,
	}, nil
}

func (r *ReplyMessage) Delete() error {
	_, err := r.Client.SendMessage(context.Background(), r.Chat, r.Client.BuildRevoke(r.Chat, types.EmptyJID, r.ID))
	return err
}
