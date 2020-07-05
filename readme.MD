# 💲USD Pricing Tool 

## Build 

`$ go build .`

## Start 

`$ usd-pricing`

## Options

| Flag     | type | default | description |
|----------|------|---------|-------------|
| `-node`  |  string    |    http://localhost:7935     |     The CLI webserver address of the Livepeer node        |
| `-price` |   string   |         |      The target USD price per MILLION pixels  as a float string     |
| `-feed`  |  string    |    coingecko     |   comma-separated list of feeds to use {coingecko}          |
| `-priceDelta`         |   string   |    0.05     |    price change required to submit price update to the Livepeer node  as a float string representing a percentage (e.g 0.05 == 5%)        |
|  `-pollingInterval`        |  time.Duration    |     1h    |   Interval on which to request prices from the price feeds          |