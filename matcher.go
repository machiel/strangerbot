package main

import "log"

func matchUsers(chatIDs <-chan int64) {

	for c := range chatIDs {

		user, err := retrieveUser(c)

		if err != nil {
			log.Printf("Error in matcher: %s", err)
			continue
		}

		if !user.Available || user.MatchChatID.Valid {
			log.Println("User already assigned")
			continue
		}

		availableUsers, err := retrieveAvailableUsers(c)

		if err != nil {
			log.Printf("Error retrieving available users: %s", err)
			continue
		}

		if len(availableUsers) == 0 {
			continue
		}

		shuffle(availableUsers)

		match := availableUsers[0]

		createMatch(user, match)
	}

}

func createMatch(user User, match User) {
	query := "UPDATE users SET match_chat_id = ? WHERE id = ?"

	db.Exec(query, user.ChatID, match.ID)
	db.Exec(query, match.ChatID, user.ID)

	telegram.SendMessage(match.ChatID, "You have been matched, have fun!", emptyOpts)
	telegram.SendMessage(user.ChatID, "You have been matched, have fun!", emptyOpts)
}
