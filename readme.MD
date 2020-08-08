# 💲USD Pricing Tool 

## Build 

`$ go build`

## Start 

`$ ./usd-pricing`

* NOTE: It's highly recommended to run this tool on the same host as the actual Livepeer node as exposing your CLI webserver endpoint over the open internet currently has serious security implications as there is no authentication for the server. 

## Options

| Flag     | type | default | description |
|----------|------|---------|-------------|
| `-node`  |  string    |    http://localhost:7935     |     The CLI webserver address of the Livepeer node        |
| `-price` |   string   |         |      The target USD price per MILLION pixels  as a float string     |
| `-feed`  |  string    |    coingecko     |   comma-separated list of feeds to use {coingecko}          |
| `-priceDelta`         |   string   |    0.05     |    price change required to submit price update to the Livepeer node  as a float string representing a percentage (e.g 0.05 == 5%)        |
|  `-pollingInterval`        |  time.Duration    |     1h    |   Interval on which to request prices from the price feeds          |

## Creating Additional Feeds

A Feed is an HTTP client that implements the `Feed` interface.

```go
type Feed interface {
	ETHUSD(context.Context) (*big.Rat, error)
}
```

The `ETHUSD` method to be implemented is a remote call to a price service. This can be a REST API such as [coingecko](https://www.coingecko.com/en/api),  but also a smart contract call over JSON-RPC (e.g. [MakerDAO Medianizer](https://developer.makerdao.com/feeds/) )

After you create a `Feed` implementation for the API of your choice, add the feed to the switch statement in the `startFeed()` function in `main.go`. If the API requires an API key you should also add a flag for that. 

Feel free to submit a PR for additional feeds ! 


### Example 

```
$ ./usd-pricing -node http://localhost:7936 -price 0.05 -priceDelta 0.00001 -pollingInterval 10s

** Welcome to the Pricer **
This service changes your node's pricing based on the current ETHUSD exchange rate
You will currently charge $ 0.05 per million pixels **

I0808 19:46:51.342354   16324 main.go:107] Starting Pricer
I0808 19:46:52.185037   16324 pricer.go:43] initial ETHUSD price: 391.67
I0808 19:46:52.185179   16324 pricer.go:49] initial pixel price: 127658488.00
I0808 19:46:52.185194   16324 feeder.go:116] Sending price per pixel update pricePerUnit=127658488 pixelsPerUnit=1
```