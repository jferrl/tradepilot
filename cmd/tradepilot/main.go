package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jferrl/go-kraken"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
	defer cancel()

	c := kraken.New(nil).
		WithAuth(
			kraken.Secrets{},
		)

	st, err := c.Market.OHCLData(ctx, kraken.OHCLDataOpts{
		Pair:     kraken.XXBTZUSD,
		Interval: 21600,
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	dataset := make([][]string, len(st.Pair))
	for i, v := range st.Pair {
		dataset[i] = []string{
			strconv.FormatInt(v.Time(), 10),
			v.Open(),
			v.Close(),
			v.High(),
			v.Low(),
			v.Volume(),
		}
	}

	series := techan.NewTimeSeries()
	for _, datum := range dataset {
		start, _ := strconv.ParseInt(datum[0], 10, 64)
		period := techan.NewTimePeriod(time.Unix(start, 0), time.Hour*24)

		candle := techan.NewCandle(period)
		candle.OpenPrice = big.NewFromString(datum[1])
		candle.ClosePrice = big.NewFromString(datum[2])
		candle.MaxPrice = big.NewFromString(datum[3])
		candle.MinPrice = big.NewFromString(datum[4])

		series.AddCandle(candle)
	}

	closePrices := techan.NewClosePriceIndicator(series)

	indicator := techan.NewEMAIndicator(closePrices, 10) // Create an exponential moving average with a window of 10

	// record trades on this object
	record := techan.NewTradingRecord()

	entryConstant := techan.NewConstantIndicator(30)
	exitConstant := techan.NewConstantIndicator(10)

	// Is satisfied when the price ema moves above 30 and the current position is new
	entryRule := techan.And(
		techan.NewCrossUpIndicatorRule(entryConstant, indicator),
		techan.PositionNewRule{})

	// Is satisfied when the price ema moves below 10 and the current position is open
	exitRule := techan.And(
		techan.NewCrossDownIndicatorRule(indicator, exitConstant),
		techan.PositionOpenRule{})

	strategy := techan.RuleStrategy{
		UnstablePeriod: 10, // Period before which ShouldEnter and ShouldExit will always return false
		EntryRule:      entryRule,
		ExitRule:       exitRule,
	}

	ok := strategy.ShouldEnter(0, record)

	fmt.Println(ok)
}
