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
		CodejsUrl = os.Getenv("CODEJS_URL")
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
		chromedp.Navigate("https://game.maj-soul.com/1/"),
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
	for {
		m := middleware.MQ.Receive()
		handTile34 := append([]int{}, m.HandTile34...)
		handTile34[m.TileGot]--
		fmt.Println("当前牌序 ", util.Tiles34ToStr(m.HandTile34), " ", util.Tile34ToStr(m.TileGot))
		fmt.Println("进张 ", util.MahjongZH[m.TileGot])
		if m.Reach {
			fmt.Println("--------Reach-------")
		}
		fmt.Println("切 ", util.MahjongZH[m.BestCard])
		// 计算手牌顺序和坐标
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
		// 出牌

		// TODO 根据当前分辨率修改坐标
		// inner 1280x720 黑边+160+55*offset
		// var gameWidth, gameHeight, windowNaviHeight float64 = 0, 0, 88
		// chromedp.Run(ctx,
		// 	chromedp.Evaluate(`(() => {return window.innerWidth})()`, &gameWidth),
		// 	chromedp.Evaluate(`(() => {return window.innerHeight})()`, &gameHeight))
		// windowHeight := gameHeight + windowNaviHeight
		// windowWidth := windowHeight * 16 / 9
		// fmt.Println(gameWidth, gameHeight)
		// fmt.Println(windowWidth, windowHeight)

		rand.Seed(time.Now().UnixNano())
		waitTime := rand.Intn(1400) + 600
		fmt.Printf("等待时间 %fs\n", float32(waitTime)/1000)
		time.Sleep(time.Millisecond * time.Duration(waitTime))

		if m.Reach {
			for i := 0; i < 5; i++ {
				chromedp.Run(ctx, chromedp.MouseClickXY(670, 490))
				time.Sleep(time.Microsecond * 5)
			}
			fmt.Println("Click 670 490")
			time.Sleep(time.Millisecond * 100)
		}

		for i := 0; i < 5; i++ {
			chromedp.Run(ctx, chromedp.MouseClickXY(posx, posy))
			time.Sleep(time.Microsecond * 5)
		}
		fmt.Printf("Click %v %v\n", posx, posy)
	}
}

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
	reg := regexp.MustCompile(`^https://game\.maj\-soul\.com/[0-9]+/[a-zA-Z0-9\.]+/code\.js$`)
	return reg.Find([]byte(u)) != nil
}

func SetRedirect(location string, reqId fetch.RequestID, e context.Context) {
	fetchResp := fetch.ContinueResponse(reqId)
	fetchResp.ResponseCode = 302
	fetchResp.ResponseHeaders = append(fetchResp.ResponseHeaders,
		&fetch.HeaderEntry{Name: "Location", Value: location})
	log.Println("code.js Redirect Success")
	fetchResp.Do(e)
}
