// #3 天気確認機能の実装
package main

// 利用したい外部のコードを読み込む
import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

// init関数はmain関数実行前の初期化のために呼び出されることがGo言語の仕様として決まっている
func init() {
	// ランダムな数値を生成する際のシード値の設定
	rand.Seed(time.Now().UnixNano())
}

// main関数は最初に呼び出されることがGo言語の仕様として決まっている
func main() {
	// ここで.envファイル全体を読み込みます。
	// この読み込み処理がないと、個々の環境変数が取得出来ません。
	// 読み込めなかったら err にエラーが入ります。
	err := godotenv.Load(".env")

	// もし err がnilではないなら、"読み込み出来ませんでした"が出力されます。
	if err != nil {
		fmt.Printf("読み込み出来ませんでした: %v", err)
	}

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
				if _, err = bot.ReplyMessage(event.ReplyToken, replyMessage).Do(); err != nil {
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
位置情報:
	その場所の天気・気温・湿度を答えるよ！
それ以外:
	それ以外にはまだ対応してないよ！ごめんね...`

// 返信を生成する
func getReplyMessage(event *linebot.Event) (replyMessage linebot.SendingMessage) {
	// 来たメッセージの種類によって行う処理を変える
	switch message := event.Message.(type) {
	// テキストメッセージが来たとき
	case *linebot.TextMessage:
		// さらに「おみくじ」という文字列が含まれているとき
		if strings.Contains(message.Text, "おみくじ") {
			// おみくじ結果を取得する
			return linebot.NewTextMessage(getFortune())
		}
		// それ以外のときはオウム返しする
		return linebot.NewTextMessage(message.Text)

	// スタンプが来たとき
	case *linebot.StickerMessage:
		return linebot.NewTextMessage(fmt.Sprintf("sticker id is %v, stickerResourceType is %v", message.StickerID, message.StickerResourceType))

		// 位置情報が来たとき
	case *linebot.LocationMessage:
		// その場所の天気
		replyMessage, err := getWeekWeather(message)
		if err != nil {
			log.Print(err)
		}
		return replyMessage

	// それ以外のとき
	default:
		return linebot.NewTextMessage(helpMessage)
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
	return oracles[rand.Intn(len(oracles))]
}

// 天気の情報で帰ってくる形式 (2)
type Weather struct {
	Icon string `json:"icon"` // 現状使わない
}

type DaysWeather struct {
	WeatherPer3h []WeatherData `json:"list"`
}

type WeatherData struct {
	MainData Main      `json:"main"`
	Humidity float32   `json:"humidity"`
	Weathers []Weather `json:"weather"`
}

type Main struct {
	TempMin  float32 `json:"temp_min"`
	TempMax  float32 `json:"temp_max"`
	Humidity float32 `json:"humidity"`
}

func getWeekWeather(location *linebot.LocationMessage) (*linebot.FlexMessage, error) {
	lat := strconv.FormatFloat(location.Latitude, 'f', 6, 64)
	lon := strconv.FormatFloat(location.Longitude, 'f', 6, 64)
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/forecast?lat=%s&lon=%s&exclude=current,minutely,hourly,alerts&appid=%s", lat, lon, os.Getenv("APP_ID"))
	// OpenWeatherMapAPIへのリクエスト
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// OpenWeatherMapAPIからのレスポンスを扱いやすい形に変換する
	weatherData := DaysWeather{}
	err = json.NewDecoder(res.Body).Decode(&weatherData)
	if err != nil {
		return nil, err
	}
	return CreateWeatherCarouseMessage(weatherData), nil
}

func max(a int, b int) int {
	if a >= b {
		return a
	}
	return b
}

func min(a int, b int) int {
	if a <= b {
		return a
	}
	return b
}

func CreateWeatherCarouseMessage(data DaysWeather) *linebot.FlexMessage {
	var tempMax = []int{math.MinInt, math.MinInt, math.MinInt}
	for i := 0; i < 24; i++ {
		tempMax[i/8] = max(tempMax[i/8], int(data.WeatherPer3h[i].MainData.TempMax-273.15))
	}
	var tempMin = []int{math.MaxInt, math.MaxInt, math.MaxInt}
	for i := 0; i < 24; i++ {
		tempMin[i/8] = min(tempMin[i/8], int(data.WeatherPer3h[i].MainData.TempMin-273.15))
	}
	var humidity = []float32{0, 0, 0}
	for i := 0; i < 24; i++ {
		humidity[i/8] += data.WeatherPer3h[i].MainData.Humidity / 8
	}

	resp := linebot.NewFlexMessage(
		"Weather Information",
		&linebot.CarouselContainer{
			Type: linebot.FlexContainerTypeCarousel,
			Contents: []*linebot.BubbleContainer{
				{
					Type:      linebot.FlexContainerTypeBubble,
					Direction: linebot.FlexBubbleDirectionTypeLTR,
					Header: &linebot.BoxComponent{
						Type:   linebot.FlexComponentTypeBox,
						Layout: linebot.FlexBoxLayoutTypeBaseline,
						Contents: []linebot.FlexComponent{
							&linebot.TextComponent{
								Type:   linebot.FlexComponentTypeText,
								Text:   "今日の天気",
								Size:   linebot.FlexTextSizeTypeLg,
								Align:  linebot.FlexComponentAlignTypeCenter,
								Weight: linebot.FlexTextWeightTypeBold,
								//Color:      "",
								//Action:     nil,
							},
						},
						CornerRadius: linebot.FlexComponentCornerRadiusTypeXxl,
						BorderColor:  "#00bfff",
						//Action: nil,
					},
					Hero: &linebot.ImageComponent{
						Type:        linebot.FlexComponentTypeImage,
						URL:         ConvertWeatherImage(data.WeatherPer3h[0].Weathers[0].Icon),
						Size:        linebot.FlexImageSizeTypeXxl,
						AspectRatio: linebot.FlexImageAspectRatioType1to1,
						AspectMode:  linebot.FlexImageAspectModeTypeFit,
						//Action:          nil,
					},
					Body: &linebot.BoxComponent{
						Type:   linebot.FlexComponentTypeBox,
						Layout: linebot.FlexBoxLayoutTypeVertical,
						Contents: []linebot.FlexComponent{
							&linebot.TextComponent{
								Type: linebot.FlexComponentTypeText,
								Text: "最高気温 : " + strconv.Itoa(tempMax[0]) + "℃\n",
								Flex: linebot.IntPtr(1),
								Size: linebot.FlexTextSizeTypeXl,
								Wrap: true,
								//Action:     nil,
								MaxLines: linebot.IntPtr(2),
							},
							&linebot.TextComponent{
								Type: linebot.FlexComponentTypeText,
								Text: "最低気温 : " + strconv.Itoa(tempMin[0]) + "℃\n",
								Flex: linebot.IntPtr(1),
								Size: linebot.FlexTextSizeTypeXl,
								Wrap: true,
								//Action:     nil,
								MaxLines: linebot.IntPtr(2),
							},
							&linebot.TextComponent{
								Type: linebot.FlexComponentTypeText,
								Text: fmt.Sprintf("湿度 : %.2f %%", humidity[0]),
								//Contents:   nil,
								Flex: linebot.IntPtr(6),
								Size: linebot.FlexTextSizeTypeSm,
								Wrap: true,
								//Color:      "",
								//Action:     nil,
								MaxLines: linebot.IntPtr(10),
							},
						},
						BorderColor: "#5cd8f7",
						//Action:          nil,
					},
					Styles: &linebot.BubbleStyle{
						Header: &linebot.BlockStyle{
							Separator:      true,
							SeparatorColor: "#2196F3",
						},
						Hero: &linebot.BlockStyle{
							Separator:      true,
							SeparatorColor: "#2196F3",
						},
						Body: &linebot.BlockStyle{
							Separator:      true,
							SeparatorColor: "#37474F",
						},
						Footer: &linebot.BlockStyle{
							Separator:      true,
							SeparatorColor: "#2196F3",
						},
					},
				},
				{
					Type:      linebot.FlexContainerTypeBubble,
					Direction: linebot.FlexBubbleDirectionTypeLTR,
					Header: &linebot.BoxComponent{
						Type:   linebot.FlexComponentTypeBox,
						Layout: linebot.FlexBoxLayoutTypeBaseline,
						Contents: []linebot.FlexComponent{
							&linebot.TextComponent{
								Type:   linebot.FlexComponentTypeText,
								Text:   "明日の天気",
								Size:   linebot.FlexTextSizeTypeLg,
								Align:  linebot.FlexComponentAlignTypeCenter,
								Weight: linebot.FlexTextWeightTypeBold,
								//Color:      "",
								//Action:     nil,
							},
						},
						CornerRadius: linebot.FlexComponentCornerRadiusTypeXxl,
						BorderColor:  "#00bfff",
						//Action: nil,
					},
					Hero: &linebot.ImageComponent{
						Type:        linebot.FlexComponentTypeImage,
						URL:         ConvertWeatherImage(data.WeatherPer3h[8].Weathers[0].Icon),
						Size:        linebot.FlexImageSizeTypeXxl,
						AspectRatio: linebot.FlexImageAspectRatioType1to1,
						AspectMode:  linebot.FlexImageAspectModeTypeFit,
						//Action:          nil,
					},
					Body: &linebot.BoxComponent{
						Type:   linebot.FlexComponentTypeBox,
						Layout: linebot.FlexBoxLayoutTypeVertical,
						Contents: []linebot.FlexComponent{
							&linebot.TextComponent{
								Type: linebot.FlexComponentTypeText,
								Text: "最高気温 : " + strconv.Itoa(tempMax[1]) + "℃\n",
								Flex: linebot.IntPtr(1),
								Size: linebot.FlexTextSizeTypeXl,
								Wrap: true,
								//Action:     nil,
								MaxLines: linebot.IntPtr(2),
							},
							&linebot.TextComponent{
								Type: linebot.FlexComponentTypeText,
								Text: "最低気温 : " + strconv.Itoa(tempMin[1]) + "℃\n",
								Flex: linebot.IntPtr(1),
								Size: linebot.FlexTextSizeTypeXl,
								Wrap: true,
								//Action:     nil,
								MaxLines: linebot.IntPtr(2),
							},
							&linebot.TextComponent{
								Type: linebot.FlexComponentTypeText,
								Text: fmt.Sprintf("湿度 : %.2f %%", humidity[1]),
								//Contents:   nil,
								Flex: linebot.IntPtr(6),
								Size: linebot.FlexTextSizeTypeSm,
								Wrap: true,
								//Color:      "",
								//Action:     nil,
								MaxLines: linebot.IntPtr(10),
							},
						},
						BorderColor: "#5cd8f7",
						//Action:          nil,
					},
					Styles: &linebot.BubbleStyle{
						Header: &linebot.BlockStyle{
							Separator:      true,
							SeparatorColor: "#2196F3",
						},
						Hero: &linebot.BlockStyle{
							Separator:      true,
							SeparatorColor: "#2196F3",
						},
						Body: &linebot.BlockStyle{
							Separator:      true,
							SeparatorColor: "#37474F",
						},
						Footer: &linebot.BlockStyle{
							Separator:      true,
							SeparatorColor: "#2196F3",
						},
					},
				},
				{
					Type:      linebot.FlexContainerTypeBubble,
					Direction: linebot.FlexBubbleDirectionTypeLTR,
					Header: &linebot.BoxComponent{
						Type:   linebot.FlexComponentTypeBox,
						Layout: linebot.FlexBoxLayoutTypeBaseline,
						Contents: []linebot.FlexComponent{
							&linebot.TextComponent{
								Type:   linebot.FlexComponentTypeText,
								Text:   "明後日の天気",
								Size:   linebot.FlexTextSizeTypeLg,
								Align:  linebot.FlexComponentAlignTypeCenter,
								Weight: linebot.FlexTextWeightTypeBold,
								//Color:      "",
								//Action:     nil,
							},
						},
						CornerRadius: linebot.FlexComponentCornerRadiusTypeXxl,
						BorderColor:  "#00bfff",
						//Action: nil,
					},
					Hero: &linebot.ImageComponent{
						Type:        linebot.FlexComponentTypeImage,
						URL:         ConvertWeatherImage(data.WeatherPer3h[16].Weathers[0].Icon),
						Size:        linebot.FlexImageSizeTypeXxl,
						AspectRatio: linebot.FlexImageAspectRatioType1to1,
						AspectMode:  linebot.FlexImageAspectModeTypeFit,
						//Action:          nil,
					},
					Body: &linebot.BoxComponent{
						Type:   linebot.FlexComponentTypeBox,
						Layout: linebot.FlexBoxLayoutTypeVertical,
						Contents: []linebot.FlexComponent{
							&linebot.TextComponent{
								Type: linebot.FlexComponentTypeText,
								Text: "最高気温 : " + strconv.Itoa(tempMax[2]) + "℃\n",
								Flex: linebot.IntPtr(1),
								Size: linebot.FlexTextSizeTypeXl,
								Wrap: true,
								//Action:     nil,
								MaxLines: linebot.IntPtr(2),
							},
							&linebot.TextComponent{
								Type: linebot.FlexComponentTypeText,
								Text: "最低気温 : " + strconv.Itoa(tempMin[2]) + "℃\n",
								Flex: linebot.IntPtr(1),
								Size: linebot.FlexTextSizeTypeXl,
								Wrap: true,
								//Action:     nil,
								MaxLines: linebot.IntPtr(2),
							},
							&linebot.TextComponent{
								Type: linebot.FlexComponentTypeText,
								Text: fmt.Sprintf("湿度 : %.2f %%", humidity[2]),
								//Contents:   nil,
								Flex: linebot.IntPtr(6),
								Size: linebot.FlexTextSizeTypeSm,
								Wrap: true,
								//Color:      "",
								//Action:     nil,
								MaxLines: linebot.IntPtr(10),
							},
						},
						BorderColor: "#5cd8f7",
						//Action:          nil,
					},
					Styles: &linebot.BubbleStyle{
						Header: &linebot.BlockStyle{
							Separator:      true,
							SeparatorColor: "#2196F3",
						},
						Hero: &linebot.BlockStyle{
							Separator:      true,
							SeparatorColor: "#2196F3",
						},
						Body: &linebot.BlockStyle{
							Separator:      true,
							SeparatorColor: "#37474F",
						},
						Footer: &linebot.BlockStyle{
							Separator:      true,
							SeparatorColor: "#2196F3",
						},
					},
				},
			},
		},
	)

	return resp
}

func ConvertWeatherImage(pngNumber string) string {
	return fmt.Sprintf("https://openweathermap.org/img/w/%s.png", pngNumber)
}
