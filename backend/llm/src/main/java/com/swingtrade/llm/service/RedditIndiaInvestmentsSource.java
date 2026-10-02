package com.swingtrade.llm.service;

import tools.jackson.databind.JsonNode;
import tools.jackson.databind.ObjectMapper;
import com.swingtrade.domain.NewsArticle;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.HashSet;
import java.util.Locale;
import java.util.List;
import java.util.Set;
import java.util.concurrent.atomic.AtomicReference;

/**
 * Fetches stock discussions from r/IndiaInvestments Reddit.
 * OAuth2: password grant (first-party) → client-credentials (standard bot) → unauthenticated fallback.
 */
@Service
public class RedditIndiaInvestmentsSource implements NewsSource {

    private static final Logger log = LoggerFactory.getLogger(RedditIndiaInvestmentsSource.class);
    private static final String SUBREDDIT = "IndiaInvestments";
    private static final String SEARCH_URL = "https://www.reddit.com/r/" + SUBREDDIT + "/search.json?q=";
    private static final String TOKEN_URL = "https://www.reddit.com/api/v1/access_token";
    private static final DateTimeFormatter DATE_FMT = DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss.SSSZ");

    private final String clientId;
    private final String clientSecret;
    private final String username;
    private final String password;
    private final boolean enabled;
    private final int maxArticles;
    private final AtomicReference<String> accessToken = new AtomicReference<>();
    private final ObjectMapper objectMapper;
    private final HttpClient httpClient;

    public RedditIndiaInvestmentsSource(
            @Value("${news.reddit.client-id:}") String clientId,
            @Value("${news.reddit.client-secret:}") String clientSecret,
            @Value("${news.reddit.username:}") String username,
            @Value("${news.reddit.password:}") String password,
            @Value("${news.source.reddit.enabled:true}") boolean enabled,
            @Value("${news.source.reddit.max-articles:15}") int maxArticles,
            ObjectMapper objectMapper) {

        this.clientId = clientId;
        this.clientSecret = clientSecret;
        this.username = username;
        this.password = password;
        this.enabled = enabled;
        this.maxArticles = maxArticles;
        this.objectMapper = objectMapper;
        this.httpClient = HttpClient.newBuilder()
                .version(HttpClient.Version.HTTP_1_1)
                .followRedirects(HttpClient.Redirect.NORMAL)
                .build();

        if (!enabled) {
            log.info("Reddit source disabled by configuration");
            return;
        }

        // Try password grant first (first-party apps), then client-credentials (standard bots)
        if (!clientId.isBlank() && !clientSecret.isBlank()) {
            if (!username.isBlank() && !password.isBlank()) {
                String token = fetchPasswordToken();
                if (token != null) {
                    accessToken.set(token);
                    log.info("Reddit OAuth via password grant successful for {}", SUBREDDIT);
                }
            }
            if (accessToken.get() == null) {
                String token = fetchClientCredentialsToken();
                if (token != null) {
                    accessToken.set(token);
                    log.info("Reddit OAuth via client-credentials successful for {}", SUBREDDIT);
                }
            }
        }

        if (accessToken.get() == null) {
            log.warn("Reddit OAuth unavailable — running unauthenticated (will likely be rate-limited)");
        }
    }

    @Override
    public String type() {
        return "reddit";
    }

