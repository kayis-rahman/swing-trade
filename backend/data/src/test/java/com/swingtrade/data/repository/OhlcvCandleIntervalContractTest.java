package com.swingtrade.data.repository;

import com.swingtrade.data.entity.OhlcvCandleEntity;
import com.swingtrade.data.service.CandleData;
import com.swingtrade.data.service.Interval;
import com.swingtrade.data.service.MarketDataClient;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.data.domain.Pageable;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalTime;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

/**
 * Contract tests for the interval dimension added to {@code ohlcv_candles} (V75).
 *
 * <p>Two guarantees are proven:
 * <ol>
 *   <li><b>Daily parity</b> — every pre-existing daily-mode query returns exactly
 *       the rows it returned before the interval dimension existed, even with
 *       15-minute bars stored for the same symbol and date.</li>
 *   <li><b>Interval correctness</b> — 15-minute queries return only 15-minute
 *       rows with their bar start times, and the widened identity
 *       {@code (symbol, interval, date, bar_time)} is enforced.</li>
 * </ol>
 */
@SpringBootTest(classes = OhlcvCandleRepositoryTestConfig.class)
@ActiveProfiles(resolver = com.swingtrade.data.test.TestProfileResolver.class)
@Transactional
class OhlcvCandleIntervalContractTest {

    private static final LocalDate SESSION_DATE = LocalDate.of(2026, 10, 5); // Monday
    private static final LocalDate NEXT_SESSION = LocalDate.of(2026, 10, 6);  // Tuesday

    @Autowired
    private OhlcvCandleRepository repository;

    private static CandleData daily(String symbol, LocalDate date, double close) {
        return CandleData.of(symbol, date,
            BigDecimal.valueOf(close - 1), BigDecimal.valueOf(close + 1),
            BigDecimal.valueOf(close - 2), BigDecimal.valueOf(close), 1_000_000L);
    }

    private static CandleData intraday(String symbol, LocalDate date, LocalTime barTime, double close) {
        return CandleData.of(symbol, date, barTime,
            BigDecimal.valueOf(close - 1), BigDecimal.valueOf(close + 1),
            BigDecimal.valueOf(close - 2), BigDecimal.valueOf(close), 500_000L,
            BigDecimal.valueOf(close), Interval.FIFTEEN_MINUTES);
    }

    private OhlcvCandleEntity saveDaily(String symbol, LocalDate date, double close) {
        return repository.save(OhlcvCandleEntity.fromCandleData(daily(symbol, date, close)));
    }

    private OhlcvCandleEntity saveIntraday(String symbol, LocalDate date, LocalTime barTime, double close) {
        return repository.save(OhlcvCandleEntity.fromCandleData(intraday(symbol, date, barTime, close)));
    }

    private void seedMixedData() {
        // Daily rows for two sessions.
        saveDaily("TCS", SESSION_DATE, 4000.0);
        saveDaily("TCS", NEXT_SESSION, 4100.0);
        // 15-minute rows for the same symbol and dates.
        saveIntraday("TCS", SESSION_DATE, LocalTime.of(9, 15), 3990.0);
        saveIntraday("TCS", SESSION_DATE, LocalTime.of(9, 30), 3995.0);
        saveIntraday("TCS", NEXT_SESSION, LocalTime.of(15, 15), 4110.0);
        // A symbol with ONLY intraday rows.
        saveIntraday("INFY", SESSION_DATE, LocalTime.of(10, 0), 1500.0);
    }

