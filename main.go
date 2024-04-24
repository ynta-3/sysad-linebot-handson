package main

// 利用したい外部のコードを読み込む
import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

// main関数外で利用するためにここで宣言する
// 詳しくは「スコープ」や「グローバル変数」で検索してください
var (
	greeting = fmt.Sprintf("Hello, %v!", "World")
)

// main関数は最初に呼び出されることがGo言語の仕様として決まっている
func main() {
	log.Println(greeting)
}
