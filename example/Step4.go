package main

// 利用したい外部のコードを読み込む
import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

// main関数外で利用するためにここで宣言する
// 詳しくは「スコープ」や「グローバル変数」で検索してください
var (
	db *sqlx.DB
)

// init関数はmain関数実行前の初期化のために呼び出されることがGo言語の仕様として決まっている
func init() {

	// ここで.envファイル全体を読み込みます。
	// この読み込み処理がないと、個々の環境変数が取得出来ません。
	// 読み込めなかったら err にエラーが入ります。
	err := godotenv.Load(".env")

	// もし err がnilではないなら、"読み込み出来ませんでした"が出力されます。
	if err != nil {
		fmt.Printf("読み込み出来ませんでした: %v", err)
	}

	// ランダムな数値を生成する際のシード値の設定
	rand.Seed(time.Now().UnixNano())
	// データベースへ接続する
	db = sqlx.MustConnect(
		"mysql",
		fmt.Sprintf(
			"%v@tcp(%v:%v)/%v?charset=utf8&parseTime=True&loc=Local",
			os.Getenv("DB_USERNAME"),
			os.Getenv("DB_HOSTNAME"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_DATABASE"),
		))
}

// main関数は最初に呼び出されることがGo言語の仕様として決まっている
func main() {

	// LINEのAPIを利用する設定
	bot, err := linebot.New(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_ACCESS_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// サーバ起動メッセージ
	log.Println("サーバが起動しました!")

	// LINEサーバからのリクエストを受け取ったときの処理
	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		log.Println("Accessed")

		// リクエストを扱いやすい形に変換する
		events, err := bot.ParseRequest(req)
		if err != nil {
			if err == linebot.ErrInvalidSignature {
				w.WriteHeader(http.StatusBadRequest)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		// LINEサーバから来たメッセージによって行う処理を変える
		for _, event := range events {
			switch event.Type {
			// メッセージが来たとき
			case linebot.EventTypeMessage:
				// 返信を生成する
				replyMessage := getReplyMessage(event)
				// 生成した返信を送信する
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do(); err != nil {
					log.Print(err)
				}
			// それ以外のとき
			default:
				continue
			}
		}
	})

	// LINEサーバからのリクエストを受け取るプロセスを起動
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
			// あるいは「todo」という文字列で始まるとき
		} else if strings.HasPrefix(message.Text, "todo") {
			// Todo用のメッセージを生成する
			return dealTodo(message)
		}

		// それ以外のときはオウム返しする
		return message.Text

	// スタンプが来たとき
	case *linebot.StickerMessage:
		replyMessage := fmt.Sprintf("sticker id is %v, stickerResourceType is %v", message.StickerID, message.StickerResourceType)
		return replyMessage

	// 位置情報が来たとき
	case *linebot.LocationMessage:
		// その場所の天気
		replyMessage, err := getWeather(message)
		if err != nil {
			log.Print(err)
		}
		return replyMessage

	// それ以外のとき
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
	// rand.Intn(10)は0～9のランダムな整数を返す
	return oracles[rand.Intn(len(oracles))]
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
	url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?lat=%v&lon=%v&APPID=%v", lat, lon, os.Getenv("APP_ID"))

	// OpenWeatherMapAPIへのリクエスト
	res, err := http.Get(url)
	if err != nil {
		return "Botサーバーでエラーが発生しました", err
	}
	defer res.Body.Close()

	// OpenWeatherMapAPIからのレスポンスを扱いやすい形に変換する
	weatherData := WeatherData{}
	err = json.NewDecoder(res.Body).Decode(&weatherData)
	if err != nil {
		return "Botサーバーでエラーが発生しました", err
	}

	// 返信メッセージの作成
	text := ` 現在の天気情報
天気 : ` + weatherData.Weather[0].Main + `
気温 : ` + fmt.Sprintf("%.2f", (weatherData.Info.Temp-273.15)) + "℃" + `
湿度 : ` + fmt.Sprintf("%.2f", weatherData.Info.Humidity) + "%"

	return text, nil

}

// データベースでTodoを扱う形式 (構造体)
type Task struct {
	ID      uint   `db:"id"`
	Todo    string `db:"todo"`
	DueDate string `db:"due_date"`
}

// Todo用のメッセージを生成
func dealTodo(message *linebot.TextMessage) string {
	// 受け取ったメッセージを空白で区切る
	token := strings.Split(message.Text, " ")

	// 区切った文字列の個数が1つ以下のときはヘルプを返す
	if len(token) <= 1 {
		return helpMessage
	}

	// Todoリスト表示
	if token[1] == "list" {
		return getTodoList()
		// TodoリストにTodoを追加
	} else if token[1] == "add" {
		return addTodo(token)
		// Todoリストから指定したIDのTodoを削除
	} else if token[1] == "done" {
		return deleteTodo(token)
	}
	return helpMessage
}

// Todoリストの取得
func getTodoList() string {
	var tasks []Task
	// MySQLデータベースへのクエリを発行して一覧を取得する
	err := db.Select(&tasks, "SELECT * from tasks")
	if err != nil {
		fmt.Print(err)
		return fmt.Sprintf("db error: %v", err)
	}

	// メッセージの生成
	replyMessage := "ID/ToDo/期限"
	for _, task := range tasks {
		replyMessage += fmt.Sprintf("\n%d/%v/%v", task.ID, task.Todo, task.DueDate)
	}
	return replyMessage
}

// TodoリストへのTodoの追加
func addTodo(token []string) string {
	// MySQLデータベースへのクエリを発行してTodoを追加する
	result, err := db.Exec("INSERT INTO tasks (todo, due_date) VALUES (?, ?)", token[2], token[3])
	if err != nil {
		log.Printf("db error: %v", err)
		return "Botサーバーでエラーが発生しました"
	}

	// (最後の)追加されたTodoのIDを取得する
	todoID, err := result.LastInsertId()
	if err != nil {
		log.Printf("db error: %v", err)
		return "Botサーバーでエラーが発生しました"
	}

	// メッセージの生成
	replyMessage := fmt.Sprintf("todo added\nID:%d\ntodo:%v\n期限:%v", todoID, token[2], token[3])
	return replyMessage
}

// Todoの削除
func deleteTodo(token []string) string {
	// IDを文字列から数値に変換する
	id, err := strconv.Atoi(token[2])
	if err != nil {
		return "Botサーバーでエラーが発生しました"
	}

	// MySQLデータベースへのクエリを発行してそのIDのTodoを削除する
	_, err = db.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		log.Printf("db error: %v", err)
		return "Botサーバーでエラーが発生しました"
	}

	// メッセージの生成
	replyMessage := fmt.Sprintf("todo deleted\nID:%d", id)
	return replyMessage
}
