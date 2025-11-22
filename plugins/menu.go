package plugins

import (
	"fmt"
	"sort"
	"strings"

	"gobot/lib"
)

func init() {
	lib.Function(map[string]interface{}{
		"pattern": "menu",
		"fromMe":  lib.Mode(),
		"desc":    "Display all available commands",
		"type":    "info",
	}, func(message *lib.Message, match string) {
		prefix := lib.GetPrefix()
		commandsByType := make(map[string][]*lib.Command)

		for _, cmd := range lib.Commands {
			if cmd.DontAddCommandList || cmd.Pattern == nil {
				continue
			}
			cmdType := cmd.Type
			if cmdType == "" {
				cmdType = "misc"
			}
			commandsByType[cmdType] = append(commandsByType[cmdType], cmd)
		}

		types := make([]string, 0, len(commandsByType))
		for t := range commandsByType {
			types = append(types, t)
		}
		sort.Strings(types)

		var menuBuilder strings.Builder
		menuBuilder.WriteString("*COMMAND MENU*\n\n")

		for _, cmdType := range types {
			commands := commandsByType[cmdType]
			typeName := strings.ToUpper(cmdType)
			menuBuilder.WriteString(fmt.Sprintf("*%s*\n", typeName))

			sort.Slice(commands, func(i, j int) bool {
				return extractCommandName(commands[i]) < extractCommandName(commands[j])
			})

			for _, cmd := range commands {
				cmdName := extractCommandName(cmd)
				desc := cmd.Desc
				if desc == "" {
					desc = "No description"
				}

				menuBuilder.WriteString(fmt.Sprintf("%s%s\n", prefix, cmdName))
				menuBuilder.WriteString(fmt.Sprintf("_%s_\n\n", desc))
			}
		}

		message.Reply(menuBuilder.String())
	})
}

func extractCommandName(cmd *lib.Command) string {
	if cmd.Pattern == nil {
		return ""
	}

	pattern := cmd.Pattern.String()
	start := strings.Index(pattern, "\\s?(")
	if start == -1 {
		start = strings.Index(pattern, "(")
		if start == -1 {
			return ""
		}
	} else {
		start += 4
	}

	end := strings.Index(pattern[start:], ")")
	if end == -1 {
		return ""
	}

	cmdName := pattern[start : start+end]
	cmdName = strings.Trim(cmdName, "()")
	return cmdName
}
