package com.swingtrade.data.service;

import okhttp3.mockwebserver.MockResponse;
import okhttp3.mockwebserver.MockWebServer;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.web.reactive.function.client.WebClient;

import java.io.IOException;
import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.List;
import java.util.Optional;
import com.swingtrade.domain.PriceBand;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class FyersServiceClientTest {

    private MockWebServer mockWebServer;
    private FyersAuthService mockAuthService;
    private FyersSymbolMasterService mockSymbolMaster;
    private FyersServiceClient client;

    @BeforeEach
    void setUp() throws IOException {
        mockWebServer = new MockWebServer();
        mockWebServer.start();

        mockAuthService = mock(FyersAuthService.class);
        when(mockAuthService.getAccessToken()).thenReturn("test-fyers-token");
        when(mockAuthService.getClientId()).thenReturn("TESTAPP-100");

        mockSymbolMaster = mock(FyersSymbolMasterService.class);

        String baseUrl = mockWebServer.url("/").toString().replaceAll("/$", "");
        WebClient.Builder builder = WebClient.builder().baseUrl(baseUrl);
        client = new FyersServiceClient(builder, mockAuthService, mockSymbolMaster, baseUrl);
    }

    @AfterEach
    void tearDown() throws IOException {
        mockWebServer.shutdown();
    }

    // -----------------------------------------------------------------------
    // Candles
    // -----------------------------------------------------------------------

    @Test
    void parsesCandleArrayResponse() {
        mockWebServer.enqueue(new MockResponse()
            .setBody("""
                {
                  "s": "success",
                  "candles": [
                    [1705272600, 2500.0, 2550.5, 2490.0, 2530.0, 5000000],
                    [1704927000, 2450.0, 2510.0, 2440.0, 2500.0, 4800000]
                  ]
                }
                """)
            .addHeader("Content-Type", "application/json")
        );

        List<CandleData> candles = client.fetchCandlesList("RELIANCE",
            LocalDate.of(2024, 1, 10), LocalDate.of(2024, 1, 15));

        assertThat(candles).hasSize(2);
        assertThat(candles.get(0).symbol()).isEqualTo("RELIANCE");
        assertThat(candles.get(0).open()).isEqualByComparingTo(new BigDecimal("2500.0"));
        assertThat(candles.get(0).high()).isEqualByComparingTo(new BigDecimal("2550.5"));
        assertThat(candles.get(0).low()).isEqualByComparingTo(new BigDecimal("2490.0"));
        assertThat(candles.get(0).close()).isEqualByComparingTo(new BigDecimal("2530.0"));
        assertThat(candles.get(0).volume()).isEqualTo(5000000L);
    }

    @Test
    void retriesRateLimitedHistoricalRequestWithBackoff() {
        mockWebServer.enqueue(new MockResponse().setResponseCode(429));
        mockWebServer.enqueue(new MockResponse()
            .setBody("{\"s\":\"success\",\"candles\":[[1705272600,2500,2550,2490,2530,5000000]]}")
            .addHeader("Content-Type", "application/json"));

        List<CandleData> candles = client.fetchCandlesList("RELIANCE",
            LocalDate.of(2024, 1, 15), LocalDate.of(2024, 1, 15));

        assertThat(candles).hasSize(1);
        assertThat(mockWebServer.getRequestCount()).isEqualTo(2);
    }

    @Test
    void fetchCandleReturnsSingleDayCandle() {
        mockWebServer.enqueue(new MockResponse()
            .setBody("""
                {"s":"success","candles":[
                  [1705272600,2500.0,2550.5,2490.0,2530.0,5000000]
                ]}
                """)
            .addHeader("Content-Type", "application/json")
        );

        CandleData candle = client.fetchCandle("RELIANCE", LocalDate.of(2024, 1, 15));

        assertThat(candle).isNotNull();
        assertThat(candle.symbol()).isEqualTo("RELIANCE");
    }

    @Test
    void fetchCandleReturnsEmptyWhenNoToken() {
        when(mockAuthService.getAccessToken()).thenReturn(null);

        List<CandleData> candles = client.fetchCandlesList("RELIANCE",
            LocalDate.of(2024, 1, 10), LocalDate.of(2024, 1, 15));

        assertThat(candles).isEmpty();
    }

    @Test
    void fetchCandleReturnsEmptyWhenApiReturnsError() {
        mockWebServer.enqueue(new MockResponse()
            .setBody("{\"s\":\"error\",\"message\":\"Invalid symbol\"}")
            .addHeader("Content-Type", "application/json")
        );

        List<CandleData> candles = client.fetchCandlesList("INVALID",
            LocalDate.of(2024, 1, 10), LocalDate.of(2024, 1, 15));

        assertThat(candles).isEmpty();
    }

    @Test
    void fetchCandleReturnsEmptyWhenApiReturnsEmptyCandles() {
        mockWebServer.enqueue(new MockResponse()
            .setBody("{\"s\":\"success\",\"candles\":[]}")
            .addHeader("Content-Type", "application/json")
        );

        CandleData candle = client.fetchCandle("RELIANCE", LocalDate.of(2024, 1, 15));
        assertThat(candle).isNull();
    }

    @Test
    void requestIncludesTokenAuthHeader() throws InterruptedException {
        mockWebServer.enqueue(new MockResponse()
            .setBody("""
                {"s":"success","candles":[
                  [1705272600,100.0,110.0,90.0,105.0,1000]
                ]}
                """)
            .addHeader("Content-Type", "application/json")
        );

        client.fetchCandle("RELIANCE", LocalDate.of(2024, 1, 15));

        var recordedRequest = mockWebServer.takeRequest();
        assertThat(recordedRequest.getHeader("Authorization")).isEqualTo("TESTAPP-100:test-fyers-token");
    }

    @Test
    void fetchesCorrectFyersSymbolFormat() throws InterruptedException {
        mockWebServer.enqueue(new MockResponse()
            .setBody("{\"s\":\"success\",\"candles\":[]}")
            .addHeader("Content-Type", "application/json")
        );

        client.fetchCandle("RELIANCE", LocalDate.of(2024, 1, 15));

        var request = mockWebServer.takeRequest();
        assertThat(request.getPath()).contains("symbol=NSE:RELIANCE-EQ");
        assertThat(request.getPath()).contains("resolution=D");
    }

    @Test
    void fetchesBseHistoryUsingBseInstrumentSymbol() throws InterruptedException {
        mockWebServer.enqueue(new MockResponse()
            .setBody("{\"s\":\"success\",\"candles\":[]}")
            .addHeader("Content-Type", "application/json"));

        client.fetchCandlesList("BSECO", "BSE", LocalDate.of(2024, 1, 15), LocalDate.of(2024, 1, 15));

        assertThat(mockWebServer.takeRequest().getPath()).contains("symbol=BSE:BSECO-EQ");
    }

    @Test
    void isConnectedDelegatesToAuthService() {
        when(mockAuthService.validateToken()).thenReturn(true);
        assertThat(client.isConnected()).isTrue();

        when(mockAuthService.validateToken()).thenReturn(false);
        assertThat(client.isConnected()).isFalse();
    }

    @Test
    void fetchLatestCandleReturnsMostRecent() {
        mockWebServer.enqueue(new MockResponse()
            .setBody("""
                {
                  "s": "success",
                  "candles": [
                    [1705272600, 2500.0, 2550.5, 2490.0, 2530.0, 5000000],
                    [1705359000, 2520.0, 2560.0, 2510.0, 2545.0, 5200000]
                  ]
                }
                """)
            .addHeader("Content-Type", "application/json")
        );

        CandleData latest = client.fetchLatestCandle("RELIANCE");

        assertThat(latest).isNotNull();
        assertThat(latest.close()).isEqualByComparingTo(new BigDecimal("2545.0"));
    }

    @Test
    void fetchInstrumentDetailsUsesSymbolMasterWhenAvailable() {
        when(mockSymbolMaster.findByTradingSymbol("RELIANCE"))
            .thenReturn(Optional.of(symbolEntity("RELIANCE", "Reliance Industries", "INE002A01018")));

        InstrumentDetails details = client.fetchInstrumentDetails("RELIANCE");

        assertThat(details).isNotNull();
        assertThat(details.symbol()).isEqualTo("RELIANCE");
        assertThat(details.name()).isEqualTo("Reliance Industries");
        assertThat(details.isin()).isEqualTo("INE002A01018");
    }

    @Test
    void fetchInstrumentDetailsFallsBackWhenNotInSymbolMaster() {
        when(mockSymbolMaster.findByTradingSymbol("UNKNOWN")).thenReturn(Optional.empty());

        InstrumentDetails details = client.fetchInstrumentDetails("UNKNOWN");

        assertThat(details).isNotNull();
        assertThat(details.symbol()).isEqualTo("UNKNOWN");
    }

    @Test
    void fetchAllStockSymbolsDelegatesToSymbolMaster() {
        when(mockSymbolMaster.allTradingSymbols()).thenReturn(List.of("RELIANCE", "TCS"));

        assertThat(client.fetchAllStockSymbols()).containsExactly("RELIANCE", "TCS");
    }

    @Test
    void searchSymbolsDelegatesToSymbolMaster() {
        when(mockSymbolMaster.search("TCS", 20))
            .thenReturn(List.of(symbolEntity("TCS", "Tata Consultancy Services", "INE467B01029")));

        List<SearchResult> results = client.searchSymbols("TCS");

        assertThat(results).hasSize(1);
        assertThat(results.get(0).symbol()).isEqualTo("TCS");
    }

    // -----------------------------------------------------------------------
    // 401 / auth-error retry behavior
    // -----------------------------------------------------------------------

    @Test
    void refreshesTokenOnceThenSucceedsAfter401() {
        mockWebServer.enqueue(new MockResponse().setResponseCode(401));
        mockWebServer.enqueue(new MockResponse()
            .setBody("{\"s\":\"success\",\"candles\":[[1705272600,100.0,110.0,90.0,105.0,1000]]}")
            .addHeader("Content-Type", "application/json")
        );

        when(mockAuthService.getAccessToken())
            .thenReturn("expired-token")
            .thenReturn("refreshed-token");

        List<CandleData> candles = client.fetchCandlesList("RELIANCE",
            LocalDate.of(2024, 1, 15), LocalDate.of(2024, 1, 15));

        assertThat(candles).hasSize(1);
        verify(mockAuthService).refreshToken();
        assertThat(mockWebServer.getRequestCount()).isEqualTo(2);
    }

    @Test
    void doesNotRetryInfinitelyOnRepeated401() {
        mockWebServer.enqueue(new MockResponse().setResponseCode(401));
        mockWebServer.enqueue(new MockResponse().setResponseCode(401));

        List<CandleData> candles = client.fetchCandlesList("RELIANCE",
            LocalDate.of(2024, 1, 15), LocalDate.of(2024, 1, 15));

        assertThat(candles).isEmpty();
        verify(mockAuthService).refreshToken();
        assertThat(mockWebServer.getRequestCount()).isEqualTo(2);
    }

    @Test
    void refreshesOnBodyErrorCodeEvenWithHttp200() {
        mockWebServer.enqueue(new MockResponse()
            .setBody("{\"s\":\"error\",\"code\":-16,\"message\":\"Could not authenticate the user\"}")
            .addHeader("Content-Type", "application/json")
        );
        mockWebServer.enqueue(new MockResponse()
            .setBody("{\"s\":\"success\",\"candles\":[[1705272600,100.0,110.0,90.0,105.0,1000]]}")
            .addHeader("Content-Type", "application/json")
        );

        List<CandleData> candles = client.fetchCandlesList("RELIANCE",
            LocalDate.of(2024, 1, 15), LocalDate.of(2024, 1, 15));

        assertThat(candles).hasSize(1);
        verify(mockAuthService).refreshToken();
    }

    // -----------------------------------------------------------------------
    // Chunking
    // -----------------------------------------------------------------------

    @Test
    void chunksRangesLongerThanOneYearIntoMultipleRequests() {
        for (int i = 0; i < 3; i++) {
            mockWebServer.enqueue(new MockResponse()
                .setBody("{\"s\":\"success\",\"candles\":[]}")
                .addHeader("Content-Type", "application/json"));
        }

        // ~800 days -> 365 + 365 + 70 -> 3 chunked requests
        client.fetchCandlesList("RELIANCE", LocalDate.of(2022, 1, 1), LocalDate.of(2024, 3, 10));

        assertThat(mockWebServer.getRequestCount()).isEqualTo(3);
    }

    // -----------------------------------------------------------------------
    // Quotes
    // -----------------------------------------------------------------------

    @Test
    void fetchQuoteUsesCorrectPathAndParsesNumericFields() throws InterruptedException {
        mockWebServer.enqueue(new MockResponse()
            .setBody("""
                {
                  "s": "success",
                  "d": [
                    {
                      "n": "NSE:RELIANCE-EQ",
                      "s": "ok",
                      "v": {
                        "ch": 12.30, "chp": 0.94, "lp": 1316.50,
                        "high_price": 1325.00, "low_price": 1300.00,
                        "prev_close_price": 1304.20, "volume": 5432100,
                        "short_name": "RELIANCE", "symbol": "RELIANCE-EQ"
                      }
                    }
                  ]
                }
                """)
            .addHeader("Content-Type", "application/json")
        );

        QuoteData quote = client.fetchQuote("RELIANCE");

        assertThat(quote).isNotNull();
        assertThat(quote.symbol()).isEqualTo("RELIANCE");
        assertThat(quote.regularMarketPrice()).isEqualByComparingTo(new BigDecimal("1316.50"));
        assertThat(quote.regularMarketChange()).isEqualByComparingTo(new BigDecimal("12.30"));
        assertThat(quote.regularMarketVolume()).isEqualTo(5432100L);

        var request = mockWebServer.takeRequest();
        assertThat(request.getPath()).contains("/data/quotes");
        assertThat(request.getPath()).contains("symbols=NSE:RELIANCE-EQ");
    }

    @Test
    void fetchQuotesBatchMapsOrderAndStripsSymbolFormat() {
        mockWebServer.enqueue(new MockResponse()
            .setBody("""
                {
                  "s": "success",
                  "d": [
                    { "n": "NSE:RELIANCE-EQ", "v": { "lp": 1300.0, "symbol": "RELIANCE-EQ", "short_name": "RELIANCE" } },
                    { "n": "NSE:TCS-EQ", "v": { "lp": 3500.0, "symbol": "TCS-EQ", "short_name": "TCS" } },
                    { "n": "NSE:INFY-EQ", "v": { "lp": 1450.0, "symbol": "INFY-EQ", "short_name": "INFY" } }
                  ]
                }
                """)
            .addHeader("Content-Type", "application/json")
        );

        List<QuoteData> quotes = client.fetchQuotes(List.of("RELIANCE", "TCS", "INFY"));

        assertThat(quotes).hasSize(3);
        assertThat(quotes.get(0).symbol()).isEqualTo("RELIANCE");
        assertThat(quotes.get(1).symbol()).isEqualTo("TCS");
        assertThat(quotes.get(2).symbol()).isEqualTo("INFY");
    }

    @Test
    void fetchPriceBandUsesAuthoritativeFyersCircuitFields() {
        mockWebServer.enqueue(new MockResponse()
            .setBody("""
                {"s":"success","d":[{"n":"NSE:RELIANCE-EQ","v":{
                  "symbol":"NSE:RELIANCE-EQ","lp":2500,"lower_ckt":2250,"upper_ckt":2750
                }}]}
                """)
            .addHeader("Content-Type", "application/json"));

        PriceBand band = client.fetchPriceBand("RELIANCE", LocalDate.of(2026, 1, 5));

        assertThat(band).isNotNull();
        assertThat(band.lowerLimit()).isEqualByComparingTo("2250");
        assertThat(band.upperLimit()).isEqualByComparingTo("2750");
    }

    @Test
    void fetchQuotesReturnsEmptyOnErrorBody() {
        mockWebServer.enqueue(new MockResponse()
            .setBody("{\"s\":\"error\"}")
            .addHeader("Content-Type", "application/json")
        );

        List<QuoteData> quotes = client.fetchQuotes(List.of("RELIANCE"));

        assertThat(quotes).isEmpty();
    }

    private static com.swingtrade.data.entity.FyersSymbolEntity symbolEntity(
            String tradingSymbol, String name, String isin) {
        com.swingtrade.data.entity.FyersSymbolEntity e = new com.swingtrade.data.entity.FyersSymbolEntity();
        e.setFyToken("1");
        e.setFyersSymbol("NSE:" + tradingSymbol + "-EQ");
        e.setTradingSymbol(tradingSymbol);
        e.setName(name);
        e.setLotSize(1);
        e.setTickSize(new BigDecimal("0.05"));
        e.setIsin(isin);
        return e;
    }
}
