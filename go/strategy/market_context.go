package strategy

import (
	"fmt"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

type indicatorKind int

const (
	indicatorKindEMA indicatorKind = iota
	indicatorKindRSI
	indicatorKindATR
	indicatorKindVolumeMA
	indicatorKindHighestHigh
	indicatorKindLowestLow
	indicatorKindBBMiddle
	indicatorKindBBUpper
	indicatorKindBBLower
	indicatorKindStdDev
	indicatorKindKeltnerMiddle
	indicatorKindKeltnerUpper
)

// IndicatorCacheKey identifies a cached indicator series by kind and period.
type IndicatorCacheKey struct {
	key    indicatorKind
	period int
}

// ScaledIndicatorCacheKey adds the multiplier used by band indicators.
type ScaledIndicatorCacheKey struct {
	key        indicatorKind
	period     int
	multiplier float64
}

// MarketContext mirrors Java's MarketContext: immutable candle series plus lazy
// indicator caches shared by every bounded view.
type MarketContext struct {
	symbol             string
	candles            []domain.OhlcvCandle
	intradayTimestamps []time.Time
	seriesClose        []decimalNum
	seriesHigh         []decimalNum
	seriesLow          []decimalNum
	seriesOpen         []decimalNum
	seriesVolume       []decimalNum
	cache              map[IndicatorCacheKey][]decimalNum
	scaledCache        map[ScaledIndicatorCacheKey][]decimalNum
}

// MarketContextOf is the panicking convenience constructor used by fixtures.
func MarketContextOf(symbol string, candles []domain.OhlcvCandle) MarketContext {
	ctx, err := NewMarketContext(symbol, candles)
	if err != nil {
		panic(err)
	}
	return ctx
}

// NewMarketContext validates and builds a daily market context.
func NewMarketContext(symbol string, candles []domain.OhlcvCandle) (MarketContext, error) {
	if symbol == "" {
		return MarketContext{}, fmt.Errorf("symbol cannot be null or blank")
	}
	if len(candles) == 0 {
		return MarketContext{}, fmt.Errorf("candles cannot be null or empty")
	}
	ctx := MarketContext{
		symbol:      symbol,
		candles:     append([]domain.OhlcvCandle(nil), candles...),
		cache:       map[IndicatorCacheKey][]decimalNum{},
		scaledCache: map[ScaledIndicatorCacheKey][]decimalNum{},
	}
	ctx.seriesOpen = candleDecimalSeries(ctx.candles, func(c domain.OhlcvCandle) coredecimal.Decimal { return c.Open })
	ctx.seriesHigh = candleDecimalSeries(ctx.candles, func(c domain.OhlcvCandle) coredecimal.Decimal { return c.High })
	ctx.seriesLow = candleDecimalSeries(ctx.candles, func(c domain.OhlcvCandle) coredecimal.Decimal { return c.Low })
	ctx.seriesClose = candleDecimalSeries(ctx.candles, func(c domain.OhlcvCandle) coredecimal.Decimal { return c.Close })
	ctx.seriesVolume = candleDecimalSeries(ctx.candles, func(c domain.OhlcvCandle) coredecimal.Decimal {
		if c.Volume == nil {
			return coredecimal.Decimal{}
		}
		return coredecimalParse(fmt.Sprintf("%d", *c.Volume))
	})
	return ctx, nil
}

// MarketContextOfIntraday is the panicking intraday constructor.
func MarketContextOfIntraday(symbol string, candles []domain.OhlcvCandle, timestamps []time.Time) MarketContext {
	ctx, err := NewMarketContextIntraday(symbol, candles, timestamps)
	if err != nil {
		panic(err)
	}
	return ctx
}

// NewMarketContextIntraday attaches session timestamps to a daily context.
func NewMarketContextIntraday(symbol string, candles []domain.OhlcvCandle, timestamps []time.Time) (MarketContext, error) {
	if len(timestamps) != len(candles) {
		return MarketContext{}, fmt.Errorf("intraday timestamps must match candle count")
	}
	ctx, err := NewMarketContext(symbol, candles)
	if err != nil {
		return MarketContext{}, err
	}
	ts := append([]time.Time(nil), timestamps...)
	ctx.intradayTimestamps = ts
	return ctx, nil
}

func candleDecimalSeries(candles []domain.OhlcvCandle, f func(domain.OhlcvCandle) coredecimal.Decimal) []decimalNum {
	out := make([]decimalNum, 0, len(candles))
	for _, c := range candles {
		d, _ := newDecimalNum(cAdjusted(c, f).String())
		out = append(out, d)
	}
	return out
}

func cAdjusted(c domain.OhlcvCandle, f func(domain.OhlcvCandle) coredecimal.Decimal) coredecimal.Decimal {
	return f(c.AdjustedForAnalysis())
}

// Symbol returns the instrument symbol.
func (ctx MarketContext) Symbol() string { return ctx.symbol }

// BarCount returns the number of candles.
func (ctx MarketContext) BarCount() int { return len(ctx.candles) }

// View returns a look-ahead-bounded view at barIndex.
func (ctx MarketContext) View(barIndex int) View {
	if barIndex < 0 || barIndex >= len(ctx.candles) {
		panic(fmt.Errorf("bar index %d out of range [0,%d]", barIndex, len(ctx.candles)-1))
	}
	return View{ctx: ctx, barIndex: barIndex}
}

// CachedIndicatorCount reports how many distinct indicator series are cached.
func (ctx MarketContext) CachedIndicatorCount() int {
	return len(ctx.cache) + len(ctx.scaledCache)
}

// View exposes bars up to a bound index and panics on look-ahead reads.
type View struct {
	ctx      MarketContext
	barIndex int
}

// BarIndex returns the view's bound bar index.
func (v View) BarIndex() int { return v.barIndex }

// Symbol returns the instrument symbol.
func (v View) Symbol() string { return v.ctx.symbol }

// Date returns the candle date at index (default: the bound bar).
func (v View) Date(index ...int) time.Time {
	i := v.barIndex
	if len(index) > 0 {
		i = v.checkView(index[0])
	}
	return v.ctx.candles[i].Date
}

// SessionTime returns the intraday timestamp, or nil for daily contexts.
func (v View) SessionTime(index int) *time.Time {
	i := v.checkView(index)
	if v.ctx.intradayTimestamps == nil {
		return nil
	}
	t := v.ctx.intradayTimestamps[i]
	return &t
}

// Close returns the adjusted close at index (default: the bound bar).
func (v View) Close(indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	return decimalToCore(v.ctx.seriesClose[i])
}

// Open returns the adjusted open at index (default: the bound bar).
func (v View) Open(indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	return decimalToCore(v.ctx.seriesOpen[i])
}

// High returns the adjusted high at index (default: the bound bar).
func (v View) High(indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	return decimalToCore(v.ctx.seriesHigh[i])
}

// Low returns the adjusted low at index (default: the bound bar).
func (v View) Low(indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	return decimalToCore(v.ctx.seriesLow[i])
}

// Volume returns the volume at index (default: the bound bar).
func (v View) Volume(indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	return decimalToCore(v.ctx.seriesVolume[i])
}

// EMA returns the exponential moving average at index.
func (v View) EMA(period int, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	key := IndicatorCacheKey{key: indicatorKindEMA, period: period}
	ser := v.ctx.cachedSeriesExternal(key, func() []decimalNum { return computeEMA(v.ctx.seriesClose, period) })
	return decimalToCore(ser[i])
}

// RSI returns the relative strength index at index.
func (v View) RSI(period int, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	key := IndicatorCacheKey{key: indicatorKindRSI, period: period}
	ser := v.ctx.cachedSeriesExternal(key, func() []decimalNum { return computeRSI(v.ctx.seriesClose, period) })
	return decimalToCore(ser[i])
}

// ATR returns the average true range at index.
func (v View) ATR(period int, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	key := IndicatorCacheKey{key: indicatorKindATR, period: period}
	ser := v.ctx.cachedSeriesExternal(key, func() []decimalNum { return computeATR(v.ctx.seriesHigh, v.ctx.seriesLow, v.ctx.seriesClose, period) })
	return decimalToCore(ser[i])
}

// VolumeMA returns the simple moving average of volume at index.
func (v View) VolumeMA(period int, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	key := IndicatorCacheKey{key: indicatorKindVolumeMA, period: period}
	ser := v.ctx.cachedSeriesExternal(key, func() []decimalNum { return computeSMA(v.ctx.seriesVolume, period) })
	return decimalToCore(ser[i])
}

// HighestHigh returns the trailing highest high at index.
func (v View) HighestHigh(period int, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	bounded := period
	if bounded > len(v.ctx.candles) {
		bounded = len(v.ctx.candles)
	}
	key := IndicatorCacheKey{key: indicatorKindHighestHigh, period: bounded}
	ser := v.ctx.cachedSeriesExternal(key, func() []decimalNum { return computeHighest(v.ctx.seriesHigh, bounded) })
	return decimalToCore(ser[i])
}

// LowestLow returns the trailing lowest low at index.
func (v View) LowestLow(period int, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	bounded := period
	if bounded > len(v.ctx.candles) {
		bounded = len(v.ctx.candles)
	}
	key := IndicatorCacheKey{key: indicatorKindLowestLow, period: bounded}
	ser := v.ctx.cachedSeriesExternal(key, func() []decimalNum { return computeLowest(v.ctx.seriesLow, bounded) })
	return decimalToCore(ser[i])
}

// BBMiddle returns the Bollinger Band middle (SMA) at index.
func (v View) BBMiddle(period int, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	key := IndicatorCacheKey{key: indicatorKindBBMiddle, period: period}
	ser := v.ctx.cachedSeriesExternal(key, func() []decimalNum { return computeSMA(v.ctx.seriesClose, period) })
	return decimalToCore(ser[i])
}

// BBUpper returns the Bollinger Band upper band at index.
func (v View) BBUpper(period int, k coredecimal.Decimal, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	dk, _ := newDecimalNum(k.String())
	key := ScaledIndicatorCacheKey{key: indicatorKindBBUpper, period: period, multiplier: kfloat(k)}
	// compute std dev series once cached implicitly
	ser := v.ctx.cachedScaledExternal(key, func() []decimalNum {
		return computeBBUpper(v.ctx.seriesClose, period, dk)
	})
	return decimalToCore(ser[i])
}

// BBLower returns the Bollinger Band lower band at index.
func (v View) BBLower(period int, k coredecimal.Decimal, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	key := ScaledIndicatorCacheKey{key: indicatorKindBBLower, period: period, multiplier: kfloat(k)}
	ser := v.ctx.cachedScaledExternal(key, func() []decimalNum {
		return computeBBLower(v.ctx.seriesClose, period, func() decimalNum { d, _ := newDecimalNum(k.String()); return d }())
	})
	return decimalToCore(ser[i])
}

// BBWidth returns (upper-lower)/middle at index.
func (v View) BBWidth(period int, k coredecimal.Decimal, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	return decimalToCore(v.bbWidthNum(period, k, i))
}

// bbWidthNum mirrors Java's MarketContext.View.bbWidth: (upper-lower)/middle
// computed from full-precision indicators and divided at scale 10 HALF_UP. It
// stays in decimalNum space because the 4dp financial scale would collapse
// distinct low-volatility widths into the same value.
func (v View) bbWidthNum(period int, k coredecimal.Decimal, index int) decimalNum {
	i := v.checkView(index)
	middle := v.ctx.cachedSeriesExternal(
		IndicatorCacheKey{key: indicatorKindBBMiddle, period: period},
		func() []decimalNum { return computeSMA(v.ctx.seriesClose, period) },
	)[i]
	if middle.coefficient.Sign() == 0 {
		return decimalNum{}
	}
	stdDev := v.ctx.cachedSeriesExternal(
		IndicatorCacheKey{key: indicatorKindStdDev, period: period},
		func() []decimalNum { return computeStdDev(v.ctx.seriesClose, period) },
	)[i]
	dk, _ := newDecimalNum(k.String())
	band := stdDev.multiply(dk)
	width := middle.add(band).sub(middle.sub(band))
	quotient, err := width.divideRounded(middle, 10)
	if err != nil {
		return decimalNum{}
	}
	return quotient
}

// KeltnerUpper returns the Keltner Channel upper band at index.
func (v View) KeltnerUpper(period int, atrMult coredecimal.Decimal, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	key := ScaledIndicatorCacheKey{key: indicatorKindKeltnerUpper, period: period, multiplier: kfloat(atrMult)}
	ser := v.ctx.cachedScaledExternal(key, func() []decimalNum {
		return computeKeltnerUpper(v.ctx.seriesClose, v.ctx.seriesHigh, v.ctx.seriesLow, period, atrMult)
	})
	return decimalToCore(ser[i])
}

func (v View) checkView(index int) int {
	if index > v.barIndex {
		panic(fmt.Errorf("look-ahead violation: bar %d is beyond this view's bound of %d", index, v.barIndex))
	}
	if index < 0 {
		panic(fmt.Errorf("bar index cannot be negative: %d", index))
	}
	return index
}

// MarketContext wrapper methods for cached indicator count etc
// compatibility for performance? expose viewOp aliases
func (ctx *MarketContext) cachedSeriesExternal(key IndicatorCacheKey, compute func() []decimalNum) []decimalNum {
	if v, ok := ctx.cache[key]; ok {
		return v
	}
	v := compute()
	ctx.cache[key] = v
	return v
}
func (ctx *MarketContext) cachedScaledExternal(key ScaledIndicatorCacheKey, compute func() []decimalNum) []decimalNum {
	if v, ok := ctx.scaledCache[key]; ok {
		return v
	}
	v := compute()
	ctx.scaledCache[key] = v
	return v
}

func decimalToCore(d decimalNum) coredecimal.Decimal {
	dec, _ := coredecimal.Parse(d.String())
	return dec
}
func kfloat(d coredecimal.Decimal) float64 {
	var fl float64
	_, _ = fmt.Sscanf(d.String(), "%f", &fl)
	return fl
}

func coredecimalParse(s string) coredecimal.Decimal {
	d, _ := coredecimal.Parse(s)
	return d
}
