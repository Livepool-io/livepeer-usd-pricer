package gecko

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
)

const endpoint = "https://api.coingecko.com/api/v3/simple/price?vs_currencies=usd&ids=ethereum"

type Gecko struct {
	http *http.Client
}

func NewGecko() *Gecko {
	return &Gecko{
		http: http.DefaultClient,
	}
}

func (g *Gecko) ETHUSD(ctx context.Context) (*big.Rat, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	res, err := g.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if 200 != res.StatusCode {
		return nil, fmt.Errorf("%s", body)
	}

	t := make(map[string]map[string]float64)

	if err := json.Unmarshal(body, &t); err != nil {
		return nil, err
	}

	price := t["ethereum"]["usd"]

	return new(big.Rat).SetFloat64(price), nil
}
