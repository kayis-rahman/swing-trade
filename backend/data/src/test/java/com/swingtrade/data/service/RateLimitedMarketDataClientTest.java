package com.swingtrade.data.service;

import com.swingtrade.domain.PriceBand;
import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.List;
import org.junit.jupiter.api.Test;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class RateLimitedMarketDataClientTest {
    private static final LocalDate DATE = LocalDate.of(2026, 1, 15);

    @Test
    void delegatesEveryMarketDataOperation() {
        MarketDataClient delegate = mock(MarketDataClient.class);
        RateLimitedMarketDataClient client = new RateLimitedMarketDataClient(delegate, 0);
        CandleData candle = CandleData.of("TCS", DATE, bd("100"), bd("105"), bd("99"), bd("104"), 1_000);
        PriceBand band = new PriceBand("TCS", DATE, bd("90"), bd("110"));
        QuoteData quote = QuoteData.of("TCS", "TCS", "TCS Limited", bd("104"), bd("1"), bd("1"),
                bd("105"), bd("99"), bd("103"), bd("120"), bd("80"), 1_000L, "INR", "REGULAR",
                bd("100"), bd("95"), bd("90"), bd("110"));
        InstrumentDetails instrument = mock(InstrumentDetails.class);
        ChartMeta meta = mock(ChartMeta.class);
        SearchResult searchResult = mock(SearchResult.class);
        when(delegate.fetchCandle("TCS", DATE)).thenReturn(candle);
        when(delegate.fetchCandles("TCS", DATE, DATE)).thenReturn(List.of(candle));
        when(delegate.fetchLatestCandle("TCS")).thenReturn(candle);
        when(delegate.fetchPriceBand("TCS", DATE)).thenReturn(band);
        when(delegate.fetchInstrumentDetails("TCS")).thenReturn(instrument);
        when(delegate.fetchChartMeta("TCS")).thenReturn(meta);
        when(delegate.fetchQuote("TCS")).thenReturn(quote);
        when(delegate.fetchQuotes(List.of("TCS"))).thenReturn(List.of(quote));
        when(delegate.searchSymbols("Tata")).thenReturn(List.of(searchResult));
        when(delegate.fetchAllStockSymbols()).thenReturn(List.of("TCS"));
        when(delegate.isConnected()).thenReturn(true);

        assertThat(client.fetchCandle("TCS", DATE)).isEqualTo(candle);
        assertThat(client.fetchCandles("TCS", DATE, DATE)).containsExactly(candle);
        assertThat(client.fetchLatestCandle("TCS")).isEqualTo(candle);
        assertThat(client.fetchPriceBand("TCS", DATE)).isEqualTo(band);
        assertThat(client.fetchInstrumentDetails("TCS")).isEqualTo(instrument);
        assertThat(client.fetchChartMeta("TCS")).isEqualTo(meta);
        assertThat(client.fetchQuote("TCS")).isEqualTo(quote);
        assertThat(client.fetchQuotes(List.of("TCS"))).containsExactly(quote);
        assertThat(client.searchSymbols("Tata")).containsExactly(searchResult);
        assertThat(client.fetchAllStockSymbols()).containsExactly("TCS");
        assertThat(client.isConnected()).isTrue();

        verify(delegate).fetchCandle("TCS", DATE);
        verify(delegate).fetchCandles("TCS", DATE, DATE);
        verify(delegate).fetchLatestCandle("TCS");
        verify(delegate).fetchPriceBand("TCS", DATE);
        verify(delegate).fetchInstrumentDetails("TCS");
        verify(delegate).fetchChartMeta("TCS");
        verify(delegate).fetchQuote("TCS");
        verify(delegate).fetchQuotes(List.of("TCS"));
        verify(delegate).searchSymbols("Tata");
        verify(delegate).fetchAllStockSymbols();
        verify(delegate).isConnected();
    }

    @Test
    void nonPositiveRateLimitIsUnlimited() {
        MarketDataClient delegate = mock(MarketDataClient.class);
        when(delegate.isConnected()).thenReturn(true);
        RateLimitedMarketDataClient client = new RateLimitedMarketDataClient(delegate, 0);

        assertThat(client.isConnected()).isTrue();
        assertThat(client.isConnected()).isTrue();
        verify(delegate, org.mockito.Mockito.times(2)).isConnected();
    }

    private static BigDecimal bd(String value) {
        return new BigDecimal(value);
    }
}
