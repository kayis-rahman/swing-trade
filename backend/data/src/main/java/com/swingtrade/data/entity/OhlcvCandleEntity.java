package com.swingtrade.data.entity;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Index;
import jakarta.persistence.Table;
import jakarta.persistence.UniqueConstraint;
import jakarta.persistence.Version;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.LocalTime;

@Entity
@Table(name = "ohlcv_candles", schema = "public", indexes = {
    @Index(name = "idx_ohlcv_symbol_date", columnList = "symbol, date"),
    @Index(name = "idx_ohlcv_date", columnList = "date")
}, uniqueConstraints = {
    // Candle identity: (symbol, interval, date, bar_time). Daily rows are ('D', date, 00:00),
    // so the pre-existing (symbol, date) uniqueness is preserved for the swing path.
    @UniqueConstraint(name = "uq_ohlcv_candles_symbol_timeframe_date_bar_time",
        columnNames = {"symbol", "timeframe", "date", "bar_time"})
})
public class OhlcvCandleEntity {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Version
    private Integer version = 0;

    @Column(nullable = false, length = 10)
    private String symbol;

    @Column(nullable = false)
    private LocalDate date;

    /**
     * Interval/timeframe code ({@link com.swingtrade.data.service.Interval#getCode()}).
     * Defaults to 'D' (daily) so existing rows and the swing path are unchanged.
     * Named {@code timeframe} because {@code interval} is a reserved keyword in H2.
     */
    @Column(name = "timeframe", nullable = false, length = 10)
    private String timeframe;

    /**
     * Bar start time within the trading date. Daily bars use 00:00; intraday bars
     * use the actual bar start (e.g. 09:15 for the first 15-minute bar).
     */
    @Column(name = "bar_time", nullable = false)
    private LocalTime barTime;

    @Column(name = "open_price", precision = 15, scale = 4)
    private BigDecimal openPrice;

    @Column(name = "high_price", precision = 15, scale = 4)
    private BigDecimal highPrice;

    @Column(name = "low_price", precision = 15, scale = 4)
    private BigDecimal lowPrice;

    @Column(name = "close_price", precision = 15, scale = 4)
    private BigDecimal closePrice;

    private Long volume;

    @Column(name = "adj_close_price", precision = 15, scale = 4)
    private BigDecimal adjClosePrice;

    @Column(name = "created_at", updatable = false)
    private LocalDateTime createdAt;

    public OhlcvCandleEntity() {}

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }

    public static OhlcvCandleEntity fromDomain(com.swingtrade.domain.OhlcvCandle candle) {
        OhlcvCandleEntity entity = new OhlcvCandleEntity();
        entity.setSymbol(candle.symbol());
        entity.setDate(candle.date());
        entity.setTimeframe(com.swingtrade.data.service.Interval.DAILY.getCode());
        entity.setBarTime(java.time.LocalTime.MIN);
        entity.setOpenPrice(candle.open());
        entity.setHighPrice(candle.high());
        entity.setLowPrice(candle.low());
        entity.setClosePrice(candle.close());
        entity.setVolume(candle.volume());
        entity.setAdjClosePrice(candle.adjClose());
        return entity;
    }

    /**
     * Maps a provider candle (which carries the interval dimension) to the entity.
     */
    public static OhlcvCandleEntity fromCandleData(com.swingtrade.data.service.CandleData candle) {
        OhlcvCandleEntity entity = new OhlcvCandleEntity();
        entity.setSymbol(candle.symbol());
        entity.setDate(candle.date());
        entity.setTimeframe(candle.interval().getCode());
        entity.setBarTime(candle.barTime());
        entity.setOpenPrice(candle.open());
        entity.setHighPrice(candle.high());
        entity.setLowPrice(candle.low());
        entity.setClosePrice(candle.close());
        entity.setVolume(candle.volume());
        entity.setAdjClosePrice(candle.adjClose());
        return entity;
    }

    /**
     * Maps back to the provider DTO, preserving the interval dimension.
     */
    public com.swingtrade.data.service.CandleData toCandleData() {
        return com.swingtrade.data.service.CandleData.of(
            symbol, date, barTime, openPrice, highPrice, lowPrice, closePrice,
            volume, adjClosePrice, com.swingtrade.data.service.Interval.fromCode(timeframe)
        );
    }

    public com.swingtrade.domain.OhlcvCandle toDomain() {
        return new com.swingtrade.domain.OhlcvCandle(
            symbol, date, openPrice, highPrice, lowPrice, closePrice, volume, adjClosePrice
        );
    }

    public String getSymbol() { return symbol; }
    public void setSymbol(String symbol) { this.symbol = symbol; }

    public LocalDate getDate() { return date; }
    public void setDate(LocalDate date) { this.date = date; }

    public String getTimeframe() { return timeframe; }
    public void setTimeframe(String timeframe) { this.timeframe = timeframe; }

    public LocalTime getBarTime() { return barTime; }
    public void setBarTime(LocalTime barTime) { this.barTime = barTime; }

    public BigDecimal getOpenPrice() { return openPrice; }
    public void setOpenPrice(BigDecimal openPrice) { this.openPrice = openPrice; }

    public BigDecimal getHighPrice() { return highPrice; }
    public void setHighPrice(BigDecimal highPrice) { this.highPrice = highPrice; }

    public BigDecimal getLowPrice() { return lowPrice; }
    public void setLowPrice(BigDecimal lowPrice) { this.lowPrice = lowPrice; }

    public BigDecimal getClosePrice() { return closePrice; }
    public void setClosePrice(BigDecimal closePrice) { this.closePrice = closePrice; }

    public Long getVolume() { return volume; }
    public void setVolume(Long volume) { this.volume = volume; }

    public BigDecimal getAdjClosePrice() { return adjClosePrice; }
    public void setAdjClosePrice(BigDecimal adjClosePrice) { this.adjClosePrice = adjClosePrice; }

    public LocalDateTime getCreatedAt() { return createdAt; }
    public void setCreatedAt(LocalDateTime createdAt) { this.createdAt = createdAt; }
}
