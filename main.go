package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/Machiel/telegrambot"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	telegram        telegrambot.TelegramBot
	db              *sqlx.DB
	emptyOpts       = telegrambot.SendMessageOptions{}
	commandHandlers = []commandHandler{
		commandDisablePictures,
		commandHelp,
		commandStart,
		commandStop,
		commandReport,
		commandMessage,
	}
	startJobs            = make(chan int64, 10000)
	messageQueue         = make(chan telegrambot.Message, 10000)
	endConversationQueue = make(chan EndConversationEvent, 10000)
	stopped              = false
)

func main() {

	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)

	var err error

	log.Println("Starting...")
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPassword := os.Getenv("MYSQL_PASSWORD")
	mysqlDatabaseName := os.Getenv("MYSQL_DATABASE_NAME")
	telegramBotKey := os.Getenv("TELEGRAM_BOT_KEY")

	dsn := fmt.Sprintf("%s:%s@(localhost:3306)/%s?parseTime=true", mysqlUser, mysqlPassword, mysqlDatabaseName)
	db, err = sqlx.Open("mysql", dsn)

	if err != nil {
		panic(err)
	}

	telegram = telegrambot.New(telegramBotKey)

	var wg sync.WaitGroup

	wg.Add(1)
	go func(jobs <-chan int64) {
		defer wg.Done()
		log.Println("Starting match user job")
		matchUsers(jobs)
	}(startJobs)

	for j := 0; j < 1; j++ {
		wg.Add(1)
		go func(jobs chan<- int64) {
			defer wg.Done()
			log.Println("Started load available user job")
			loadAvailableUsers(jobs)
		}(startJobs)
	}

	var workerWg sync.WaitGroup
	for i := 0; i < 3; i++ {
		workerWg.Add(1)
		go func(queue <-chan telegrambot.Message) {
			defer workerWg.Done()
			log.Println("Started a message worker...")
			messageWorker(queue)
		}(messageQueue)
	}

	for x := 0; x < 1; x++ {
		wg.Add(1)

		go func(queue <-chan EndConversationEvent) {
			defer wg.Done()
			log.Println("Started end convo worker...")
			endConversationWorker(queue)
		}(endConversationQueue)
	}

	var receiverWg sync.WaitGroup
	receiverWg.Add(1)
	go func() {
		defer receiverWg.Done()
		log.Println("Started update worker")

		var offset int64

		for {
			log.Println("Requesting updates")
			offset = processUpdates(offset)
			log.Println("Request completed")
			time.Sleep(500 * time.Millisecond)

			if stopped {
				break
			}
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	signal.Notify(sigs, syscall.SIGTERM)
	done := make(chan bool, 1)

	go func() {
		<-sigs
		done <- true
	}()

	<-done

	log.Printf("Stopping...")

	stopped = true

	receiverWg.Wait()

	close(messageQueue)

	workerWg.Wait()

	close(startJobs)
	close(endConversationQueue)

	log.Printf("Waiting for goroutines to stop...")

	wg.Wait()

	log.Printf("Closed...")
}

func loadAvailableUsers(startJobs chan<- int64) {

	for {

		u, err := retrieveAllAvailableUsers()

		if err != nil {
			log.Printf("Error retrieving everyone available: %s", err)
		} else {
			for _, x := range u {
				startJobs <- x.ChatID
			}
		}

		time.Sleep(10 * time.Second)
	}

}

// User holds user data
type User struct {
	ID            int64         `db:"id"`
	ChatID        int64         `db:"chat_id"`
	Available     bool          `db:"available"`
	LastActivity  time.Time     `db:"last_activity"`
	MatchChatID   sql.NullInt64 `db:"match_chat_id"`
	RegisterDate  time.Time     `db:"register_date"`
	PreviousMatch sql.NullInt64 `db:"previous_match"`
	AllowPictures bool          `db:"allow_pictures"`
	BannedUntil   NullTime      `db:"banned_until"`
}

func retrieveUser(chatID int64) (User, error) {
	var u User
	err := db.Get(&u, "SELECT * FROM users WHERE chat_id = ?", chatID)
	return u, err
}

func retrieveOrCreateUser(chatID int64) (User, error) {
	var u User
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM users WHERE chat_id = ?", chatID)

	if err != nil {
		return u, err
	}

	if count == 0 {
		_, err = db.Exec("INSERT INTO users(chat_id, available, last_activity, register_date, allow_pictures) VALUES (?, ?, ?, ?, 1)", chatID, false, time.Now(), time.Now())

		if err != nil {
			return u, err
		}

		telegram.SendMessage(chatID, `Welcome! Nice to meet you! I will try to match you with interesting people!

		To get started enter:

		/start

		If you're bored of a conversation, type:

		/bye

		If you want another chat partner, type /start again after typing /bye!

		Have fun,

		StrangerBot!`, emptyOpts)
	}

	return retrieveUser(chatID)
}

func updateLastActivity(id int64) {
	db.Exec("UPDATE users SET last_activity = ? WHERE id = ?", time.Now(), id)
}

func retrieveAllAvailableUsers() ([]User, error) {
	var u []User
	err := db.Select(&u, "SELECT * FROM users WHERE available = 1 AND match_chat_id IS NULL")
	return u, err
}

func retrieveAvailableUsers(c int64) ([]User, error) {
	var u []User
	err := db.Select(&u, "SELECT * FROM users WHERE chat_id != ? AND available = 1 AND match_chat_id IS NULL", c)
	return u, err
}

func shuffle(a []User) {
	for i := range a {
		j := rand.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}

func handleMessage(message telegrambot.Message) {

	u, err := retrieveOrCreateUser(message.Chat.ID)

	if err != nil {
		log.Println(err)
		return
	}

	if u.BannedUntil.Valid && time.Now().Before(u.BannedUntil.Time) {
		date := u.BannedUntil.Time.Format("02 January 2006")
		response := fmt.Sprintf("You are banned until %s", date)
		telegram.SendMessage(message.Chat.ID, response, emptyOpts)
		return
	}

	sendToHandler(u, message)

	// @TODO: Add this to a worker as well
	updateLastActivity(u.ID)
}

func sendToHandler(u User, message telegrambot.Message) {
	for _, handler := range commandHandlers {
		res := handler(u, message)

		if res {
			return
		}
	}
}

func processUpdates(offset int64) int64 {

	log.Printf("Fetching with offset %d", offset)
	updates, err := telegram.GetUpdates(offset, 20)

	if err != nil {
		log.Println(err)
	}

	return handleUpdates(updates, offset)

}

func messageWorker(messages <-chan telegrambot.Message) {
	for message := range messages {
		handleMessage(message)
	}
}

func handleUpdates(updates telegrambot.Update, offset int64) int64 {
	for _, update := range updates.Result {

		if update.ID >= offset {
			if update.ID%1000 == 0 {
				log.Printf("Update ID: %d", update.ID)
			}
			offset = (update.ID + 1)
		}

		messageQueue <- update.Message
	}
	return offset
}
