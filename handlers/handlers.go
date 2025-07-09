// Copyright (C) 2025 CypherGoat
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/CypherGoat/nojs/views"

	"github.com/CypherGoat/nojs/cmd/api"

	"github.com/labstack/echo/v4"
)

var blockedExchangesForAnonymousNetworks = map[string]bool{
	"stealthex":  true,
	"changenow":  true,
	"fixedfloat": true,
	"simpleswap": true,
	"exolix":     true,
	"nanswap":    true,
}

func isExchangeBlocked(exchangeName string, isAnonymousNetwork bool) bool {
	if !isAnonymousNetwork {
		return false
	}
	return blockedExchangesForAnonymousNetworks[strings.ToLower(exchangeName)]
}

var (
	transactions = make(map[string]api.Transaction)
	mu           sync.Mutex
)

func IndexHandler(c echo.Context) error {
	return views.IndexTempl().Render(c.Request().Context(), c.Response())
}

func ContactHandler(c echo.Context) error {
	return views.Contact().Render(c.Request().Context(), c.Response())
}

func AffiliateTerms(c echo.Context) error {
	return views.AffiliateTerms().Render(c.Request().Context(), c.Response())
}

func AboutHandler(c echo.Context) error {
	return views.About().Render(c.Request().Context(), c.Response())

}

func TermsHandler(c echo.Context) error {
	return views.Terms().Render(c.Request().Context(), c.Response())

}

func PrivacyPolicyHandler(c echo.Context) error {
	return views.Privacy().Render(c.Request().Context(), c.Response())

}

func AffiliateHandler(c echo.Context) error {
	return views.Affiliate().Render(c.Request().Context(), c.Response())
}

func NotFoundHandler(c echo.Context) error {
	return views.NotFoundPage().Render(c.Request().Context(), c.Response())

}

type ExchangeInfo struct {
	ImageURL  string
	RequireIP bool
}

var exchangeInfo = map[string]ExchangeInfo{
	"alfacash":     {"/exchanges/alfacash.svg", true},
	"changee":      {"/exchanges/changee.svg", true},
	"changehero":   {"/exchanges/changehero.svg", true},
	"changenow":    {"/exchanges/changenow.svg", true},
	"coincraddle":  {"/exchanges/coincraddle.png", true},
	"exch.cx":      {"/exchanges/exchcx.png", false},
	"fixedfloat":   {"/exchanges/fixedfloat-v2.svg", true},
	"majesticbank": {"/exchanges/majesticbank.png", false},
	"nanswap":      {"/exchanges/nanswap.svg", true},
	"simpleswap":   {"/exchanges/simpleswap.svg", true},
	"wizardswap":   {"/exchanges/wizardswap.png", false},
	"stealthex":    {"/exchanges/stealthex.svg", true},
	"exolix":       {"/exchanges/exolix.png", true},
	"swapuz":       {"/exchanges/swapuz.svg", false},
	"bitcoinvn":    {"/exchanges/bitcoinvn.png", false},
	"pegasusswap":  {"/exchanges/pegasusswap.png", false},
	"godex":        {"/exchanges/godex.svg", true},
}

type EstimateWithBlocking struct {
	api.Estimate
	Blocked bool
}

func EstimateHandler(c echo.Context) error {
	amount := c.QueryParam("amount")
	coin1 := c.QueryParam("coin1")
	coin2 := c.QueryParam("coin2")
	sorting := c.QueryParam("sort")

	coin1 = strings.ToLower(coin1)
	coin2 = strings.ToLower(coin2)

	query := url.Values{}
	for key, values := range c.QueryParams() {
		if key == "sort" {
			continue
		}
		for _, v := range values {
			query.Add(key, v)
		}
	}
	baseQuery := query.Encode()

	if sorting != "kyc" {
		sorting = "rate"
	}

	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return views.EstimateTempl(err.Error(), []api.Estimate{}, "", baseQuery).Render(c.Request().Context(), c.Response())
	}
	if amountFloat == 0 {
		return views.EstimateTempl("amount can't be zero", []api.Estimate{}, "", baseQuery).Render(c.Request().Context(), c.Response())
	}

	coin1Ticker, network1 := parseCoinValue(coin1)
	coin2Ticker, network2 := parseCoinValue(coin2)

	estimates, err := api.FetchEstimateFromAPI(coin1Ticker, coin2Ticker, amountFloat, false, network1, network2)
	if err != nil {
		return views.EstimateTempl(err.Error(), []api.Estimate{}, "", baseQuery).Render(c.Request().Context(), c.Response())
	}

	isAnonymousNetwork, _ := c.Get("isAnonymousNetwork").(bool)

	for i := range estimates {
		// fmt.Println(apiEstimates[i].ExchangeName)

		name := strings.ToLower(estimates[i].ExchangeName)

		if info, ok := exchangeInfo[name]; ok {
			estimates[i].ImageURL = info.ImageURL
		} else {
			estimates[i].ImageURL = ""
		}

		estimates[i].Log = exchangeInfo[name].RequireIP
		estimates[i].Blocked = isExchangeBlocked(estimates[i].ExchangeName, isAnonymousNetwork)

	}

	if sorting == "kyc" {
		sort.SliceStable(estimates, func(i, j int) bool {
			if estimates[i].KYCScore == estimates[j].KYCScore {
				return estimates[i].ReceiveAmount > estimates[j].ReceiveAmount
			}
			return estimates[i].KYCScore < estimates[j].KYCScore
		})
	}

	return views.EstimateTempl("", estimates, sorting, baseQuery).Render(c.Request().Context(), c.Response())
}