    @Override
    public List<NewsArticle> fetch(String symbol) {
        if (!enabled) return List.of();
        String query = symbol + " India stock";
        String url = SEARCH_URL + URLEncoder.encode(query, StandardCharsets.UTF_8) + "&limit=25";

        try {
            HttpRequest.Builder req = HttpRequest.newBuilder()
                    .uri(URI.create(url))
                    .header("User-Agent", "SwingTrade/1.0")
                    .timeout(java.time.Duration.ofSeconds(8));

            String token = accessToken.get();
            if (token != null) {
                req.header("Authorization", "Bearer " + token);
            }

            HttpResponse<String> resp = httpClient.send(
                    req.build(),
                    HttpResponse.BodyHandlers.ofString());

            if (resp.statusCode() == 429) {
                log.warn("Reddit rate-limited for {}. Token may have expired.", symbol);
                accessToken.set(null);
                return List.of();
            }

            if (resp.statusCode() != 200) {
                log.warn("Reddit HTTP {} for {} (body: {})", resp.statusCode(), symbol,
                        resp.body().length() > 200 ? resp.body().substring(0, 200) : resp.body());
                return List.of();
            }

            JsonNode root = objectMapper.readTree(resp.body());
            JsonNode children = root.path("data").path("children");

            Set<String> seen = new HashSet<>();
            List<NewsArticle> articles = new ArrayList<>();

            for (JsonNode child : children) {
                JsonNode post = child.path("data");
                String title = post.path("title").asText().trim();
                String link = "https://www.reddit.com" + post.path("permalink").asText();
                String selfText = post.path("selftext").asText().trim();
                long created = post.path("created_utc").asLong(0);

                if (title.isBlank()) continue;
                if (!title.toLowerCase(Locale.ENGLISH).contains(symbol.toLowerCase(Locale.ENGLISH))) continue;

                String key = title.toLowerCase(Locale.ENGLISH);
                if (seen.add(key) && articles.size() < maxArticles) {
                    ZonedDateTime pubDate = created > 0
                            ? ZonedDateTime.ofInstant(
                                    java.time.Instant.ofEpochSecond(created),
                                    java.time.ZoneOffset.UTC)
                            : ZonedDateTime.now();

                    articles.add(NewsArticle.builder()
                            .symbol(symbol)
                            .title(title)
                            .link(link)
                            .description(selfText.length() > 500 ? selfText.substring(0, 500) : selfText)
                            .publishedDate(pubDate)
                            .source("reddit")
                            .rawContent(selfText)
                            .build());
                }
            }

            log.debug("Reddit: {} articles for {}", articles.size(), symbol);
            return articles;

        } catch (Exception e) {
            log.warn("Reddit fetch failed for {}: {}", symbol, e.getMessage());
            return List.of();
        }
    }

    private String fetchPasswordToken() {
        try {
            String auth = java.util.Base64.getEncoder()
                    .encodeToString((clientId + ":" + clientSecret).getBytes(StandardCharsets.UTF_8));

            HttpResponse<String> resp = httpClient.send(
                    HttpRequest.newBuilder()
                            .uri(URI.create(TOKEN_URL))
                            .header("Authorization", "Basic " + auth)
                            .header("User-Agent", "SwingTrade/1.0")
                            .header("Content-Type", "application/x-www-form-urlencoded")
                            .POST(HttpRequest.BodyPublishers.ofString(
                                    "grant_type=password&username=" + URLEncoder.encode(username, StandardCharsets.UTF_8)
                                            + "&password=" + URLEncoder.encode(password, StandardCharsets.UTF_8)))
                            .build(),
                    HttpResponse.BodyHandlers.ofString());

            if (resp.statusCode() == 200) {
                JsonNode root = objectMapper.readTree(resp.body());
                return root.path("access_token").asText();
            }
        } catch (Exception e) {
            log.debug("Password grant failed: {}", e.getMessage());
        }
        return null;
    }

    private String fetchClientCredentialsToken() {
        try {
            String auth = java.util.Base64.getEncoder()
                    .encodeToString((clientId + ":" + clientSecret).getBytes(StandardCharsets.UTF_8));

            HttpResponse<String> resp = httpClient.send(
                    HttpRequest.newBuilder()
                            .uri(URI.create(TOKEN_URL))
                            .header("Authorization", "Basic " + auth)
                            .header("User-Agent", "SwingTrade/1.0")
                            .header("Content-Type", "application/x-www-form-urlencoded")
                            .POST(HttpRequest.BodyPublishers.ofString("grant_type=client_credentials"))
                            .build(),
                    HttpResponse.BodyHandlers.ofString());

            if (resp.statusCode() == 200) {
                JsonNode root = objectMapper.readTree(resp.body());
                return root.path("access_token").asText();
            }
        } catch (Exception e) {
            log.debug("Client-credentials grant failed: {}", e.getMessage());
        }
        return null;
    }
}
