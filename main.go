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

	sheets "google.golang.org/api/sheets/v4"

	"golang.org/x/oauth2/jwt"
)

const (
	maxNoOfCurrencies = 5
	maxNoOfRates      = 65
	url1              = "https://cryptorank.io/ru/"
	url2              = "https://www.coingecko.com/"
	spreadsheetId     = "1GCPhTKxdkJ4pM0irfJx3bMzWMl5UGHEb8_QpEX4TtK4"
)

type result struct {
	name, tag string
	rate      string
	timestamp int64
}

var results []result

func main() {
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{url1},
		ParseFunc: parseUrl1,
	}).Start()

	writeRes(&results, 1)

	results = nil

	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{url2},
		ParseFunc: parseUrl2,
	}).Start()

	writeRes(&results, 2)
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

func writeRes(res *[]result, urlNo int) {
	jsonFile, err := os.Open("credentials.json")
	if err != nil {
		fmt.Println("Error in JSON opening: %v", err)
		os.Exit(1)
	}
	defer jsonFile.Close()

	type jsonData struct {
		Email        string `json:"client_email"`
		PrivateKey   string `json:"private_key"`
		PrivateKeyID string `json:"private_key_id"`
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
		Scopes:       []string{"https://www.googleapis.com/auth/spreadsheets"},
		TokenURL:     "https://oauth2.googleapis.com/token",
	}

	client := conf.Client(context.Background())

	srv, err := sheets.New(client)
	if err != nil {
		fmt.Println("Unable to retrieve Sheets client: %v", err)
		os.Exit(1)
	}

	var writeRange string
	var vr sheets.ValueRange
	var myval []interface{}

	if urlNo == 1 {
		writeRange = "Лист1!A1"
		vr.Values = append(vr.Values, []interface{}{"Name", "Tag", "TimeStamp"})
	} else {
		writeRange = "Лист2!A1"
		vr.Values = append(vr.Values, []interface{}{"Name", "Rate", "TimeStamp"})
	}

	for _, val := range *res {
		if urlNo == 1 {
			myval = []interface{}{val.name, val.tag, strconv.FormatInt(val.timestamp, 10)}
		} else {
			myval = []interface{}{val.name, val.rate, strconv.FormatInt(val.timestamp, 10)}
		}
		vr.Values = append(vr.Values, myval)
	}

	_, err = srv.Spreadsheets.Values.Update(spreadsheetId, writeRange, &vr).ValueInputOption("RAW").Do()
	if err != nil {
		fmt.Println("Unable to access the sheet. %v", err)
		os.Exit(1)
	}
}
