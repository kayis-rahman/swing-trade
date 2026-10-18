# Phase 0 financial golden vectors (partial)

Outputs were emitted directly by the Java 21 classes in `backend/strategy` on 2026-10-17. The probe covers the
`FinancialScale` rounding/saturation rules and both current cost models; it is not the complete indicator,
strategy, gate, or engine vector set required by G0.

## Java output

```text
money_half_up=1.0001
percent_half_up=12.3457
indicator_half_up=1.234568
percent_of=16.6667
percent_of_zero_denominator=0.0000
whole_shares_down=12
whole_shares_saturated=2147483647
delivery_round_trip_cost=41.86
intraday_round_trip_cost=47.57
```

The values correspond to the Java implementations in
[FinancialScale.java](../../../../backend/strategy/src/main/java/com/swingtrade/strategy/FinancialScale.java),
[ZerodhaDeliveryCostModel.java](../../../../backend/strategy/src/main/java/com/swingtrade/strategy/ZerodhaDeliveryCostModel.java),
and [IntradayCostModel.java](../../../../backend/strategy/src/main/java/com/swingtrade/strategy/IntradayCostModel.java).
The final intraday value independently agrees with the existing hand-checked replay assertion in
`IntradayBacktestEngineTest#orbHandCheckedRoundTripUsesIndependentIntradayCosts`.
