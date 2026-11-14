import com.swingtrade.strategy.TechnicalIndicators;
import com.swingtrade.strategy.TechnicalIndicators.BollingerBands;
import com.swingtrade.strategy.TechnicalIndicators.CandleWithPrices;
import com.swingtrade.strategy.TechnicalIndicators.StochasticValues;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.util.ArrayList;
import java.util.List;
import java.util.StringJoiner;

/**
 * Emits Java TechnicalIndicators outputs for a fixed, deterministic OHLCV series as JSON.
 * The Go test go/strategy/indicators_java_vectors_test.go replays the same inputs through the Go port
 * and requires bit-identical doubles. Run from the repository root with the compiled strategy classes:
 *
 *   java -cp "backend/strategy/build/classes/java/main:<ta4j-core.jar>:<slf4j-api.jar>:<spring-context.jar>" \
 *       docs/plans/go-migration/evidence/scripts/IndicatorVectors.java > go/strategy/testdata/java_indicator_vectors.json
 */
public final class IndicatorVectors {
    private static final int BARS = 120;
    private static final int[] PREFIXES = {30, 60, 120};

    public static void main(String[] args) {
        List<CandleWithPrices> candles = new ArrayList<>();
        StringJoiner candleJson = new StringJoiner(",\n    ", "[\n    ", "\n  ]");
        for (int i = 0; i < BARS; i++) {
            // Trend plus two incommensurate waves gives rises, falls, gaps and range changes.
            double base = 1000 + 2.5 * i + 40 * Math.sin(i / 5.0) + 15 * Math.cos(i / 2.3);
            BigDecimal close = money(base);
            BigDecimal open = money(base - 6 * Math.sin(i / 3.1));
            BigDecimal high = close.max(open).add(money(3 + 4 * Math.abs(Math.sin(i / 1.7))));
            BigDecimal low = close.min(open).subtract(money(2 + 5 * Math.abs(Math.cos(i / 2.9))));
            BigDecimal volume = BigDecimal.valueOf(100000 + (i * 7919L) % 50000);
            candles.add(new CandleWithPrices(open, high, low, close, volume));
            candleJson.add(String.format("{\"open\": %s, \"high\": %s, \"low\": %s, \"close\": %s, \"volume\": %s}",
                    open.toPlainString(), high.toPlainString(), low.toPlainString(), close.toPlainString(),
                    volume.toPlainString()));
        }

        TechnicalIndicators indicators = new TechnicalIndicators();
        StringJoiner cases = new StringJoiner(",\n    ", "[\n    ", "\n  ]");
        for (int bars : PREFIXES) {
            List<CandleWithPrices> window = candles.subList(0, bars);
            List<BigDecimal> closes = window.stream().map(CandleWithPrices::getClose).toList();
            List<BigDecimal> volumes = window.stream().map(CandleWithPrices::getVolume).toList();
            for (int period : new int[] {5, 20}) {
                cases.add(single(bars, "SMA", "[" + period + "]", indicators.calculateSMA(closes, period)));
                cases.add(single(bars, "EMA", "[" + period + "]", indicators.calculateEMA(closes, period)));
                cases.add(single(bars, "VolumeMA", "[" + period + "]", indicators.calculateVolumeMA(volumes, period)));
            }
            for (int period : new int[] {2, 14}) {
                cases.add(single(bars, "RSI", "[" + period + "]", indicators.calculateRSI(closes, period)));
            }
            cases.add(single(bars, "MACD", "[12, 26, 9]", indicators.calculateMACD(closes, 12, 26, 9)));
            cases.add(single(bars, "ATR", "[14]", indicators.calculateATR(window, 14)));
            cases.add(single(bars, "ADX", "[14]", indicators.calculateADX(window, 14)));
            cases.add(single(bars, "VWAP", "[]", indicators.calculateVWAP(window)));
            StochasticValues stochastic = indicators.calculateStochastic(window, 14, 3);
            cases.add(vector(bars, "Stochastic", "[14, 3]", stochastic.k(), stochastic.d()));
            BollingerBands bands = indicators.calculateBollingerBands(closes, 20, 2.0);
            cases.add(vector(bars, "BollingerBands", "[20, 2]", bands.upper(), bands.middle(), bands.lower()));
        }
        // Partial D window: TA4J's SMA divides by the observations available, not by three.
        StochasticValues early = indicators.calculateStochastic(candles.subList(0, 2), 1, 3);
        cases.add(vector(2, "Stochastic", "[1, 3]", early.k(), early.d()));
        System.out.println("{\n  \"generator\": \"docs/plans/go-migration/evidence/scripts/IndicatorVectors.java\",\n"
                + "  \"candles\": " + candleJson + ",\n  \"cases\": " + cases + "\n}");
    }

    private static BigDecimal money(double value) {
        return BigDecimal.valueOf(value).setScale(2, RoundingMode.HALF_UP);
    }

    private static String single(int bars, String name, String params, Double value) {
        return vector(bars, name, params, value);
    }

    private static String vector(int bars, String name, String params, Double... values) {
        StringJoiner out = new StringJoiner(", ", "[", "]");
        for (Double value : values) {
            out.add(String.valueOf(value));
        }
        return String.format("{\"bars\": %d, \"indicator\": \"%s\", \"params\": %s, \"values\": %s}",
                bars, name, params, out);
    }
}
