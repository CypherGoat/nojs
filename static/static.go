package static

type Coin struct {
	Ticker   string  `json:"ticker"`
	Name     string  `json:"name"`
	Network  string  `json:"network"`
	IsStable bool    `json:"isStable"`
	Icon     string  `json:"icon"`
	Min      float64 `json:"min"`
}
