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
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"

	"github.com/PuerkitoBio/goquery"

	"encoding/base64"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/joho/godotenv"
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

	var html string
	err := chromedp.Run(
		ctx,
		chromedp.Emulate(device.IPadPro11landscape),
		Login(urls[0].LoginUrl, urls[0].Username, urls[0].Password),
		Screenshot("login"),
		NavInvestmentPage(),
		Screenshot("investments"),
		TakeInvestmentValues(),
		TakeInvestments(),
		GetInvestmentTable(&html),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("writing investment table output")
	os.WriteFile("table.html", []byte(html), 0644)

	data := ExtractTableValues(html)
	WriteToGSheet(data, TrimCellText)
}

func Login(loginUrl string, username string, password string) chromedp.Tasks {
	usernameField := `//*[@id="vg-auth0-login-username"]`
	passwordField := `//*[@id="vg-auth0-login-password"]`
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

func GetInvestmentTable(html *string) chromedp.Tasks {
	var tableNode []*cdp.Node
	return chromedp.Tasks{
		chromedp.Nodes(`//table`, &tableNode),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			*html, err = dom.GetOuterHTML().WithNodeID(tableNode[0].NodeID).Do(ctx)
			return err
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

// Extract Vanguard Investment Table value, returns array of values
func ExtractTableValues(html string) [][]string {
	var rows [][]string
	var headings, row []string
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

	doc.Find("table").Each(func(index int, tablehtml *goquery.Selection) {
		tablehtml.Find("tr").Each(func(indextr int, rowhtml *goquery.Selection) {
			rowhtml.Find("th").Each(func(indexth int, tableheading *goquery.Selection) {
				headings = append(headings, tableheading.Text())
			})
			rowhtml.Find("td").Each(func(indexth int, tablecell *goquery.Selection) {
				row = append(row, tablecell.Text())
			})
			if len(row) != 0 {
				rows = append(rows, row)
			}
			row = nil
		})
	})
	fmt.Println("####### headings = ", len(headings), headings)
	fmt.Println("####### rows = ", len(rows), rows)

	return rows
}

func WriteToGSheet(data [][]string, cellTextProcessor func(string) string) {
	godotenv.Load(".env")

	key := os.Getenv("GSHEET_KEY")
	sheetName := os.Getenv("GSHEET_NAME")
	spreadsheetId := os.Getenv("GSHEET_ID")

	ctx := context.Background()
	credBytes, _ := base64.StdEncoding.DecodeString(key)
	config, _ := google.JWTConfigFromJSON(credBytes, "https://www.googleapis.com/auth/spreadsheets")
	client := config.Client(ctx)
	srv, _ := sheets.NewService(ctx, option.WithHTTPClient(client))

	t := time.Now()
	values := make([][]interface{}, len(data))
	for n, row := range data {
		columns := make([]interface{}, len(row))
		for i, s := range row {
			columns[i] = cellTextProcessor(s)
		}
		columns = append(columns, t)
		values[n] = columns
	}

	row := &sheets.ValueRange{
		Values: values,
	}

	response, _ := srv.Spreadsheets.Values.Append(
		spreadsheetId, sheetName, row).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
	fmt.Println(response)
}

func TrimCellText(s string) string {
	r0 := strings.Replace(s, "actionsTop-upSellSwitch", "", 1)
	r1 := strings.Replace(r0, "Invest", "", 1)
	r2 := strings.Replace(r1, "Change (£)", "", 1)
	r3 := strings.Replace(r2, "£", "", 1)
	r4 := strings.Replace(r3, "–", "", 1)
	return strings.Trim(r4, " ")
}
