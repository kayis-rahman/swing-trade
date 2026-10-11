package com.swingtrade.data.service;

import org.junit.jupiter.api.Test;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalTime;
import java.util.ArrayList;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;

class IntradayDataReadinessReportTest {
    private static final LocalDate SESSION = LocalDate.of(2026, 10, 5);

    @Test
    void reportsRequestedReturnedMissingAndOutOfSessionTimestamps() {
        List<CandleData> candles = new ArrayList<>();
        for (int i = 0; i < 24; i++) {
            LocalTime time = NseSessionCalendar.NORMAL_SESSION_START.plusMinutes((long) i * 15);
            candles.add(bar(SESSION, time));
        }
        candles.add(bar(SESSION, LocalTime.of(15, 30)));

        List<IntradayDataReadinessReport.LedgerRow> rows = IntradayDataReadinessReport.analyze(
            "TCS", Interval.FIFTEEN_MINUTES, List.of(SESSION), candles, 45, "AVAILABLE");

        assertThat(rows).containsExactly(new IntradayDataReadinessReport.LedgerRow(
            "TCS", Interval.FIFTEEN_MINUTES, SESSION, 25, 25, 24, 1, 1, 0.04, "AVAILABLE", 45));
    }

    @Test
    void weekdaySessionSelectionExcludesWeekendsAndRejectsInvalidRanges() {
        assertThat(IntradayDataReadinessReport.weekdaySessions(
            LocalDate.of(2026, 10, 2), LocalDate.of(2026, 10, 5)))
            .containsExactly(LocalDate.of(2026, 10, 2), LocalDate.of(2026, 10, 5));
        assertThat(org.assertj.core.api.Assertions.catchThrowable(() ->
            IntradayDataReadinessReport.weekdaySessions(SESSION.plusDays(1), SESSION)))
            .isInstanceOf(IllegalArgumentException.class);
    }

    private static CandleData bar(LocalDate date, LocalTime time) {
        return CandleData.of("TCS", date, time, BigDecimal.TEN, BigDecimal.TEN,
            BigDecimal.TEN, BigDecimal.TEN, 1_000L, BigDecimal.TEN, Interval.FIFTEEN_MINUTES);
    }
}
