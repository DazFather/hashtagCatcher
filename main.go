package main

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/DazFather/parrbot/message"
	"github.com/DazFather/parrbot/robot"
	"github.com/DazFather/parrbot/tgui"
)

var (
	// Map hashtag -> times used
	trending = make(map[int64]map[string]int, 0)
	// Convert number between 0 and 10 into their emoji
	number = [11]string{"0️⃣", "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
)

func main() {
	// Define the list of commands of the bot
	var commandList = []robot.Command{
		{ReplyAt: message.MESSAGE, CallFunc: messageHandler},
		{
			Description: "Start the bot",
			Trigger:     "/start",
			ReplyAt:     message.MESSAGE,
			CallFunc:    startHandler,
		},
		{
			Description: "Show trending hashtags",
			Trigger:     "/show",
			ReplyAt:     message.MESSAGE,
			CallFunc:    showHandler,
		},
		{
			Description: "Reset saved trending hashtags",
			Trigger:     "/restart",
			ReplyAt:     message.MESSAGE,
			CallFunc: func(bot *robot.Bot, update *message.Update) message.Any {
				trending = make(map[int64]map[string]int, 0)
				return message.Text{"Counter has been resetted", nil}
			},
		},
		helpHandler.UseMenu("Help menu", "/help"),
	}
	// Make the bot alive
	robot.Start(commandList)
}

// Start function
func startHandler(bot *robot.Bot, update *message.Update) message.Any {
	sosPage := tgui.InlineButton{Text: "🆘 How to use me", CallbackData: "/help"}

	var msg = message.Text{"🦜 Welcome!", nil}
	msg.ClipInlineKeyboard([][]tgui.InlineButton{{sosPage}})
	return msg
}

// Message hashtags extractor
func messageHandler(bot *robot.Bot, update *message.Update) message.Any {
	chatID := extractGroupID(update.Message)
	if chatID == nil {
		return nil
	}

	tags := extractHashtags(update.Message.Text)
	if tags == nil || len(tags) == 0 {
		return nil
	}

	if trending[*chatID] == nil {
		trending[*chatID] = make(map[string]int)
	}
	for _, tag := range tags {
		trending[*chatID][tag]++
	}
	return nil
}

func extractGroupID(msg *message.UpdateMessage) *int64 {
	if msg == nil || msg.Chat == nil {
		return nil
	}
	return &msg.Chat.ID
}

// Show trending hashtags
func showHandler(bot *robot.Bot, update *message.Update) message.Any {

	// controls and set trend to the trending hashtags of the current group chat
	var trend map[string]int
	if chatID := extractGroupID(update.Message); chatID == nil {
		return message.Text{"You are not in a group", nil}
	} else if trend = trending[*chatID]; trend == nil || len(trend) == 0 {
		return message.Text{"No hashtag used in this group", nil}
	}

	// Sort the trending hashtag
	keys := make([]string, 0, len(trend))
	for key := range trend {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return trend[keys[i]] > trend[keys[j]]
	})

	// Take the first 10 results
	if len(keys) > 10 {
		keys = keys[:10]
	}

	// Build the final message
	msg := "🔥 Trending hashtag:\n\n"
	for i, tag := range keys {
		msg += fmt.Sprint(number[i+1], " ", tag, " - used: ", trend[tag], "\n")
	}
	return message.Text{msg, nil}
}

func extractHashtags(text string) (tags []string) {
	// search hashtag using regex and retive a list of indexes for the results
	pattern := regexp.MustCompile(`#\w+`)
	var found = pattern.FindAllStringIndex(text, -1)
	if found == nil {
		return
	}

	// Check if the message start with an hashtag
	if startAt := found[0][0]; startAt == 0 {
		tags = append(tags, text[startAt:found[0][1]])
		found = found[1:]
	}

	// Check for each result found if berofore "#" there is a white space
	for _, position := range found {
		min, max := position[0], position[1]
		if match, _ := regexp.MatchString(`\s`, string(text[min-1])); match {
			tags = append(tags, text[min:max])
		}
	}
	return
}

// Help menu
var helpHandler = tgui.Menu{
	Pages: []tgui.MenuPage{
		tgui.StaticPage("Work in progress", nil),
		tgui.StaticPage("Work in progress", nil),
		tgui.StaticPage(
			"This bot is still work in progress and is being developed with ❤️ by @DazFather. Feel free to contract me",
			tgui.GenReplyMarkupOpt(nil, 1, tgui.InlineButton{Text: "Contact developer", URL: "t.me/DazFather"}),
		),
	},
}
