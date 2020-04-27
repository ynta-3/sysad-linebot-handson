// #3 天気確認機能の実装
package main

// 利用したい外部のコードを読み込む
import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/line/line-bot-sdk-go/linebot"
)

const verifyToken = "00000000000000000000000000000000"

// main関数は最初に呼び出されることが決まっている
func main() {
	// ランダムな数値を利用する際のシード値の設定
	rand.Seed(time.Now().UnixNano())

	// LINEのAPIを利用する設定
	bot, err := linebot.New(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_ACCESS_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// LINEサーバからのリクエストを受け取ったときの処理
	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		fmt.Print("Accessed\n")

		// リクエストを扱いやすい形に変換する
		events, err := bot.ParseRequest(req)
		// 変換に失敗したとき
		if err != nil {
			fmt.Println("ParseRequest error:", err)
			if err == linebot.ErrInvalidSignature {
				w.WriteHeader(400)
			} else {
				w.WriteHeader(500)
			}
			return
		}

		// LINEサーバから来たメッセージによってやる処理を変える
		for _, event := range events {
			// LINEサーバのverify時は何もしない
			if event.ReplyToken == verifyToken {
				return
			}

			// メッセージが来たとき
			if event.Type == linebot.EventTypeMessage {
				// 返信を生成する
				replyMessage := getReplyMessage(event)
				// 生成した返信を送信する
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do(); err != nil {
					log.Print(err)
				}
			}
		}
	})

	// LINEサーバからのリクエストを受け取る
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatal(err)
	}
}

const helpMessage = `使い方
テキストメッセージ:
	"おみくじ"がメッセージに入ってれば今日の運勢を占うよ！
	それ以外はやまびこを返すよ！
スタンプ:
	スタンプの情報を答えるよ！
位置情報:
	その場所の天気・気温・湿度を答えるよ！
それ以外:
	それ以外にはまだ対応してないよ！ごめんね...`

// 返信を生成する
func getReplyMessage(event *linebot.Event) (replyMessage string) {
	// 来たメッセージの種類によって分岐する
	switch message := event.Message.(type) {
		// テキストメッセージが来たとき
	case *linebot.TextMessage:
		// さらに「おみくじ」という文字列が含まれているとき
		if strings.Contains(message.Text, "おみくじ") {
			// おみくじ結果を取得する
			return getFortune()
		}
		// そうじゃないときはオウム返しする
		return message.Text

	// スタンプが来たとき
	case *linebot.StickerMessage:
		replyMessage := fmt.Sprintf("sticker id is %s, stickerResourceType is %s", message.StickerID, message.StickerResourceType)
		return replyMessage

	// 位置情報が来たとき
	case *linebot.LocationMessage:
		// その場所の天気
		replyMessage, err := getWeather(message)
		if err != nil {
			log.Print(err)
		}
		return replyMessage

	// どっちでもないとき
	default:
		return helpMessage
	}
}

// おみくじ結果の生成
func getFortune() string {
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
	// rand.Intn(10)は1～10のランダムな整数を返す
	return oracles[rand.Intn(10)]
}

// 天気の情報で帰ってくる形式 (1)
type WeatherData struct {
	Weather []Weather `json:"weather"`
	Info    Info      `json:"main"`
}

// 天気の情報で帰ってくる形式 (2)
type Weather struct {
	Main string `json:"main"`
	Icon string `json:"icon"` // 現状使わない
}

// 天気の情報で帰ってくる形式 (3)
type Info struct {
	Temp     float32 `json:"temp"`     // 気温(K)
	Humidity float32 `json:"humidity"` // 湿度(%)
}

// 天気の情報の文字列をつくる
func getWeather(location *linebot.LocationMessage) (string, error) {
	// 緯度経度からOpenWeatherMapAPIのURLを作成
	lat := strconv.FormatFloat(location.Latitude, 'f', 6, 64)
	lon := strconv.FormatFloat(location.Longitude, 'f', 6, 64)
	url := "http://api.openweathermap.org/data/2.5/weather?lat=" + lat + "&lon=" + lon + "&APPID=" + os.Getenv("APP_ID")

	// OpenWeatherMapAPIへのリクエスト
	res, err := http.Get(url)
	if err != nil {
		return "内部でエラーが発生しました", err
	}
	defer res.Body.Close()

	// OpenWeatherMapAPIからのレスポンスを扱いやすい形に変換する
	weatherData := WeatherData{}
	err = json.NewDecoder(res.Body).Decode(&weatherData)
	if err != nil {
		return "内部でエラーが発生しました", err
	}

	// 返信メッセージの作成
	text := ` 現在の天気情報
天気 : ` + weatherData.Weather[0].Main + `
気温 : ` + fmt.Sprintf("%.2f", (weatherData.Info.Temp-273.15)) + "℃" + `
湿度 : ` + fmt.Sprintf("%.2f", weatherData.Info.Humidity) + "%"

	return text, nil
}
