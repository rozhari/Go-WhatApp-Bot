package lib

import (
    "os"
	"fmt"
	"time"
	"regexp"
	"strings"
	"strconv"

	"go.mau.fi/whatsmeow/types"
)

func NumToJid(num string) types.JID {
	if num == "" {
		return types.EmptyJID
	}
	return types.NewJID(num, types.DefaultUserServer)
}

func FormatTime(seconds float64) string {
	duration := time.Duration(seconds) * time.Second

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	secs := int(duration.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, secs)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func getEnvBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return b
}

func Mode() bool {
	return Config.MODE != "public"
}

func GetPrefix() string {
	if strings.HasPrefix(Config.HANDLERS, "^") {
		re := regexp.MustCompile(`\[(\W*)\]`)
		matches := re.FindStringSubmatch(Config.HANDLERS)
		if len(matches) > 1 && len(matches[1]) > 0 {
			return string(matches[1][0])
		}
	}

	prefix := strings.ReplaceAll(Config.HANDLERS, "[", "")
	prefix = strings.ReplaceAll(prefix, "]", "")
	return strings.TrimSpace(prefix)
}

