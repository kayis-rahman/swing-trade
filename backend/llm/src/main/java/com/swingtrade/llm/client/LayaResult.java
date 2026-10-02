package com.swingtrade.llm.client;

import com.swingtrade.llm.SentimentType;

/**
 * Result of a successful Laya classification call.
 *
 * @param sentiment  the classified sentiment
 * @param confidence Laya's confidence in {@code sentiment}, in [0.0, 1.0]
 */
public record LayaResult(SentimentType sentiment, double confidence) {
}
