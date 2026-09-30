package com.swingtrade.api.service;

import com.swingtrade.api.app.SwingTradeApiApplication;
import com.swingtrade.broker.entity.PaperTradingOrderEntity;
import com.swingtrade.broker.repository.PaperTradingOrderRepository;
import com.swingtrade.broker.scheduler.PendingOrderExecutionScheduler;
import com.swingtrade.data.entity.PositionEntity;
import com.swingtrade.data.repository.PositionRepository;
import com.swingtrade.domain.OhlcvCandle;
import com.swingtrade.domain.SentimentResult;
import com.swingtrade.domain.Signal;
import com.swingtrade.domain.Trade;
import com.swingtrade.domain.store.CandleStore;
import com.swingtrade.domain.store.SentimentStore;
import com.swingtrade.domain.store.SignalStore;
import com.swingtrade.domain.store.TradeStore;
import com.swingtrade.domain.service.TradingService;
import com.swingtrade.llm.service.SentimentService;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.testcontainers.service.connection.ServiceConnection;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.test.util.ReflectionTestUtils;
import org.testcontainers.containers.PostgreSQLContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.ArrayList;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.when;

/**
 * Isolated database verification of the BUY-to-paper-position path. Market data and sentiment
 * are deterministic; the real signal engine, orchestrator gate, paper engine, persistence, and
 * next-session order scheduler remain in the path.
 */
@SpringBootTest(classes = SwingTradeApiApplication.class)
@Testcontainers
@ActiveProfiles("test")
@TestPropertySource(properties = {
    "job.orchestrator.llm-analysis.enabled=false",
    "paper.trading.pending-order-cron=-",
    "broker.slippage.percentage=0",
    "spring.ai.openai.api-key=dummy"
})
class BuyPaperTradeIntegrationTest {

    private static final String SYMBOL = "BUYTEST";

    @Container
    @ServiceConnection
    static PostgreSQLContainer<?> postgres = new PostgreSQLContainer<>("postgres:16")
        .withDatabaseName("swing_trade_buy_test")
        .withUsername("swing_trade_buy_test")
        .withPassword("swing_trade_buy_test");

    @Autowired private SignalPipeline signalPipeline;
    @Autowired private JobOrchestratorService jobOrchestratorService;
    @Autowired private PendingOrderExecutionScheduler pendingOrderExecutionScheduler;
    @Autowired private CandleStore candleStore;
    @Autowired private SentimentStore sentimentStore;
    @Autowired private SignalStore signalStore;
    @Autowired private TradeStore tradeStore;
    @Autowired private TradingService tradingService;
    @Autowired private PositionRepository positionRepository;
    @Autowired private PaperTradingOrderRepository orderRepository;

    @MockitoBean private SentimentService sentimentService;
    @MockitoBean private LiveEligibilityService liveEligibilityService;

