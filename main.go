package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"

	"github.com/ahmdrz/goinsta"
	"github.com/ahmdrz/goinsta/store"
	"github.com/ashwanthkumar/slack-go-webhook"
	"github.com/joho/godotenv"
	"strings"
	"time"
)

var insta *goinsta.Instagram

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("[Info] Could not open .env file, continuing with OS environment: %v", err)
	}

	login()
	likeFriendsFeed()
}

// login will try to reload a previous session, and will create a new one if it can't
func login() {
	err := reloadSession()
	if err != nil {
		insta = goinsta.New(os.Getenv("USERNAME"), os.Getenv("PASSWORD"))
		err := insta.Login()
		check(err)

		key := createKey()
		bytes, err := store.Export(insta, key)
		check(err)
		err = ioutil.WriteFile("session", bytes, 0644)
		check(err)
		log.Println("Created and saved the session")
	}
}

// reloadSession will attempt to recover a previous session
func reloadSession() error {
	if _, err := os.Stat("session"); os.IsNotExist(err) {
		return errors.New("No session found")
	}

	session, err := ioutil.ReadFile("session")
	check(err)

	key, err := ioutil.ReadFile("key")
	check(err)

	insta, err = store.Import(session, key)
	if err != nil {
		return errors.New("Couldn't recover the session")
	}

	return nil
}

// createKey creates a key and saves it to file
func createKey() (key []byte) {
	key = make([]byte, 32)
	_, err := rand.Read(key)
	check(err)
	err = ioutil.WriteFile("key", key, 0644)
	check(err)
	return
}

func likeFriendsFeed() {
	following, err := insta.SelfTotalUserFollowing()
	check(err)
	var report []string

	for _, user := range following.Users {
		r, err := insta.LatestUserFeed(user.ID)
		check(err)
		for _, item := range r.Items {
			if !item.HasLiked {
				insta.Like(item.ID)
				time.Sleep(20 * time.Second)
				report = append(report, user.FullName+" has been liked")
			}
		}
	}

	Slack(strings.Join(report, "\n"), true)
}

func check(err error) {
	if err != nil {
		Slack(err.Error(), false)
	}
}

// Send message to Slack
func Slack(body string, success bool) {
	status := func() string {
		if success {
			return "Success"
		}
		return "Failure"
	}()

	attachment := slack.Attachment{}
	attachment.AddField(slack.Field{Title: "Status", Value: status})
	payload := slack.Payload{
		Text:        body,
		Username:    "Insta-bot",
		Channel:     "#general",
		IconEmoji:   ":sunrise_over_mountains:",
		Attachments: []slack.Attachment{attachment},
	}
	err := slack.Send(os.Getenv("SLACK"), "", payload)
	if len(err) > 0 {
		fmt.Printf("error: %s\n", err)
	}
}
