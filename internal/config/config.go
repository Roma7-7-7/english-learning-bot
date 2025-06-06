package config

import (
	"fmt"
	"strconv"
	"strings"
)

func parseChatIDs(chatIDsStr string) ([]int64, error) {
	if chatIDsStr == "" {
		return nil, nil
	}

	chatIDStrings := strings.Split(chatIDsStr, ",")
	chatIDs := make([]int64, 0, len(chatIDStrings))
	for _, chatIDString := range chatIDStrings {
		chatID, err := strconv.ParseInt(chatIDString, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse chat IDs: invalid chat ID %s: %w", chatIDString, err)
		}
		chatIDs = append(chatIDs, chatID)
	}

	return chatIDs, nil
}
