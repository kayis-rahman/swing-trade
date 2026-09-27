package com.swingtrade.api.controller;

import com.swingtrade.data.entity.WatchlistEntity;
import com.swingtrade.data.service.DataIngestionService;
import com.swingtrade.data.service.WatchlistService;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.http.ResponseEntity;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyBoolean;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class WatchlistControllerTest {

    @Mock
    private WatchlistService watchlistService;

    @Mock
    private DataIngestionService dataIngestionService;

    private WatchlistController controller;

    @BeforeEach
    void setUp() {
        controller = new WatchlistController(watchlistService, dataIngestionService);
    }

    @Test
    void toggleWatchlist_withActivateTrue_passesTrueThroughToService() {
        WatchlistEntity entity = new WatchlistEntity();
        when(watchlistService.toggleActive(eq("RELIANCE"), anyBoolean())).thenReturn(entity);

        controller.toggleWatchlist("RELIANCE", true);

        ArgumentCaptor<Boolean> captor = ArgumentCaptor.forClass(Boolean.class);
        verify(watchlistService).toggleActive(eq("RELIANCE"), captor.capture());
        assertThat(captor.getValue())
            .as("activate=true must result in the symbol being ACTIVATED, not deactivated")
            .isTrue();
    }

    @Test
    void toggleWatchlist_withActivateFalse_passesFalseThroughToService() {
        WatchlistEntity entity = new WatchlistEntity();
        when(watchlistService.toggleActive(eq("RELIANCE"), anyBoolean())).thenReturn(entity);

        controller.toggleWatchlist("RELIANCE", false);

        ArgumentCaptor<Boolean> captor = ArgumentCaptor.forClass(Boolean.class);
        verify(watchlistService).toggleActive(eq("RELIANCE"), captor.capture());
        assertThat(captor.getValue())
            .as("activate=false must result in the symbol being DEACTIVATED, not activated")
            .isFalse();
    }

    @Test
    void toggleWatchlist_whenSymbolNotFound_returns404() {
        when(watchlistService.toggleActive(any(), anyBoolean())).thenReturn(null);

        ResponseEntity<?> response = controller.toggleWatchlist("UNKNOWN", true);

        assertThat(response.getStatusCode().value()).isEqualTo(404);
    }

    @Test
    void addToWatchlist_rejectsSymbolLongerThanDatabaseColumnBeforePersistence() {
        ResponseEntity<?> response = controller.addToWatchlist("CODQA90508093", "", "NSE");

        assertThat(response.getStatusCode().value()).isEqualTo(400);
        verify(watchlistService, never()).getBySymbol(any());
        verify(watchlistService, never()).addToWatchlist(any(), any(), any());
    }

    @Test
    void addToWatchlist_normalizesBeforeCheckingForDuplicates() {
        WatchlistEntity existing = new WatchlistEntity();
        when(watchlistService.getBySymbol("RELIANCE")).thenReturn(java.util.Optional.of(existing));

        ResponseEntity<?> response = controller.addToWatchlist(" reliance ", "", "NSE");

        assertThat(response.getStatusCode().value()).isEqualTo(400);
        verify(watchlistService).getBySymbol("RELIANCE");
        verify(watchlistService, never()).addToWatchlist(any(), any(), any());
    }

    @Test
    void addToWatchlist_reactivatesExistingInactiveSymbol() {
        WatchlistEntity existing = new WatchlistEntity();
        existing.setIsActive(false);
        when(watchlistService.getBySymbol("RELIANCE")).thenReturn(java.util.Optional.of(existing));
        when(watchlistService.addToWatchlist("RELIANCE", "", "NSE")).thenReturn(existing);

        ResponseEntity<?> response = controller.addToWatchlist("RELIANCE", "", "NSE");

        assertThat(response.getStatusCode().value()).isEqualTo(200);
        verify(watchlistService).addToWatchlist("RELIANCE", "", "NSE");
    }
}
