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

    private static CandidateScanResultEntity result(UUID runId, String symbol, boolean qualified, boolean activated) {
        CandidateScanResultEntity result = new CandidateScanResultEntity();
        result.setRunId(runId);
        result.setSymbol(symbol);
        result.setQualified(qualified);
        result.setActivated(activated);
        return result;
    }
}
