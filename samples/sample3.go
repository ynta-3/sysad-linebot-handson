package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
		fmt.Print("Accessed\n")
		events, err := bot.ParseRequest(req)
		if err != nil {
			fmt.Println("ParseRequest error:", err)
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
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(parse(message.Text))).Do(); err != nil {
						log.Print(err)
					}
				case *linebot.StickerMessage:
					replyMessage := fmt.Sprintf(
						"sticker id is %s, stickerResourceType is %s", message.StickerID, message.StickerResourceType)
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do(); err != nil {
						log.Print(err)
					}
				case *linebot.LocationMessage:
					text, err := weather(message)
					if err != nil {
						log.Print(err)
					}
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(text)).Do(); err != nil {
						log.Print(err)
					}
				default:
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(helpMessage)).Do(); err != nil {
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

var helpMessage = `使い方
テキストメッセージ: 
	"おみくじ"がメッセージに入ってれば今日の運勢を占うよ！
	それ以外はやまびこを返すよ！
スタンプ: 
	スタンプの情報を答えるよ！
位置情報:
	その場所の天気・気温・湿度を答えるよ！
それ以外:
	それ以外にはまだ対応してないよ！ごめんね...`

func parse(message string) string {
	if strings.Contains(message, "おみくじ") {
		return fortune()
	}
	return message
}

func fortune() string {
	oracles := map[int]string{
		0: "大吉",
		1: "中吉",
		2: "小吉",
		3: "末吉",
		4: "吉",
		5: "凶",
		6: "末凶",
		7: "小凶",
		8: "中凶",
		9: "大凶",
	}

	rand.Seed(time.Now().UnixNano())
	return oracles[rand.Intn(10)]
}

// WeatherData represents Json core fields.
type WeatherData struct {
	Weather []Weather `json:"weather"`
	Info    Info      `json:"main"`
}

// Weather represents weather item.
type Weather struct {
	Main string `json:"main"`
	Icon string `json:"icon"`
}

// Info represents main item.
type Info struct {
	Temp     float32 `json:"temp"`     // 気温(K)
	Humidity float32 `json:"humidity"` // 湿度(%)
}

func weather(location *linebot.LocationMessage) (string, error) {

	// 緯度経度からOWMのURLを作成
	lat := strconv.FormatFloat(location.Latitude, 'f', 6, 64)
	lon := strconv.FormatFloat(location.Longitude, 'f', 6, 64)
	url := "http://api.openweathermap.org/data/2.5/weather?lat=" + lat + "&lon=" + lon + "&APPID=" + os.Getenv("APP_ID")

	// 天気情報を取得
	res, err := http.Get(url)
	if err != nil {
		return "内部でエラーが発生しました", err
	}

	defer res.Body.Close()
	byteArray, _ := ioutil.ReadAll(res.Body)
	jsonBytes := ([]byte)(string(byteArray[:]))

	weatherData := new(WeatherData)
	if err := json.Unmarshal(jsonBytes, weatherData); err != nil {
		return "内部でエラーが発生しました", err
	}

	//メッセージ作成
	text := ` 現在の天気情報
天気 : ` + weatherData.Weather[0].Main + `
気温 : ` + fmt.Sprintf("%.2f", (weatherData.Info.Temp-273.15)) + "℃" + `
湿度 : ` + fmt.Sprintf("%.2f", weatherData.Info.Humidity) + "%"

	return text, nil

}
