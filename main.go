
package main

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

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/line/line-bot-sdk-go/linebot"
)

var (
	db *sqlx.DB
)

func main() {
	rand.Seed(time.Now().UnixNano())

	_db, err := sqlx.Connect(
		"mysql",
		fmt.Sprintf(
			"%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
			os.Getenv("DB_USERNAME"),
			os.Getenv("DB_HOSTNAME"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_DATABASE"),
			))

	if err != nil {
		log.Fatalf("Cannot Connect to Database: %s", err)
	}
	db = _db

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
				const verifyToken = "00000000000000000000000000000000"
				if event.ReplyToken == verifyToken {
					return
				}
				replyMessage := getReplyMessage(event)
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do(); err != nil {
					log.Print(err)
				}
			}
		}
	})

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
TodoList:
	"todo"に続けて実行したい操作を入力してね！
		list
		add "タスク名" "期限"
		done "タスクID"
	例:
		todo list
		todo add レポート 2/24
		todo done 12
それ以外:
	それ以外にはまだ対応してないよ！ごめんね...`

func getReplyMessage(event *linebot.Event) (replyMessage string) {

	switch message := event.Message.(type) {
	case *linebot.TextMessage:
		if strings.Contains(message.Text, "おみくじ") {
			return getFortune()
		} else if strings.HasPrefix(message.Text, "todo") {
			return dealTodo(message)
		}

		return message.Text

	case *linebot.StickerMessage:
		replyMessage := fmt.Sprintf("sticker id is %s, stickerResourceType is %s", message.StickerID, message.StickerResourceType)
		return replyMessage

	case *linebot.LocationMessage:
		replyMessage, err := getWeather(message)
		if err != nil {
			log.Print(err)
		}
		return replyMessage

	default:
		return helpMessage

	}

}

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
	Icon string `json:"icon"` // 現状使わない
}

// Info represents main item.
type Info struct {
	Temp     float32 `json:"temp"`     // 気温(K)
	Humidity float32 `json:"humidity"` // 湿度(%)
}

func getWeather(location *linebot.LocationMessage) (string, error) {

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

	weatherData := WeatherData{}
	err = json.NewDecoder(res.Body).Decode(&weatherData)
	if err != nil {
		return "内部でエラーが発生しました", err
	}

	//メッセージ作成
	text := ` 現在の天気情報
天気 : ` + weatherData.Weather[0].Main + `
気温 : ` + fmt.Sprintf("%.2f", (weatherData.Info.Temp-273.15)) + "℃" + `
湿度 : ` + fmt.Sprintf("%.2f", weatherData.Info.Humidity) + "%"

	return text, nil

}

type Task struct {
	ID uint			`db:"id"`
	Todo string  `db:"todo"`
	DueDate string  `db:"due_date"`
}

type Tasks []Task

func dealTodo(message *linebot.TextMessage) string {
	token := strings.Split(message.Text, " ")
	if len(token) <= 1 {
			return helpMessage
	}
	if token[1] == "list" {
		return getTodoList()
	} else if token[1] == "add" {
		return addTodo(token)
	} else if token[1] == "done" {
		return deleteTodo(token)
	}
	return helpMessage
}

func getTodoList() string {
	tasks := Tasks{}
	err := db.Select(&tasks, "SELECT * from tasks")
	if err != nil {
		fmt.Print(err)
		return  fmt.Sprintf("db error: %v", err)
	}
	replyMessage := "ID/ToDo/期限"
	for _, task := range tasks {
		replyMessage += fmt.Sprintf("\n%d/%s/%s", task.ID, task.Todo, task.DueDate)
	}
	return replyMessage
}

func addTodo(token []string) string {
	result, err := db.Exec("INSERT INTO tasks (todo, due_date) VALUES (?, ?)", token[2], token[3])
	if err != nil {
		fmt.Print(err)
		return fmt.Sprintf("db error: %v", err)
	}
	todoID, err := result.LastInsertId()
	if err != nil {
		fmt.Print(err)
		return fmt.Sprintf("db error: %v", err)
	}
	replyMessage := fmt.Sprintf("todo added\nID:%d\ntodo:%s\n期限:%s", todoID, token[2], token[3])
	return replyMessage
}

func deleteTodo(token []string) string {
	id, err := strconv.Atoi(token[2])
	if err != nil {
		return "内部でエラーが発生しました"
	}
	_, err = db.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		fmt.Print(err)
		return fmt.Sprintf("db error: %v", err)
	}
	replyMessage := fmt.Sprintf("todo deleted\nID:%d", id)
	return replyMessage
}
