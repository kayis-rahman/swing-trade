package com.swingtrade.api.scheduler;

import com.swingtrade.api.service.JobOrchestratorService;
import com.swingtrade.api.service.RunRequest;
import com.swingtrade.data.entity.CandidateScanResultEntity;
import com.swingtrade.data.entity.CandidateScanRunEntity;
import com.swingtrade.data.repository.CandidateScanResultRepository;
import com.swingtrade.data.repository.CandidateScanRunRepository;
import com.swingtrade.data.repository.JobRunRepository;
import com.swingtrade.domain.JobRun;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.time.LocalDateTime;
import java.util.Optional;
import java.util.UUID;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class CandidateScanHandoffDispatcherTest {

    @Test
    void startsAFullJobScopedToQualifiedActivatedCandidates() {
        CandidateScanRunRepository runs = mock(CandidateScanRunRepository.class);
        JobRunRepository jobs = mock(JobRunRepository.class);
        CandidateScanResultRepository results = mock(CandidateScanResultRepository.class);
        JobOrchestratorService orchestrator = mock(JobOrchestratorService.class);
        CandidateScanRunEntity scan = new CandidateScanRunEntity();
        scan.setRunId(UUID.randomUUID());
        scan.setStatus("COMPLETED");
        scan.setOrchestrationStatus("PENDING");
        scan.setQualifiedSymbols(1);
        CandidateScanResultEntity qualified = result(scan.getRunId(), "INFY", true, true);
        CandidateScanResultEntity rejected = result(scan.getRunId(), "TCS", false, false);
        when(runs.findByStatusAndOrchestrationStatus("COMPLETED", "PENDING")).thenReturn(List.of(scan));
        when(jobs.findFirstByCandidateScanRunIdOrderByStartedAtDesc(scan.getRunId())).thenReturn(Optional.empty());
        when(orchestrator.findActiveRun()).thenReturn(Optional.empty());
        when(results.findByRunIdOrderBySymbolAsc(scan.getRunId())).thenReturn(List.of(qualified, rejected));
        when(orchestrator.startRun(eq(JobRun.TriggerType.SCHEDULED), eq(scan.getRunId()), any(RunRequest.class)))
            .thenReturn(new JobRun(UUID.randomUUID(), JobRun.TriggerType.SCHEDULED, JobRun.Status.RUNNING,
                LocalDateTime.now(), null, 1, 0, 0, null));

        new CandidateScanHandoffDispatcher(runs, jobs, results, orchestrator).dispatchPendingHandoffs();

        verify(orchestrator).startRun(eq(JobRun.TriggerType.SCHEDULED), eq(scan.getRunId()),
            eq(new RunRequest(List.of("INFY"), null, null, null, null)));
    }

    @Test
    void manualScanHandoffWaitsForOperatorScopeSelection() {
        CandidateScanRunRepository runs = mock(CandidateScanRunRepository.class);
        JobRunRepository jobs = mock(JobRunRepository.class);
        CandidateScanResultRepository results = mock(CandidateScanResultRepository.class);
        JobOrchestratorService orchestrator = mock(JobOrchestratorService.class);
        CandidateScanRunEntity scan = new CandidateScanRunEntity();
        scan.setRunId(UUID.randomUUID());
        scan.setStatus("COMPLETED");
        scan.setScanTrigger("MANUAL");
        scan.setOrchestrationStatus("PENDING");
        scan.setQualifiedSymbols(0);
        when(runs.findByStatusAndOrchestrationStatus("COMPLETED", "PENDING")).thenReturn(List.of(scan));

        new CandidateScanHandoffDispatcher(runs, jobs, results, orchestrator).dispatchPendingHandoffs();

        verify(orchestrator, never()).startRun(eq(JobRun.TriggerType.MANUAL), eq(scan.getRunId()), any());
    }

    @Test
    void selectedQualifiedCandidatesAreExplicitlyScopedForAnalysis() {
        CandidateScanRunRepository runs = mock(CandidateScanRunRepository.class);
        JobRunRepository jobs = mock(JobRunRepository.class);
        CandidateScanResultRepository results = mock(CandidateScanResultRepository.class);
        JobOrchestratorService orchestrator = mock(JobOrchestratorService.class);
        CandidateScanRunEntity scan = new CandidateScanRunEntity();
        scan.setRunId(UUID.randomUUID());
        scan.setStatus("COMPLETED");
        scan.setScanTrigger("MANUAL");
        when(runs.findByRunId(scan.getRunId())).thenReturn(Optional.of(scan));
        when(results.findByRunIdOrderBySymbolAsc(scan.getRunId())).thenReturn(List.of(
            result(scan.getRunId(), "INFY", true, true), result(scan.getRunId(), "TCS", true, true)));
        when(orchestrator.findActiveRun()).thenReturn(Optional.empty());
        JobRun job = new JobRun(UUID.randomUUID(), JobRun.TriggerType.MANUAL, JobRun.Status.RUNNING,
            LocalDateTime.now(), null, 1, 0, 0, null);
        when(orchestrator.startRun(eq(JobRun.TriggerType.MANUAL), eq(scan.getRunId()), any(RunRequest.class)))
            .thenReturn(job);

        new CandidateScanHandoffDispatcher(runs, jobs, results, orchestrator)
            .startManualAnalysis(scan.getRunId(), "SELECTED", List.of("INFY"));

        verify(orchestrator).startRun(JobRun.TriggerType.MANUAL, scan.getRunId(),
            new RunRequest(List.of("INFY"), null, null, null, null));
    }

    private static CandidateScanResultEntity result(UUID runId, String symbol, boolean qualified, boolean activated) {
        CandidateScanResultEntity result = new CandidateScanResultEntity();
        result.setRunId(runId);
        result.setSymbol(symbol);
        result.setQualified(qualified);
        result.setActivated(activated);
        return result;
    }
}
