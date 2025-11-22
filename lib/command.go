package lib

import (
	"regexp"
	"strings"
)

type CommandFunc func(*Message, string)

type Command struct {
	Pattern            *regexp.Regexp
	On                 string
	Ev                 string
	FromMe             bool
	OnlyGroup          bool
	OnlyPm             bool
	Desc               string
	Type               string
	DontAddCommandList bool
	Function           CommandFunc
}

var Commands []*Command
var validTypes = []string{"photo", "image", "text", "message", "video", "number", "viewonce", "sticker", "audio", "messages.upsert"}
var PREFIX string
var RAGEX string

func init() {
	LoadConfig()
	handlers := Config.HANDLERS
	if handlers == "false" || handlers == "null" {
		PREFIX = ""
	} else if !strings.HasPrefix(handlers, "^") && handlers != "" {
		PREFIX = strings.ReplaceAll(handlers, "[", "")
		PREFIX = strings.ReplaceAll(PREFIX, "]", "")
		PREFIX = strings.ReplaceAll(PREFIX, ".", "[.]")
	} else {
		PREFIX = handlers
	}

	if handlers == "null" {
		RAGEX = "^[^]?"
	} else {
		RAGEX = "^"
	}

	Config.HANDLERS = PREFIX
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func Function(info map[string]interface{}, function CommandFunc) *Command {
	cmd := &Command{
		FromMe:             true,
		OnlyGroup:          false,
		OnlyPm:             false,
		Desc:               "",
		Type:               "misc",
		DontAddCommandList: false,
		Function:           function,
	}

	if v, ok := info["fromMe"].(bool); ok {
		cmd.FromMe = v
	}
	if v, ok := info["onlyGroup"].(bool); ok {
		cmd.OnlyGroup = v
	}
	if v, ok := info["onlyPm"].(bool); ok {
		cmd.OnlyPm = v
	}
	if v, ok := info["desc"].(string); ok {
		cmd.Desc = v
	}
	if v, ok := info["type"].(string); ok {
		cmd.Type = v
	}
	if v, ok := info["dontAddCommandList"].(bool); ok {
		cmd.DontAddCommandList = v
	}

	_, hasOn := info["on"]
	_, hasPattern := info["pattern"]
	_, hasEv := info["ev"]

	if !hasOn && !hasPattern && !hasEv {
		cmd.On = "message"
		cmd.FromMe = false
	} else if !hasOn && !hasPattern && hasEv {
		if v, ok := info["ev"].(string); ok {
			cmd.Ev = v
		}
	} else {
		if onValue, ok := info["on"].(string); ok && contains(validTypes, onValue) {
			cmd.On = onValue
			if pattern, ok := info["pattern"].(string); ok {
				handler := true
				if h, ok := info["handler"].(bool); ok {
					handler = h
				}
				flags := ""
				if f, ok := info["flags"].(string); ok {
					flags = f
				}
				var patternStr string
				if handler {
					patternStr = "(?" + flags + ")" + Config.HANDLERS + pattern
				} else {
					patternStr = "(?" + flags + ")" + pattern
				}
				cmd.Pattern = regexp.MustCompile(patternStr)
			}
		} else if pattern, ok := info["pattern"].(string); ok {
			handler := true
			if h, ok := info["handler"].(bool); ok {
				handler = h
			}
			flags := "is"
			if f, ok := info["flags"].(string); ok {
				flags = f
			}
			prefixToUse := PREFIX
			if !strings.HasPrefix(prefixToUse, "^") {
				prefixToUse = RAGEX + PREFIX
			}
			var patternStr string
			if handler {
				patternStr = "(?" + flags + ")" + prefixToUse + `\s?(` + pattern + `)(.*)`
			} else {
				patternStr = "(?" + flags + ")" + pattern
			}
			cmd.Pattern = regexp.MustCompile(patternStr)
		}
	}

	Commands = append(Commands, cmd)
	return cmd
}