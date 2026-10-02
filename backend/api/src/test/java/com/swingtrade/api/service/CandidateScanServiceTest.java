package com.swingtrade.api.service;

import com.swingtrade.data.entity.CandidateScanResultEntity;
import com.swingtrade.data.entity.CandidateScanRunEntity;
import com.swingtrade.data.entity.FyersSymbolEntity;
import com.swingtrade.data.repository.CandidateScanResultRepository;
import com.swingtrade.data.repository.CandidateScanRunRepository;
import com.swingtrade.data.repository.FyersSymbolRepository;
import com.swingtrade.data.service.AppSettingsService;
import com.swingtrade.data.service.DataIngestionService;
import com.swingtrade.domain.store.CandleStore;
import com.swingtrade.strategy.BacktestEngine;
import com.swingtrade.strategy.PriceActionSignalEngine;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.springframework.data.domain.PageImpl;
import org.springframework.data.domain.Pageable;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class CandidateScanServiceTest {

    private CandidateScanRunRepository runRepository;
    private CandidateScanResultRepository resultRepository;
    private FyersSymbolRepository symbolRepository;
    private AppSettingsService settingsService;
    private CandidateScanService service;

    @BeforeEach
    void setUp() {
        runRepository = mock(CandidateScanRunRepository.class);
        resultRepository = mock(CandidateScanResultRepository.class);
        symbolRepository = mock(FyersSymbolRepository.class);
        settingsService = mock(AppSettingsService.class);
        service = new CandidateScanService(
            symbolRepository,
            runRepository,
            resultRepository,
            mock(DataIngestionService.class),
            settingsService,
            mock(CandleStore.class),
            mock(PriceActionSignalEngine.class),
            mock(BacktestEngine.class),
            3,
            0,
            3);
    }

    @AfterEach
    void tearDown() {
        service.shutdown();
    }

    private static CandidateScanRunEntity run(String status) {
        CandidateScanRunEntity run = new CandidateScanRunEntity();
        run.setRunId(UUID.randomUUID());
        run.setStatus(status);
        run.setStartedAt(LocalDateTime.now());
        return run;
    }

    @Nested
    class Recovery {

        @Test
        void marksRunningScansCancelledAfterRestart() {
            CandidateScanRunEntity run = run("RUNNING");
            when(runRepository.findByStatus("RUNNING")).thenReturn(List.of(run));
            when(runRepository.findByStatus("PAUSED")).thenReturn(List.of());

            service.recoverInterruptedRuns();

            assertThat(run.getStatus()).isEqualTo("CANCELLED");
            assertThat(run.getCompletedAt()).isNotNull();
            assertThat(run.getErrorMessage()).isEqualTo("Scan interrupted by API restart.");
            verify(runRepository).save(run);
        }

        @Test
        void leavesRepositoryUntouchedWhenNothingWasInterrupted() {
            when(runRepository.findByStatus("RUNNING")).thenReturn(List.of());
            when(runRepository.findByStatus("PAUSED")).thenReturn(List.of());

            service.recoverInterruptedRuns();

            verify(runRepository, never()).save(any());
        }

        @Test
        void alsoCancelsPausedScansAfterRestart() {
            CandidateScanRunEntity run = run("PAUSED");
            when(runRepository.findByStatus("RUNNING")).thenReturn(List.of());
            when(runRepository.findByStatus("PAUSED")).thenReturn(List.of(run));

            service.recoverInterruptedRuns();

            assertThat(run.getStatus()).isEqualTo("CANCELLED");
            assertThat(run.getErrorMessage()).isEqualTo("Scan interrupted by API restart.");
            verify(runRepository).save(run);
        }
    }

    @Nested
    class Lifecycle {

        @Test
        void manualScansUseTheNseSymbolMasterInsteadOfTheActiveWatchlist() {
            FyersSymbolEntity first = new FyersSymbolEntity();
            first.setTradingSymbol("INFY");
            FyersSymbolEntity second = new FyersSymbolEntity();
            second.setTradingSymbol("TCS");
            when(runRepository.existsByStatus("RUNNING")).thenReturn(false);
            when(symbolRepository.findByExchangeIgnoreCaseOrderByTradingSymbolAsc("NSE"))
                .thenReturn(List.of(first, second));

            CandidateScanRunEntity scan = service.start();

            assertThat(scan.getScanScope()).isEqualTo("NSE_BROAD");
            assertThat(scan.getTotalSymbols()).isEqualTo(2);
            verify(symbolRepository).findByExchangeIgnoreCaseOrderByTradingSymbolAsc("NSE");
        }

        @Test
        void pausesAndResumesRunningScan() {
            CandidateScanRunEntity run = run("RUNNING");
            when(runRepository.findByRunId(run.getRunId())).thenReturn(Optional.of(run));

            assertThat(service.pause(run.getRunId())).isTrue();
            assertThat(run.getStatus()).isEqualTo("PAUSED");

            assertThat(service.resume(run.getRunId())).isTrue();
            assertThat(run.getStatus()).isEqualTo("RUNNING");
            verify(runRepository, org.mockito.Mockito.times(2)).save(run);
        }

        @Test
        void rejectsTransitionsFromTerminalScan() {
            CandidateScanRunEntity run = run("COMPLETED");
            when(runRepository.findByRunId(run.getRunId())).thenReturn(Optional.of(run));

            assertThat(service.pause(run.getRunId())).isFalse();
            assertThat(service.resume(run.getRunId())).isFalse();
            assertThat(service.cancel(run.getRunId())).isFalse();
            verify(runRepository, never()).save(any());
        }

        @Test
        void cancellationIsAllowedWhilePaused() {
            CandidateScanRunEntity run = run("PAUSED");
            when(runRepository.findByRunId(run.getRunId())).thenReturn(Optional.of(run));

            assertThat(service.cancel(run.getRunId())).isTrue();

            assertThat(run.getStatus()).isEqualTo("CANCELLED");
            assertThat(run.getCompletedAt()).isNotNull();
            verify(runRepository).save(run);
        }
    }

    @Nested
    class Results {

        @Test
        void appliesNormalizedFiltersAndBoundedPagination() {
            UUID runId = UUID.randomUUID();
            CandidateScanResultEntity result = new CandidateScanResultEntity();
            result.setRunId(runId);
            result.setSymbol("RELIANCE");
            result.setDataStatus("READY");
            when(resultRepository.search(eq(runId), eq("rel"), eq("BUY"), any(Pageable.class)))
                .thenAnswer(invocation -> {
                    Pageable pageable = invocation.getArgument(3);
                    assertThat(pageable.getPageNumber()).isEqualTo(2);
                    assertThat(pageable.getPageSize()).isEqualTo(10);
                    return new PageImpl<>(List.of(result), pageable, 21);
                });

            CandidateScanService.ResultPage page =
                service.getResultsPage(runId, 20, 10, "  rel  ", " buy ");

            assertThat(page.items()).containsExactly(result);
            assertThat(page.total()).isEqualTo(21);
            assertThat(page.offset()).isEqualTo(20);
            assertThat(page.limit()).isEqualTo(10);
        }

        @Test
        void clampsInvalidOffsetAndOversizedLimit() {
            UUID runId = UUID.randomUUID();
            when(resultRepository.search(eq(runId), eq(""), eq(""), any(Pageable.class)))
                .thenAnswer(invocation -> {
                    Pageable pageable = invocation.getArgument(3);
                    assertThat(pageable.getPageNumber()).isZero();
                    assertThat(pageable.getPageSize()).isEqualTo(100);
                    return new PageImpl<>(List.of(), pageable, 0);
                });

            CandidateScanService.ResultPage page =
                service.getResultsPage(runId, -5, 500, null, null);

            assertThat(page.offset()).isZero();
            assertThat(page.limit()).isEqualTo(100);
        }
    }

    @Nested
    class Settings {

        @Test
        void savesValidScanningSettings() {
            stubSettingReads();

            Map<String, String> updated = service.updateScanSettings(Map.of(
                "candidate-scan.min-win-rate", "52.5",
                "candidate-scan.min-total-return", "1.25",
                "candidate-scan.min-trades", "20",
                "candidate-scan.out-of-sample-days", "180",
                "candidate-scan.out-of-sample-folds", "4",
                "candidate-scan.max-concurrent", "6",
                "candidate-scan.backfill-years", "5"));

            verify(settingsService).set("candidate-scan.min-win-rate", "52.5");
            verify(settingsService).set("candidate-scan.min-total-return", "1.25");
            verify(settingsService).set("candidate-scan.min-trades", "20");
            verify(settingsService).set("candidate-scan.out-of-sample-days", "180");
            verify(settingsService).set("candidate-scan.out-of-sample-folds", "4");
            verify(settingsService).set("candidate-scan.max-concurrent", "6");
            verify(settingsService).set("candidate-scan.backfill-years", "5");
            assertThat(updated).containsEntry("candidate-scan.max-concurrent", "6");
        }

        @Test
        void rejectsOutOfRangeWalkForwardFolds() {
            assertThatThrownBy(() -> service.updateScanSettings(Map.of(
                "candidate-scan.out-of-sample-folds", "9")))
                .isInstanceOf(IllegalArgumentException.class)
                .hasMessageContaining("between 1 and 8");

            verify(settingsService, never()).set(any(), any());
        }

        @Test
        void rejectsOutOfRangeConcurrency() {
            assertThatThrownBy(() -> service.updateScanSettings(Map.of(
                "candidate-scan.max-concurrent", "13")))
                .isInstanceOf(IllegalArgumentException.class)
                .hasMessageContaining("between 1 and 12");

            verify(settingsService, never()).set(any(), any());
        }

        private void stubSettingReads() {
            when(settingsService.get("candidate-scan.min-win-rate", "45.0")).thenReturn("52.5");
            when(settingsService.get("candidate-scan.min-total-return", "0.0")).thenReturn("1.25");
            when(settingsService.get("candidate-scan.min-trades", "15")).thenReturn("20");
            when(settingsService.get("candidate-scan.out-of-sample-days", "252")).thenReturn("180");
            when(settingsService.get("candidate-scan.out-of-sample-folds", "3")).thenReturn("4");
            when(settingsService.get("candidate-scan.max-concurrent", "3")).thenReturn("6");
            when(settingsService.get("candidate-scan.backfill-years", "3")).thenReturn("5");
        }
    }

}
