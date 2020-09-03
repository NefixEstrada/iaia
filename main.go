package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	twitterscraper "github.com/n0madic/twitter-scraper"
	tb "gopkg.in/tucnak/telebot.v2"
)

const compte = "bondiabonanit"

var (
	grups       map[int64]*tb.Chat = map[int64]*tb.Chat{}
	bonDia                         = []salutació{}
	bonDiaLock  sync.Mutex
	bonaNit     = []salutació{}
	bonaNitLock sync.Mutex
	grupsLock   sync.Mutex
	últimTuit   string
)

type salutació struct {
	Missatge string
	Foto     string
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	bot, err := tb.NewBot(tb.Settings{
		Token:  os.Getenv("TELEGRAM_TOKEN"),
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatalln(err)
	}

	bot.Handle(tb.OnSticker, func(m *tb.Message) {
		if m.Chat.Type == tb.ChatGroup {
			grupsLock.Lock()
			if _, ok := grups[m.Chat.ID]; !ok {
				fmt.Printf("grup %d descobert\n", m.ID)
				grups[m.Chat.ID] = m.Chat
			}
			grupsLock.Unlock()
		}
	})

	go bot.Start()

	actualitzarSalutacions()

	go func() {
		for range time.NewTicker(24 * time.Hour).C {
			actualitzarSalutacions()
		}
	}()

	for range time.NewTicker(time.Minute).C {
		ara := time.Now()
		if ara.Hour() == 7 && ara.Minute() == 00 {
			bonDiaLock.Lock()
			s := triar(bonDia)
			bonDiaLock.Unlock()

			enviar(bot, s)
		}

		if ara.Hour() == 23 && ara.Minute() == 0 {
			bonaNitLock.Lock()
			s := triar(bonaNit)
			bonaNitLock.Unlock()

			enviar(bot, s)
		}
	}
}

func actualitzarSalutacions() {
	p, err := twitterscraper.GetProfile(compte)
	if err != nil {
		log.Println(err)
		return
	}

	primerTuit := ""
	for tweet := range twitterscraper.GetTweets(context.Background(), compte, p.TweetsCount) {
		if tweet.Error != nil {
			log.Println(tweet.Error)
			break
		}

		if primerTuit == "" {
			primerTuit = tweet.ID
		}

		if tweet.ID == últimTuit {
			break
		}

		text := strings.ToLower(tweet.Text)
		if strings.Contains(text, "dia") {
			bonDiaLock.Lock()
			bonDia = afegirSalutació(tweet, bonDia)
			bonDiaLock.Unlock()
		} else {
			if strings.Contains(text, "nit") {
				bonaNitLock.Lock()
				bonaNit = afegirSalutació(tweet, bonaNit)
				bonaNitLock.Unlock()
			}
		}
	}

	últimTuit = primerTuit

	fmt.Printf("trobats %d bon dies\n", len(bonDia))
	fmt.Printf("trobats %d bona nits\n", len(bonaNit))
}

func afegirSalutació(t *twitterscraper.Result, s []salutació) []salutació {
	if len(t.Photos) == 0 {
		return s
	}

	return append(s, salutació{
		Missatge: t.Text,
		Foto:     t.Photos[0],
	})
}

func triar(opcions []salutació) salutació {
	return opcions[rand.Intn(len(opcions))]
}

func enviar(bot *tb.Bot, s salutació) {
	grupsLock.Lock()
	defer grupsLock.Unlock()

	for _, g := range grups {
		_, err := bot.Send(g, &tb.Photo{
			Caption: s.Missatge,
			File:    tb.FromURL(s.Foto),
		})
		if err != nil {
			log.Println(err)
		}
	}
}
