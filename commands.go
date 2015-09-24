package main

import (
	"log"
	"strings"
	"time"

	"github.com/Machiel/telegrambot"
)

// CommandHandler supplies an interface for handling messages
type commandHandler func(u User, m telegrambot.Message) bool

func commandDisablePictures(u User, m telegrambot.Message) bool {
	if len(m.Text) < 7 || strings.ToLower(m.Text[0:7]) != "/nopics" {
		return false
	}

	if u.AllowPictures {
		db.Exec("UPDATE users SET allow_pictures = 0 WHERE id = ?", u.ID)
		telegram.SendMessage(u.ChatID, "Strangers won't be able to send you photos anymore!", emptyOpts)
		return true
	}

	db.Exec("UPDATE users SET allow_pictures = 1 WHERE id = ?", u.ID)
	telegram.SendMessage(u.ChatID, "Strangers can now send you photos!", emptyOpts)
	return true
}

func commandStart(u User, m telegrambot.Message) bool {

	if len(m.Text) < 6 {
		return false
	}

	if strings.ToLower(m.Text[0:6]) != "/start" {
		return false
	}

	if u.Available {
		return false
	}

	if u.MatchChatID.Valid {
		return false
	}

	db.Exec("UPDATE users SET available = 1 WHERE chat_id = ?", u.ChatID)

	telegram.SendMessage(u.ChatID, "Looking for a stranger to match you with... Hold on!", emptyOpts)
	startJobs <- u.ChatID

	return true
}

func commandStop(u User, m telegrambot.Message) bool {

	if len(m.Text) < 4 {
		return false
	}

	rightCommand := strings.ToLower(m.Text[0:4]) == "/bye" || strings.ToLower(m.Text[0:4]) == "/end"

	if !rightCommand {
		return false
	}

	if !u.Available {
		return false
	}

	telegram.SendMessage(u.ChatID, "We're ending the conversation...", emptyOpts)

	endConversationQueue <- EndConversationEvent{ChatID: u.ChatID}

	return true
}

func commandReport(u User, m telegrambot.Message) bool {

	if len(m.Text) < 7 || strings.ToLower(m.Text[0:7]) != "/report" {
		return false
	}

	if !u.Available || !u.MatchChatID.Valid {
		return false
	}

	report := m.Text[7:]
	report = strings.TrimSpace(report)

	if len(report) == 0 {
		telegram.SendMessage(u.ChatID, "Usage /report: /report <reason>", emptyOpts)
		return true
	}

	partner, err := retrieveUser(u.MatchChatID.Int64)

	if err != nil {
		log.Println("Error retrieving partner")
		return true
	}

	db.Exec("INSERT INTO reports (user_id, report, reporter_id, created_at) VALUES (?, ?, ?, ?)", partner.ID, report, u.ID, time.Now())

	telegram.SendMessage(u.ChatID, "User has been reported!", emptyOpts)

	return true
}

func commandMessage(u User, m telegrambot.Message) bool {

	if !u.Available {
		return false
	}

	if !u.MatchChatID.Valid {
		return false
	}

	chatID := u.MatchChatID.Int64
	partner, err := retrieveUser(chatID)

	if err != nil {
		log.Println("[ERROR] Could not retrieve partner %d", chatID)
		return false
	}

	if len(m.Photo) > 0 {

		if !partner.AllowPictures {
			telegram.SendMessage(chatID, "Stranger tried to send you a photo, but you disabled this,  you can enable photos by using the /nopics command", emptyOpts)
			telegram.SendMessage(u.ChatID, "Stranger disabled photos, and will not receive your photos", emptyOpts)
			return true
		}

		var toSend telegrambot.PhotoSize

		for _, t := range m.Photo {
			if t.FileSize > toSend.FileSize {
				toSend = t
			}
		}

		telegram.SendMessage(chatID, "Stranger sends you a photo!", emptyOpts)
		_, err = telegram.SendPhoto(chatID, toSend.FileID, emptyOpts)

	} else if m.Sticker != (telegrambot.Sticker{}) {
		telegram.SendMessage(chatID, "Stranger sends you a sticker!", emptyOpts)
		_, err = telegram.SendSticker(chatID, m.Sticker.FileID, emptyOpts)
	} else if m.Location != (telegrambot.Location{}) {
		telegram.SendMessage(chatID, "Stranger sends you a location!", emptyOpts)
		_, err = telegram.SendLocation(chatID,
			m.Location.Latitude,
			m.Location.Longitude,
			emptyOpts,
		)
	} else if m.Document != (telegrambot.Document{}) {
		telegram.SendMessage(chatID, "Stranger sends you a document!", emptyOpts)
		_, err = telegram.SendDocument(chatID, m.Document.FileID, emptyOpts)
	} else if m.Audio != (telegrambot.Audio{}) {
		telegram.SendMessage(chatID, "Stranger sends you an audio file!", emptyOpts)
		_, err = telegram.SendAudio(chatID, m.Audio.FileID, emptyOpts)
	} else if m.Video != (telegrambot.Video{}) {
		telegram.SendMessage(chatID, "Stranger sends you a video file!", emptyOpts)
		_, err = telegram.SendVideo(chatID, m.Video.FileID, emptyOpts)
	} else {
		_, err = telegram.SendMessage(chatID, "Stranger: "+m.Text, emptyOpts)
	}

	if err != nil {
		log.Printf("Forward error: %s", err)
	}

	return true

}

func commandHelp(u User, m telegrambot.Message) bool {

	if len(m.Text) < 5 {
		return false
	}

	if strings.ToLower(m.Text[0:5]) != "/help" {
		return false
	}

	telegram.SendMessage(m.Chat.ID, `Help:

Use /start to start looking for a conversational partner, once you're matched you can use /end to end the conversation.

Use /report to report a user, use it as follows:
/report <reason>

Use /nopics to disable receiving photos, and /nopics if you want to enable it again.

Sending images and videos are a beta functionality, but appear to be working fine.

If you have any suggestions or require help, please contact @MachielMolenaar on Twitter, or follow @MachielMolenaar on Twitter for updates. (http://twitter.com/MachielMolenaar)`, emptyOpts)

	return true
}