func parseCoinValue(value string) (string, string) {
	re := regexp.MustCompile(`^([^\(]+)(?:\(([^)]+)\))?$`)
	matches := re.FindStringSubmatch(value)
	if len(matches) == 3 {
		ticker := matches[1]
		network := matches[2]
		if network == "" {
			network = ticker
		}
		return ticker, network
	}
	return value, value
}

func Step2Handler(c echo.Context) error {
	coin1 := c.QueryParam("coin1")
	coin2 := c.QueryParam("coin2")
	network1 := c.QueryParam("network1")
	network2 := c.QueryParam("network2")
	amountStr := c.QueryParam("amount")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Error parsing amount")
	}
	partner := c.QueryParam("partner")
	receiveamountStr := c.QueryParam("receiveamount")
	receiveamount, err := strconv.ParseFloat(receiveamountStr, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Error parsing receive amount")
	}

	isAnonymousNetwork, _ := c.Get("isAnonymousNetwork").(bool)
	if isExchangeBlocked(partner, isAnonymousNetwork) {
		return c.JSON(http.StatusForbidden, "Exchange not available via anonymous networks")
	}

	var estimate api.Estimate
	estimate.Coin1 = coin1
	estimate.Coin2 = coin2
	estimate.SendAmount = amount
	estimate.ExchangeName = partner
	estimate.ReceiveAmount = receiveamount
	estimate.Network1 = network1
	estimate.Network2 = network2

	name := strings.ToLower(estimate.ExchangeName)
	if info, ok := exchangeInfo[name]; ok {
		estimate.ImageURL = info.ImageURL
	} else {
		estimate.ImageURL = "https://example.com/images/default.png"
	}

	return views.AddressForm(estimate, false).Render(c.Request().Context(), c.Response())
}

func Step3Handler(c echo.Context) error {
	coin1 := c.QueryParam("coin1")
	coin2 := c.QueryParam("coin2")
	network1 := c.QueryParam("network1")
	network2 := c.QueryParam("network2")
	amountStr := c.QueryParam("amount")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "error parsing amount")
	}
	partner := c.QueryParam("partner")
	address := c.QueryParam("address")

	info := api.Info{
		IP:        "",
		UserAgent: "",
		LangList:  "",
	}
	partnerLower := strings.ToLower(partner)

	isAnonymousNetwork, _ := c.Get("isAnonymousNetwork").(bool)
	if isExchangeBlocked(partner, isAnonymousNetwork) {
		return c.JSON(http.StatusForbidden, "Exchange not available via anonymous networks")
	}

	if exchangeData, ok := exchangeInfo[partnerLower]; ok && exchangeData.RequireIP {
		ip := c.RealIP()
		userAgent := c.Request().Header.Get("User-Agent")
		langList := c.Request().Header.Get("Accept-Language")

		info = api.Info{
			IP:        ip,
			UserAgent: userAgent,
			LangList:  langList,
		}
		// fmt.Printf("Debug - Info values for %s: IP=%s, UA=%s, Lang=%s\n",
		// partner, ip, userAgent, langList)
	}

	affiliate := ""
	affiliateCookie, err := c.Cookie("affiliate")
	if err == nil && affiliateCookie.Value != "" {
		affiliate = affiliateCookie.Value
	}

	err, transaction := api.CreateTradeFromAPI(coin1, coin2, amount, address, partner, network1, network2, affiliate, info)
	if err != nil || transaction.Id == "" {
		var estimate api.Estimate
		estimate.Coin1 = coin1
		estimate.Coin2 = coin2
		estimate.SendAmount = amount
		estimate.ExchangeName = partner
		estimate.ReceiveAmount = amount
		estimate.Network1 = network1
		estimate.Network2 = network2

		return views.AddressForm(estimate, true).Render(c.Request().Context(), c.Response())
	}

	fmt.Printf("Redirecting to transaction: /transaction/%s\n", transaction.CGID)

	return c.Redirect(http.StatusSeeOther, "/transaction/"+transaction.CGID)
}

