// 测试 Open-Meteo 对广州的返回值：对比 current 与 daily 的天气代码及全天最低/最高温度。
// 运行：go run ./test/weather
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	lat = "23.1291"
	lon = "113.2644"
)

type response struct {
	Timezone string `json:"timezone"`
	Current  struct {
		Time        string  `json:"time"`
		Temperature float64 `json:"temperature_2m"`
		Code        int     `json:"weather_code"`
	} `json:"current"`
	Daily struct {
		Time []string  `json:"time"`
		Code []int     `json:"weather_code"`
		Max  []float64 `json:"temperature_2m_max"`
		Min  []float64 `json:"temperature_2m_min"`
	} `json:"daily"`
}

func main() {
	q := url.Values{}
	q.Set("latitude", lat)
	q.Set("longitude", lon)
	q.Set("current", "temperature_2m,weather_code")
	q.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min")
	q.Set("timezone", "auto")
	q.Set("forecast_days", "4")
	endpoint := "https://api.open-meteo.com/v1/forecast?" + q.Encode()
	fmt.Println("GET", endpoint)

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Get(endpoint)
	if err != nil {
		fmt.Println("请求失败:", err)
		os.Exit(1)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		fmt.Println("读取响应失败:", err)
		os.Exit(1)
	}
	fmt.Println("HTTP", res.StatusCode)
	if res.StatusCode != http.StatusOK {
		fmt.Println(string(body))
		os.Exit(1)
	}

	var payload response
	if err := json.Unmarshal(body, &payload); err != nil {
		fmt.Println("解析 JSON 失败:", err)
		os.Exit(1)
	}

	fmt.Printf("时区: %s\n", payload.Timezone)
	fmt.Printf("current: %s 温度 %.1f°C 天气代码 %d (%s)\n\n",
		payload.Current.Time, payload.Current.Temperature, payload.Current.Code, describe(payload.Current.Code))

	fmt.Println("daily:")
	for i, date := range payload.Daily.Time {
		var code int
		var max, min float64
		if i < len(payload.Daily.Code) {
			code = payload.Daily.Code[i]
		}
		if i < len(payload.Daily.Max) {
			max = payload.Daily.Max[i]
		}
		if i < len(payload.Daily.Min) {
			min = payload.Daily.Min[i]
		}
		fmt.Printf("  %s  代码 %d (%s)  最低 %.1f°C / 最高 %.1f°C\n",
			date, code, describe(code), min, max)
	}

	if len(payload.Daily.Time) > 0 {
		fmt.Printf("\n今日全天: 天气代码 %d (%s), 最低 %.1f°C / 最高 %.1f°C\n",
			payload.Daily.Code[0], describe(payload.Daily.Code[0]), payload.Daily.Min[0], payload.Daily.Max[0])
	}
}

// describe 参考 internal/weather.Describe 的 WMO 代码映射。
func describe(code int) string {
	switch code {
	case 0:
		return "晴"
	case 1:
		return "晴间多云"
	case 2:
		return "多云"
	case 3:
		return "阴"
	case 45, 48:
		return "有雾"
	case 51, 53, 55:
		return "毛毛雨"
	case 56, 57:
		return "冻毛毛雨"
	case 61, 63, 65:
		return "下雨"
	case 66, 67:
		return "冻雨"
	case 71, 73, 75:
		return "下雪"
	case 77:
		return "米雪"
	case 80, 81, 82:
		return "阵雨"
	case 85, 86:
		return "阵雪"
	case 95:
		return "雷阵雨"
	case 96, 99:
		return "雷阵雨伴冰雹"
	default:
		return "天气未知"
	}
}
