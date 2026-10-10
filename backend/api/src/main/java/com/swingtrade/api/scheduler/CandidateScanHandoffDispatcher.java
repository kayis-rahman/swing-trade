package com.swingtrade.api.scheduler;

import com.swingtrade.api.service.JobOrchestratorService;
import com.swingtrade.data.entity.CandidateScanRunEntity;
import com.swingtrade.data.repository.CandidateScanResultRepository;
import com.swingtrade.data.repository.CandidateScanRunRepository;
import com.swingtrade.api.service.RunRequest;
import com.swingtrade.data.repository.JobRunRepository;
import com.swingtrade.domain.JobRun;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

/** Recovers scheduled scan handoffs after a process restart; no in-memory waiter is required. */
@Component
public class CandidateScanHandoffDispatcher {
    private static final Logger logger = LoggerFactory.getLogger(CandidateScanHandoffDispatcher.class);
    private final CandidateScanRunRepository runs;
    private final JobRunRepository jobRuns;
    private final CandidateScanResultRepository results;
    private final JobOrchestratorService orchestrator;
    private final Object dispatchLock = new Object();
    public CandidateScanHandoffDispatcher(CandidateScanRunRepository runs, JobRunRepository jobRuns,
                                          CandidateScanResultRepository results,
                                          JobOrchestratorService orchestrator) {
        this.runs = runs;
        this.jobRuns = jobRuns;
        this.results = results;
        this.orchestrator = orchestrator;
    }
    @Scheduled(fixedDelayString = "${candidate-scan.handoff-dispatch-ms:60000}")
    public void dispatchPendingHandoffs() {
        synchronized (dispatchLock) {
            for (CandidateScanRunEntity scan : runs.findByStatusAndOrchestrationStatus("COMPLETED", "PENDING")) {
                if (scan.getQualifiedSymbols() == 0) {
                    markNotRequired(scan);
                    continue;
                }
                var existingJob = jobRuns.findFirstByCandidateScanRunIdOrderByStartedAtDesc(scan.getRunId());
                if (existingJob.isPresent()) {
                    var job = existingJob.get();
                    scan.setOrchestrationJobRunId(job.getRunId());
                    if ("RUNNING".equals(job.getStatus())) {
                        if (orchestrator.findActiveRun()
                                .map(active -> active.runId().equals(job.getRunId()))
                                .orElse(false)) {
                            continue;
                        }
                        orchestrator.cancelRun(job.getRunId());
                        continue;
                    } else if ("COMPLETED".equals(job.getStatus())) {
                        scan.setOrchestrationStatus("STARTED");
                        scan.setOrchestrationError(null);
                    } else {
                        scan.setOrchestrationStatus("PENDING");
                        scan.setOrchestrationError(job.getErrorMessage());
                        scan.setOrchestrationJobRunId(null);
                        jobRuns.clearCandidateScanRunId(job.getRunId());
                    }
                    runs.save(scan);
                    continue;
                }
                if (orchestrator.findActiveRun().isPresent()) return;
                try {
                    var symbols = results.findByRunIdOrderBySymbolAsc(scan.getRunId()).stream()
                        .filter(result -> result.isQualified() && result.isActivated())
                        .map(result -> result.getSymbol()).toList();
                    if (symbols.isEmpty()) {
                        markNotRequired(scan);
                        continue;
                    }
                    JobRun job = orchestrator.startRun(JobRun.TriggerType.SCHEDULED, scan.getRunId(),
                        new RunRequest(symbols, null, null, null, null));
                    scan.setOrchestrationJobRunId(job.runId());
                    scan.setOrchestrationStatus("STARTED");
                    scan.setOrchestrationError(null);
                    runs.save(scan);
                    logger.info("Started orchestration {} for candidate scan {}", job.runId(), scan.getRunId());
                } catch (JobOrchestratorService.ConcurrentRunException ignored) {
                    return;
                } catch (Exception e) {
                    jobRuns.findFirstByCandidateScanRunIdOrderByStartedAtDesc(scan.getRunId())
                        .filter(job -> "RUNNING".equals(job.getStatus()))
                        .ifPresent(job -> orchestrator.cancelRun(job.getRunId()));
                    scan.setOrchestrationStatus("PENDING");
                    scan.setOrchestrationError(e.getMessage());
                    runs.save(scan);
                    logger.warn("Candidate scan handoff {} failed: {}", scan.getRunId(), e.getMessage());
                }
            }
        }
    }
    @Transactional
    void markNotRequired(CandidateScanRunEntity scan) {
        scan.setOrchestrationStatus("NOT_REQUIRED");
        runs.save(scan);
    }
}
