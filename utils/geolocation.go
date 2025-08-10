package utils

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"

    "github.com/joho/godotenv"
)

// 定义高德地图 IP 定位 API 的响应结构体
// 注意：高德的返回字段与 ip-api.com 不同
type AmapIPResponse struct {
    Status    string `json:"status"`    // 返回状态，1表示成功，0表示失败
    Info      string `json:"info"`      // 返回的状态信息
    Infocode  string `json:"infocode"`  // 返回的状态码
    Province  string `json:"province"`  // 省份名称
    City      string `json:"city"`      // 城市名称
    Adcode    string `json:"adcode"`    // 城市编码
    Rectangle string `json:"rectangle"` // 城市中心点经纬度矩形区域
    // 高德IP定位API不直接返回经纬度、ISP、Org、AS等详细信息
    // 如果需要经纬度，可以从rectangle中解析，或者通过其他API（如地理编码）获取城市中心点
}


const AmapAPIKey = os.Getenv("AmapAPIKey")

// GetGeolocation 函数用于根据 IP 地址获取地理位置信息 (使用高德地图 API)
func GetGeolocation(ip string) (*AmapIPResponse, error) {
    // 检查 Key 是否已设置
    if AmapAPIKey == "" {
        return nil, fmt.Errorf("Amap API Key is not set. Please replace 'YOUR_AMAP_API_KEY' with your actual key.")
    }

    // 构造高德 IP 定位 API 的 URL
    url := fmt.Sprintf("https://restapi.amap.com/v3/ip?ip=%s&key=%s", ip, AmapAPIKey)

    resp, err := http.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to make HTTP request to Amap API: %w", err)
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read Amap API response: %w", err)
    }

    var amapInfo AmapIPResponse
    if err := json.Unmarshal(body, &amapInfo); err != nil {
        return nil, fmt.Errorf("failed to unmarshal Amap API JSON response: %w", err)
    }

    // 检查高德 API 返回的状态码
    // 高德API成功时 status 为 "1"
    if amapInfo.Status != "1" {
        // 高德API失败时，info 和 infocode 字段会包含错误信息
        return nil, fmt.Errorf("Amap API returned status: %s, info: %s, infocode: %s for IP: %s", amapInfo.Status, amapInfo.Info, amapInfo.Infocode, ip)
    }

    return &amapInfo, nil
}