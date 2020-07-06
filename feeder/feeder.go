package feeder

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/golang/glog"
)

var httpTimeout = 5 * time.Second

var bcastEndpoint = "/setBroadcastMaxPrice"
var orchEndpint = "/setOrchestratorPrice"

type Feed interface {
	ETHUSD(context.Context) (*big.Rat, error)
}

type Feeder struct {
	feeds  []Feed
	node   string
	isOrch bool
}

func NewFeeder(node string, feeds []Feed) (*Feeder, error) {
	res, err := http.Get(node + "/IsOrchestrator")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var isOrch bool
	if err := json.Unmarshal(body, &isOrch); err != nil {
		return nil, err
	}

	return &Feeder{
		feeds,
		node,
		isOrch,
	}, nil
}

func (f *Feeder) ETHUSD(ctx context.Context) (*big.Rat, error) {
	// Get USD prices from the feeds
	var wg sync.WaitGroup
	num := len(f.feeds)
	prices := make([]*big.Rat, 0)
	priceCh := make(chan *big.Rat, num)

	getPrice := func(feed Feed) {
		ctx, cancel := context.WithTimeout(ctx, httpTimeout)
		defer cancel()
		price, err := feed.ETHUSD(ctx)
		if err != nil {
			glog.Errorf("Unable to get price for feed=%t", feed)
			return
		}
		priceCh <- price
		wg.Done()
	}

	for _, feed := range f.feeds {
		wg.Add(1)
		go getPrice(feed)
	}

	wg.Wait()
	close(priceCh)

	for price := range priceCh {
		prices = append(prices, price)
	}

	// calculate the median
	median := getMedian(prices)

	// discard outliers
	var cleanedPrices []*big.Rat
	for _, p := range prices {
		if !isOutlier(p, median) {
			cleanedPrices = append(cleanedPrices, p)
		}
	}

	// TODO: make this the mean
	return getMedian(cleanedPrices), nil
}

func (f *Feeder) PostPriceUpdate(ctx context.Context, pricePerPixel *big.Rat) error {
	uri := f.node + orchEndpint
	if !f.isOrch {
		uri = f.node + bcastEndpoint
	}

	val := url.Values{
		"pricePerUnit":  {pricePerPixel.Num().String()},
		"pixelsPerUnit": {pricePerPixel.Denom().String()},
	}

	glog.Infof("Sending price per pixel update pricePerUnit=%v pixelsPerUnit=%v", pricePerPixel.Num(), pricePerPixel.Denom())
	return httpPostWithParams(uri, val)
}

func getMedian(prices []*big.Rat) *big.Rat {
	if len(prices) == 0 {
		return nil
	}

	if len(prices) == 1 {
		return prices[0]
	}

	sort.SliceStable(prices, func(i, j int) bool {
		return prices[i].Cmp(prices[j]) < 0
	})

	if isOdd(len(prices)) {
		return prices[len(prices)/2]
	}

	a := prices[len(prices)/2-1]
	b := prices[len(prices)/2]

	return new(big.Rat).Mul(
		new(big.Rat).Add(a, b),
		big.NewRat(1, 2),
	)
}

func isOdd(n int) bool {
	return n%2 > 0
}

func isOutlier(val, median *big.Rat) bool {
	min := new(big.Rat).Mul(median, big.NewRat(95, 100))
	max := new(big.Rat).Mul(median, big.NewRat(105, 100))
	return val.Cmp(min) < 0 || val.Cmp(max) > 0
}

func httpPostWithParams(url string, val url.Values) error {
	body := bytes.NewBufferString(val.Encode())

	resp, err := http.Post(url, "application/x-www-form-urlencoded", body)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil || string(result) == "" {
		return err
	}

	return nil
}
