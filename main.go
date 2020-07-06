package main

import (
	"flag"
	"fmt"
	"livepool/usd-pricing/feeder"
	gecko "livepool/usd-pricing/feeds/coingecko"
	"livepool/usd-pricing/pricer"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/golang/glog"
)

const httpPort = "7935"

func defaultAddr(addr, defaultHost, defaultPort string) string {
	if addr == "" {
		return defaultHost + ":" + defaultPort
	}
	if addr[0] == ':' {
		return defaultHost + addr
	}
	// not IPv6 safe
	if !strings.Contains(addr, ":") {
		return addr + ":" + defaultPort
	}
	return addr
}

func startFeed(feed string) feeder.Feed {
	switch feed {
	case "coingecko":
		return gecko.NewGecko()
	default:
		return nil
	}
}

func main() {
	flag.Set("logtostderr", "true")
	vFlag := flag.Lookup("v")

	//We preserve this flag before resetting all the flags.  Not a scalable approach, but it'll do for now.  More discussions here - https://github.com/livepeer/go-livepeer/pull/617
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	node := flag.String("node", "", "CLI webserver address of the Livepeer node")
	price := flag.String("price", "0.0002", "Price per MILLION PIXELS as a float string")
	feed := flag.String("feed", "coingecko", "the pricing feed(s) to use")
	minUpdateDelta := flag.String("priceDelta", "0.05", "percentage as a floatstring of price delta required to update pixel pricing on the node")
	pollingInterval := flag.Duration("pollingInterval", 1*time.Hour, "polling interval to fetch ethereum price")
	verbosity := flag.String("v", "4", "Log verbosity.  {4|5|6}")

	flag.Parse()
	vFlag.Value.Set(*verbosity)

	fmt.Println("** Welcome to the Pricer **")
	fmt.Println("This service changes your node's pricing based on the current ETHUSD exchange rate")
	fmt.Printf("You will currently charge $ %v per million pixels **\n", *price)
	fmt.Println("")
	nodeAddr := defaultAddr(*node, "127.0.0.1", httpPort)
	if !strings.HasPrefix(nodeAddr, "http") {
		nodeAddr = "http://" + nodeAddr
	}

	priceRatMillion, ok := new(big.Rat).SetString(*price)
	if !ok {
		glog.Errorf("provided price is not a valid float string: %v", price)
		return
	}
	priceRat := priceRatMillion.Mul(priceRatMillion, big.NewRat(1, 1000000))

	delta, ok := new(big.Rat).SetString(*minUpdateDelta)
	if !ok {
		glog.Errorf("provided price delta is not a valid float string: %v", *minUpdateDelta)
	}

	var feeds []feeder.Feed
	if len(*feed) > 0 {
		for _, feed := range strings.Split(*feed, ",") {
			feed = strings.TrimSpace(feed)
			feedClient := startFeed(feed)
			if feedClient == nil {
				glog.Errorf("provided feed '%v' is not valid", feed)
				continue
			}
			feeds = append(feeds, feedClient)
		}
	}
	if len(feeds) == 0 {
		glog.Errorf("No feeds to fetch price from")
		return
	}

	feeder, err := feeder.NewFeeder(nodeAddr, feeds)
	if err != nil {
		glog.Errorf("Unable to start feeder: %v", err)
		return
	}

	pricer := pricer.NewPricer(feeder, priceRat, delta, *pollingInterval)

	errCh := make(chan error)
	go func() {
		errCh <- pricer.Start()
	}()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	select {
	case <-c:
	case err := <-errCh:
		if err != nil {
			glog.Error(err)
		}
		return
	}
}
