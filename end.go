package main

import "log"

// EndConversationEvent contains the data of the conversation that has to be
// ended
type EndConversationEvent struct {
	ChatID int64
}

func endConversationWorker(jobs <-chan EndConversationEvent) {
	for e := range jobs {
		u, err := retrieveUser(e.ChatID)

		if err != nil {
			log.Printf("Could not retrieve user in worker %s", err)
			return
		}

		// Check if is valid
		if u.MatchChatID.Valid {
			db.Exec("UPDATE users SET match_chat_id = NULL, available = 0, previous_match = ? WHERE chat_id = ?", u.MatchChatID, u.ChatID)
			db.Exec("UPDATE users SET match_chat_id = NULL, available = 0, previous_match = ? WHERE chat_id = ?", u.ChatID, u.MatchChatID)

			telegram.SendMessage(u.MatchChatID.Int64, "Your conversation partner left the chat", emptyOpts)
			telegram.SendMessage(u.MatchChatID.Int64, "Type /start to get matched with a new partner", emptyOpts)
		} else {
			db.Exec("UPDATE users SET available = 0 WHERE chat_id = ?", u.ChatID)
		}

		telegram.SendMessage(u.ChatID, "Your conversation is over, I hope you enjoyed it :)", emptyOpts)
		telegram.SendMessage(u.ChatID, "Type /start to get matched with a new partner", emptyOpts)
	}
}
