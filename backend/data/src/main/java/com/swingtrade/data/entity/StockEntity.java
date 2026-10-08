package com.swingtrade.data.entity;

import com.swingtrade.domain.Stock;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Index;
import jakarta.persistence.Table;
import jakarta.persistence.Version;

import java.time.LocalDate;
import java.time.LocalDateTime;

/**
 * JPA entity for the Stock domain model.
 */
@Entity
@Table(name = "stocks", indexes = {
    @Index(name = "idx_stocks_symbol", columnList = "symbol", unique = true),
    @Index(name = "idx_stocks_exchange", columnList = "exchange"),
    @Index(name = "idx_stocks_sector", columnList = "sector")
})
public class StockEntity {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Version
    private Integer version = 0;

    @Column(nullable = false, length = 10)
    private String symbol;

    @Column(length = 255)
    private String name;

    @Column(length = 50)
    private String exchange;

    @Column(length = 100)
    private String sector;

    @Column(length = 100)
    private String industry;

    private Long marketCap;

    @Column(precision = 10, scale = 2)
    private java.math.BigDecimal peRatio;

    @Column(length = 13)
    private String isin;

    private Integer lotSize;

    @Column(name = "added_on")
    private LocalDate addedOn;

    @Column(name = "created_at", updatable = false)
    private LocalDateTime createdAt;

    @Column(name = "updated_at")
    private LocalDateTime updatedAt;

    public StockEntity() {
    }

    public StockEntity(Stock stock) {
        this.symbol = stock.symbol();
        this.name = stock.name();
        this.exchange = stock.exchange().name();
        this.sector = stock.sector() != null ? stock.sector().name() : null;
        this.industry = stock.industry();
        this.marketCap = stock.marketCap();
        this.peRatio = stock.peRatio();
        this.isin = stock.isin();
        this.lotSize = stock.lotSize();
        this.addedOn = stock.addedOn();
    }

    public static StockEntity fromDomain(Stock stock) {
        StockEntity entity = new StockEntity();
        entity.setSymbol(stock.symbol());
        entity.setName(stock.name());
        entity.setExchange(stock.exchange().name());
        entity.setSector(stock.sector() != null ? stock.sector().name() : null);
        entity.setIndustry(stock.industry());
        entity.setMarketCap(stock.marketCap());
        entity.setPeRatio(stock.peRatio());
        entity.setIsin(stock.isin());
        entity.setLotSize(stock.lotSize());
        entity.setAddedOn(stock.addedOn());
        return entity;
    }

    public Stock toDomain() {
        return new Stock(
            symbol,
            exchange != null ? Stock.Exchange.valueOf(exchange) : null,
            name,
            sector != null ? Stock.Sector.fromDbName(sector) : null,
            industry,
            marketCap,
            peRatio,
            isin,
            lotSize,
            addedOn
        );
    }

    // Getters and Setters
    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }

    public String getIndustry() {
        return industry;
    }

    public void setIndustry(String industry) {
        this.industry = industry;
    }

    public Long getMarketCap() {
        return marketCap;
    }

    public void setMarketCap(Long marketCap) {
        this.marketCap = marketCap;
    }

    public java.math.BigDecimal getPeRatio() {
        return peRatio;
    }

    public void setPeRatio(java.math.BigDecimal peRatio) {
        this.peRatio = peRatio;
    }

    public String getSymbol() {
        return symbol;
    }

    public void setSymbol(String symbol) {
        this.symbol = symbol;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getExchange() {
        return exchange;
    }

    public void setExchange(String exchange) {
        this.exchange = exchange;
    }

    public String getSector() {
        return sector;
    }

    public void setSector(String sector) {
        this.sector = sector;
    }

    public String getIsin() {
        return isin;
    }

    public void setIsin(String isin) {
        this.isin = isin;
    }

    public Integer getLotSize() {
        return lotSize;
    }

    public void setLotSize(Integer lotSize) {
        this.lotSize = lotSize;
    }

    public LocalDate getAddedOn() {
        return addedOn;
    }

    public void setAddedOn(LocalDate addedOn) {
        this.addedOn = addedOn;
    }

    public LocalDateTime getCreatedAt() {
        return createdAt;
    }

    public void setCreatedAt(LocalDateTime createdAt) {
        this.createdAt = createdAt;
    }

    public LocalDateTime getUpdatedAt() {
        return updatedAt;
    }

    public void setUpdatedAt(LocalDateTime updatedAt) {
        this.updatedAt = updatedAt;
    }
}
