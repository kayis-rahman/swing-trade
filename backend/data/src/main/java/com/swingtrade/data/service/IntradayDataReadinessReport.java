package com.swingtrade.data.service;

import java.time.DayOfWeek;
import java.time.LocalDate;
import java.time.LocalTime;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

/** Deterministic per-session completeness and timestamp checks for Yahoo readiness runs. */
public final class IntradayDataReadinessReport {
    private IntradayDataReadinessReport() {}

    public record LedgerRow(String symbol, Interval interval, LocalDate sessionDate,
                            int requestedBars, int returnedBars, int validSessionBars,
                            int missingBars, int invalidTimestampBars, double missingRate,
                            String providerStatus, long elapsedMillis) {}

    public static List<LedgerRow> analyze(String symbol, Interval interval,
                                          List<LocalDate> expectedSessions,
                                          List<CandleData> candles, long elapsedMillis,
                                          String providerStatus) {
        int barsPerSession = interval == Interval.FIFTEEN_MINUTES
            ? NseSessionCalendar.NORMAL_SESSION_BAR_COUNT : 1;
        Map<LocalDate, List<CandleData>> byDate = candles.stream()
            .filter(candle -> candle.interval() == interval)
            .collect(Collectors.groupingBy(CandleData::date));
        List<LedgerRow> rows = new ArrayList<>();
        for (LocalDate date : expectedSessions) {
            List<CandleData> returned = byDate.getOrDefault(date, List.of());
            Set<LocalTime> expectedTimes = interval == Interval.FIFTEEN_MINUTES
                ? Set.copyOf(expectedIntradayTimes()) : Set.of(LocalTime.MIDNIGHT);
            long valid = returned.stream().filter(candle -> expectedTimes.contains(candle.barTime())).count();
            int invalid = returned.size() - (int) valid;
            int missing = Math.max(0, barsPerSession - (int) valid);
            rows.add(new LedgerRow(symbol, interval, date, barsPerSession, returned.size(), (int) valid,
                missing, invalid, (double) missing / barsPerSession, providerStatus, elapsedMillis));
        }
        return List.copyOf(rows);
    }

    public static List<LocalDate> weekdaySessions(LocalDate from, LocalDate to) {
        if (from == null || to == null || to.isBefore(from)) {
            throw new IllegalArgumentException("A valid inclusive date range is required");
        }
        List<LocalDate> sessions = new ArrayList<>();
        for (LocalDate date = from; !date.isAfter(to); date = date.plusDays(1)) {
            if (date.getDayOfWeek() != DayOfWeek.SATURDAY && date.getDayOfWeek() != DayOfWeek.SUNDAY) {
                sessions.add(date);
            }
        }
        return List.copyOf(sessions);
    }

    private static List<LocalTime> expectedIntradayTimes() {
        List<LocalTime> times = new ArrayList<>(NseSessionCalendar.NORMAL_SESSION_BAR_COUNT);
        LocalTime time = NseSessionCalendar.NORMAL_SESSION_START;
        for (int i = 0; i < NseSessionCalendar.NORMAL_SESSION_BAR_COUNT; i++) {
            times.add(time);
            time = time.plusMinutes(NseSessionCalendar.BAR_MINUTES);
        }
        return times;
    }
}
