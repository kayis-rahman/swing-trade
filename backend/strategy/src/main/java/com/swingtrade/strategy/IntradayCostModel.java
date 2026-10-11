package com.swingtrade.strategy;

import java.math.BigDecimal;
import java.math.RoundingMode;

/** NSE intraday-equity costs; intentionally independent of the delivery cost model. */
public final class IntradayCostModel {
    private static final BigDecimal STT_SELL = new BigDecimal("0.00025");
    private static final BigDecimal STAMP_BUY = new BigDecimal("0.00003");
    private static final BigDecimal EXCHANGE = new BigDecimal("0.0000307");
    private static final BigDecimal SEBI = new BigDecimal("0.0000001");
    private static final BigDecimal GST_RATE = new BigDecimal("0.18");
    private final BigDecimal brokeragePerOrder;
    private final BigDecimal slippageRate;

    public IntradayCostModel() { this(new BigDecimal("20"), new BigDecimal("0.0005")); }
    public IntradayCostModel(BigDecimal brokeragePerOrder, BigDecimal slippageRate) {
        if (brokeragePerOrder == null || slippageRate == null || brokeragePerOrder.signum() < 0 || slippageRate.signum() < 0) {
            throw new IllegalArgumentException("Costs must be non-negative");
        }
        this.brokeragePerOrder = brokeragePerOrder;
        this.slippageRate = slippageRate;
    }

    public BigDecimal buyFill(BigDecimal reference) { return reference.multiply(BigDecimal.ONE.add(slippageRate)); }
    public BigDecimal sellFill(BigDecimal reference) { return reference.multiply(BigDecimal.ONE.subtract(slippageRate)); }

    public BigDecimal roundTripCost(BigDecimal buy, BigDecimal sell, int quantity) {
        if (quantity <= 0) throw new IllegalArgumentException("Quantity must be positive");
        BigDecimal buyTurnover = buy.multiply(BigDecimal.valueOf(quantity));
        BigDecimal sellTurnover = sell.multiply(BigDecimal.valueOf(quantity));
        BigDecimal brokerage = brokeragePerOrder.multiply(BigDecimal.valueOf(2));
        BigDecimal exchange = buyTurnover.add(sellTurnover).multiply(EXCHANGE);
        BigDecimal sebi = buyTurnover.add(sellTurnover).multiply(SEBI);
        BigDecimal taxable = brokerage.add(exchange).add(sebi);
        BigDecimal gst = taxable.multiply(GST_RATE);
        return brokerage.add(exchange).add(sebi).add(gst)
            .add(sellTurnover.multiply(STT_SELL)).add(buyTurnover.multiply(STAMP_BUY))
            .setScale(2, RoundingMode.HALF_UP);
    }
}
