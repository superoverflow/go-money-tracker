// Command submit is a chromedp example demonstrating how to fill out and
// submit a form.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
)

var urlsFile = flag.String("inputfile", "urls.json", "a file containing urls and login credentials")

type Url struct {
	LoginUrl string `json:"LoginUrl"`
	Account  string `json:"Account"`
	Username string `json:"Username"`
	Password string `json:"Password"`
}

func main() {
	flag.Parse()
	content, _ := os.ReadFile(*urlsFile)
	var urls []Url
	json.Unmarshal(content, &urls)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	err := chromedp.Run(
		ctx,
		chromedp.Emulate(device.IPadPro11landscape),
		Login(urls[0].LoginUrl, urls[0].Username, urls[0].Password),
		Screenshot("login"),
		NavInvestmentPage(),
		Screenshot("investments"),
		TakeInvestmentValues(),
		TakeInvestments(),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func Login(loginUrl string, username string, password string) chromedp.Tasks {
	usernameField := `//*[@id="__GUID_1007"]`
	passwordField := `//*[@id="__GUID_1008"]`
	submitButton := `//button[contains(text(), "Log in")]`

	return chromedp.Tasks{
		chromedp.Navigate(loginUrl),
		chromedp.WaitVisible(usernameField),
		chromedp.SendKeys(usernameField, username),
		chromedp.SendKeys(passwordField, password),
		chromedp.Click(submitButton),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("waiting login completes")
			return nil
		}),
		chromedp.Sleep(3 * time.Second),
	}

}

func NavInvestmentPage() chromedp.Tasks {
	investmentLink := `//li/a[contains(@href, "investments")]`
	investmentHoldingPanel := `//meta[@content="investments-holdings"]`
	return chromedp.Tasks{
		chromedp.Click(investmentLink),
		chromedp.WaitReady(investmentHoldingPanel),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("waiting investments to load")
			return nil
		}),
		chromedp.Sleep(3 * time.Second),
	}
}

func TakeInvestmentValues() chromedp.Tasks {
	var currentValueNodes []*cdp.Node
	return chromedp.Tasks{
		chromedp.Nodes(`//td[@data-css="Current value"]/span/text()`, &currentValueNodes),

		chromedp.ActionFunc(func(ctx context.Context) error {
			for i, n := range currentValueNodes {
				log.Printf("value [%d] [%s]", i, n.NodeValue)
			}
			return nil
		}),
	}
}

func TakeInvestments() chromedp.Tasks {
	var instrumentNodes []*cdp.Node
	return chromedp.Tasks{
		chromedp.Nodes(`//div[contains(@class, "content-product-name")]/span/a`, &instrumentNodes),
		chromedp.ActionFunc(func(ctx context.Context) error {
			for _, n := range instrumentNodes {
				dom.RequestChildNodes(n.NodeID).WithDepth(-1).Do(ctx)
			}
			return nil
		}),
		chromedp.Sleep(time.Second),
		chromedp.ActionFunc(func(c context.Context) error {
			for i, n := range instrumentNodes {
				log.Printf("Instruments [%d] [%s]", i, n.Children[0].NodeValue)
			}
			return nil
		}),
	}
}

func Screenshot(filename string) chromedp.Tasks {
	var buf []byte
	quality := 80
	return chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("taking screenshot [%s]", filename)
			return nil
		}),
		chromedp.FullScreenshot(&buf, quality),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return os.WriteFile(fmt.Sprintf("%s.png", filename), buf, 0o644)
		}),
	}
}
