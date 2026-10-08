package com.swingtrade.data.entity;

import jakarta.persistence.Column;
import jakarta.persistence.ElementCollection;
import jakarta.persistence.Embeddable;
import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Table;
import jakarta.persistence.Version;

import java.time.LocalDateTime;
import java.time.LocalDate;
import java.util.UUID;
import java.util.ArrayList;
import java.util.List;
import java.math.BigDecimal;

@Entity
@Table(name = "candidate_scan_results")
public class CandidateScanResultEntity {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;
    @Version
    private Integer version = 0;
    @Column(name = "run_id", nullable = false)
    private UUID runId;
    @Column(nullable = false, length = 32)
    private String symbol;
    @Column(name = "data_status", nullable = false, length = 24)
    private String dataStatus;
    @Column(name = "candle_count", nullable = false)
    private int candleCount;
    @Column(name = "signal_type", length = 16)
    private String signalType;
    @Column(name = "strategy_buy_count", nullable = false)
    private int strategyBuyCount;
    @Column(name = "strategy_evaluation_count", nullable = false)
    private int strategyEvaluationCount;
    @ElementCollection
    @jakarta.persistence.CollectionTable(name = "candidate_scan_strategy_outcomes",
        joinColumns = @jakarta.persistence.JoinColumn(name = "candidate_scan_result_id"))
    private List<StrategyOutcome> strategyOutcomes = new ArrayList<>();
    @Column(name = "total_trades")
    private Integer totalTrades;
    @Column(name = "win_rate")
    private Double winRate;
    @Column(name = "total_return")
    private Double totalReturn;
    @Column(name = "max_drawdown_pct")
    private Double maxDrawdownPct;
    @Column(name = "oos_start_date")
    private LocalDate oosStartDate;
    @Column(name = "oos_end_date")
    private LocalDate oosEndDate;
    @Column(name = "oos_total_trades")
    private Integer oosTotalTrades;
    @Column(name = "oos_win_rate")
    private Double oosWinRate;
    @Column(name = "oos_total_return")
    private Double oosTotalReturn;
    @Column(name = "oos_max_drawdown_pct")
    private Double oosMaxDrawdownPct;
    @Column(nullable = false)
    private boolean qualified;
    @Column(nullable = false)
    private boolean activated;
    @Column(columnDefinition = "TEXT")
    private String reason;
    @Column(name = "error_message", columnDefinition = "TEXT")
    private String errorMessage;
    @Column(name = "source_outcome", length = 32) private String sourceOutcome;
    @Column(name = "invalid_rows", nullable = false) private int invalidRows;
    @Column(name = "first_available_date") private LocalDate firstAvailableDate;
    @Column(name = "last_available_date") private LocalDate lastAvailableDate;
    @Column(name = "retry_after") private LocalDate retryAfter;
    @Column(name = "created_at", nullable = false)
    private LocalDateTime createdAt = LocalDateTime.now();

