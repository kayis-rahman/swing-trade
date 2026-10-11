package com.swingtrade.strategy;

import com.swingtrade.domain.OhlcvCandle;
import org.ta4j.core.Bar;
import org.ta4j.core.BarSeries;
import org.ta4j.core.BaseBar;
import org.ta4j.core.BaseBarSeries;
import org.ta4j.core.Indicator;
import org.ta4j.core.indicators.ATRIndicator;
import org.ta4j.core.indicators.EMAIndicator;
import org.ta4j.core.indicators.RSIIndicator;
import org.ta4j.core.indicators.SMAIndicator;
import org.ta4j.core.indicators.bollinger.BollingerBandsLowerIndicator;
import org.ta4j.core.indicators.bollinger.BollingerBandsMiddleIndicator;
import org.ta4j.core.indicators.bollinger.BollingerBandsUpperIndicator;
import org.ta4j.core.indicators.helpers.ClosePriceIndicator;
import org.ta4j.core.indicators.helpers.HighPriceIndicator;
import org.ta4j.core.indicators.helpers.HighestValueIndicator;
import org.ta4j.core.indicators.helpers.LowestValueIndicator;
import org.ta4j.core.indicators.helpers.LowPriceIndicator;
import org.ta4j.core.indicators.helpers.OpenPriceIndicator;
import org.ta4j.core.indicators.helpers.VolumeIndicator;
import org.ta4j.core.indicators.keltner.KeltnerChannelMiddleIndicator;
import org.ta4j.core.indicators.keltner.KeltnerChannelUpperIndicator;
import org.ta4j.core.num.DecimalNum;
import org.ta4j.core.num.Num;

