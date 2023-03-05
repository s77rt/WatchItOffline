package main

import (
	"fmt"
	"io"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kkdai/youtube/v2"
)

var client youtube.Client
var bot *tgbotapi.BotAPI

func init() {
	client = youtube.Client{}

	var err error
	bot, err = tgbotapi.NewBotAPI(os.Getenv("WATCHITOFFLINE_TOKEN"))
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			handleMessageUpdate(update)
		}
	}
}

func handleMessageUpdate(update tgbotapi.Update) {
	log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Checking..."))

	video, err := client.GetVideo(update.Message.Text)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Unable to get video. Error: %s", err)))
		return
	}

	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Video: %s (%s)", video.Title, video.Duration)))

	formats := video.Formats.WithAudioChannels()
	formats.Sort()
	stream, _, err := client.GetStream(video, &formats[0])
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Unable to get video stream. Error: %s", err)))
		return
	}

	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Downloading..."))

	file, err := os.CreateTemp("", "WatchItOffline")
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Unable to create temp file. Error: %s", err)))
		return
	}
	defer os.Remove(file.Name())

	_, err = io.Copy(file, stream)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Unable to copy stream to file. Error: %s", err)))
		return
	}

	bot.Send(tgbotapi.NewChatAction(update.Message.Chat.ID, tgbotapi.ChatUploadVideo))

	videoPacket := tgbotapi.NewVideo(update.Message.Chat.ID, tgbotapi.FilePath(file.Name()))
	videoPacket.Caption = video.Title
	if _, err := bot.Send(videoPacket); err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Unable to send video. Error: %s", err)))
		return
	}
}