    @Test
    void dailyModeQueriesReturnExactlyWhatTheyReturnedBefore() {
        seedMixedData();

        // findBySymbolAndDate: the daily row, not any of the 15-minute rows.
        OhlcvCandleEntity candle = repository.findBySymbolAndDate("TCS", SESSION_DATE).orElseThrow();
        assertThat(candle.getTimeframe()).isEqualTo("D");
        assertThat(candle.getBarTime()).isEqualTo(LocalTime.MIN);
        assertThat(candle.getClosePrice()).isEqualByComparingTo(BigDecimal.valueOf(4000.0));

        // findLatestBySymbol / findEarliestBySymbol.
        assertThat(repository.findLatestBySymbol("TCS").orElseThrow().getDate()).isEqualTo(NEXT_SESSION);
        assertThat(repository.findEarliestBySymbol("TCS").orElseThrow().getDate()).isEqualTo(SESSION_DATE);

        // Range queries return only daily rows.
        List<OhlcvCandleEntity> range = repository.findBySymbolAndDateRange(
            "TCS", SESSION_DATE, NEXT_SESSION, Pageable.unpaged());
        assertThat(range).hasSize(2);
        assertThat(range).allSatisfy(c -> assertThat(c.getTimeframe()).isEqualTo("D"));

        List<OhlcvCandleEntity> top = repository.findTopBySymbolOrderByDateDesc("TCS", Pageable.ofSize(1));
        assertThat(top).hasSize(1);
        assertThat(top.get(0).getDate()).isEqualTo(NEXT_SESSION);
        assertThat(top.get(0).getTimeframe()).isEqualTo("D");

        List<OhlcvCandleEntity> all = repository.findAllBySymbolOrderByDateDesc("TCS");
        assertThat(all).hasSize(2);
        assertThat(all).allSatisfy(c -> assertThat(c.getTimeframe()).isEqualTo("D"));

        // Existence and count are daily-only.
        assertThat(repository.existsBySymbolAndDate("TCS", SESSION_DATE)).isTrue();
        assertThat(repository.existsBySymbolAndDate("TCS", LocalDate.of(2026, 10, 7))).isFalse();
        assertThat(repository.countBySymbol("TCS")).isEqualTo(2);

        // Positional and cross-symbol queries stay daily.
        // The daily row on SESSION_DATE exists, so "on or after" returns it.
        assertThat(repository.findFirstBySymbolAndDateAfterOrderByDateAsc("TCS", SESSION_DATE)
            .orElseThrow().getDate()).isEqualTo(SESSION_DATE);
        assertThat(repository.findFirstBySymbolAndDateAfterOrderByDateAsc("TCS", LocalDate.of(2026, 10, 7)))
            .isEmpty();
        assertThat(repository.findLatestBySymbolBeforeDate("TCS", NEXT_SESSION)
            .orElseThrow().getDate()).isEqualTo(SESSION_DATE);
        assertThat(repository.findNthBySymbolAndDateAfterOrderByDateAsc("TCS", SESSION_DATE, 0)
            .orElseThrow().getDate()).isEqualTo(NEXT_SESSION);
        assertThat(repository.findLastNBySymbolBeforeDateAsc("TCS", NEXT_SESSION, 5))
            .hasSize(2);
        assertThat(repository.findByDateRange(SESSION_DATE, NEXT_SESSION, Pageable.unpaged()))
            .hasSize(2);
        assertThat(repository.findBySymbolsAndDateRange(List.of("TCS"), SESSION_DATE, NEXT_SESSION,
            Pageable.unpaged())).hasSize(2);

        // Distinct symbols come from daily rows only: INFY has 15-minute rows only.
        assertThat(repository.findAllDistinctSymbols()).containsExactly("TCS");
    }

