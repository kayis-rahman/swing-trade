package com.swingtrade.data.service;

import com.swingtrade.data.entity.NseCalendarCoverageEntity;
import com.swingtrade.data.entity.NseHolidayEntity;
import com.swingtrade.data.repository.NseCalendarCoverageRepository;
import org.junit.jupiter.api.Test;

import java.time.LocalDate;
import java.time.LocalTime;
import java.util.List;
import java.util.Optional;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

/**
 * Session-model tests pinned to real exchange dates: NSE-published FULL holidays,
 * exchange-declared PARTIAL (Muhurat) sessions — including Muhurat days that fall
 * on a weekend — and the normal 09:15–15:30 session grid.
 */
class NseSessionCalendarTest {

    // Real NSE FULL holidays (2026 equity calendar, V1/V62 seeds).
    private static final LocalDate REPUBLIC_DAY = LocalDate.of(2026, 1, 26);      // Monday
    private static final LocalDate GANDHI_JAYANTI = LocalDate.of(2026, 10, 2);    // Friday
    private static final LocalDate DIWALI_FULL = LocalDate.of(2026, 10, 20);      // Tuesday
    private static final LocalDate CHRISTMAS = LocalDate.of(2026, 12, 25);       // Friday

    // Real NSE Muhurat (PARTIAL) sessions — Diwali Laxmi Pujan special trading.
    private static final LocalDate MUHRAT_2026 = LocalDate.of(2026, 11, 8);      // Sunday
    private static final LocalDate MUHRAT_2025 = LocalDate.of(2025, 10, 21);     // Tuesday
    private static final LocalDate MUHRAT_2024 = LocalDate.of(2024, 11, 1);      // Friday
    private static final LocalDate MUHRAT_2023 = LocalDate.of(2023, 11, 12);     // Sunday
    private static final LocalDate MUHRAT_2022 = LocalDate.of(2022, 10, 24);     // Monday
    private static final LocalDate MUHRAT_2021 = LocalDate.of(2021, 11, 4);      // Thursday
    private static final LocalDate MUHRAT_2020 = LocalDate.of(2020, 11, 14);     // Saturday

    private static final LocalDate NORMAL_MONDAY = LocalDate.of(2026, 10, 5);

    private final NseHolidayService holidays = mock(NseHolidayService.class);
    private final NseCalendarCoverageRepository coverage = mock(NseCalendarCoverageRepository.class);
    private final NseSessionCalendar calendar =
        new NseSessionCalendar(new MarketCalendar(holidays, coverage));

    private void seedFullHoliday(LocalDate date) {
        when(holidays.getHoliday(date)).thenReturn(Optional.of(
            new NseHolidayEntity(date, "Full holiday", "FULL")));
    }

    private void seedMuhurat(LocalDate date) {
        when(holidays.getHoliday(date)).thenReturn(Optional.of(
            new NseHolidayEntity(date, "Diwali Laxmi Pujan / Muhurat Trading", "PARTIAL")));
    }

    @Test
    void weekendsAreClosedAllDay() {
        LocalDate saturday = LocalDate.of(2026, 10, 3);
        LocalDate sunday = LocalDate.of(2026, 10, 4);

        assertThat(calendar.isTradingSession(saturday)).isFalse();
        assertThat(calendar.isTradingSession(sunday)).isFalse();
        assertThat(calendar.phaseAt(saturday, LocalTime.of(10, 30)))
            .isEqualTo(NseSessionCalendar.SessionPhase.CLOSED);
        assertThat(calendar.expectedBarStartTimes(saturday)).isEmpty();
    }

    @Test
    void fullHolidaysAreClosedAllDay() {
        seedFullHoliday(REPUBLIC_DAY);
        seedFullHoliday(GANDHI_JAYANTI);
        seedFullHoliday(DIWALI_FULL);
        seedFullHoliday(CHRISTMAS);

        for (LocalDate date : List.of(REPUBLIC_DAY, GANDHI_JAYANTI, DIWALI_FULL, CHRISTMAS)) {
            assertThat(calendar.isTradingSession(date)).isFalse();
            assertThat(calendar.phaseAt(date, LocalTime.of(10, 30)))
                .isEqualTo(NseSessionCalendar.SessionPhase.CLOSED);
            assertThat(calendar.expectedBarStartTimes(date)).isEmpty();
            assertThat(calendar.isSquareOffCutoffReached(date, LocalTime.of(15, 25))).isFalse();
        }
    }

    @Test
    void muhuratPartialSessionsAreTradingSessionsIncludingOnWeekends() {
        seedMuhurat(MUHRAT_2026);
        seedMuhurat(MUHRAT_2025);
        seedMuhurat(MUHRAT_2024);
        seedMuhurat(MUHRAT_2023);
        seedMuhurat(MUHRAT_2022);
        seedMuhurat(MUHRAT_2021);
        seedMuhurat(MUHRAT_2020);

        // Every real Muhurat date — including the Sunday (2026-11-08, 2023-11-12)
        // and Saturday (2020-11-14) sessions — is a trading session.
        for (LocalDate muhurat : List.of(MUHRAT_2026, MUHRAT_2025, MUHRAT_2024, MUHRAT_2023,
                MUHRAT_2022, MUHRAT_2021, MUHRAT_2020)) {
            assertThat(calendar.isTradingSession(muhurat))
                .as("Muhurat session %s must be a trading session", muhurat)
                .isTrue();
            assertThat(calendar.phaseAt(muhurat, LocalTime.of(10, 30)))
                .isEqualTo(NseSessionCalendar.SessionPhase.NORMAL);
        }
    }