func GetTransactionHandler(c echo.Context) error {
	id := c.Param("id")

	err, transaction := api.GetTransactionFromAPI(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("error getting transaction: %v", err))
	}

	return views.TransactionPage(transaction).Render(c.Request().Context(), c.Response())

}

func getCoinName(ticker string) string {
	defaultName := strings.ToUpper(ticker)

	jsonFile, err := os.ReadFile("static/coins.json")
	if err != nil {
		fmt.Printf("Error reading coins.json: %v\n", err)
		return defaultName
	}

	type Coin struct {
		Ticker  string `json:"ticker"`
		Name    string `json:"name"`
		Network string `json:"network"`
	}

	var coins []Coin
	if err := json.Unmarshal(jsonFile, &coins); err != nil {
		fmt.Printf("Error parsing coins.json: %v\n", err)
		return defaultName
	}

	for _, coin := range coins {
		if strings.EqualFold(coin.Ticker, ticker) &&
			(coin.Network == ticker || coin.Network == "") {
			return coin.Name
		}
	}

	for _, coin := range coins {
		if strings.EqualFold(coin.Ticker, ticker) {
			return coin.Name
		}
	}

	return defaultName
}

func coinExists(ticker string) (string, bool) {
	jsonFile, err := os.ReadFile("static/coins.json")
	if err != nil {
		fmt.Printf("Error reading coins.json: %v\n", err)
		return "", false
	}

	type Coin struct {
		Ticker  string `json:"ticker"`
		Name    string `json:"name"`
		Network string `json:"network"`
	}

	var coins []Coin
	if err := json.Unmarshal(jsonFile, &coins); err != nil {
		fmt.Printf("Error parsing coins.json: %v\n", err)
		return "", false
	}

	for _, coin := range coins {
		if strings.EqualFold(coin.Ticker, ticker) &&
			(coin.Network == ticker || coin.Network == "") {
			return coin.Name, true
		}
	}

	for _, coin := range coins {
		if strings.EqualFold(coin.Ticker, ticker) {
			return coin.Name, true
		}
	}

	return "", false
}

func HealthHandler(c echo.Context) error {
	c.String(http.StatusOK, "OK")
	return nil
}

func RobotsHandler(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, "text/plain")

	robots := `User-agent: *
Disallow: /
`

	return c.String(http.StatusOK, robots)
}
func SitemapHandler(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, "application/xml")

	sitemap := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`

	// Add static pages
	staticPages := []struct {
		Path       string
		ChangeFreq string
		Priority   string
	}{
		{"/", "daily", "1.0"},
		{"/about", "monthly", "0.8"},
		{"/terms", "monthly", "0.7"},
		{"/privacy", "monthly", "0.7"},
		{"/contact", "monthly", "0.8"},
		{"/affiliate", "weekly", "0.9"},
		{"/affiliate/terms", "monthly", "0.7"},
	}

	baseURL := "https://cyphergoat.com"

	for _, page := range staticPages {
		sitemap += fmt.Sprintf(`
  <url>
    <loc>%s%s</loc>
    <changefreq>%s</changefreq>
    <priority>%s</priority>
  </url>`, baseURL, page.Path, page.ChangeFreq, page.Priority)
	}

	jsonFile, err := os.ReadFile("static/coins.json")
	if err == nil {
		type Coin struct {
			Ticker  string `json:"ticker"`
			Name    string `json:"name"`
			Network string `json:"network"`
		}

		var coins []Coin
		if err := json.Unmarshal(jsonFile, &coins); err == nil {
			uniqueTickers := make(map[string]bool)

			for _, coin := range coins {
				ticker := strings.ToLower(coin.Ticker)
				if _, exists := uniqueTickers[ticker]; !exists {
					uniqueTickers[ticker] = true

					sitemap += fmt.Sprintf(`
  <url>
    <loc>%s/swap/%s</loc>
    <changefreq>weekly</changefreq>
    <priority>0.8</priority>
  </url>`, baseURL, ticker)
				}
			}
		}
	}

	sitemap += `
</urlset>`

	return c.String(http.StatusOK, sitemap)
}