    public Long getId() { return id; }
    public UUID getRunId() { return runId; }
    public void setRunId(UUID value) { this.runId = value; }
    public String getSymbol() { return symbol; }
    public void setSymbol(String value) { this.symbol = value; }
    public String getDataStatus() { return dataStatus; }
    public void setDataStatus(String value) { this.dataStatus = value; }
    public int getCandleCount() { return candleCount; }
    public void setCandleCount(int value) { this.candleCount = value; }
    public String getSignalType() { return signalType; }
    public void setSignalType(String value) { this.signalType = value; }
    public int getStrategyBuyCount() { return strategyBuyCount; }
    public void setStrategyBuyCount(int value) { strategyBuyCount = value; }
    public int getStrategyEvaluationCount() { return strategyEvaluationCount; }
    public void setStrategyEvaluationCount(int value) { strategyEvaluationCount = value; }
    public List<StrategyOutcome> getStrategyOutcomes() { return strategyOutcomes; }
    public void setStrategyOutcomes(List<StrategyOutcome> value) {
        strategyOutcomes = value == null ? new ArrayList<>() : new ArrayList<>(value);
    }
    public Integer getTotalTrades() { return totalTrades; }
    public void setTotalTrades(Integer value) { this.totalTrades = value; }
    public Double getWinRate() { return winRate; }
    public void setWinRate(Double value) { this.winRate = value; }
    public Double getTotalReturn() { return totalReturn; }
    public void setTotalReturn(Double value) { this.totalReturn = value; }
    public Double getMaxDrawdownPct() { return maxDrawdownPct; }
    public void setMaxDrawdownPct(Double value) { this.maxDrawdownPct = value; }
    public LocalDate getOosStartDate() { return oosStartDate; }
    public void setOosStartDate(LocalDate value) { this.oosStartDate = value; }
    public LocalDate getOosEndDate() { return oosEndDate; }
    public void setOosEndDate(LocalDate value) { this.oosEndDate = value; }
    public Integer getOosTotalTrades() { return oosTotalTrades; }
    public void setOosTotalTrades(Integer value) { this.oosTotalTrades = value; }
    public Double getOosWinRate() { return oosWinRate; }
    public void setOosWinRate(Double value) { this.oosWinRate = value; }
    public Double getOosTotalReturn() { return oosTotalReturn; }
    public void setOosTotalReturn(Double value) { this.oosTotalReturn = value; }
    public Double getOosMaxDrawdownPct() { return oosMaxDrawdownPct; }
    public void setOosMaxDrawdownPct(Double value) { this.oosMaxDrawdownPct = value; }
    public boolean isQualified() { return qualified; }
    public void setQualified(boolean value) { this.qualified = value; }
    public boolean isActivated() { return activated; }
    public void setActivated(boolean value) { this.activated = value; }
    public String getReason() { return reason; }
    public void setReason(String value) { this.reason = value; }
    public String getErrorMessage() { return errorMessage; }
    public void setErrorMessage(String value) { this.errorMessage = value; }
    public String getSourceOutcome() { return sourceOutcome; }
    public void setSourceOutcome(String value) { sourceOutcome = value; }
    public int getInvalidRows() { return invalidRows; }
    public void setInvalidRows(int value) { invalidRows = value; }
    public LocalDate getFirstAvailableDate() { return firstAvailableDate; }
    public void setFirstAvailableDate(LocalDate value) { firstAvailableDate = value; }
    public LocalDate getLastAvailableDate() { return lastAvailableDate; }
    public void setLastAvailableDate(LocalDate value) { lastAvailableDate = value; }
    public LocalDate getRetryAfter() { return retryAfter; }
    public void setRetryAfter(LocalDate value) { retryAfter = value; }
    public LocalDateTime getCreatedAt() { return createdAt; }
    public void setCreatedAt(LocalDateTime value) { this.createdAt = value; }

    @Embeddable
    public static class StrategyOutcome {
        @Column(name = "variant_id", nullable = false, length = 128)
        private String variantId;
        @Column(name = "signal_type", nullable = false, length = 16)
        private String signalType;
        @Column(precision = 18, scale = 8)
        private BigDecimal score;
        @Column(columnDefinition = "TEXT")
        private String detail;
        @Column(name = "backtest_total_trades") private Integer backtestTotalTrades;
        @Column(name = "backtest_win_rate") private Double backtestWinRate;
        @Column(name = "backtest_total_return") private Double backtestTotalReturn;
        @Column(name = "oos_total_trades") private Integer oosTotalTrades;
        @Column(name = "oos_win_rate") private Double oosWinRate;
        @Column(name = "oos_total_return") private Double oosTotalReturn;
        @Column(name = "performance_status", nullable = false, length = 16) private String performanceStatus = "NOT_RUN";

        protected StrategyOutcome() { }

        public StrategyOutcome(String variantId, String signalType, BigDecimal score, String detail) {
            this.variantId = variantId;
            this.signalType = signalType;
            this.score = score;
            this.detail = detail;
        }

        public StrategyOutcome(String variantId, String signalType, BigDecimal score, String detail,
                               Integer backtestTotalTrades, Double backtestWinRate, Double backtestTotalReturn,
                               Integer oosTotalTrades, Double oosWinRate, Double oosTotalReturn,
                               String performanceStatus) {
            this(variantId, signalType, score, detail);
            this.backtestTotalTrades = backtestTotalTrades;
            this.backtestWinRate = backtestWinRate;
            this.backtestTotalReturn = backtestTotalReturn;
            this.oosTotalTrades = oosTotalTrades;
            this.oosWinRate = oosWinRate;
            this.oosTotalReturn = oosTotalReturn;
            this.performanceStatus = performanceStatus == null ? "NOT_RUN" : performanceStatus;
        }

        public String getVariantId() { return variantId; }
        public String getSignalType() { return signalType; }
        public BigDecimal getScore() { return score; }
        public String getDetail() { return detail; }
        public Integer getBacktestTotalTrades() { return backtestTotalTrades; }
        public Double getBacktestWinRate() { return backtestWinRate; }
        public Double getBacktestTotalReturn() { return backtestTotalReturn; }
        public Integer getOosTotalTrades() { return oosTotalTrades; }
        public Double getOosWinRate() { return oosWinRate; }
        public Double getOosTotalReturn() { return oosTotalReturn; }
        public String getPerformanceStatus() { return performanceStatus; }
    }
}
