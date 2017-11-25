package main

import (
	"fmt"
	"github.com/bitly/go-simplejson"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const JsonUrl = "https://www.huomao.com/channels/channel.json?page=1&game_url_rule=dota2"
const FlvTemplate = "http://live-yf-hdl.huomaotv.cn/live/%s.flv"
const FileNameTemplate = "hm_%d.pls"
const OldFilesPattern = "hm_*.pls"

const (
	NetErrStr    = "火猫网络错误"
	DataErrStr   = "火猫数据异常"
	ConfigErrStr = "配置错误"
)

type HuomaoItem struct {
	Name       string `json:"name"`
	FlvAddress string `json:"flv_address"`
	Online     int    `json:"online"`
}

type HuomaoList []*HuomaoItem

func (p HuomaoList) Len() int           { return len(p) }
func (p HuomaoList) Less(i, j int) bool { return p[i].Online > p[j].Online }
func (p HuomaoList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func DecodeHuomao(data []byte) []*HuomaoItem {
	list := make([]*HuomaoItem, 0)
	m, err := simplejson.NewJson(data)
	if err != nil {
		return list
	}
	channels := m.GetPath("data", "channelList")
	channelArray, err := channels.Array()
	DataErr(err)
	for i := 0; i < len(channelArray); i++ {
		item := new(HuomaoItem)
		channel := channels.GetIndex(i)
		isAlive, err := channel.Get("is_live").Int()
		DataErr(err)
		if isAlive == 1 {
			item.Name, err = channel.Get("nickname").String()
			DataErr(err)
			item.Online, err = channel.Get("originviews").Int()
			DataErr(err)
			m3u8, err := channel.Get("m3u8").Get("address").GetIndex(0).String()
			DataErr(err)
			item.FlvAddress = M3u8ToFlv(m3u8)
			list = append(list, item)
		}
	}
	if len(list) == 0 {
		Err(DataErrStr)
	}
	sort.Sort(HuomaoList(list))
	return list
}

func M3u8ToFlv(m3u8 string) string {
	i := strings.Replace(m3u8, "https://live-ws-hls.huomaotv.cn/live/", "", -1)
	i = strings.Replace(i, "_100/playlist.m3u8", "", -1)
	return fmt.Sprintf(FlvTemplate, i)
}

func SaveToTemp(list []*HuomaoItem) string {
	content := "[playlist]\n"
	content += fmt.Sprintf("NumberOfEntries=%d\n", len(list))
	for idx, v := range list {
		fmt.Println(v)
		content += fmt.Sprintf("File%d=%s\n", idx+1, v.FlvAddress)
		content += fmt.Sprintf("Title%d=%s(%d)\n", idx+1, v.Name, v.Online)
	}
	temp := os.Getenv("TEMP")
	if temp == "" {
		Err(ConfigErrStr)
	}
	files, _ := filepath.Glob(filepath.Join(temp, OldFilesPattern))
	for _, v := range files {
		os.Remove(v)
	}
	filename := filepath.Join(temp, fmt.Sprintf(FileNameTemplate, time.Now().Unix()))
	file, err := os.Create(filename)
	if err != nil {
		Err(ConfigErrStr)
	}
	buf := []byte(content)
	n, err := file.Write(buf)
	if n != len(buf) || err != nil {
		Err(ConfigErrStr)
	}
	return filename
}

func Err(err string) {
	fmt.Println(err)
	time.Sleep(time.Millisecond * 1500)
	os.Exit(-1)
}

func DataErr(err error) {
	if err != nil {
		Err(DataErrStr)
	}
}

func main() {
	rsp, err := http.Get(JsonUrl)
	if err != nil {
		Err(NetErrStr)
	}
	data, err := ioutil.ReadAll(rsp.Body)
	defer rsp.Body.Close()
	DataErr(err)
	list := DecodeHuomao(data)
	filname := SaveToTemp(list)
	command := exec.Command("cmd", "/C", "start", filname)
	err = command.Start()
	if err != nil {
		Err(ConfigErrStr)
	}
}
