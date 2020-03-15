package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/line/line-bot-sdk-go/linebot"
)

func main() {

	bot, err := linebot.New(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_ACCESS_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Webhook endpoint
	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		fmt.Print(Accessed\n)
		events, err := bot.ParseRequest(req)
		if err != nil {
			if err == linebot.ErrInvalidSignature {
				w.WriteHeader(400)
			} else {
				w.WriteHeader(500)
			}
			return
		}
		for _, event := range events {
			if event.Type == linebot.EventTypeMessage {
				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					if event.ReplyToken == "00000000000000000000000000000000" {
						return
					}
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(parse(message.Text)).Do(); err != nil {
						log.Print(err)
					}
				case *linebot.StickerMessage:
					replyMessage := fmt.Sprintf(
						"sticker id is %s, stickerResourceType is %s", message.StickerID, message.StickerResourceType)
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do(); err != nil {
						log.Print(err)
					}
				}
			}
		}
	})

	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatal(err)
	}

}

func parse(message string) string {
	if startsWith(message, "路線一覧:") {
		return fetchText(message[13:len(message)])
	} else if startsWith(message, "駅一覧:") {
		code, err := strconv.Atoi(message[10:len(message)])
		if err != nil {
			return "エラー:\n路線一覧で取得した路線番号を入力してください"
		}
		return getLineData(code)
	} else if startsWith(message, "駅情報:") {
		code, err := strconv.Atoi(message[10:len(message)])
		if err != nil {
			return "エラー:\n駅一覧で取得した駅番号を入力してください"
		}
		return getStationData(code)
	} else if startsWith(message, "所属路線一覧:") {
		code, err := strconv.Atoi(message[19:len(message)])
		if err != nil {
			return "エラー:\n駅一覧で取得した駅番号を入力してください"
		}
		return getGroupData(code)
	} else if startsWith(message, "隣接駅:") {
		code, err := strconv.Atoi(message[10:len(message)])
		if err != nil {
			return "エラー:\n駅一覧で取得した駅番号を入力してください"
		}
		return getJoinData(code)
	}
	return helpMessage
}