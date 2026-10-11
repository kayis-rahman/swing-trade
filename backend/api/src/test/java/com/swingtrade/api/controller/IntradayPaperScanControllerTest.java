package com.swingtrade.api.controller;

import com.swingtrade.api.dto.IntradayPaperScanRequest;
import com.swingtrade.broker.service.intraday.IntradayPaperScanResult;
import com.swingtrade.broker.service.intraday.IntradayPaperScanService;
import com.swingtrade.strategy.IntradayBacktestConfig;
import org.junit.jupiter.api.Test;
import org.springframework.http.HttpStatus;

import java.time.LocalDate;
import java.time.LocalTime;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class IntradayPaperScanControllerTest {
    @Test
    void returnsEphemeralScanAndAppliesDefaultRiskConfig() {
        IntradayPaperScanService service = mock(IntradayPaperScanService.class);
        LocalDate date = LocalDate.of(2026, 10, 9);
        LocalTime asOf = LocalTime.of(10, 0);
        IntradayPaperScanResult expected = new IntradayPaperScanResult("PAPER_SCAN_AS_OF", date, asOf,
            "ACTIVE_NSE_WATCHLIST", 3, 2, 1, 0, List.of("BSECO"), List.of());
        when(service.scan(date, asOf, IntradayBacktestConfig.defaults())).thenReturn(expected);
        IntradayPaperScanController controller = new IntradayPaperScanController(service);

        var response = controller.scan(new IntradayPaperScanRequest(date, asOf, null));

        assertThat(response.getStatusCode()).isEqualTo(HttpStatus.OK);
        assertThat(response.getBody()).isSameAs(expected);
        verify(service).scan(date, asOf, IntradayBacktestConfig.defaults());
    }

    @Test
    void rejectsOversizedWatchlistWithoutLeakingImplementationDetails() {
        IntradayPaperScanService service = mock(IntradayPaperScanService.class);
        when(service.scan(org.mockito.ArgumentMatchers.any(), org.mockito.ArgumentMatchers.any(),
            org.mockito.ArgumentMatchers.any())).thenThrow(new IllegalStateException("too many symbols"));
        IntradayPaperScanController controller = new IntradayPaperScanController(service);

        var response = controller.scan(new IntradayPaperScanRequest(
            LocalDate.of(2026, 10, 9), LocalTime.of(10, 0), null));

        assertThat(response.getStatusCode()).isEqualTo(HttpStatus.UNPROCESSABLE_ENTITY);
        assertThat(response.getBody()).isNull();
    }
}
