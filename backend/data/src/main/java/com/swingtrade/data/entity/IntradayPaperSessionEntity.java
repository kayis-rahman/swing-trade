package com.swingtrade.data.entity;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.Id;
import jakarta.persistence.Table;

import java.time.Instant;
import java.time.LocalDate;

@Entity
@Table(name = "intraday_paper_sessions")
public class IntradayPaperSessionEntity {
    @Id
    @Column(name = "run_id", nullable = false, length = 36)
    private String runId;
    @Column(nullable = false, length = 32)
    private String symbol;
    @Column(name = "session_date", nullable = false)
    private LocalDate sessionDate;
    @Column(name = "created_at", nullable = false)
    private Instant createdAt;
    @Column(name = "result_json", nullable = false, columnDefinition = "TEXT")
    private String resultJson;

    public IntradayPaperSessionEntity() { }
    public IntradayPaperSessionEntity(String runId, String symbol, LocalDate sessionDate, Instant createdAt,
                                      String resultJson) {
        this.runId = runId;
        this.symbol = symbol;
        this.sessionDate = sessionDate;
        this.createdAt = createdAt;
        this.resultJson = resultJson;
    }
    public String getRunId() { return runId; }
    public String getSymbol() { return symbol; }
    public LocalDate getSessionDate() { return sessionDate; }
    public Instant getCreatedAt() { return createdAt; }
    public String getResultJson() { return resultJson; }
}
