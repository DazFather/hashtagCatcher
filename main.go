package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/DazFather/parrbot/message"
	"github.com/DazFather/parrbot/robot"
	"github.com/DazFather/parrbot/tgui"
)

// Set default auto-reset of hashtags time to 24 Hours
const RESET_TIME time.Duration = 24 * time.Hour

var (
	// Map groupID -> ChatInfo
	trending = map[int64]*ChatInfo{}
	// Convert number between 0 and 10 into their emoji
	number = [11]string{"0️⃣", "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
	// Default page options
	pageOpt = tgui.EditOptions{
		DisableWebPagePreview: true,
		ParseMode:             "Markdown",
	}
)

func main() {
	robot.Start(
		// Define the list of commands of the bot
		messageHandler,
		startHandler,
		showHandler,
		resetHandler,
		helpHandler,
	)
}

// Command handlers
var (

	// Message hashtags extractor - extract and save hashtags sent in a group
	messageHandler = robot.Command{
		ReplyAt: message.MESSAGE,
		CallFunc: func(bot *robot.Bot, update *message.Update) message.Any {
			// Get the chatID of the current group chat
			chatID := extractGroupID(update.Message)
			if chatID == nil {
				return nil
			}

			// Extract the hashtags from message and save them on ChatInfo of current group
			if watcher := trending[*chatID]; watcher != nil {
				watcher.Save(update.Message.ExtractEntitiesOfType("hashtag")...)
			}

			return nil
		},
	}

	// Start command - welcome user in private chat or initialize current group if sent by admin
	startHandler = robot.Command{
		Description: "👤/👥 Start the bot",
		Trigger:     "/start",
		ReplyAt:     message.MESSAGE,
		CallFunc: func(bot *robot.Bot, update *message.Update) message.Any {
			// Get the chatID of the current group chat
			var chatID *int64 = extractGroupID(update.Message)

			// Private chat: Send welcome message
			if update.Message.Chat.Type == "private" {
				m := message.Text{
					"👋 Welcome " + update.Message.From.FirstName + "!\n" +
						"Add me to your group and send /start to keep it up to date with the most used hashtags",
					nil,
				}
				return *m.ClipInlineKeyboard([][]tgui.InlineButton{{tgui.InlineCaller("🆘 Help & Info ℹ️", "/help")}})
			}

			// Group chat: start listening for hashtags
			if !isFromAdmin(*update.Message) {
				return nil
			}
			var timer *time.Duration
			if payload := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, "/start")); payload != "" {
				t, err := time.ParseDuration(payload)
				if err != nil {
					return message.Text{"Invalid time unit, please use format like this \"3h5m\" to indicate 3 hours and 5 minutes", nil}
				}
				timer = &t
			}
			watchGroup(*chatID, timer)

			return message.Text{"Group setted!👌 Now I will start catching all the #hashtags for you", nil}
		},
	}

	// Shows trending hashtags - shows the 10 top trending hashtags in current group
	showHandler = robot.Command{
		Description: "👥 Show trending hashtags in current group",
		Trigger:     "/show",
		ReplyAt:     message.MESSAGE,
		CallFunc: func(bot *robot.Bot, update *message.Update) (msg message.Any) {
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
			if groupTrends := trending[*chatID]; groupTrends != nil {
				if m := buildTrendingMessage(*groupTrends); m != nil {
					msg = *m
				}
			}
			if msg == nil {
				msg = message.Text{"No recent hashtag used in this group", nil}
			}

			return
		},
	}

	// Reset the counter and disable auto-reset of hashtags
	resetHandler = robot.Command{
		Description: "👥 Reset saved hashtags and turn off auto-reset",
		Trigger:     "/reset",
		ReplyAt:     message.MESSAGE,
		CallFunc: func(bot *robot.Bot, update *message.Update) message.Any {
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
		},
	}

	// Help menu - shows help pages (just on provate chat)
	helpHandler robot.Command = genPageMenu(
		"/help",
		"👤 How to use and set-up the bot and other infos",
		page{title: "Command list", caption: []string{
			"👤 *Private commands*:",
			"/start - Welcome message",
			"/help - How to use the bot and it's info. What you are seeing right now",
			"\n👥 *Group commands* (admin only):",
			"/start - Start listening for hashtags on the current group and turn auto-reset on",
			"/show - Shows the top 10 most popular hashtags for the current group",
			"/reset - Reset the hashtag counter and turn off auto-reset for the current group",
		}},
		page{title: "What is auto-reset mode", caption: []string{
			"This mode will cause the reset of all saved hashtags every regular time interval, counter will start when /start command is sent on group.",
			"By default the time interval is of " + fmt.Sprint(DEFAULT_RESET_TIME) + ", but it can be specified when the command is sent. ex: \"/start 48h3m20s\"",
			"If \"0\" is used as time interval auto-reset mode will not be active.",
			"When auto-reset is on the bot will also show to the group the top 10 most used hashtags just before they reset.\n",
			"Use /reset to turn off the auto-reset mode and delete all saved hashtags. Send /start to turn it on again",
		}},
		page{title: "Why use this bot", caption: []string{
			"💸 *Free* - No payments required to use this bot. [Donations](https://paypal.me/DazFather) to the developer are still welcome",
			"\n⏱ *Ready to go* - Just add this bot to a group to stay up-to-date with the trending hashtag.",
			"\n🔒 *Privacy focused* - No log or referce to the sent message will be saved, there is no database and the [code is open](https://github.com/DazFather/hashtagCatcher/)",
		}},
		page{title: "Developer info", caption: []string{
			"This bot is still work in progress and is being actively developed with ❤️ by @DazFather.",
			"It is powerade by the [Parr(B)ot](https://github.com/DazFather/parrbot) framework and is written in [Go](https://go.dev/)",
			"Feel free to contact me on Telegram or [contribute to the project](https://github.com/DazFather/hashtagCatcher/)",
		}},
	)
)

/* --- UTILITY --- */

func watchGroup(groupID int64, autoReset *time.Duration) {
	var info *ChatInfo = trending[groupID]
	if info == nil {
		info = new(ChatInfo)
		trending[groupID] = info
	}

	if autoReset == nil {
		autoReset = new(time.Duration)
		*autoReset = DEFAULT_RESET_TIME
	}
	if *autoReset == 0 {
		return
	}

	info.SetAutoReset(*autoReset, func(info ChatInfo) {
		if msg := buildTrendingMessage(info); msg != nil {
			msg.Send(groupID)
		}
	})
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

	res, err := message.API().GetChatMember(msg.Chat.ID, msg.From.ID)
	if err != nil {
		return false
	}

	return res.Result.Status == "creator" || res.Result.Status == "administrator"
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

type page struct {
	title   string
	caption []string
}

func (p page) paginate(n, tot int) tgui.Page {
	return tgui.StaticPage(
		fmt.Sprint(
			"*", strings.ToUpper(p.title), "*\n\n",
			strings.Join(p.caption, "\n"),
			"\n\n` -- page ", n, "/", tot, "` 📄",
		),
		&pageOpt,
	)
}

func genPageMenu(trigger, description string, pages ...page) robot.Command {
	var (
		tot  = len(pages)
		menu = tgui.PagedMenu{Pages: make([]tgui.Page, len(pages))}
	)

	for i, p := range pages {
		menu.Pages[i] = p.paginate(i+1, tot)
	}

	return tgui.UseMenu(&menu, trigger, description)
}
