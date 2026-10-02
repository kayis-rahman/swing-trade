package com.swingtrade.llm.service;

import org.jsoup.Connection;
import org.jsoup.Jsoup;
import org.jsoup.nodes.Document;
import org.junit.jupiter.api.Test;
import org.mockito.MockedStatic;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.*;

class BseAnnouncementsSourceTest {
    @Test
    void parsesHtmlAnnouncementsAndClassifiesFiling() throws Exception {
        Connection connection = mock(Connection.class);
        Document document = Jsoup.parse("<table><tr><td>16-Sep-2026</td><td><a href='/notice.pdf'>Dividend declared</a></td><td>valid</td></tr></table>");
        when(connection.userAgent(anyString())).thenReturn(connection);
        when(connection.timeout(anyInt())).thenReturn(connection);
        when(connection.data(anyString(), anyString())).thenReturn(connection);
        when(connection.post()).thenReturn(document);

        try (MockedStatic<Jsoup> jsoup = mockStatic(Jsoup.class)) {
            jsoup.when(() -> Jsoup.connect(anyString())).thenReturn(connection);
            var filings = new BseAnnouncementsSource(10, true).fetchFilings("500112");

            assertThat(filings).singleElement().satisfies(filing -> {
                assertThat(filing.type()).isEqualTo(StructuredFiling.FilingType.DIVIDEND);
                assertThat(filing.date().toString()).isEqualTo("2026-09-16");
                assertThat(filing.link()).isEqualTo("https://www.bseindia.com/notice.pdf");
            });
        }
    }
}
