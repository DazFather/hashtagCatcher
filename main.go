package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/DazFather/parrbot/message"
	"github.com/DazFather/parrbot/robot"
	"github.com/DazFather/parrbot/tgui"
)

// Set default auto-reset of hashtags time to 24 Hours
const RESET_TIME time.Duration = 24 * time.Hour

type ChatInfo struct {
	hashtags map[string]int // Map hashtag -> times used
	resetter *time.Ticker   // function that will loop causing the hashtag to reset
	stop     bool
}

// Get the firsts 'cap' most used hashtags
func (i ChatInfo) Trending(cap int) (trend []string) {
	var tags map[string]int = i.hashtags
	if tags == nil {
		return
	}

	// Sort the trending hashtag
	trend = make([]string, 0, len(tags))
	for tag := range tags {
		trend = append(trend, tag)
	}
	sort.SliceStable(trend, func(i, j int) bool {
		return tags[trend[i]] > tags[trend[j]]
	})

	// Take the first 10 results
	if len(trend) > cap {
		trend = trend[:cap]
	}
	return
}

// Save one or more tags on the hashtags map and increase used conuter
func (i *ChatInfo) Save(tags ...string) {
	if len(tags) == 0 {
		return
	}

	if i.hashtags == nil {
		i.hashtags = make(map[string]int, len(tags))
	}

	for _, tag := range tags {
		i.hashtags[tag]++
	}
}

// Set auto-reset of hashtags for a given time interval and call given function before reset
func (i *ChatInfo) SetAutoReset(interval time.Duration, beforeReset func(ChatInfo)) {
	if i.resetter != nil {
		i.resetter.Reset(interval)
	} else {
		i.resetter = time.NewTicker(interval)
	}

	go func() {
		defer i.resetter.Stop()

		for range i.resetter.C {
			if i.stop {
				i.stop = false
				break
			}
			go beforeReset(*i)
			i.hashtags = nil
		}

	}()

	return
}

// Stop auto-reset and clear saved hashtags
func (i *ChatInfo) StopAutoReset() {
	i.stop = true
	i.hashtags = nil
}

var (
	// Map groupID -> ChatInfo
	trending = map[int64]*ChatInfo{}
	// Convert number between 0 and 10 into their emoji
	number = [11]string{"0️⃣", "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
	// Default page options
	pageOpt = &tgui.PageOptions{
		DisableWebPagePreview: true,
		ParseMode:             "Markdown",
	}
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
			Description: "Reset saved trending hashtags and turn off auto-reset",
			Trigger:     "/reset",
			ReplyAt:     message.MESSAGE,
			CallFunc:    resetHandler,
		},
		helpHandler.UseMenu("Help menu", "/help"),
	}
	// Make the bot alive
	robot.Start(commandList)
}

// Start function
func startHandler(bot *robot.Bot, update *message.Update) message.Any {
	// Get the chatID of the current group chat
	var chatID *int64 = extractGroupID(update.Message)

	// Private chat: Send welcome message
	if update.Message.Chat.Type == "private" {
		m := message.Text{"👋 Welcome!\nAdd me to your group and send /start to keep it up to date with the most used hashtags", nil}
		return *m.ClipInlineKeyboard([][]tgui.InlineButton{{{Text: "🆘 Help & Info ℹ️", CallbackData: "/help"}}})
	}

	// Group chat: start listening for hashtags
	if !isFromAdmin(*update.Message) {
		return nil
	}
	watchGroup(*chatID, true)

	return message.Text{"Group setted!👌 Now I will start catching all the #hashtags for you", nil}
}

func watchGroup(groupID int64, autoReset bool) {
	var info *ChatInfo = trending[groupID]
	if info == nil {
		info = new(ChatInfo)
		trending[groupID] = info
	}

	if !autoReset {
		return
	}

	info.SetAutoReset(RESET_TIME, func(info ChatInfo) {
		if msg := buildTrendingMessage(info); msg != nil {
			msg.Send(groupID)
		}
	})
}

// Message hashtags extractor
func messageHandler(bot *robot.Bot, update *message.Update) message.Any {
	// Get the chatID of the current group chat
	chatID := extractGroupID(update.Message)
	if chatID == nil {
		return nil
	}

	// Extract the hashtags from message and save them on ChatInfo of current group
	tags := extractHashtags(update.Message.Text)

	if watcher := trending[*chatID]; watcher != nil {
		watcher.Save(tags...)
	}

	return nil
}