import java.math.BigDecimal;
import java.time.Duration;
import java.time.LocalDate;
import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Per-symbol market data plus lazily-computed, cached indicators, shared by every
 * {@link SignalStrategy} variant evaluated for that symbol on a given job/backtest run
 * (plan §3.1). An indicator (e.g. {@code ema(20)}) is computed at most once per
 * {@code (IndicatorKey, period)} regardless of how many variants request it - see
 * {@link #cachedIndicatorCount()}.
 *
 * <p>{@link MarketContext} itself exposes no bar-level accessors; the only way to read data is
 * through {@link #view(int)}, whose {@link View} enforces that no bar after the requested
 * {@code barIndex} is ever read. This is the no-look-ahead guard required so the same
 * {@link SignalStrategy} evaluation code is safe to call live (barIndex == last bar) and at
 * every bar of a backtest.
 */
public final class MarketContext {

    private static final ZoneId MARKET_ZONE = ZoneId.of("Asia/Kolkata");

    private final String symbol;
    private final List<OhlcvCandle> candles;
    private final BarSeries series;

    private final ClosePriceIndicator closePriceIndicator;
    private final OpenPriceIndicator openPriceIndicator;
    private final HighPriceIndicator highPriceIndicator;
    private final LowPriceIndicator lowPriceIndicator;
    private final VolumeIndicator volumeIndicator;

    private final Map<IndicatorCacheKey, Indicator<Num>> cache = new ConcurrentHashMap<>();
    private final Map<ScaledIndicatorCacheKey, Indicator<Num>> scaledCache = new ConcurrentHashMap<>();

    private MarketContext(String symbol, List<OhlcvCandle> chronologicalCandles) {
        this(symbol, chronologicalCandles, null);
    }

    private MarketContext(String symbol, List<OhlcvCandle> chronologicalCandles,
                          List<ZonedDateTime> intradayTimestamps) {
        this.symbol = symbol;
        this.candles = List.copyOf(chronologicalCandles);
        this.series = intradayTimestamps == null ? buildBarSeries(symbol, this.candles)
            : buildIntradayBarSeries(symbol, this.candles, intradayTimestamps);
        this.closePriceIndicator = new ClosePriceIndicator(series);
        this.openPriceIndicator = new OpenPriceIndicator(series);
        this.highPriceIndicator = new HighPriceIndicator(series);
        this.lowPriceIndicator = new LowPriceIndicator(series);
        this.volumeIndicator = new VolumeIndicator(series);
    }

    /**
     * Builds a context over a chronologically-ordered (oldest first) candle series for a symbol.
     *
     * @throws IllegalArgumentException if symbol is null/blank or candles is null/empty
     */
    public static MarketContext of(String symbol, List<OhlcvCandle> chronologicalCandles) {
        validateInput(symbol, chronologicalCandles);
        return new MarketContext(symbol, chronologicalCandles);
    }

    /** Builds the same bounded strategy context over 15-minute candles without changing daily bars. */
    public static MarketContext ofIntraday(String symbol, List<OhlcvCandle> chronologicalCandles,
                                           List<ZonedDateTime> timestamps) {
        validateInput(symbol, chronologicalCandles);
        if (timestamps == null || timestamps.size() != chronologicalCandles.size()) {
            throw new IllegalArgumentException("Intraday timestamps must match candle count");
        }
        return new MarketContext(symbol, chronologicalCandles, List.copyOf(timestamps));
    }

    private static void validateInput(String symbol, List<OhlcvCandle> candles) {
        if (symbol == null || symbol.isBlank()) throw new IllegalArgumentException("Symbol cannot be null or blank");
        if (candles == null || candles.isEmpty()) throw new IllegalArgumentException("Candles cannot be null or empty");
    }

    public String symbol() {
        return symbol;
    }

    /** Total number of bars available in this context. */
    public int barCount() {
        return series.getBarCount();
    }

    /**
     * Returns a view bounded at {@code barIndex}: every read through it is limited to bars
     * {@code <= barIndex}.
     *
     * @throws IndexOutOfBoundsException if barIndex is outside [0, barCount())
     */
    public View view(int barIndex) {
        checkRawIndex(barIndex);
        return new View(barIndex);
    }

    private int checkRawIndex(int index) {
        if (index < 0 || index >= series.getBarCount()) {
            throw new IndexOutOfBoundsException(
                "Bar index " + index + " out of range [0," + (series.getBarCount() - 1) + "]");
        }
        return index;
    }

    @SuppressWarnings("unchecked")
    private EMAIndicator ema(int period) {
        return (EMAIndicator) cache.computeIfAbsent(new IndicatorCacheKey(IndicatorKey.EMA, period),
            key -> new EMAIndicator(closePriceIndicator, period));
    }

    @SuppressWarnings("unchecked")
    private RSIIndicator rsi(int period) {
        return (RSIIndicator) cache.computeIfAbsent(new IndicatorCacheKey(IndicatorKey.RSI, period),
            key -> new RSIIndicator(closePriceIndicator, period));
    }

    @SuppressWarnings("unchecked")
    private ATRIndicator atr(int period) {
        return (ATRIndicator) cache.computeIfAbsent(new IndicatorCacheKey(IndicatorKey.ATR, period),
            key -> new ATRIndicator(series, period));
    }

    @SuppressWarnings("unchecked")
    private SMAIndicator volumeMa(int period) {
        return (SMAIndicator) cache.computeIfAbsent(new IndicatorCacheKey(IndicatorKey.VOLUME_MA, period),
            key -> new SMAIndicator(volumeIndicator, period));
    }

    @SuppressWarnings("unchecked")
    private HighestValueIndicator highestHigh(int period) {
        int boundedPeriod = Math.min(period, series.getBarCount());
        return (HighestValueIndicator) cache.computeIfAbsent(
            new IndicatorCacheKey(IndicatorKey.HIGHEST_HIGH, boundedPeriod),
            key -> new HighestValueIndicator(highPriceIndicator, boundedPeriod));
    }

    @SuppressWarnings("unchecked")
    private LowestValueIndicator lowestLow(int period) {
        int boundedPeriod = Math.min(period, series.getBarCount());
        return (LowestValueIndicator) cache.computeIfAbsent(
            new IndicatorCacheKey(IndicatorKey.LOWEST_LOW, boundedPeriod),
            key -> new LowestValueIndicator(lowPriceIndicator, boundedPeriod));
    }

    @SuppressWarnings("unchecked")
    private BollingerBandsMiddleIndicator bbMiddle(int period) {
        return (BollingerBandsMiddleIndicator) cache.computeIfAbsent(
            new IndicatorCacheKey(IndicatorKey.BB_MIDDLE, period),
            key -> new BollingerBandsMiddleIndicator(new SMAIndicator(closePriceIndicator, period)));
    }

    @SuppressWarnings("unchecked")
    private BollingerBandsUpperIndicator bbUpper(int period, BigDecimal k) {
        return (BollingerBandsUpperIndicator) scaledCache.computeIfAbsent(
            new ScaledIndicatorCacheKey(IndicatorKey.BB_UPPER, period, k.doubleValue()),
            key -> new BollingerBandsUpperIndicator(bbMiddle(period),
                new org.ta4j.core.indicators.statistics.StandardDeviationIndicator(closePriceIndicator, period),
                DecimalNum.valueOf(k)));
    }

    @SuppressWarnings("unchecked")
    private BollingerBandsLowerIndicator bbLower(int period, BigDecimal k) {
        return (BollingerBandsLowerIndicator) scaledCache.computeIfAbsent(
            new ScaledIndicatorCacheKey(IndicatorKey.BB_LOWER, period, k.doubleValue()),
            key -> new BollingerBandsLowerIndicator(bbMiddle(period),
                new org.ta4j.core.indicators.statistics.StandardDeviationIndicator(closePriceIndicator, period),
                DecimalNum.valueOf(k)));
    }

    @SuppressWarnings("unchecked")
    private KeltnerChannelMiddleIndicator keltnerMiddle(int period) {
        return (KeltnerChannelMiddleIndicator) cache.computeIfAbsent(
            new IndicatorCacheKey(IndicatorKey.KELTNER_MIDDLE, period),
            key -> new KeltnerChannelMiddleIndicator(series, period));
    }

    @SuppressWarnings("unchecked")
    private KeltnerChannelUpperIndicator keltnerUpper(int period, BigDecimal atrMult) {
        return (KeltnerChannelUpperIndicator) scaledCache.computeIfAbsent(
            new ScaledIndicatorCacheKey(IndicatorKey.KELTNER_UPPER, period, atrMult.doubleValue()),
            key -> new KeltnerChannelUpperIndicator(keltnerMiddle(period), atrMult.doubleValue(), period));
    }

    /**
     * Number of distinct (indicator, period) instances computed so far. Test-only - proves the
     * lazy-cache requirement (plan §3.5: "12 variants -> each indicator computed once").
     */
    int cachedIndicatorCount() {
        return cache.size();
    }

    static BarSeries buildBarSeries(String symbol, List<OhlcvCandle> chronologicalCandles) {
        // Preserve the existing daily series shape and indicator behavior exactly.
        BarSeries barSeries = new BaseBarSeries(symbol, DecimalNum.valueOf(0));
        for (OhlcvCandle candle : chronologicalCandles) {
            addBar(barSeries, candle, candle.date().atStartOfDay(MARKET_ZONE), Duration.ofDays(1));
        }
        return barSeries;
    }

    private static BarSeries buildIntradayBarSeries(String symbol, List<OhlcvCandle> candles,
                                                     List<ZonedDateTime> timestamps) {
        BarSeries barSeries = new BaseBarSeries(symbol, DecimalNum.valueOf(0));
        for (int i = 0; i < candles.size(); i++) {
            addBar(barSeries, candles.get(i), timestamps.get(i).plusMinutes(15), Duration.ofMinutes(15));
        }
        return barSeries;
    }

    private static void addBar(BarSeries series, OhlcvCandle candle, ZonedDateTime endTime, Duration duration) {
        OhlcvCandle analytical = candle.adjustedForAnalysis();
        Long volume = analytical.volume();
        series.addBar(new BaseBar(duration, endTime, analytical.open(), analytical.high(), analytical.low(),
            analytical.close(), BigDecimal.valueOf(volume != null ? volume : 0L)));
    }

    private static BigDecimal numToBigDecimal(Num value) {
        return (BigDecimal) value.getDelegate();
    }

    /**
     * A read-only window onto its {@link MarketContext} bounded at {@link #barIndex()}. Every
     * accessor that takes an explicit index throws {@link IllegalStateException} if asked to
     * read a bar after {@link #barIndex()} - the no-look-ahead guard (plan §3.1).
     */
    public final class View {

        private final int barIndex;

        private View(int barIndex) {
            this.barIndex = barIndex;
        }

        public int barIndex() {
            return barIndex;
        }

        public String symbol() {
            return symbol;
        }

        public LocalDate date() {
            return date(barIndex);
        }

        public LocalDate date(int index) {
            return candles.get(checkView(index)).date();
        }

        public BigDecimal close() {
            return close(barIndex);
        }

        public BigDecimal close(int index) {
            return numToBigDecimal(closePriceIndicator.getValue(checkView(index)));
        }

        public BigDecimal open() {
            return open(barIndex);
        }

        public BigDecimal open(int index) {
            return numToBigDecimal(openPriceIndicator.getValue(checkView(index)));
        }

        public BigDecimal high() {
            return high(barIndex);
        }

        public BigDecimal high(int index) {
            return numToBigDecimal(highPriceIndicator.getValue(checkView(index)));
        }

        public BigDecimal low() {
            return low(barIndex);
        }

        public BigDecimal low(int index) {
            return numToBigDecimal(lowPriceIndicator.getValue(checkView(index)));
        }

        public BigDecimal volume() {
            return volume(barIndex);
        }

        public BigDecimal volume(int index) {
            return numToBigDecimal(volumeIndicator.getValue(checkView(index)));
        }

        public BigDecimal ema(int period) {
            return ema(period, barIndex);
        }

        public BigDecimal ema(int period, int index) {
            return numToBigDecimal(MarketContext.this.ema(period).getValue(checkView(index)));
        }

        public BigDecimal rsi(int period) {
            return rsi(period, barIndex);
        }

        public BigDecimal rsi(int period, int index) {
            return numToBigDecimal(MarketContext.this.rsi(period).getValue(checkView(index)));
        }

        public BigDecimal atr(int period) {
            return atr(period, barIndex);
        }

        public BigDecimal atr(int period, int index) {
            return numToBigDecimal(MarketContext.this.atr(period).getValue(checkView(index)));
        }

        public BigDecimal volumeMa(int period) {
            return volumeMa(period, barIndex);
        }

        public BigDecimal volumeMa(int period, int index) {
            return numToBigDecimal(MarketContext.this.volumeMa(period).getValue(checkView(index)));
        }

        public BigDecimal highestHigh(int period) {
            return highestHigh(period, barIndex);
        }

        public BigDecimal highestHigh(int period, int index) {
            return numToBigDecimal(MarketContext.this.highestHigh(period).getValue(checkView(index)));
        }

        public BigDecimal lowestLow(int period) {
            return lowestLow(period, barIndex);
        }

        public BigDecimal lowestLow(int period, int index) {
            return numToBigDecimal(MarketContext.this.lowestLow(period).getValue(checkView(index)));
        }

        public BigDecimal bbMiddle(int period) {
            return bbMiddle(period, barIndex);
        }

        public BigDecimal bbMiddle(int period, int index) {
            return numToBigDecimal(MarketContext.this.bbMiddle(period).getValue(checkView(index)));
        }

        public BigDecimal bbUpper(int period, BigDecimal k) {
            return bbUpper(period, k, barIndex);
        }

        public BigDecimal bbUpper(int period, BigDecimal k, int index) {
            return numToBigDecimal(MarketContext.this.bbUpper(period, k).getValue(checkView(index)));
        }

        public BigDecimal bbLower(int period, BigDecimal k) {
            return bbLower(period, k, barIndex);
        }

        public BigDecimal bbLower(int period, BigDecimal k, int index) {
            return numToBigDecimal(MarketContext.this.bbLower(period, k).getValue(checkView(index)));
        }

        /** {@code (upper-lower)/middle} - the raw Bollinger Band width ratio (plan §5.3). */
        public BigDecimal bbWidth(int period, BigDecimal k, int index) {
            BigDecimal middle = bbMiddle(period, index);
            if (middle.signum() == 0) {
                return BigDecimal.ZERO;
            }
            return bbUpper(period, k, index).subtract(bbLower(period, k, index))
                .divide(middle, 10, java.math.RoundingMode.HALF_UP);
        }

        public BigDecimal keltnerUpper(int period, BigDecimal atrMult) {
            return keltnerUpper(period, atrMult, barIndex);
        }

        public BigDecimal keltnerUpper(int period, BigDecimal atrMult, int index) {
            return numToBigDecimal(MarketContext.this.keltnerUpper(period, atrMult).getValue(checkView(index)));
        }

        private int checkView(int index) {
            if (index > barIndex) {
                throw new IllegalStateException(
                    "Look-ahead violation: bar " + index + " is beyond this view's bound of " + barIndex);
            }
            if (index < 0) {
                throw new IllegalArgumentException("Bar index cannot be negative: " + index);
            }
            return index;
        }
    }

    private record IndicatorCacheKey(IndicatorKey key, int period) {
    }

    /**
     * Cache key for indicators keyed on both a period and a decimal multiplier (Bollinger
     * {@code k}, Keltner ATR multiplier) - plan §5.3.
     */
    private record ScaledIndicatorCacheKey(IndicatorKey key, int period, double multiplier) {
    }
}
