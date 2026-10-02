package com.swingtrade.llm.service;

import tools.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.Test;
import org.mockito.MockedStatic;

import java.net.http.HttpClient;
import java.net.http.HttpResponse;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

class RedditIndiaInvestmentsSourceTest {
    @Test
    void returnsMatchingDeduplicatedPostsUpToConfiguredLimit() throws Exception {
        HttpClient.Builder builder = mock(HttpClient.Builder.class);
        HttpClient client = mock(HttpClient.class);
        HttpResponse<String> response = mock(HttpResponse.class);
        when(builder.version(any())).thenReturn(builder);
        when(builder.followRedirects(any())).thenReturn(builder);
        when(builder.build()).thenReturn(client);
        when(response.statusCode()).thenReturn(200);
        when(response.body()).thenReturn("{\"data\":{\"children\":[" +
                "{\"data\":{\"title\":\"TCS outlook improves\",\"permalink\":\"/r/IndiaInvestments/a\",\"selftext\":\"Strong demand\",\"created_utc\":1726400000}}," +
                "{\"data\":{\"title\":\"TCS outlook improves\",\"permalink\":\"/r/IndiaInvestments/b\",\"selftext\":\"Duplicate\",\"created_utc\":1726400001}}," +
                "{\"data\":{\"title\":\"Unrelated market chat\",\"permalink\":\"/r/IndiaInvestments/c\",\"selftext\":\"No match\",\"created_utc\":1726400002}}]}}");
        doAnswer(invocation -> response).when(client).send(any(), any());

        try (MockedStatic<HttpClient> http = mockStatic(HttpClient.class)) {
            http.when(HttpClient::newBuilder).thenReturn(builder);
            var source = new RedditIndiaInvestmentsSource("", "", "", "", true, 10, new ObjectMapper());

            List<com.swingtrade.domain.NewsArticle> articles = source.fetch("TCS");
            assertThat(articles).singleElement().satisfies(article -> {
                assertThat(article.title()).isEqualTo("TCS outlook improves");
                assertThat(article.description()).isEqualTo("Strong demand");
                assertThat(article.source()).isEqualTo("reddit");
            });
        }
    }
}
