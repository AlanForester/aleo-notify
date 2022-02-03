package main

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var queue chan string
var regs map[int64]bool

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	regs := make(map[int64]bool, 0)
	queue = make(chan string, 0)

	token := os.Getenv("TELEGRAM_APITOKEN")

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		panic(err)
	}

	bot.Debug = true

	// Create a new UpdateConfig struct with an offset of 0. Offsets are used
	// to make sure Telegram knows we've handled previous values and we don't
	// need them repeated.
	updateConfig := tgbotapi.NewUpdate(0)

	// Tell Telegram we should wait up to 30 seconds on each request for an
	// update. This way we can get information just as quickly as making many
	// frequent requests without having to send nearly as many.
	updateConfig.Timeout = 30

	// Start polling Telegram for updates.
	updates := bot.GetUpdatesChan(updateConfig)

	go func(updates tgbotapi.UpdatesChannel) {
		// Let's go through each update that we're getting from Telegram.
		for update := range updates {
			if update.Message != nil {
				log.Printf("Message %#v", update)
				//msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Im ALEO bot")
				// Telegram can send many types of updates depending on what your Bot
				// is up to. We only want to look at messages for now, so we can
				// discard any other updates.
				if update.Message == nil {
					continue
				}

				// Now that we know we've gotten a new message, we can construct a
				// reply! We'll take the Chat ID and Text from the incoming message
				// and use it to create a new message.

				if update.Message.Text == "/stats" {
					miners := getMiners()
					var minersKb [][]tgbotapi.InlineKeyboardButton
					totalHr := float64(0)

					for _, m := range miners {
						if !m.Active {
							continue
						}
						totalHr += m.HashRate
						row := tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("⛏ %s - %0.2f h/s", m.Ip, m.HashRate), m.Ip))
						minersKb = append(minersKb, row)
					}

					var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
						minersKb...,
					)
					message := fmt.Sprintf("💎 Общий хешрейт %0.2f", totalHr)
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
					msg.ReplyMarkup = numericKeyboard
					if _, err := bot.Send(msg); err != nil {
						// Note that panics are a bad way to handle errors. Telegram can
						// have service outages or network errors, you should retry sending
						// messages or more gracefully handle failures.
						panic(err)
					}
				}

				if update.Message.Text == "/start" {
					regs[update.Message.Chat.ID] = true
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Registered! Im aleo bot")
					msg.ReplyToMessageID = update.Message.MessageID
					// Okay, we're sending our message off! We don't care about the message
					// we just sent, so we'll discard it.
					if _, err := bot.Send(msg); err != nil {
						// Note that panics are a bad way to handle errors. Telegram can
						// have service outages or network errors, you should retry sending
						// messages or more gracefully handle failures.
						panic(err)
					}
				}
			} else if update.CallbackQuery != nil {
				// Respond to the callback query, telling Telegram to show the user
				// a message with the data received.
				callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
				if _, err := bot.Request(callback); err != nil {
					panic(err)
				}

				// And finally, send a message containing the data received.
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Data)
				if _, err := bot.Send(msg); err != nil {
					panic(err)
				}
			}

		}
	}(updates)

	go func() {
		for {
			message := <-queue
			for chat, _ := range regs {
				unascaped, _ := url.QueryUnescape(message)
				msg := tgbotapi.NewMessage(chat, unascaped)
				// Okay, we're sending our message off! We don't care about the message
				// we just sent, so we'll discard it.
				if _, err := bot.Send(msg); err != nil {
					// Note that panics are a bad way to handle errors. Telegram can
					// have service outages or network errors, you should retry sending
					// messages or more gracefully handle failures.
					panic(err)
				}
			}
			log.Printf("%v", message)
		}
	}()

	http.HandleFunc("/", handler) // each request calls handler
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

type Stats struct {
	StartTime  string      `json:"start_time"`
	Servers    []Miner     `json:"servers"`
	Pool       interface{} `json:"pool"`
	ActivePool interface{} `json:"active_pool"`
}

type Miner struct {
	Ip       string  `json:"ip"`
	Blocks   int     `json:"blocks_count"`
	Active   bool    `json:"active"`
	HashRate float64 `json:"hashrate"`
}

func getMiners() []Miner {
	u := os.Getenv("STATS_URL")

	client := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", os.Getenv("STATS_AUTH")))

	res, getErr := client.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	stat := Stats{}
	jsonErr := json.Unmarshal(body, &stat)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	fmt.Printf("%+v", stat.Servers)

	return stat.Servers
}

func handler(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatalln(err)
	}

	msg := strings.Replace(string(b), "text=", "", 1)

	queue <- msg

	//fmt.Fprintf(w, "URL.Path = %q\n", copyBody)
}