    @Test
    void positiveSentimentBuy_isQueuedAndFilledFromNextSessionOpen() {
        List<OhlcvCandle> candles = buildBuyCandles();
        candles.forEach(candleStore::save);
        OhlcvCandle signalCandle = candles.getLast();

        Signal signal = signalPipeline.generatePrimarySignal(SYMBOL).orElseThrow();
        assertThat(signal.type()).isEqualTo(Signal.SignalType.BUY);
        assertThat(signal.date()).isEqualTo(signalCandle.date());
        assertThat(signal.id()).isNotNull();
        assertThat(signalStore.findUnprocessed()).extracting(Signal::id).contains(signal.id());

        when(sentimentService.analyzeStockSentiment(eq(SYMBOL), any(LocalDate.class)))
            .thenAnswer(invocation -> {
                LocalDate date = invocation.getArgument(1);
                SentimentResult positive = SentimentResult.create(SYMBOL, date,
                    SentimentResult.SentimentScore.POSITIVE, "deterministic positive test result",
                    "fixture", 0.95);
                sentimentStore.saveOrUpdate(positive);
                return positive;
            });
        when(liveEligibilityService.assess(eq(SYMBOL), eq(signal.date()), any(BigDecimal.class)))
            .thenReturn(new LiveEligibilityService.EligibilityDecision(true, List.of(), List.of()));

        jobOrchestratorService.stageSentiment(SYMBOL, signal.date());
        assertThat(sentimentStore.findBySymbolAndDate(SYMBOL, signal.date())).isPresent();
        assertThat(sentimentStore.findBySymbolAndDate(SYMBOL, signal.date()).orElseThrow().score())
            .isEqualTo(SentimentResult.SentimentScore.POSITIVE);

        String queueSummary = jobOrchestratorService.stagePaperTrade(SYMBOL);
        assertThat(queueSummary).contains("1 trade(s) executed");
        assertThat(signalStore.findUnprocessed()).extracting(Signal::id).doesNotContain(signal.id());

        PaperTradingOrderEntity queued = orderRepository.findAllByOrderByCreatedAtDesc().stream()
            .filter(order -> SYMBOL.equals(order.getSymbol()))
            .findFirst().orElseThrow();
        assertThat(queued.getStatus()).isEqualTo("PENDING");
        assertThat(queued.getSignalId()).isEqualTo(signal.id().toString());
        assertThat(tradingService.getPendingOrders()).hasSize(1);
        assertThat(positionRepository.findBySymbol(SYMBOL)).isEmpty();

        BigDecimal nextOpen = signalCandle.close().multiply(new BigDecimal("1.01"))
            .setScale(2, java.math.RoundingMode.HALF_UP);
        LocalDate nextDate = signalCandle.date().plusDays(1);
        candleStore.save(OhlcvCandle.of(SYMBOL, nextDate, nextOpen,
            nextOpen.multiply(new BigDecimal("1.01")), nextOpen.multiply(new BigDecimal("0.99")),
            nextOpen, 1_000_000L));

        ReflectionTestUtils.setField(pendingOrderExecutionScheduler, "slippagePercentage", 0.0);
        pendingOrderExecutionScheduler.executePendingOrders();

        PaperTradingOrderEntity filled = orderRepository.findByOrderId(queued.getOrderId()).orElseThrow();
        assertThat(filled.getStatus()).isEqualTo("FILLED");
        assertThat(filled.getPrice()).isEqualByComparingTo(nextOpen);
        assertThat(filled.getExecutedAt()).isNotNull();

        PositionEntity position = positionRepository.findOpenBySymbol(SYMBOL).orElseThrow();
        assertThat(position.getEntryPrice()).isEqualByComparingTo(nextOpen);
        assertThat(position.getSignalId()).isEqualTo(signal.id());
        Trade trade = tradeStore.findOpenByPositionId(position.getId()).orElseThrow();
        assertThat(trade.tradeStatus()).isEqualTo(Trade.TradeStatus.OPEN);
        assertThat(trade.entryPrice()).isEqualByComparingTo(nextOpen);
        assertThat(trade.quantity()).isEqualTo(position.getQuantity());
        assertThat(trade.fees()).isEqualByComparingTo(
            new BigDecimal("0.05").multiply(BigDecimal.valueOf(position.getQuantity())));
        assertThat(tradingService.getPendingOrders()).isEmpty();
    }

    private static List<OhlcvCandle> buildBuyCandles() {
        List<OhlcvCandle> candles = new ArrayList<>();
        double price = 100.0;
        LocalDate date = LocalDate.of(2025, 1, 1);
        for (int i = 0; i < 299; i++) {
            double changePercent = i % 3 == 2 ? -0.75 : 0.5;
            double close = price * (1 + changePercent / 100.0);
            candles.add(OhlcvCandle.of(SYMBOL, date, BigDecimal.valueOf(price),
                BigDecimal.valueOf(Math.max(price, close) * 1.001),
                BigDecimal.valueOf(Math.min(price, close) * 0.999), BigDecimal.valueOf(close),
                1_000_000L));
            price = close;
            date = date.plusDays(1);
        }
        BigDecimal close = BigDecimal.valueOf(price).multiply(new BigDecimal("1.005"));
        candles.add(OhlcvCandle.of(SYMBOL, date, BigDecimal.valueOf(price),
            close.multiply(new BigDecimal("1.001")), BigDecimal.valueOf(price).multiply(new BigDecimal("0.999")),
            close, 2_000_000L));
        return candles;
    }
}
