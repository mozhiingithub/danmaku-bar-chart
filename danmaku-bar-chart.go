package main

import (
	"compress/flate"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/charts"
)

const (
	cookie    = "main_confirmation=mVukukEuyrz3oaSUBpsnGuz/I+nadYz+IgHTdqmLbPg=; CURRENT_FNVAL=16; _uuid=08DEB053-A632-90F6-CEAD-212C77E534C768556infoc; buvid3=2E9CFDF8-E296-404A-9AE3-410BA6077F7953921infoc; sid=b3dqjfnj; LIVE_BUVID=AUTO2115857123723278"
	userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:74.0) Gecko/20100101 Firefox/74.0"
	cidReg    = "upgcxcode/[\\d]+/[\\d]+/[\\d]+/"
	lenReg    = "timelength\":[\\d]+,"
	// xmlURL    = "https://comment.bilibili.com/%s.xml"
	xmlURL = "https://api.bilibili.com/x/v1/dm/list.so?oid=%s"
)

// DanmakuList 用于解构弹幕xml文档的结构体。
// ChatID为视频的cid。
// Ds为弹幕列表。
type DanmakuList struct {
	XMLName    xml.Name `xml:"i"`
	ChatServer string   `xml:"chatserver"`
	ChatID     int      `xml:"chatid"`
	Mission    int      `xml:"mission"`
	MaxLimit   int      `xml:"maxlimit"`
	State      int      `xml:"state"`
	RealName   int      `xml:"real_name"`
	Source     string   `xml:"source"`
	Ds         []d      `xml:"d"`
}

// d是Danmaku的子结构体，用于解构弹幕xml文件当中的d标签。
// P是d标签的参数，其中包含了弹幕所处视频时间、弹幕发送时间、弹幕发送者标识等信息。
// Comment为弹幕文本。
type d struct {
	P       string `xml:"p,attr"`
	Comment string `xml:",chardata"`
}

func main() {

	// 程序运行的命令格式为： go run danmaku_lite.go url
	// url存于os.Args[1]中。若os.Args切片长度小于2,说明没有输入url。
	if len(os.Args) < 2 {
		checkError(errors.New("No url"))
	}

	// 定义变量
	var (
		e      error
		bs     []byte
		url    string
		s      string
		cidStr string
		lenStr string
		lenInt int
		t      int
		f      *os.File
	)

	// 根据给定的url，获取网页源码
	url = os.Args[1]
	bs, e = getBytes(url)
	checkError(e)
	s = string(bs)

	// 使用正则表达式匹配cid及视频长度

	// 根据cidReg格式，找到源码中包含cid的部分
	// 获取后，对内容按“/”进行切割，其中第4部分即为cid
	// 实例：
	// cid存在于视频的实际网址中：
	// https://cn-sh-ix-bcache-06.bilivideo.com/upgcxcode/53/29/170132953/170132953-1-30112.m4s
	// 其中170132953是cid号，因此，用正则表达式，匹配“upgcxcode”及其后续四个左斜线的内容
	// 得到：upgcxcode/53/29/170132953/
	// 再将其按左斜线切割，得到的第4部分即为cid：170132953
	cidStr = regexp.MustCompile(cidReg).FindString(s)
	if "" == cidStr {
		checkError(errors.New("cid: Cannot match"))
	}
	cidStr = strings.Split(cidStr, "/")[3]

	// 根据lenReg格式，找到源码中包含length的部分
	// 再使用"[\\d]+"匹配其中的数字部分，即为lenStr
	// 实例：
	// 视频长度存在于：
	// "format":"flv360","timelength":2539267,"accept_format":"flv_p60,
	// 其中2539267是视频长度，因此，用正则表达式，匹配：timelength":2539267,
	// 再使用"[\\d]+"匹配其中的数字部分，即为2539267
	lenStr = regexp.MustCompile(lenReg).FindString(s)
	if "" == lenStr {
		checkError(errors.New("length: Cannot match"))
	}
	lenStr = regexp.MustCompile("[\\d]+").FindString(lenStr)
	lenInt, e = strconv.Atoi(lenStr)
	checkError(e)

	// 将视频长度单位从毫秒改为秒
	lenInt /= 1000

	//添加冗余
	lenInt += 2

	// 根据cid号，获取弹幕xml文件
	url = fmt.Sprintf(xmlURL, cidStr)
	bs, e = getBytes(url)
	checkError(e)

	// 解构xml文件
	dm := &DanmakuList{}
	e = xml.Unmarshal(bs, dm)
	checkError(e)

	// 以秒为单位，统计每一秒的弹幕数
	countSlice := make([]int, lenInt)
	for _, d := range dm.Ds {
		t, e = getTime(d.P)
		checkError(e)
		countSlice[t]++
	}

	// 制作一个时间戳字符串切片
	x := make([]string, lenInt)
	for i := 0; i < lenInt; i++ {
		x[i] = (time.Duration(i) * time.Second).String()
	}

	// 生成弹幕分布图
	bar := charts.NewBar()
	bar.SetGlobalOptions(charts.TitleOpts{Title: "弹幕分布图"}, charts.ToolboxOpts{Show: true})
	bar.AddXAxis(x).AddYAxis("弹幕量", countSlice)

	bar.Width = "1000px"
	bar.Height = "500px"

	f, e = os.Create(cidStr + ".html")
	checkError(e)
	bar.Render(f)
}

// 根据url获取字节流
func getBytes(url string) (bs []byte, e error) {

	var (
		request    *http.Request
		response   *http.Response
		readCloser io.ReadCloser
	)

	// 定义一个网络请求
	request, e = http.NewRequest("GET", url, nil)
	checkError(e)
	request.Header.Add("User-Agent", userAgent)
	request.Header.Add("Cookie", cookie)

	// 定义一个client，发送请求,获取响应
	client := &http.Client{}
	response, e = client.Do(request)
	checkError(e)

	defer response.Body.Close()

	// B站弹幕xml文档需要解压，而网页源码无需解压
	if "deflate" == response.Header.Get("Content-Encoding") {
		readCloser = flate.NewReader(response.Body)
	} else {
		readCloser = response.Body
	}

	// 读取内容
	bs, e = ioutil.ReadAll(readCloser)
	return
}

// 检查错误是否非空，是则终止程序运行
func checkError(e error) {
	if nil != e {
		log.Println(e)
		os.Exit(1)
	}
}

// 弹幕时间戳位于参数p的第一部分，为秒制浮点数，用小数点切割后的第一部分即为弹幕秒数
func getTime(p string) (t int, e error) {
	t, e = strconv.Atoi(strings.Split(p, ".")[0])
	return
}
