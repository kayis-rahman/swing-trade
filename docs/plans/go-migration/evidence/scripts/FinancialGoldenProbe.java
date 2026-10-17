package com.swingtrade.strategy;

import java.math.BigDecimal;

/** Emits deterministic Java financial-scale and cost-model outputs for Phase 0 fixtures. */
public final class FinancialGoldenProbe {
    private FinancialGoldenProbe() { }

    public static void main(String[] args) {
        System.out.println("money_half_up=" + FinancialScale.money(new BigDecimal("1.00005")));
        System.out.println("percent_half_up=" + FinancialScale.percent(new BigDecimal("12.34565")));
        System.out.println("indicator_half_up=" + FinancialScale.indicator(new BigDecimal("1.2345675")));
        System.out.println("percent_of=" + FinancialScale.percentOf(new BigDecimal("1"), new BigDecimal("6")));
        System.out.println("percent_of_zero_denominator=" + FinancialScale.percentOf(new BigDecimal("1"), BigDecimal.ZERO));
        System.out.println("whole_shares_down=" + FinancialScale.wholeShares(new BigDecimal("12.999")));
        System.out.println("whole_shares_saturated=" + FinancialScale.wholeShares(new BigDecimal("2147483648")));
        System.out.println("delivery_round_trip_cost=" + new ZerodhaDeliveryCostModel().roundTripCost(
                new BigDecimal("100"), new BigDecimal("110"), 10, new BigDecimal("20")));
        System.out.println("intraday_round_trip_cost=" + new IntradayCostModel(new BigDecimal("20"), BigDecimal.ZERO)
                .roundTripCost(new BigDecimal("102"), new BigDecimal("104.04"), 10));
    }
}
