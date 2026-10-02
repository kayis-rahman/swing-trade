package com.swingtrade.data.entity;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Table;
import jakarta.persistence.Version;

import java.time.LocalDateTime;
import java.util.UUID;

@Entity
@Table(name = "candidate_scan_runs")
public class CandidateScanRunEntity {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;
    @Version
    private Integer version = 0;
    @Column(name = "run_id", nullable = false, unique = true)
    private UUID runId;
    @Column(nullable = false, length = 16)
    private String status;
    @Column(name = "total_symbols", nullable = false)
    private int totalSymbols;
    /** Explicit provenance for the symbol universe; null is retained for legacy rows. */
    @Column(name = "scan_scope", length = 16)
    private String scanScope;
    @Column(name = "completed_symbols", nullable = false)
    private int completedSymbols;
    @Column(name = "failed_symbols", nullable = false)
    private int failedSymbols;
    @Column(name = "qualified_symbols", nullable = false)
    private int qualifiedSymbols;
    @Column(name = "started_at", nullable = false)
    private LocalDateTime startedAt;
    @Column(name = "completed_at")
    private LocalDateTime completedAt;
    @Column(name = "error_message", columnDefinition = "TEXT")
    private String errorMessage;
    @Column(name = "orchestration_status", nullable = false, length = 16)
    private String orchestrationStatus = "NOT_REQUIRED";
    @Column(name = "orchestration_job_run_id") private UUID orchestrationJobRunId;
    @Column(name = "orchestration_error", columnDefinition = "TEXT") private String orchestrationError;

    public Long getId() { return id; }
    public UUID getRunId() { return runId; }
    public void setRunId(UUID runId) { this.runId = runId; }
    public String getStatus() { return status; }
    public void setStatus(String status) { this.status = status; }
    public int getTotalSymbols() { return totalSymbols; }
    public void setTotalSymbols(int value) { this.totalSymbols = value; }
    public String getScanScope() { return scanScope; }
    public void setScanScope(String value) { this.scanScope = value; }
    public int getCompletedSymbols() { return completedSymbols; }
    public void setCompletedSymbols(int value) { this.completedSymbols = value; }
    public int getFailedSymbols() { return failedSymbols; }
    public void setFailedSymbols(int value) { this.failedSymbols = value; }
    public int getQualifiedSymbols() { return qualifiedSymbols; }
    public void setQualifiedSymbols(int value) { this.qualifiedSymbols = value; }
    public LocalDateTime getStartedAt() { return startedAt; }
    public void setStartedAt(LocalDateTime value) { this.startedAt = value; }
    public LocalDateTime getCompletedAt() { return completedAt; }
    public void setCompletedAt(LocalDateTime value) { this.completedAt = value; }
    public String getErrorMessage() { return errorMessage; }
    public void setErrorMessage(String value) { this.errorMessage = value; }
    public String getOrchestrationStatus() { return orchestrationStatus; }
    public void setOrchestrationStatus(String value) { this.orchestrationStatus = value; }
    public UUID getOrchestrationJobRunId() { return orchestrationJobRunId; }
    public void setOrchestrationJobRunId(UUID value) { this.orchestrationJobRunId = value; }
    public String getOrchestrationError() { return orchestrationError; }
    public void setOrchestrationError(String value) { this.orchestrationError = value; }
}
