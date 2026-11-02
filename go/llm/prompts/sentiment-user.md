Analyse the following for a swing trade entry decision on {symbol}.

Recent news headlines (last 7 days):
{newsContent}

{marketContext}

Task: Determine if news sentiment supports a 1-4 week swing trade entry.

CRITICAL: Respond with ONLY a JSON object. No explanation, no reasoning, no other text.
Start your response with { and end with }.

{
  "score": "POSITIVE|NEUTRAL|NEGATIVE",
  "confidence": 0.0-1.0,
  "summary": "2 sentence max reasoning",
  "red_flags": ["[article index] specific risk, or [] if none"],
  "catalysts": ["[article index] upcoming catalyst, or [] if none"]
}

Every red flag and catalyst MUST cite at least one supplied article using its
exact index in square brackets, such as [1] or [2]. Do not infer or invent
details. If no supplied article supports an item, omit it.

/no_think
