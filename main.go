//test
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"

	"google.golang.org/api/option"
	sheets "google.golang.org/api/sheets/v4"

	"golang.org/x/oauth2/jwt"
)

const (
	maxNoOfCurrencies    = 3
	maxNoOfRates         = 65
	url1                 = "https://cryptorank.io/ru/"
	url2                 = "https://www.coingecko.com/"
	spreadsheetId        = "1GCPhTKxdkJ4pM0irfJx3bMzWMl5UGHEb8_QpEX4TtK4" // Change with your spreadSheetId
	googleApiCredentials = "credentials.json"                             // Google Cloud API credentials file
	googleApiUrl         = "https://www.googleapis.com/auth/spreadsheets"
	sheet1Name           = "Лист1"
	sheet2Name           = "Лист2"
)

type result struct {
	name, tag string
	rate      string
	timestamp int64
}

var results []result

func main() {
	srv := makeService()
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{url1},
		ParseFunc: parseUrl1,
	}).Start()

	writeResults(&results, srv, sheet1Name, "Tag")

	results = nil

	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{url2},
		ParseFunc: parseUrl2,
	}).Start()

	writeResults(&results, srv, sheet2Name, "Rate")
}

func parseUrl1(g *geziyor.Geziyor, r *client.Response) {
	r.HTMLDoc.Find("div.data-table__table-content").Each(func(i int, s *goquery.Selection) {
		names := s.Find("span.table-coin-link__Name-sc-1oywjh8-0")
		tags := s.Find("span.table-coin-link__Symbol-sc-1oywjh8-1")
		for i := 0; i < maxNoOfCurrencies; i++ {
			rs := result{}
			rs.name, rs.tag, rs.timestamp = strings.TrimSpace(names.Eq(i).Text()), strings.TrimSpace(tags.Eq(i).Text()), time.Now().Unix()
			results = append(results, rs)
		}
	})
}

func parseUrl2(g *geziyor.Geziyor, r *client.Response) {
	r.HTMLDoc.Find("div.coingecko-table").Each(func(i int, s *goquery.Selection) {
		names := s.Find("a.tw-hidden")
		rates := s.Find("td.td-price")
		for i := 0; i < maxNoOfRates; i++ {
			rs := result{}
			rs.name, rs.rate, rs.timestamp = strings.TrimSpace(names.Eq(i).Text()), strings.TrimSpace(rates.Eq(2*i+1).Text()), time.Now().Unix()
			results = append(results, rs)
		}
	})
}

func writeResults(res *[]result, srv *sheets.Service, sheetName, columnName string) {
	var vr sheets.ValueRange
	var myval []interface{}
	writeRange := sheetName + "!A1"

	sheetIsEmpty, err := checkSheet(sheetName, srv)
	if err != nil {
		fmt.Printf("Sheet %v is not accessible", sheetName)
		os.Exit(1)
	}
	if sheetIsEmpty {
		vr.Values = append(vr.Values, []interface{}{"Name", columnName, "TimeStamp"})
	}

	for _, val := range *res {
		if columnName == "Tag" {
			myval = []interface{}{val.name, val.tag, strconv.FormatInt(val.timestamp, 10)}
		} else {
			myval = []interface{}{val.name, val.rate, strconv.FormatInt(val.timestamp, 10)}
		}
		vr.Values = append(vr.Values, myval)
	}

	if _, err := srv.Spreadsheets.Values.Append(spreadsheetId, writeRange, &vr).ValueInputOption("USER_ENTERED").Do(); err != nil {
		fmt.Printf("Unable to access the sheet. %v", err)
		os.Exit(1)
	}
}

func makeService() *sheets.Service {
	jsonFile, err := os.Open(googleApiCredentials)
	if err != nil {
		fmt.Printf("Error in JSON opening: %v", err)
		os.Exit(1)
	}
	defer jsonFile.Close()

	type jsonData struct {
		Email        string `json:"client_email"`
		PrivateKey   string `json:"private_key"`
		PrivateKeyID string `json:"private_key_id"`
		TokenUri     string `json:"token_uri"`
	}

	var apiData jsonData

	byteValue, _ := ioutil.ReadAll(jsonFile)

	err = json.Unmarshal([]byte(byteValue), &apiData)
	if err != nil {
		fmt.Println("Error in JSON parsing: ", err)
		os.Exit(1)
	}

	conf := &jwt.Config{
		Email:        apiData.Email,
		PrivateKey:   []byte(apiData.PrivateKey),
		PrivateKeyID: apiData.PrivateKeyID,
		Scopes:       []string{googleApiUrl},
		TokenURL:     apiData.TokenUri,
	}

	ctx := context.Background()

	client := conf.Client(ctx)

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		fmt.Printf("Unable to retrieve Sheets client: %v", err)
		os.Exit(1)
	}
	return srv
}

// Check if sheet exists and empty
func checkSheet(sheetName string, srv *sheets.Service) (bool, error) {
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, sheetName+"!A1:A3").Do()
	return len(resp.Values) == 0, err
}
