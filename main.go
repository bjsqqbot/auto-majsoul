package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"regexp"
	"time"

	"github.com/bjsqqbot/auto-majsoul/helper"
	"github.com/bjsqqbot/auto-majsoul/helper/util"
	"github.com/bjsqqbot/auto-majsoul/middleware"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	CodejsUrl = "https://endlesscheng.gitee.io/public/js/majsoul/code-zh.js"
	CacheDir  = "/cache"
)

func main() {
	go RunMajsoul()
	helper.Run()
}

func RunMajsoul() {
	dir, _ := os.Getwd()
	options := []chromedp.ExecAllocatorOption{
		chromedp.WindowSize(1280, 720),
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
			gopool.Go(func() { ListenMessageQueue(ctx) })
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
		fmt.Println("手牌 ", m.HandTile34)
		fmt.Println("进张 ", util.MahjongZH[m.TileGot])
		if m.Reach {
			fmt.Println("--------Reach-------")
		}
		fmt.Println("切 ", util.MahjongZH[m.BestCard])
		// 计算手牌顺序和坐标
		handTile34 := append([]int{}, m.HandTile34...)
		handTile34[m.TileGot]--
		fmt.Println("当前牌序 ", util.Tiles34ToStr(m.HandTile34), " ", util.Tile34ToStr(m.TileGot))
		posOffset := 0
		for tileId, tileCount := range handTile34 {
			if tileCount != 0 && tileId == m.BestCard {
				break
			}
			posOffset += tileCount
		}
		posx, posy := 230+55*(posOffset), 600
		// 出牌
		rand.Seed(time.Now().UnixNano())
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(1400)+600))
		if m.Reach {
			for i := 0; i < 5; i++ {
				chromedp.Run(ctx, chromedp.MouseClickXY(670, 490))
				time.Sleep(time.Microsecond * 5)
			}
			fmt.Println("Click 670 490")
		}
		time.Sleep(time.Millisecond * 100)
		for i := 0; i < 5; i++ {
			chromedp.Run(ctx, chromedp.MouseClickXY(float64(posx), float64(posy)))
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
