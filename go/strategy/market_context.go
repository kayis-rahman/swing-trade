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
	indicatorKindKeltnerMiddle
	indicatorKindKeltnerUpper
)

type IndicatorCacheKey struct {
	key    indicatorKind
	period int
}

type ScaledIndicatorCacheKey struct {
	key         indicatorKind
	period      int
	multiplier  float64
}

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

func MarketContextOf(symbol string, candles []domain.OhlcvCandle) MarketContext {
	ctx, err := NewMarketContext(symbol, candles)
	if err != nil {
		panic(err)
	}
	return ctx
}

func NewMarketContext(symbol string, candles []domain.OhlcvCandle) (MarketContext, error) {
	if symbol == "" {
		return MarketContext{}, fmt.Errorf("Symbol cannot be null or blank")
	}
	if len(candles) == 0 {
		return MarketContext{}, fmt.Errorf("Candles cannot be null or empty")
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

func MarketContextOfIntraday(symbol string, candles []domain.OhlcvCandle, timestamps []time.Time) MarketContext {
	ctx, err := NewMarketContextIntraday(symbol, candles, timestamps)
	if err != nil {
		panic(err)
	}
	return ctx
}

func NewMarketContextIntraday(symbol string, candles []domain.OhlcvCandle, timestamps []time.Time) (MarketContext, error) {
	if len(timestamps) != len(candles) {
		return MarketContext{}, fmt.Errorf("Intraday timestamps must match candle count")
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

func (ctx MarketContext) Symbol() string { return ctx.symbol }

func (ctx MarketContext) BarCount() int { return len(ctx.candles) }

func (ctx MarketContext) View(barIndex int) View {
	if barIndex < 0 || barIndex >= len(ctx.candles) {
		panic(fmt.Errorf("Bar index %d out of range [0,%d]", barIndex, len(ctx.candles)-1))
	}
	return View{ctx: ctx, barIndex: barIndex}
}

func (ctx MarketContext) CachedIndicatorCount() int {
	return len(ctx.cache) + len(ctx.scaledCache)
}

func (ctx *MarketContext) cachedSeries(key IndicatorCacheKey, compute func() []decimalNum) []decimalNum {
	if v, ok := ctx.cache[key]; ok {
		return v
	}
	v := compute()
	ctx.cache[key] = v
	return v
}

func (ctx *MarketContext) cachedScaledSeries(key ScaledIndicatorCacheKey, compute func() []decimalNum) []decimalNum {
	if v, ok := ctx.scaledCache[key]; ok {
		return v
	}
	v := compute()
	ctx.scaledCache[key] = v
	return v
}

type View struct {
	ctx      MarketContext
	barIndex int
}

func (v View) BarIndex() int  { return v.barIndex }
func (v View) Symbol() string { return v.ctx.symbol }

func (v View) Date(index ...int) time.Time {
	i := v.barIndex
	if len(index) > 0 {
		i = v.checkView(index[0])
	}
	return v.ctx.candles[i].Date
}

func (v View) SessionTime(index int) *time.Time {
	i := v.checkView(index)
	if v.ctx.intradayTimestamps == nil {
		return nil
	}
	t := v.ctx.intradayTimestamps[i]
	return &t
}

func (v View) Close(indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	return decimalToCore(v.ctx.seriesClose[i])
}
func (v View) Open(indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	return decimalToCore(v.ctx.seriesOpen[i])
}
func (v View) High(indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	return decimalToCore(v.ctx.seriesHigh[i])
}
func (v View) Low(indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	return decimalToCore(v.ctx.seriesLow[i])
}
func (v View) Volume(indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	return decimalToCore(v.ctx.seriesVolume[i])
}

// coredecimal back-compat no-index methods used by tests
func (v View) close() coredecimal.Decimal { return v.Close() }
func (v View) open() coredecimal.Decimal  { return v.Open() }
func (v View) high() coredecimal.Decimal  { return v.High() }
func (v View) low() coredecimal.Decimal   { return v.Low() }

func (v View) EMA(period int, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	key := IndicatorCacheKey{key: indicatorKindEMA, period: period}
	ser := v.ctx.cachedSeriesExternal(key, func() []decimalNum { return computeEMA(v.ctx.seriesClose, period) })
	return decimalToCore(ser[i])
}
func (v View) RSI(period int, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	key := IndicatorCacheKey{key: indicatorKindRSI, period: period}
	ser := v.ctx.cachedSeriesExternal(key, func() []decimalNum { return computeRSI(v.ctx.seriesClose, period) })
	return decimalToCore(ser[i])
}
func (v View) ATR(period int, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	key := IndicatorCacheKey{key: indicatorKindATR, period: period}
	ser := v.ctx.cachedSeriesExternal(key, func() []decimalNum { return computeATR(v.ctx.seriesHigh, v.ctx.seriesLow, v.ctx.seriesClose, period) })
	return decimalToCore(ser[i])
}
func (v View) VolumeMA(period int, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	key := IndicatorCacheKey{key: indicatorKindVolumeMA, period: period}
	ser := v.ctx.cachedSeriesExternal(key, func() []decimalNum { return computeSMA(v.ctx.seriesVolume, period) })
	return decimalToCore(ser[i])
}
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
func (v View) BBMiddle(period int, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	key := IndicatorCacheKey{key: indicatorKindBBMiddle, period: period}
	ser := v.ctx.cachedSeriesExternal(key, func() []decimalNum { return computeSMA(v.ctx.seriesClose, period) })
	return decimalToCore(ser[i])
}
func (v View) BBUpper(period int, k coredecimal.Decimal, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	dk, _ := newDecimalNum(k.String())
	mult := 0.0 // unused
	_ = mult
	key := ScaledIndicatorCacheKey{key: indicatorKindBBUpper, period: period, multiplier: kfloat(k)}
	// compute std dev series once cached implicitly
	ser := v.ctx.cachedScaledExternal(key, func() []decimalNum {
		return computeBBUpper(v.ctx.seriesClose, period, dk)
	})
	return decimalToCore(ser[i])
}
func (v View) BBLower(period int, k coredecimal.Decimal, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	key := ScaledIndicatorCacheKey{key: indicatorKindBBLower, period: period, multiplier: kfloat(k)}
	ser := v.ctx.cachedScaledExternal(key, func() []decimalNum { return computeBBLower(v.ctx.seriesClose, period, func() decimalNum { d, _ := newDecimalNum(k.String()); return d }()) })
	return decimalToCore(ser[i])
}
func (v View) BBWidth(period int, k coredecimal.Decimal, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	middle := v.BBMiddle(period, i)
	if middle.Cmp(coredecimal.Decimal{}) == 0 {
		return coredecimal.Decimal{}
	}
	upper := v.BBUpper(period, k, i)
	lower := v.BBLower(period, k, i)
	diff := upper.Sub(lower)
	dm, _ := newDecimalNum(middle.String())
	du, _ := newDecimalNum(diff.String())
	q, _ := du.divide(dm)
	return decimalToCore(q)
}
func (v View) KeltnerUpper(period int, atrMult coredecimal.Decimal, indices ...int) coredecimal.Decimal {
	i := v.barIndex
	if len(indices) > 0 {
		i = v.checkView(indices[0])
	}
	key := ScaledIndicatorCacheKey{key: indicatorKindKeltnerUpper, period: period, multiplier: kfloat(atrMult)}
	ser := v.ctx.cachedScaledExternal(key, func() []decimalNum { return computeKeltnerUpper(v.ctx.seriesClose, v.ctx.seriesHigh, v.ctx.seriesLow, period, atrMult) })
	return decimalToCore(ser[i])
}

func (v View) checkView(index int) int {
	if index > v.barIndex {
		panic(fmt.Errorf("Look-ahead violation: bar %d is beyond this view's bound of %d", index, v.barIndex))
	}
	if index < 0 {
		panic(fmt.Errorf("Bar index cannot be negative: %d", index))
	}
	return index
}


func (v View) ema(period int, index int) coredecimal.Decimal  { return v.EMA(period, index) }
func (v View) rsi(period int, index int) coredecimal.Decimal  { return v.RSI(period, index) }
func (v View) atr(period int, index int) coredecimal.Decimal  { return v.ATR(period, index) }
func (v View) PivotHigh(period int)                          {}

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
func newDecimalCore(value string) coredecimal.Decimal {
	dec, _ := coredecimal.Parse(value)
	return dec
}
func kfloat(d coredecimal.Decimal) float64 {
	var fl float64
	fmt.Sscanf(d.String(), "%f", &fl)
	return fl
}

func coredecimalParse(s string) coredecimal.Decimal {
	d, _ := coredecimal.Parse(s)
	return d
}
