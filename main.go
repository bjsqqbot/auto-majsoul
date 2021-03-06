package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"regexp"
	"strconv"
	"time"

	"github.com/bjsqqbot/auto-majsoul/helper"
	"github.com/bjsqqbot/auto-majsoul/helper/util"
	"github.com/bjsqqbot/auto-majsoul/middleware"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/joho/godotenv"
)

var (
	CodejsUrl       string = "https://endlesscheng.gitee.io/public/js/majsoul/code-zh.js"
	CacheDir        string = "cache"
	WindowWidth     int    = 1280
	WindowHeight    int    = 720
	EnableAutoClick bool   = true
)

func main() {
	LoadEnv()
	go RunMajsoul()
	helper.Run()
}

func LoadEnv() {
	if err := godotenv.Load(".env"); err == nil {
		CacheDir = os.Getenv("CACHE_DIR")
		if w, err := strconv.Atoi(os.Getenv("WINDOW_WIDTH")); err == nil {
			WindowWidth = w
		}
		if h, err := strconv.Atoi(os.Getenv("WINDOW_HEIGHT")); err == nil {
			WindowHeight = h
		}
		if os.Getenv("ENABLE_AUTO_CLICK") == "FALSE" {
			EnableAutoClick = false
		}
		if os.Getenv("CONSOLE_MODE") == "TRUE" {
			WindowWidth += 555
		}
	}
}

func RunMajsoul() {
	dir, _ := os.Getwd()
	options := []chromedp.ExecAllocatorOption{
		chromedp.WindowSize(WindowWidth, WindowHeight),
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-sync", false),
		chromedp.UserDataDir(path.Join(dir, CacheDir)),
		chromedp.Flag("allow-insecure-localhost", "Enable"),
		chromedp.Flag("blink-settings", "imageEnable=false"),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.54 Safari/537.36 Edg/101.0.1210.39`),
		chromedp.NoFirstRun,
	}
	ctx, _ := chromedp.NewExecAllocator(context.Background(), options...)
	ctx, cancel := chromedp.NewContext(ctx)
	defer cancel()
	ListenRequests(ctx)
	err := chromedp.Run(ctx,
		network.Enable(),
		fetch.Enable(),
		chromedp.Navigate("https://game.maj-soul.net/1/"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			go ListenMessageQueue(ctx)
			return nil
		}),
		chromedp.WaitVisible(`div[class="List-item"]`),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func ListenMessageQueue(ctx context.Context) {
	go ListenSelfDrawEvent(ctx)
	go ListenMeldEvent(ctx)
}

// ??????????????????
func ListenMeldEvent(ctx context.Context) {
	for {
		m := <-middleware.MQ.MeldCh

		WaitForButton()

		if m.Long {
			Click(670, 490, ctx)
			continue
		}

		Click(835, 490, ctx) // ??????
		// TODO ????????????
	}
}

// ??????????????????
func ListenSelfDrawEvent(ctx context.Context) {
	reachMode := false
	gopool.Go(func() {
		for {
			if reachMode { // ?????????????????????
				Click(670, 490, ctx)
				time.Sleep(time.Second)
			}
		}
	})
	for {
		m := <-middleware.MQ.SelfDrawCh

		reachMode = false

		if m.BestCard == -1 { // ????????????-1????????????
			WaitForButton()
			Click(670, 490, ctx)
			continue
		}
		handTile34 := make([]int, len(m.HandTile34))
		copy(handTile34, m.HandTile34)
		handTile34[m.TileGot]--
		fmt.Println("???????????? ", util.Tiles34ToStr(m.HandTile34), " ", util.Tile34ToStr(m.TileGot))
		fmt.Println("?????? ", util.MahjongZH[m.TileGot])
		if m.Reach {
			fmt.Println("--------Reach-------")
		}
		fmt.Println("??? ", util.MahjongZH[m.BestCard])
		// ???????????????????????????
		if !EnableAutoClick {
			continue
		}
		posOffset := 0
		for tileId, tileCount := range handTile34 {
			if tileCount != 0 && tileId == m.BestCard {
				break
			}
			posOffset += tileCount
		}
		var posx, posy float64 = 230 + 55*(float64(posOffset)), 600
		// ??????

		// TODO ?????????????????????????????????
		// inner 1280x720 ??????+160+55*offset
		// var gameWidth, gameHeight, windowNaviHeight float64 = 0, 0, 88
		// chromedp.Run(ctx,
		// 	chromedp.Evaluate(`(() => {return window.innerWidth})()`, &gameWidth),
		// 	chromedp.Evaluate(`(() => {return window.innerHeight})()`, &gameHeight))
		// windowHeight := gameHeight + windowNaviHeight
		// windowWidth := windowHeight * 16 / 9
		// fmt.Println(gameWidth, gameHeight)
		// fmt.Println(windowWidth, windowHeight)

		WaitForButton()

		// ??????
		if m.Reach {
			reachMode = true
			Click(670, 490, ctx)
			time.Sleep(time.Millisecond * 100)
		}

		Click(posx, posy, ctx)
	}
}

// ??????(x,y)?????? ctx:???????????????
func Click(x, y float64, ctx context.Context) {
	for i := 0; i < 5; i++ {
		chromedp.Run(ctx, chromedp.MouseClickXY(x, y))
		time.Sleep(time.Microsecond * 5)
	}
	fmt.Printf("Click %v %v\n", x, y)
}

// ???????????????????????? 600ms~2000ms
func WaitForButton() {
	rand.Seed(time.Now().UnixNano())
	waitTime := rand.Intn(1400) + 600
	fmt.Printf("???????????? %fs\n", float32(waitTime)/1000)
	time.Sleep(time.Millisecond * time.Duration(waitTime))
}

// ????????????
func ListenRequests(ctx context.Context) {
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *fetch.EventRequestPaused:
			gopool.Go(func() {
				c := chromedp.FromContext(ctx)
				e := cdp.WithExecutor(ctx, c.Target)
				fetchReq := fetch.ContinueRequest(ev.RequestID)
				if MatchedCodejs(ev.Request.URL) {
					go SetRedirect(CodejsUrl, ev.RequestID, e)
					return
				}
				fetchReq.Do(e)
			})
		}
	})
}

func MatchedCodejs(u string) bool {
	reg := regexp.MustCompile(`^https://game\.maj\-soul\.net/[0-9]+/[a-zA-Z0-9\.]+/code\.js$`)
	return reg.Find([]byte(u)) != nil
}

// ???????????????
// location:???????????????
// reqId:??????????????????ID
// e:????????????????????????
func SetRedirect(location string, reqId fetch.RequestID, e context.Context) {
	fetchResp := fetch.ContinueResponse(reqId)
	fetchResp.ResponseCode = 302
	fetchResp.ResponseHeaders = append(fetchResp.ResponseHeaders,
		&fetch.HeaderEntry{Name: "Location", Value: location})
	log.Println("code.js Redirect Success")
	fetchResp.Do(e)
}
