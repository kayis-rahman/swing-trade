package com.swingtrade.data.service;

import org.springframework.stereotype.Component;

import java.time.LocalDate;
import java.time.LocalTime;
import java.util.ArrayList;
import java.util.List;

/**
 * NSE session-window model for the intraday data plane.
 *
 * <p>Models the exchange session structure (NSE market timings, updated 04-Aug-2025):
 * the pre-open window, the normal trading session, the closing auction, and the
 * intraday square-off cutoff. Whether a date is a trading session at all reuses
 * {@link MarketCalendar} — weekends are closed, FULL holidays are closed, and
 * exchange-declared PARTIAL days (Muhurat trading) are sessions.
 *
 * <p>Muhurat sessions are conducted on Diwali Laxmi Pujan with timings notified
 * separately by the exchange; this calendar treats a Muhurat day as a trading
 * session but does not assume its clock follows the normal 09:15–15:30 grid.
 */
@Component
public class NseSessionCalendar {

    /** Regular pre-open session (order entry for the opening auction). */
    public static final LocalTime PRE_OPEN_START = LocalTime.of(9, 0);
    public static final LocalTime PRE_OPEN_END = LocalTime.of(9, 8);

    /** Normal / odd-lot trading session. */
    public static final LocalTime NORMAL_SESSION_START = LocalTime.of(9, 15);
    public static final LocalTime NORMAL_SESSION_END = LocalTime.of(15, 30);

    /** Closing session / closing auction (CAS). */
    public static final LocalTime CLOSING_AUCTION_START = LocalTime.of(15, 40);
    public static final LocalTime CLOSING_AUCTION_END = LocalTime.of(16, 0);

    /**
     * Intraday square-off cutoff: intraday positions must be flattened by this
     * time — no overnight carry (stage-0 MVP rule from the intraday expansion plan).
     */
    public static final LocalTime SQUARE_OFF_CUTOFF = LocalTime.of(15, 20);

    /** Fifteen-minute bars per normal session: 09:15 through 15:15 inclusive. */
    public static final int NORMAL_SESSION_BAR_COUNT = 25;
    public static final int BAR_MINUTES = 15;

    /** The trading session phases an intraday system must distinguish. */
    public enum SessionPhase {
        PRE_OPEN,
        NORMAL,
        CLOSING_AUCTION,
        CLOSED
    }

    private final MarketCalendar marketCalendar;

    public NseSessionCalendar(MarketCalendar marketCalendar) {
        this.marketCalendar = marketCalendar;
    }

    /**
     * Whether the date is an NSE trading session at all (weekend/holiday check),
     * reusing the exchange holiday calendar. PARTIAL (Muhurat) days are sessions.
     */
    public boolean isTradingSession(LocalDate date) {
        return marketCalendar.isNseTradingSession(date);
    }

    /**
     * The session phase at a given date and time (IST). Non-trading dates and the
     * gaps between windows (e.g. 09:08–09:15 order matching, 15:30–15:40) are
     * {@link SessionPhase#CLOSED}.
     */
    public SessionPhase phaseAt(LocalDate date, LocalTime time) {
        if (!isTradingSession(date) || time == null) {
            return SessionPhase.CLOSED;
        }
        if (!time.isBefore(PRE_OPEN_START) && time.isBefore(PRE_OPEN_END)) {
            return SessionPhase.PRE_OPEN;
        }
        if (!time.isBefore(NORMAL_SESSION_START) && time.isBefore(NORMAL_SESSION_END)) {
            return SessionPhase.NORMAL;
        }
        if (!time.isBefore(CLOSING_AUCTION_START) && time.isBefore(CLOSING_AUCTION_END)) {
            return SessionPhase.CLOSING_AUCTION;
        }
        return SessionPhase.CLOSED;
    }

    /** Whether the given date and time (IST) fall inside the normal 09:15–15:30 session. */
    public boolean isWithinNormalSession(LocalDate date, LocalTime time) {
        return phaseAt(date, time) == SessionPhase.NORMAL;
    }

    /** Whether the given date and time (IST) fall inside the pre-open window. */
    public boolean isWithinPreOpen(LocalDate date, LocalTime time) {
        return phaseAt(date, time) == SessionPhase.PRE_OPEN;
    }

    /** Whether the given date and time (IST) fall inside the closing auction. */
    public boolean isWithinClosingAuction(LocalDate date, LocalTime time) {
        return phaseAt(date, time) == SessionPhase.CLOSING_AUCTION;
    }

    /**
     * Whether a bar start time is a valid normal-session bar start: 09:15 through
     * 15:15 inclusive (25 fifteen-minute bars). Used to filter provider bars that
     * fall outside the session (pre-open, closing auction).
     */
    public boolean isNormalSessionBarTime(LocalTime barTime) {
        return barTime != null
            && !barTime.isBefore(NORMAL_SESSION_START)
            && barTime.isBefore(NORMAL_SESSION_END);
    }

    /**
     * Whether the intraday square-off cutoff (15:20 IST) has been reached on a
     * trading session — the point after which no new intraday positions may be taken.
     */
    public boolean isSquareOffCutoffReached(LocalDate date, LocalTime time) {
        return isTradingSession(date) && time != null && !time.isBefore(SQUARE_OFF_CUTOFF);
    }

    /**
     * The expected 15-minute bar start times for a trading session, in ascending
     * order (09:15, 09:30, …, 15:15). Empty for non-trading dates.
     */
    public List<LocalTime> expectedBarStartTimes(LocalDate date) {
        List<LocalTime> barTimes = new ArrayList<>(NORMAL_SESSION_BAR_COUNT);
        if (!isTradingSession(date)) {
            return barTimes;
        }
        LocalTime barTime = NORMAL_SESSION_START;
        for (int i = 0; i < NORMAL_SESSION_BAR_COUNT; i++) {
            barTimes.add(barTime);
            barTime = barTime.plusMinutes(BAR_MINUTES);
        }
        return barTimes;
    }
}
