# Hashtag Catcher

Telegram bot for catching the most trending hashtags on a group chat.
Powerade by my own framework [Parr(B)ot](https://github.com/DazFather/parrbot)


> **Warning: W.I.P.**
> This project is still work in progress. All the features are highly experimental and can not be consider by any mean stable or secure.


## Set-up
 - Clone this repository and build using the command "`go build`" on your terminal (make sure to have [Go](https://go.dev/) installed, check [go.mod](./go.mod) for the minimal version required)
 - Use [@BotFather](https:/t.me/BotFather) to create your own bot and copy the API TOKEN. Remember to set [privacy mode](https://core.telegram.org/bots#privacy-mode) off to be able to catch also hashtags in messages that don't start with "/"
 - Run the bot and use as argument or the API TOKEN, or save it on a _".txt"_ file and use  `--readfrom ` followed by the file path. Like this: `.\hashtagCatcher.exe --readfrom myFile.txt`
