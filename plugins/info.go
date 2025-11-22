package plugins

import (
	"fmt"
	"time"

	"gobot/lib"
)

func init() {
	lib.Function(map[string]interface{}{
		"pattern": "ping",
		"fromMe":  lib.Mode(),
		"desc":    "Bot response in milliseconds.",
		"type":    "info",
	}, func(message *lib.Message, match string) {
		start := time.Now()
		message.Reply("*Ping!*")
		responseTime := time.Since(start).Milliseconds()
		message.Reply(fmt.Sprintf("*Pong!*\nLatency: %dms", responseTime))
	})

	lib.Function(map[string]interface{}{
		"pattern": "jid",
		"fromMe":  lib.Mode(),
		"desc":    "To get remoteJid",
		"type":    "whatsapp",
	}, func(message *lib.Message, match string) {
		jid := message.Chat.String()
		if len(message.MentionedJid) > 0 {
			jid = message.MentionedJid[0].String()
		} else if message.Quoted != nil {
			jid = message.Quoted.Sender.String()
		}
		message.Reply(jid)
	})

	lib.Function(map[string]interface{}{
		"pattern": "uptime",
		"fromMe":  lib.Mode(),
		"desc":    "Get bots runtime",
		"type":    "info",
	}, func(message *lib.Message, match string) {
		uptime := time.Since(lib.StartTime).Seconds()
		message.Reply(lib.FormatTime(uptime))
	})
}
