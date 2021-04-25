// #2 おみくじの実装
package main

// 利用したい外部のコードを読み込む
import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

const verifyToken = "00000000000000000000000000000000"

// init関数はmain関数実行前の初期化のために呼び出されることがGo言語の仕様として決まっている
func init() {
	// ランダムな数値を生成する際のシード値の設定
	rand.Seed(time.Now().UnixNano())
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

	// LINEサーバからのリクエストを受け取ったときの処理
	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		log.Println("Accessed")

		// リクエストを扱いやすい形に変換する
		events, err := bot.ParseRequest(req)
		switch err {
		case nil:
		// 変換に失敗したとき
		case linebot.ErrInvalidSignature:
			log.Println("ParseRequest error:", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		default:
			log.Println("ParseRequest error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// LINEサーバから来たメッセージによって行う処理を変える
		for _, event := range events {
			// LINEサーバからのverify時は何もしない
			if event.ReplyToken == verifyToken {
				return
			}

			switch event.Type {
			// メッセージが来たとき
			case linebot.EventTypeMessage:
				// 返信を生成する
				replyMessage := getReplyMessage(event)
				// 生成した返信を送信する
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do(); err != nil {
					log.Print(err)
				}
			// それ以外
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
それ以外:
	それ以外にはまだ対応してないよ！ごめんね...`

// 返信を生成する
func getReplyMessage(event *linebot.Event) (replyMessage string) {
	// 来たメッセージの種類によって行う処理を変える
	switch message := event.Message.(type) {
	// テキストメッセージが来たとき
	case *linebot.TextMessage:
		// さらに「おみくじ」という文字列が含まれているとき
		if strings.Contains(message.Text, "おみくじ") {
			// おみくじ結果を取得する
			return getFortune()
		}
		// それ以外のときはオウム返しする
		return message.Text

	// スタンプが来たとき
	case *linebot.StickerMessage:
		return fmt.Sprintf("sticker id is %s, stickerResourceType is %s", message.StickerID, message.StickerResourceType)

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