    @Test
    void fifteenMinuteQueriesReturnIntervalCorrectRows() {
        seedMixedData();

        List<OhlcvCandleEntity> bars = repository.findBySymbolAndIntervalAndDateRange(
            "TCS", Interval.FIFTEEN_MINUTES.getCode(), SESSION_DATE, NEXT_SESSION, Pageable.unpaged());

        assertThat(bars).hasSize(3);
        assertThat(bars).allSatisfy(c -> assertThat(c.getTimeframe()).isEqualTo("15m"));
        // Ascending by date then bar start time.
        assertThat(bars.get(0).getBarTime()).isEqualTo(LocalTime.of(9, 15));
        assertThat(bars.get(1).getBarTime()).isEqualTo(LocalTime.of(9, 30));
        assertThat(bars.get(2).getBarTime()).isEqualTo(LocalTime.of(15, 15));
        assertThat(bars.get(2).getClosePrice()).isEqualByComparingTo(BigDecimal.valueOf(4110.0));

        OhlcvCandleEntity latest = repository.findLatestBySymbolAndInterval(
            "TCS", Interval.FIFTEEN_MINUTES.getCode()).orElseThrow();
        assertThat(latest.getDate()).isEqualTo(NEXT_SESSION);
        assertThat(latest.getBarTime()).isEqualTo(LocalTime.of(15, 15));

        assertThat(repository.findBySymbolAndIntervalAndDateAndBarTime(
            "TCS", Interval.FIFTEEN_MINUTES.getCode(), SESSION_DATE, LocalTime.of(9, 30)))
            .isPresent();
        assertThat(repository.countBySymbolAndInterval("TCS", Interval.FIFTEEN_MINUTES.getCode()))
            .isEqualTo(3);
        assertThat(repository.countBySymbolAndInterval("TCS", Interval.DAILY.getCode())).isEqualTo(2);
    }

    @Test
    void entityRoundTripsIntervalDimension() {
        seedMixedData();

        OhlcvCandleEntity daily = repository.findBySymbolAndDate("TCS", SESSION_DATE).orElseThrow();
        CandleData dailyDto = daily.toCandleData();
        assertThat(dailyDto.interval()).isEqualTo(Interval.DAILY);
        assertThat(dailyDto.barTime()).isEqualTo(LocalTime.MIN);
        assertThat(dailyDto.close()).isEqualByComparingTo(BigDecimal.valueOf(4000.0));

        OhlcvCandleEntity bar = repository.findBySymbolAndIntervalAndDateAndBarTime(
            "TCS", Interval.FIFTEEN_MINUTES.getCode(), SESSION_DATE, LocalTime.of(9, 30)).orElseThrow();
        CandleData barDto = bar.toCandleData();
        assertThat(barDto.interval()).isEqualTo(Interval.FIFTEEN_MINUTES);
        assertThat(barDto.barTime()).isEqualTo(LocalTime.of(9, 30));
        assertThat(barDto.close()).isEqualByComparingTo(BigDecimal.valueOf(3995.0));
    }

    @Test
    void dailyIdentityIsUnchangedForTheSameSymbolAndDate() {
        saveDaily("TCS", SESSION_DATE, 4000.0);
        // The same symbol and date may exist as a 15-minute bar ...
        saveIntraday("TCS", SESSION_DATE, LocalTime.of(9, 15), 3990.0);
        // ... but a second daily row for that symbol and date violates the identity.
        assertThatThrownBy(() -> saveDaily("TCS", SESSION_DATE, 4001.0))
            .isInstanceOf(DataIntegrityViolationException.class);
    }

    @Test
    void intradayIdentityIncludesIntervalDateAndBarTime() {
        saveIntraday("TCS", SESSION_DATE, LocalTime.of(9, 15), 3990.0);
        // Same symbol/date/bar time under the daily interval is a different identity.
        saveDaily("TCS", SESSION_DATE, 4000.0);
        // A different bar start time is a new identity.
        saveIntraday("TCS", SESSION_DATE, LocalTime.of(9, 30), 3995.0);
        assertThat(repository.countBySymbolAndInterval("TCS", Interval.FIFTEEN_MINUTES.getCode()))
            .isEqualTo(2);
        // A duplicate 15-minute bar violates the identity even with different values.
        // Kept last: a constraint violation leaves the persistence context with a
        // null-identifier entry that fails the next flush.
        assertThatThrownBy(() -> saveIntraday("TCS", SESSION_DATE, LocalTime.of(9, 15), 3999.0))
            .isInstanceOf(DataIntegrityViolationException.class);
    }

    @Test
    void dailyIntervalCodeMatchesTheEnum() {
        // The repository's compile-time query constant and the enum must not drift.
        assertThat(OhlcvCandleRepository.DAILY_INTERVAL).isEqualTo(Interval.DAILY.getCode());
    }
}
