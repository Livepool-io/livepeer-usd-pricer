package pricer

import (
	"context"
	"fmt"
	"livepool/usd-pricing/feeder"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
)

const maxDecimals = 18

type Pricer struct {
	feeder            *feeder.Feeder
	ticker            *time.Ticker
	basePriceUSD      *big.Rat
	minDeltaForUpdate *big.Rat
	quit              chan struct{}
}

func NewPricer(feeder *feeder.Feeder, basepriceUSD, minDeltaForUpdate *big.Rat, pollingInterval time.Duration) *Pricer {
	ticker := time.NewTicker(pollingInterval)
	return &Pricer{
		feeder,
		ticker,
		basepriceUSD,
		minDeltaForUpdate,
		make(chan struct{}),
	}
}

func (p *Pricer) Start() error {
	ctx := context.Background()
	last, err := p.feeder.ETHUSD(ctx)
	if err != nil {
		return err
	}
	for {
		select {
		case <-p.quit:
			return nil
		case <-p.ticker.C:
			newPrice, err := p.feeder.ETHUSD(context.Background())
			if err != nil {
				return err
			}

			// if the price is within an acceptable range of the last price
			// there is no need to update prematurely
			if isWithinDelta(newPrice, last, p.minDeltaForUpdate) {
				continue
			}

			glog.Infof("ETHUSD price change last=%v new=%v", last, newPrice)

			pixelPrice, err := p.pixelPrice(newPrice)
			if err != nil {
				glog.Error(err)
				continue
			}

			if err := p.feeder.PostPriceUpdate(context.Background(), pixelPrice); err != nil {
				glog.Error(err)
			}
		}

	}
}

func (p *Pricer) Stop() {
	close(p.quit)
}

func (p *Pricer) pixelPrice(usdPrice *big.Rat) (*big.Rat, error) {
	wei, err := toBaseAmount(new(big.Rat).Mul(
		p.basePriceUSD,
		usdPrice.Inv(usdPrice),
	).FloatString(18))

	if err != nil {
		return nil, err
	}

	return new(big.Rat).SetFrac(wei, big.NewInt(1)), nil
}

func isWithinDelta(newPrice, last, delta *big.Rat) bool {
	min := new(big.Rat).Sub(last, new(big.Rat).Mul(last, delta))
	max := new(big.Rat).Add(last, new(big.Rat).Mul(last, delta))
	return newPrice.Cmp(min) >= 0 && newPrice.Cmp(max) <= 0
}

func toBaseAmount(v string) (*big.Int, error) {
	// check that string is a float represented as a string with "." as seperator
	ok, err := regexp.MatchString("^[-+]?[0-9]*.?[0-9]+$", v)
	if !ok || err != nil {
		return nil, fmt.Errorf("submitted value %v is not a valid float", v)
	}
	splitNum := strings.Split(v, ".")
	// If '.' is absent from string, add an empty string to become the decimal part
	if len(splitNum) == 1 {
		splitNum = append(splitNum, "")
	}
	intPart := splitNum[0]
	decPart := splitNum[1]

	// If decimal part is longer than 18 decimals return an error
	if len(decPart) > maxDecimals {
		return nil, fmt.Errorf("submitted value has more than 18 decimals")
	}

	// If decimal part is shorter than 18 digits, pad with "0"
	for len(decPart) < maxDecimals {
		decPart += "0"
	}

	// Combine intPart and decPart into a big.Int
	baseAmount, ok := new(big.Int).SetString(intPart+decPart, 10)
	if !ok {
		return nil, fmt.Errorf("unable to convert floatString %v into big.Int", intPart+decPart)
	}

	return baseAmount, nil
}
