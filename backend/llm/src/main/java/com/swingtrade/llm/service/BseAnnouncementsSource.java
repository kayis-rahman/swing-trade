package com.swingtrade.llm.service;

import com.swingtrade.domain.NewsArticle;
import com.swingtrade.llm.service.StructuredFiling.FilingType;
import org.jsoup.Jsoup;
import org.jsoup.nodes.Document;
import org.jsoup.nodes.Element;
import org.jsoup.select.Elements;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.time.LocalDate;
import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.util.Locale;
import java.util.stream.Collectors;
import java.util.ArrayList;
import java.util.List;

/**
 * Fetches corporate announcements from BSE India website.
 */
@Service
public class BseAnnouncementsSource implements NewsSource {

    private static final Logger log = LoggerFactory.getLogger(BseAnnouncementsSource.class);
    private static final String URL = "https://www.bseindia.com/Corporate";
    private static final ZoneId IST = ZoneId.of("Asia/Kolkata");

    private final int maxArticles;
    private final boolean enabled;

    public BseAnnouncementsSource(
            @Value("${news.source.bse.max-articles:10}") int maxArticles,
            @Value("${news.source.bse.enabled:true}") boolean enabled) {
        this.maxArticles = maxArticles;
        this.enabled = enabled;
        if (!enabled) log.info("BSE announcements source disabled by configuration");
    }

    @Override
    public String type() {
        return "bse";
    }

    @Override
    public List<NewsArticle> fetch(String symbol) {
        if (!enabled) return List.of();
        List<StructuredFiling> filings = fetchFilings(symbol);
        return filings.stream()
                .map(f -> NewsArticle.builder()
                        .symbol(symbol)
                        .title(f.title())
                        .link(f.link())
                        .description(f.description())
                        .publishedDate(f.date() != null ? f.date().atStartOfDay(IST) : ZonedDateTime.now(IST))
                        .source("bse")
                        .rawContent(f.description())
                        .build())
                .limit(maxArticles)
                .collect(Collectors.toList());
    }

    @Override
    public List<StructuredFiling> fetchFilings(String symbol) {
        if (!enabled) return List.of();
        List<StructuredFiling> filings = new ArrayList<>();

        try {
            Document doc = Jsoup.connect(URL)
                    .userAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
                    .timeout(8000)
                    .data("sharecode", symbol)
                    .data("cmd", "search")
                    .post();

            for (Element row : doc.select("table tr")) {
                Elements cells = row.select("td");
                if (cells.size() < 3) continue;

                String dateStr = cells.get(0).text().trim();
                Element titleEl = cells.get(1).selectFirst("a");
                String title = titleEl != null ? titleEl.text().trim() : cells.get(1).text().trim();
                String link = titleEl != null && !titleEl.attr("href").isEmpty()
                        ? titleEl.attr("href")
                        : "https://www.bseindia.com";

                if (title.isBlank()) continue;
                LocalDate ld = parseLocalDate(dateStr);
                if (ld == null) continue;

                filings.add(new StructuredFiling(
                        classify(title), ld, title, "",
                        link.startsWith("http") ? link : "https://www.bseindia.com" + link));
            }
        } catch (Exception e) {
            log.warn("BSE fetch failed for '{}': {}", symbol, e.getMessage());
        }

        return filings;
    }

    private LocalDate parseLocalDate(String s) {
        if (s == null || s.isBlank()) return null;
        try {
            return LocalDate.parse(s.trim(), java.time.format.DateTimeFormatter.ofPattern("dd-MMM-yyyy", Locale.ENGLISH));
        } catch (Exception e) {
            return null;
        }
    }

    private FilingType classify(String title) {
        String t = title.toLowerCase(Locale.ENGLISH);
        if (t.contains("board meeting") || t.contains("boardmeet")) return FilingType.BOARD_MEETING;
        if (t.contains("shareholding") || t.contains("share holding")) return FilingType.SHAREHOLDING;
        if (t.contains("dividend")) return FilingType.DIVIDEND;
        if (t.contains("split") || t.contains("bonus") || t.contains("right")) return FilingType.CORPORATE_ACTION;
        if (t.contains("result") || t.contains("quarterly") || t.contains("quarter")) return FilingType.PERFORMANCE_RESULT;
        return FilingType.OTHER;
    }
}