// Reset the counter and disable auto-reset of hashtags
func resetHandler(bot *robot.Bot, update *message.Update) message.Any {
	// Get the chatID of the current group chat
	chatID := extractGroupID(update.Message)
	if chatID == nil {
		return message.Text{"You are not in a group", nil}
	}

	// Check if sending user is authorized
	if !isFromAdmin(*update.Message) {
		return nil
	}

	// If ChatInfo for current group is available then stop auto reset
	if watcher := trending[*chatID]; watcher != nil {
		watcher.StopAutoReset()
		return message.Text{"Counter has been resetted. Use /start to turn auto-reset on", nil}
	}

	return message.Text{"I'm not listening this group. Use /start to start catching", nil}
}

func extractGroupID(msg *message.UpdateMessage) *int64 {
	if msg == nil || msg.Chat == nil {
		return nil
	}
	return &msg.Chat.ID
}

func isFromAdmin(msg message.UpdateMessage) bool {
	if msg.Chat == nil || msg.From == nil || msg.Chat.ID == msg.From.ID {
		return false
	}

	res, err := message.GetAPI().GetChatMember(msg.Chat.ID, msg.From.ID)
	if err != nil {
		return false
	}

	return res.Result.Status == "creator" || res.Result.Status == "administrator"
}

// Show trending hashtags
func showHandler(bot *robot.Bot, update *message.Update) message.Any {
	// Get the chatID of the current group chat
	chatID := extractGroupID(update.Message)
	if chatID == nil {
		return message.Text{"You are not in a group", nil}
	}

	// Check if sending user is authorized
	if !isFromAdmin(*update.Message) {
		return nil
	}

	// use the ChatInfo to build the message that display the top 10 trending hashtags
	msg := buildTrendingMessage(*trending[*chatID])
	if msg == nil {
		return message.Text{"No hashtag used in this group", nil}
	}

	return msg
}

func buildTrendingMessage(info ChatInfo) *message.Text {
	// Take the first 10 trending hashtags
	var trend []string = info.Trending(10)
	if trend == nil || len(trend) == 0 {
		return nil
	}

	// Build the final message
	msg := "🔥 Trending hashtag:\n\n"
	for i, tag := range trend {
		msg += fmt.Sprint(number[i+1], " ", tag, " - used: ", info.hashtags[tag], "\n")
	}
	return &message.Text{msg, nil}
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

func genPage(title string, page, tot uint8, lines ...string) tgui.MenuPage {
	return tgui.StaticPage(
		fmt.Sprint(
			"*", strings.ToUpper(title), "*\n\n",
			strings.Join(lines, "\n"),
			"\n\n` -- page ", page, "/", tot, "` 📄",
		),
		pageOpt,
	)
}

// Help menu
var helpHandler = tgui.Menu{Pages: []tgui.MenuPage{
	genPage("Command list", 1, 4,
		"👤 *Private commands*:",
		"/start - Welcome message",
		"/help - How to use the bot and it's info. What you are seeing right now",
		"\n👥 *Group commands* (admin only):",
		"/start - Start listening for hashtags on the current group and turn auto-reset on",
		"/show - Shows the top 10 most popular hashtags for the current group",
		"/reset - Reset the hashtag counter and turn off auto-reset for the current group",
	),

	genPage("What is auto-reset mode", 2, 4,
		"This mode will cause the reset of all saved hashtags every 24h since the last /start command has been sent.",
		"It's on by default but you can easily turn it off using /reset and on again with /start.",
		"When auto-reset is on the bot will show to the group the top 10 most used hashtags just before they reset",
	),

	genPage("Why use this bot", 3, 4,
		"💸 *Free* - No payments required to use this bot. [Donations](https://paypal.me/DazFather) to the developer are still welcome",
		"\n⏱ *Ready to go* - Just add this bot to a group to stay up-to-date with the trending hashtag.",
		"\n🔒 *Privacy focused* - No log or referce to the sent message will be saved, there is no database and the [code is open](https://github.com/DazFather/hashtagCatcher/)",
	),

	genPage("Developer info", 4, 4,
		"This bot is still work in progress and is being actively developed with ❤️ by @DazFather.\n",
		"Feel free to contract me on Telegram or [contribute to the project](https://github.com/DazFather/hashtagCatcher/)",
	),
}}