    @Test
    void normalDayPhaseBoundariesFollowTheExchangeSessionWindows() {
        assertThat(calendar.phaseAt(NORMAL_MONDAY, LocalTime.of(8, 59)))
            .isEqualTo(NseSessionCalendar.SessionPhase.CLOSED);
        assertThat(calendar.phaseAt(NORMAL_MONDAY, LocalTime.of(9, 0)))
            .isEqualTo(NseSessionCalendar.SessionPhase.PRE_OPEN);
        assertThat(calendar.phaseAt(NORMAL_MONDAY, LocalTime.of(9, 7)))
            .isEqualTo(NseSessionCalendar.SessionPhase.PRE_OPEN);
        // 09:08–09:15 is the opening-auction gap, not a trading phase.
        assertThat(calendar.phaseAt(NORMAL_MONDAY, LocalTime.of(9, 8)))
            .isEqualTo(NseSessionCalendar.SessionPhase.CLOSED);
        assertThat(calendar.phaseAt(NORMAL_MONDAY, LocalTime.of(9, 14)))
            .isEqualTo(NseSessionCalendar.SessionPhase.CLOSED);
        assertThat(calendar.phaseAt(NORMAL_MONDAY, LocalTime.of(9, 15)))
            .isEqualTo(NseSessionCalendar.SessionPhase.NORMAL);
        assertThat(calendar.phaseAt(NORMAL_MONDAY, LocalTime.of(12, 0)))
            .isEqualTo(NseSessionCalendar.SessionPhase.NORMAL);
        assertThat(calendar.phaseAt(NORMAL_MONDAY, LocalTime.of(15, 29)))
            .isEqualTo(NseSessionCalendar.SessionPhase.NORMAL);
        assertThat(calendar.phaseAt(NORMAL_MONDAY, LocalTime.of(15, 30)))
            .isEqualTo(NseSessionCalendar.SessionPhase.CLOSED);
        // 15:30–15:40 is the session break before the closing auction.
        assertThat(calendar.phaseAt(NORMAL_MONDAY, LocalTime.of(15, 39)))
            .isEqualTo(NseSessionCalendar.SessionPhase.CLOSED);
        assertThat(calendar.phaseAt(NORMAL_MONDAY, LocalTime.of(15, 40)))
            .isEqualTo(NseSessionCalendar.SessionPhase.CLOSING_AUCTION);
        assertThat(calendar.phaseAt(NORMAL_MONDAY, LocalTime.of(15, 59)))
            .isEqualTo(NseSessionCalendar.SessionPhase.CLOSING_AUCTION);
        assertThat(calendar.phaseAt(NORMAL_MONDAY, LocalTime.of(16, 0)))
            .isEqualTo(NseSessionCalendar.SessionPhase.CLOSED);
    }

    @Test
    void squareOffCutoffIsFifteenTwentyIstOnTradingSessionsOnly() {
        assertThat(calendar.isSquareOffCutoffReached(NORMAL_MONDAY, LocalTime.of(15, 19))).isFalse();
        assertThat(calendar.isSquareOffCutoffReached(NORMAL_MONDAY, LocalTime.of(15, 20))).isTrue();
        assertThat(calendar.isSquareOffCutoffReached(NORMAL_MONDAY, LocalTime.of(15, 25))).isTrue();

        seedFullHoliday(DIWALI_FULL);
        assertThat(calendar.isSquareOffCutoffReached(DIWALI_FULL, LocalTime.of(15, 25))).isFalse();
    }

    @Test
    void normalSessionBarGridIsTwentyFiveFifteenMinuteBars() {
        List<LocalTime> barTimes = calendar.expectedBarStartTimes(NORMAL_MONDAY);

        assertThat(barTimes).hasSize(25);
        assertThat(barTimes.getFirst()).isEqualTo(LocalTime.of(9, 15));
        assertThat(barTimes.getLast()).isEqualTo(LocalTime.of(15, 15));
        assertThat(barTimes.get(1)).isEqualTo(LocalTime.of(9, 30));

        assertThat(calendar.isNormalSessionBarTime(LocalTime.of(9, 15))).isTrue();
        assertThat(calendar.isNormalSessionBarTime(LocalTime.of(15, 15))).isTrue();
        assertThat(calendar.isNormalSessionBarTime(LocalTime.of(9, 14))).isFalse();
        assertThat(calendar.isNormalSessionBarTime(LocalTime.of(15, 30))).isFalse();
        assertThat(calendar.isNormalSessionBarTime(LocalTime.of(9, 0))).isFalse();
    }

    @Test
    void barGridIsEmptyOnHolidaysAndWeekends() {
        seedFullHoliday(GANDHI_JAYANTI);
        assertThat(calendar.expectedBarStartTimes(GANDHI_JAYANTI)).isEmpty();
        assertThat(calendar.expectedBarStartTimes(LocalDate.of(2026, 10, 3))).isEmpty();
    }
}
