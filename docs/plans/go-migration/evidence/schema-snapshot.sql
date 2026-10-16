--
-- PostgreSQL database dump
--

\restrict gBPHXGmUNfL0bpg4hk0VpTJa4cnVeCBgT47hQQ8BeAo1sYa8ZzAT8l8CDLgUTDd

-- Dumped from database version 16.15 (Homebrew)
-- Dumped by pg_dump version 16.15 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: app_settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_settings (
    id bigint NOT NULL,
    key character varying(64) NOT NULL,
    value text,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    version integer
);


--
-- Name: app_settings_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.app_settings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: app_settings_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.app_settings_id_seq OWNED BY public.app_settings.id;


--
-- Name: backtest_result; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.backtest_result (
    id bigint NOT NULL,
    symbol character varying(255) NOT NULL,
    run_date date NOT NULL,
    total_trades integer NOT NULL,
    winning_trades integer NOT NULL,
    losing_trades integer NOT NULL,
    win_rate double precision NOT NULL,
    avg_gain_pct double precision NOT NULL,
    avg_loss_pct double precision NOT NULL,
    max_drawdown_pct double precision NOT NULL,
    sharpe_ratio double precision NOT NULL,
    total_return double precision NOT NULL,
    expectancy double precision NOT NULL,
    profit_factor double precision NOT NULL,
    has_enough_data boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: backtest_result_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.backtest_result_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: backtest_result_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.backtest_result_id_seq OWNED BY public.backtest_result.id;


--
-- Name: candidate_history_eligibility; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.candidate_history_eligibility (
    symbol character varying(32) NOT NULL,
    candle_count integer DEFAULT 0 NOT NULL,
    first_available_date date,
    last_available_date date,
    source_outcome character varying(32) NOT NULL,
    invalid_rows integer DEFAULT 0 NOT NULL,
    retry_after date,
    error_message text,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    version integer DEFAULT 0 NOT NULL
);


--
-- Name: candidate_scan_results; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.candidate_scan_results (
    id bigint NOT NULL,
    run_id uuid NOT NULL,
    symbol character varying(32) NOT NULL,
    data_status character varying(24) NOT NULL,
    candle_count integer DEFAULT 0 NOT NULL,
    signal_type character varying(16),
    total_trades integer,
    win_rate double precision,
    total_return double precision,
    max_drawdown_pct double precision,
    qualified boolean DEFAULT false NOT NULL,
    activated boolean DEFAULT false NOT NULL,
    reason text,
    error_message text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    version integer DEFAULT 0 NOT NULL,
    oos_start_date date,
    oos_end_date date,
    oos_total_trades integer,
    oos_win_rate double precision,
    oos_total_return double precision,
    oos_max_drawdown_pct double precision,
    source_outcome character varying(32),
    invalid_rows integer DEFAULT 0 NOT NULL,
    first_available_date date,
    last_available_date date,
    retry_after date,
    strategy_buy_count integer DEFAULT 0 NOT NULL,
    strategy_evaluation_count integer DEFAULT 0 NOT NULL
);


--
-- Name: candidate_scan_results_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.candidate_scan_results_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: candidate_scan_results_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.candidate_scan_results_id_seq OWNED BY public.candidate_scan_results.id;


--
-- Name: candidate_scan_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.candidate_scan_runs (
    id bigint NOT NULL,
    run_id uuid NOT NULL,
    status character varying(16) NOT NULL,
    total_symbols integer DEFAULT 0 NOT NULL,
    completed_symbols integer DEFAULT 0 NOT NULL,
    failed_symbols integer DEFAULT 0 NOT NULL,
    qualified_symbols integer DEFAULT 0 NOT NULL,
    started_at timestamp without time zone NOT NULL,
    completed_at timestamp without time zone,
    error_message text,
    version integer DEFAULT 0 NOT NULL,
    orchestration_status character varying(16) DEFAULT 'NOT_REQUIRED'::character varying NOT NULL,
    orchestration_job_run_id uuid,
    orchestration_error text,
    scan_scope character varying(16),
    scan_trigger character varying(16)
);


--
-- Name: candidate_scan_runs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.candidate_scan_runs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: candidate_scan_runs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.candidate_scan_runs_id_seq OWNED BY public.candidate_scan_runs.id;


--
-- Name: candidate_scan_strategy_outcomes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.candidate_scan_strategy_outcomes (
    candidate_scan_result_id bigint NOT NULL,
    variant_id character varying(128) NOT NULL,
    signal_type character varying(16) NOT NULL,
    score numeric(18,8),
    detail text,
    backtest_total_trades integer,
    backtest_win_rate double precision,
    backtest_total_return double precision,
    oos_total_trades integer,
    oos_win_rate double precision,
    oos_total_return double precision,
    performance_status character varying(16) DEFAULT 'NOT_RUN'::character varying NOT NULL
);


--
-- Name: corporate_actions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.corporate_actions (
    id bigint NOT NULL,
    symbol character varying(20) NOT NULL,
    effective_date date NOT NULL,
    action_type character varying(40) NOT NULL,
    adjustment_factor numeric(20,8),
    cash_amount numeric(20,8),
    source character varying(80) NOT NULL,
    recorded_at timestamp with time zone NOT NULL,
    CONSTRAINT ck_corporate_action_cash_nonnegative CHECK (((cash_amount IS NULL) OR (cash_amount >= (0)::numeric))),
    CONSTRAINT ck_corporate_action_factor_positive CHECK (((adjustment_factor IS NULL) OR (adjustment_factor > (0)::numeric))),
    CONSTRAINT ck_corporate_action_value CHECK (((adjustment_factor IS NOT NULL) OR (cash_amount IS NOT NULL)))
);


--
-- Name: corporate_actions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.corporate_actions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: corporate_actions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.corporate_actions_id_seq OWNED BY public.corporate_actions.id;


--
-- Name: daily_loss_circuit_breaker_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.daily_loss_circuit_breaker_state (
    id bigint NOT NULL,
    circuit_open boolean DEFAULT false NOT NULL,
    circuit_opened_at timestamp without time zone,
    loss_at_open numeric(15,2),
    last_reset_date date NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    version integer,
    CONSTRAINT uq_single_row CHECK ((id = 1))
);


--
-- Name: daily_loss_circuit_breaker_state_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.daily_loss_circuit_breaker_state_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: daily_loss_circuit_breaker_state_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.daily_loss_circuit_breaker_state_id_seq OWNED BY public.daily_loss_circuit_breaker_state.id;


--
-- Name: flyway_schema_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.flyway_schema_history (
    installed_rank integer NOT NULL,
    version character varying(50),
    description character varying(200) NOT NULL,
    type character varying(20) NOT NULL,
    script character varying(1000) NOT NULL,
    checksum integer,
    installed_by character varying(100) NOT NULL,
    installed_on timestamp without time zone DEFAULT now() NOT NULL,
    execution_time integer NOT NULL,
    success boolean NOT NULL
);


--
-- Name: fyers_symbol_master; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.fyers_symbol_master (
    id bigint NOT NULL,
    fy_token character varying(32) NOT NULL,
    fyers_symbol character varying(64) NOT NULL,
    trading_symbol character varying(32) NOT NULL,
    name text,
    exchange character varying(16) DEFAULT 'NSE'::character varying NOT NULL,
    segment character varying(16),
    lot_size integer,
    tick_size numeric(10,4),
    isin character varying(13),
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    version integer
);


--
-- Name: fyers_symbol_master_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.fyers_symbol_master_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: fyers_symbol_master_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.fyers_symbol_master_id_seq OWNED BY public.fyers_symbol_master.id;


--
-- Name: gate_effectiveness_audit; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.gate_effectiveness_audit (
    id bigint NOT NULL,
    symbol character varying(20) NOT NULL,
    signal_date date NOT NULL,
    gate_name character varying(40) NOT NULL,
    verdict character varying(24) NOT NULL,
    reason text,
    recorded_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    strategy character varying(30) DEFAULT 'DEFAULT'::character varying NOT NULL
);


--
-- Name: gate_effectiveness_audit_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.gate_effectiveness_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: gate_effectiveness_audit_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.gate_effectiveness_audit_id_seq OWNED BY public.gate_effectiveness_audit.id;


--
-- Name: intraday_paper_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.intraday_paper_sessions (
    run_id character varying(36) NOT NULL,
    symbol character varying(32) NOT NULL,
    session_date date NOT NULL,
    created_at timestamp with time zone NOT NULL,
    result_json text NOT NULL
);


--
-- Name: job_run_stages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.job_run_stages (
    id bigint NOT NULL,
    run_id uuid NOT NULL,
    symbol character varying(16) NOT NULL,
    stage_name character varying(32) NOT NULL,
    status character varying(16) DEFAULT 'PENDING'::character varying NOT NULL,
    started_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    completed_at timestamp without time zone,
    duration_ms bigint,
    error_message text,
    log_details text,
    result_summary text,
    version integer,
    details text
);


--
-- Name: job_run_stages_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.job_run_stages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: job_run_stages_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.job_run_stages_id_seq OWNED BY public.job_run_stages.id;


--
-- Name: job_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.job_runs (
    id bigint NOT NULL,
    run_id uuid DEFAULT gen_random_uuid() NOT NULL,
    trigger_type character varying(16) DEFAULT 'MANUAL'::character varying NOT NULL,
    status character varying(32) DEFAULT 'RUNNING'::character varying NOT NULL,
    started_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    completed_at timestamp without time zone,
    symbols_count integer DEFAULT 0 NOT NULL,
    completed_count integer DEFAULT 0 NOT NULL,
    failed_count integer DEFAULT 0 NOT NULL,
    error_message text,
    version integer,
    candidate_scan_run_id uuid,
    trigger_options text
);


--
-- Name: job_runs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.job_runs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: job_runs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.job_runs_id_seq OWNED BY public.job_runs.id;


--
-- Name: kill_switch; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.kill_switch (
    id bigint DEFAULT 1 NOT NULL,
    active boolean DEFAULT false NOT NULL,
    enabled_at timestamp without time zone,
    reason text,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: llm_analysis_audit; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.llm_analysis_audit (
    id bigint NOT NULL,
    request_id character varying(36) NOT NULL,
    symbol character varying(20) NOT NULL,
    analysis_date date NOT NULL,
    provider character varying(32) NOT NULL,
    model_version character varying(255),
    prompt_hash character varying(64),
    system_prompt text,
    user_prompt text,
    raw_response text,
    parsed_score character varying(20),
    parsed_confidence double precision,
    status character varying(20) NOT NULL,
    error_message text,
    fallback_used boolean DEFAULT false NOT NULL,
    max_tokens integer,
    temperature double precision,
    started_at timestamp with time zone NOT NULL,
    completed_at timestamp with time zone,
    latency_ms bigint
);


--
-- Name: llm_analysis_audit_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.llm_analysis_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: llm_analysis_audit_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.llm_analysis_audit_id_seq OWNED BY public.llm_analysis_audit.id;


--
-- Name: llm_analysis_result; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.llm_analysis_result (
    id bigint NOT NULL,
    run_id uuid NOT NULL,
    symbol character varying(255) NOT NULL,
    analysis_date date NOT NULL,
    recommendation character varying(255),
    confidence double precision,
    narrative text,
    key_drivers text[],
    bullish_factors text[],
    bearish_factors text[],
    composite_score integer,
    composite_signal character varying(255),
    success boolean NOT NULL,
    fallback_used boolean DEFAULT false NOT NULL,
    error_message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: llm_analysis_result_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.llm_analysis_result_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: llm_analysis_result_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.llm_analysis_result_id_seq OWNED BY public.llm_analysis_result.id;


--
-- Name: news_articles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.news_articles (
    id bigint NOT NULL,
    symbol character varying(20) NOT NULL,
    source character varying(50) NOT NULL,
    title text NOT NULL,
    link text,
    summary text,
    published_at timestamp with time zone,
    raw_content text,
    created_at timestamp with time zone DEFAULT now(),
    version integer,
    article_key character varying(64),
    first_seen_at timestamp with time zone
);


--
-- Name: news_articles_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.news_articles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: news_articles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.news_articles_id_seq OWNED BY public.news_articles.id;


--
-- Name: news_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.news_items (
    id bigint NOT NULL,
    symbol character varying(10) NOT NULL,
    headline text NOT NULL,
    source character varying(50) NOT NULL,
    published_at timestamp without time zone NOT NULL,
    url text,
    raw_content text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    version integer
);


--
-- Name: news_items_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.news_items_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: news_items_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.news_items_id_seq OWNED BY public.news_items.id;


--
-- Name: nse_calendar_coverage; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.nse_calendar_coverage (
    calendar_year integer NOT NULL,
    status character varying(16) NOT NULL,
    source character varying(255),
    verified_at timestamp without time zone,
    CONSTRAINT nse_calendar_coverage_status_check CHECK (((status)::text = ANY ((ARRAY['VERIFIED'::character varying, 'PENDING'::character varying])::text[])))
);


--
-- Name: nse_holidays; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.nse_holidays (
    id bigint NOT NULL,
    holiday_date date NOT NULL,
    occasion character varying(128) NOT NULL,
    holiday_type character varying(32) DEFAULT 'FULL'::character varying NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    version integer,
    CONSTRAINT nse_holidays_holiday_type_check CHECK (((holiday_type)::text = ANY ((ARRAY['FULL'::character varying, 'PARTIAL'::character varying])::text[])))
);


--
-- Name: nse_holidays_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.nse_holidays_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: nse_holidays_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.nse_holidays_id_seq OWNED BY public.nse_holidays.id;


--
-- Name: ohlcv_candles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ohlcv_candles (
    id bigint NOT NULL,
    symbol character varying(10) NOT NULL,
    date date NOT NULL,
    open_price numeric(15,4),
    high_price numeric(15,4),
    low_price numeric(15,4),
    close_price numeric(15,4),
    volume bigint,
    adj_close_price numeric(15,4),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    version integer,
    timeframe character varying(10) DEFAULT 'D'::character varying NOT NULL,
    bar_time time without time zone DEFAULT '00:00:00'::time without time zone NOT NULL
);


--
-- Name: ohlcv_candles_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.ohlcv_candles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: ohlcv_candles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.ohlcv_candles_id_seq OWNED BY public.ohlcv_candles.id;


--
-- Name: paper_trading_orders; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.paper_trading_orders (
    id bigint NOT NULL,
    order_id character varying(32) NOT NULL,
    symbol character varying(16) NOT NULL,
    order_type character varying(16) DEFAULT 'MARKET'::character varying NOT NULL,
    direction character varying(10) NOT NULL,
    quantity integer NOT NULL,
    price numeric(15,2) NOT NULL,
    limit_price numeric(15,2),
    stop_price numeric(15,2),
    status character varying(16) DEFAULT 'PENDING'::character varying NOT NULL,
    commission numeric(15,4) DEFAULT 0 NOT NULL,
    executed_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    version integer,
    signal_id character varying(64),
    portfolio_id character varying(40)
);


--
-- Name: paper_trading_orders_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.paper_trading_orders_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: paper_trading_orders_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.paper_trading_orders_id_seq OWNED BY public.paper_trading_orders.id;


--
-- Name: paper_trading_portfolio; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.paper_trading_portfolio (
    id bigint NOT NULL,
    portfolio_id character varying(40) DEFAULT 'default'::character varying NOT NULL,
    initial_capital numeric(15,2) DEFAULT 10000000 NOT NULL,
    current_capital numeric(15,2) DEFAULT 10000000 NOT NULL,
    total_realized_pnl numeric(15,2) DEFAULT 0 NOT NULL,
    total_unrealized_pnl numeric(15,2) DEFAULT 0 NOT NULL,
    open_position_count integer DEFAULT 0 NOT NULL,
    max_concurrent_positions integer DEFAULT 5 NOT NULL,
    max_capital_per_position_pct numeric(5,2) DEFAULT 20 NOT NULL,
    commission_rate numeric(6,4) DEFAULT 0.05 NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    version integer
);


--
-- Name: paper_trading_portfolio_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.paper_trading_portfolio_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: paper_trading_portfolio_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.paper_trading_portfolio_id_seq OWNED BY public.paper_trading_portfolio.id;


--
-- Name: paper_trading_portfolio_snapshots; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.paper_trading_portfolio_snapshots (
    id bigint NOT NULL,
    snapshot_time timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    total_value numeric(15,2) NOT NULL,
    cash_balance numeric(15,2) NOT NULL,
    market_value numeric(15,2) DEFAULT 0 NOT NULL,
    total_pnl numeric(15,2) DEFAULT 0 NOT NULL,
    return_pct numeric(8,4) DEFAULT 0 NOT NULL,
    open_positions integer DEFAULT 0 NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    version integer
);


--
-- Name: paper_trading_portfolio_snapshots_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.paper_trading_portfolio_snapshots_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: paper_trading_portfolio_snapshots_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.paper_trading_portfolio_snapshots_id_seq OWNED BY public.paper_trading_portfolio_snapshots.id;


--
-- Name: pdf_extractions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pdf_extractions (
    id bigint NOT NULL,
    symbol character varying(10) NOT NULL,
    document_type character varying(20) NOT NULL,
    extracted_json jsonb NOT NULL,
    source_url text,
    extraction_date date NOT NULL,
    model_used character varying(50),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    version integer
);


--
-- Name: pdf_extractions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.pdf_extractions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: pdf_extractions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.pdf_extractions_id_seq OWNED BY public.pdf_extractions.id;


--
-- Name: positions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.positions (
    id bigint NOT NULL,
    symbol character varying(10) NOT NULL,
    entry_price numeric(15,4) NOT NULL,
    entry_date date NOT NULL,
    quantity integer,
    stop_loss numeric(15,4),
    target numeric(15,4),
    status character varying(20),
    entry_reason text,
    current_price numeric(15,4),
    direction character varying(10) DEFAULT 'LONG'::character varying,
    pnl numeric(15,4),
    broker_type character varying(10) DEFAULT 'PAPER'::character varying,
    position_id character varying(32),
    exchange character varying(10) DEFAULT 'NSE'::character varying,
    average_price numeric(15,2),
    unrealized_pnl numeric(15,2),
    realized_pnl numeric(15,2),
    margin_utilized numeric(15,2),
    entry_time timestamp without time zone,
    exit_time timestamp without time zone,
    exit_reason character varying(64),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    version integer,
    broker_position_id character varying(64),
    partial_exit_taken boolean DEFAULT false NOT NULL,
    signal_id bigint,
    portfolio_id character varying(40)
);


--
-- Name: positions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.positions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: positions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.positions_id_seq OWNED BY public.positions.id;


--
-- Name: price_band; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.price_band (
    id bigint NOT NULL,
    symbol character varying(20) NOT NULL,
    band_date date NOT NULL,
    lower_limit numeric(15,4) NOT NULL,
    upper_limit numeric(15,4) NOT NULL,
    CONSTRAINT ck_price_band_limits_ordered CHECK ((lower_limit < upper_limit)),
    CONSTRAINT ck_price_band_limits_positive CHECK (((lower_limit > (0)::numeric) AND (upper_limit > (0)::numeric)))
);


--
-- Name: price_band_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.price_band_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: price_band_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.price_band_id_seq OWNED BY public.price_band.id;


--
-- Name: reconciliation_audits; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.reconciliation_audits (
    id bigint NOT NULL,
    symbol character varying(20) NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL,
    status character varying(32) NOT NULL,
    missing_count integer DEFAULT 0 NOT NULL,
    inserted_count integer DEFAULT 0 NOT NULL,
    invalid_stored_removed_count integer DEFAULT 0 NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: reconciliation_audits_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.reconciliation_audits_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: reconciliation_audits_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.reconciliation_audits_id_seq OWNED BY public.reconciliation_audits.id;


--
-- Name: sentiment_accuracy; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sentiment_accuracy (
    id bigint NOT NULL,
    symbol character varying(20) NOT NULL,
    analysis_date date NOT NULL,
    llm_score character varying(20) NOT NULL,
    llm_confidence real NOT NULL,
    numeric_score real NOT NULL,
    actual_return_1d numeric(10,6),
    actual_return_5d numeric(10,6),
    actual_return_21d numeric(10,6),
    ground_truth_label character varying(10),
    was_correct boolean,
    pnl_pct numeric(10,6),
    market_regime character varying(10),
    prompt_hash character varying(64),
    model_version character varying(255),
    composite_score integer,
    composite_signal character varying(10),
    composite_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    evaluated_at timestamp with time zone,
    version integer,
    excess_return_1d numeric(10,6),
    excess_return_5d numeric(10,6),
    excess_return_21d numeric(10,6),
    ground_truth_basis character varying(20),
    sentiment_source character varying(20)
);


--
-- Name: sentiment_accuracy_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sentiment_accuracy_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sentiment_accuracy_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.sentiment_accuracy_id_seq OWNED BY public.sentiment_accuracy.id;


--
-- Name: sentiment_classification_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sentiment_classification_log (
    id bigint NOT NULL,
    version integer DEFAULT 0 NOT NULL,
    symbol character varying(20) NOT NULL,
    analysis_date date NOT NULL,
    model_used character varying(10) NOT NULL,
    sentiment character varying(10) NOT NULL,
    confidence double precision,
    latency_ms bigint,
    shadow_mode boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: sentiment_classification_log_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sentiment_classification_log_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sentiment_classification_log_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.sentiment_classification_log_id_seq OWNED BY public.sentiment_classification_log.id;


--
-- Name: sentiment_evaluation_audits; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sentiment_evaluation_audits (
    id bigint NOT NULL,
    trigger character varying(16) NOT NULL,
    status character varying(16) NOT NULL,
    processed_count integer NOT NULL,
    started_at timestamp without time zone NOT NULL,
    completed_at timestamp without time zone NOT NULL,
    error_message text
);


--
-- Name: sentiment_evaluation_audits_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sentiment_evaluation_audits_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sentiment_evaluation_audits_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.sentiment_evaluation_audits_id_seq OWNED BY public.sentiment_evaluation_audits.id;


--
-- Name: sentiment_results; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sentiment_results (
    id bigint NOT NULL,
    symbol character varying(10) NOT NULL,
    date date NOT NULL,
    sentiment_score character varying(20) NOT NULL,
    summary text,
    raw_content text,
    confidence real,
    analyzed_at date,
    red_flags text[],
    catalysts text[],
    prompt_hash character varying(64),
    model_version character varying(255),
    article_count integer DEFAULT 0,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    version integer,
    source character varying(16) DEFAULT 'DEFAULT'::character varying NOT NULL,
    article_ids bigint[],
    audit_request_id character varying(36)
);


--
-- Name: sentiment_results_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sentiment_results_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sentiment_results_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.sentiment_results_id_seq OWNED BY public.sentiment_results.id;


--
-- Name: signal_selection; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.signal_selection (
    id bigint NOT NULL,
    symbol character varying(16) NOT NULL,
    selection_date date NOT NULL,
    winner_variant_id character varying(40) NOT NULL,
    winner_version integer NOT NULL,
    winner_signal_id bigint,
    winner_confidence numeric(8,4) NOT NULL,
    candidates jsonb DEFAULT '[]'::jsonb NOT NULL,
    reason character varying(255),
    status character varying(16) DEFAULT 'PENDING'::character varying NOT NULL,
    status_detail character varying(255),
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: signal_selection_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.signal_selection_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: signal_selection_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.signal_selection_id_seq OWNED BY public.signal_selection.id;


--
-- Name: signals; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.signals (
    id bigint NOT NULL,
    symbol character varying(10) NOT NULL,
    date date NOT NULL,
    signal_type character varying(10) NOT NULL,
    confidence_score numeric(5,2),
    reasoning text,
    entry_price numeric(15,4),
    stop_loss numeric(15,4),
    target numeric(15,4),
    risk_reward numeric(5,2),
    indicators character varying(255),
    warning_flag character varying(50),
    generated_at date,
    processed boolean DEFAULT false,
    strategy character varying(40) DEFAULT 'DEFAULT'::character varying NOT NULL,
    sentiment_score character varying(12),
    sentiment_reasoning text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    version integer,
    strategy_version integer DEFAULT 1 NOT NULL,
    params_hash character varying(64)
);


--
-- Name: signals_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.signals_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: signals_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.signals_id_seq OWNED BY public.signals.id;


--
-- Name: stocks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stocks (
    id bigint NOT NULL,
    symbol character varying(10) NOT NULL,
    name character varying(255),
    exchange character varying(50),
    sector character varying(100),
    industry character varying(100),
    market_cap bigint,
    pe_ratio numeric(10,2),
    isin character varying(13),
    lot_size integer,
    added_on date,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    version integer
);


--
-- Name: stocks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.stocks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: stocks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.stocks_id_seq OWNED BY public.stocks.id;


--
-- Name: strategy_config; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.strategy_config (
    id bigint NOT NULL,
    variant_id character varying(40) NOT NULL,
    version integer NOT NULL,
    strategy_type character varying(30) NOT NULL,
    params jsonb NOT NULL,
    overlays jsonb DEFAULT '{}'::jsonb NOT NULL,
    params_hash character varying(64) NOT NULL,
    mode character varying(16) DEFAULT 'OFF'::character varying NOT NULL,
    paper_capital numeric(15,2) DEFAULT 500000 NOT NULL,
    is_current boolean DEFAULT true NOT NULL,
    notes text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    lock_version integer DEFAULT 0 NOT NULL,
    CONSTRAINT ck_strategy_config_mode CHECK (((mode)::text = ANY ((ARRAY['OFF'::character varying, 'BACKTEST_ONLY'::character varying, 'SHADOW'::character varying, 'CHAMPION'::character varying])::text[]))),
    CONSTRAINT ck_strategy_config_paper_capital_nonnegative CHECK ((paper_capital >= (0)::numeric)),
    CONSTRAINT ck_strategy_config_version_positive CHECK ((version > 0))
);


--
-- Name: strategy_config_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.strategy_config_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: strategy_config_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.strategy_config_id_seq OWNED BY public.strategy_config.id;


--
-- Name: synthesis_evaluations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.synthesis_evaluations (
    id bigint NOT NULL,
    version integer DEFAULT 0 NOT NULL,
    symbol character varying(20) NOT NULL,
    analysis_date date NOT NULL,
    recommendation character varying(10),
    confidence double precision NOT NULL,
    conflict_detected boolean DEFAULT false NOT NULL,
    event_risk_detected boolean DEFAULT false NOT NULL,
    event_risk_reason character varying(500),
    recorded_at timestamp with time zone NOT NULL,
    outcome_horizon_days integer,
    outcome_forward_return_pct numeric(12,6),
    outcome_correct boolean,
    outcome_measured_at timestamp with time zone
);


--
-- Name: synthesis_evaluations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.synthesis_evaluations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: synthesis_evaluations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.synthesis_evaluations_id_seq OWNED BY public.synthesis_evaluations.id;


--
-- Name: trade_labels; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.trade_labels (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    trade_id bigint NOT NULL,
    position_id bigint NOT NULL,
    exit_reason character varying(255) NOT NULL,
    exit_confidence numeric(38,2),
    notes text,
    labelled_by character varying(255),
    labelled_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    version integer
);


--
-- Name: trades; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.trades (
    id bigint NOT NULL,
    position_id bigint NOT NULL,
    symbol character varying(10) NOT NULL,
    entry_date date NOT NULL,
    exit_date date,
    entry_price numeric(15,4) NOT NULL,
    exit_price numeric(15,4),
    quantity integer,
    total_pnl numeric(15,4),
    duration_days integer,
    trade_status character varying(20) NOT NULL,
    entry_reason text,
    exit_reason character varying(50),
    fees numeric(15,4),
    direction character varying(10),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    version integer
);


--
-- Name: trades_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.trades_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: trades_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.trades_id_seq OWNED BY public.trades.id;


--
-- Name: universe_snapshots; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.universe_snapshots (
    id bigint NOT NULL,
    symbol character varying(20) NOT NULL,
    snapshot_date date NOT NULL,
    exchange character varying(12),
    isin character varying(12),
    included boolean NOT NULL,
    source character varying(80) NOT NULL,
    captured_at timestamp with time zone NOT NULL
);


--
-- Name: universe_snapshots_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.universe_snapshots_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: universe_snapshots_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.universe_snapshots_id_seq OWNED BY public.universe_snapshots.id;


--
-- Name: watchlist; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.watchlist (
    id bigint NOT NULL,
    symbol character varying(10) NOT NULL,
    name character varying(255),
    exchange character varying(5) DEFAULT 'NSE'::character varying,
    is_active boolean DEFAULT true,
    added_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    last_synced_at timestamp without time zone,
    candle_count integer DEFAULT 0,
    version integer
);


--
-- Name: watchlist_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.watchlist_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: watchlist_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.watchlist_id_seq OWNED BY public.watchlist.id;


--
-- Name: app_settings id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_settings ALTER COLUMN id SET DEFAULT nextval('public.app_settings_id_seq'::regclass);


--
-- Name: backtest_result id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.backtest_result ALTER COLUMN id SET DEFAULT nextval('public.backtest_result_id_seq'::regclass);


--
-- Name: candidate_scan_results id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.candidate_scan_results ALTER COLUMN id SET DEFAULT nextval('public.candidate_scan_results_id_seq'::regclass);


--
-- Name: candidate_scan_runs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.candidate_scan_runs ALTER COLUMN id SET DEFAULT nextval('public.candidate_scan_runs_id_seq'::regclass);


--
-- Name: corporate_actions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.corporate_actions ALTER COLUMN id SET DEFAULT nextval('public.corporate_actions_id_seq'::regclass);


--
-- Name: daily_loss_circuit_breaker_state id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daily_loss_circuit_breaker_state ALTER COLUMN id SET DEFAULT nextval('public.daily_loss_circuit_breaker_state_id_seq'::regclass);


--
-- Name: fyers_symbol_master id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fyers_symbol_master ALTER COLUMN id SET DEFAULT nextval('public.fyers_symbol_master_id_seq'::regclass);


--
-- Name: gate_effectiveness_audit id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.gate_effectiveness_audit ALTER COLUMN id SET DEFAULT nextval('public.gate_effectiveness_audit_id_seq'::regclass);


--
-- Name: job_run_stages id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_run_stages ALTER COLUMN id SET DEFAULT nextval('public.job_run_stages_id_seq'::regclass);


--
-- Name: job_runs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_runs ALTER COLUMN id SET DEFAULT nextval('public.job_runs_id_seq'::regclass);


--
-- Name: llm_analysis_audit id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_analysis_audit ALTER COLUMN id SET DEFAULT nextval('public.llm_analysis_audit_id_seq'::regclass);


--
-- Name: llm_analysis_result id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_analysis_result ALTER COLUMN id SET DEFAULT nextval('public.llm_analysis_result_id_seq'::regclass);


--
-- Name: news_articles id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.news_articles ALTER COLUMN id SET DEFAULT nextval('public.news_articles_id_seq'::regclass);


--
-- Name: news_items id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.news_items ALTER COLUMN id SET DEFAULT nextval('public.news_items_id_seq'::regclass);


--
-- Name: nse_holidays id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.nse_holidays ALTER COLUMN id SET DEFAULT nextval('public.nse_holidays_id_seq'::regclass);


--
-- Name: ohlcv_candles id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ohlcv_candles ALTER COLUMN id SET DEFAULT nextval('public.ohlcv_candles_id_seq'::regclass);


--
-- Name: paper_trading_orders id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.paper_trading_orders ALTER COLUMN id SET DEFAULT nextval('public.paper_trading_orders_id_seq'::regclass);


--
-- Name: paper_trading_portfolio id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.paper_trading_portfolio ALTER COLUMN id SET DEFAULT nextval('public.paper_trading_portfolio_id_seq'::regclass);


--
-- Name: paper_trading_portfolio_snapshots id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.paper_trading_portfolio_snapshots ALTER COLUMN id SET DEFAULT nextval('public.paper_trading_portfolio_snapshots_id_seq'::regclass);


--
-- Name: pdf_extractions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pdf_extractions ALTER COLUMN id SET DEFAULT nextval('public.pdf_extractions_id_seq'::regclass);


--
-- Name: positions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.positions ALTER COLUMN id SET DEFAULT nextval('public.positions_id_seq'::regclass);


--
-- Name: price_band id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.price_band ALTER COLUMN id SET DEFAULT nextval('public.price_band_id_seq'::regclass);


--
-- Name: reconciliation_audits id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reconciliation_audits ALTER COLUMN id SET DEFAULT nextval('public.reconciliation_audits_id_seq'::regclass);


--
-- Name: sentiment_accuracy id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sentiment_accuracy ALTER COLUMN id SET DEFAULT nextval('public.sentiment_accuracy_id_seq'::regclass);


--
-- Name: sentiment_classification_log id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sentiment_classification_log ALTER COLUMN id SET DEFAULT nextval('public.sentiment_classification_log_id_seq'::regclass);


--
-- Name: sentiment_evaluation_audits id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sentiment_evaluation_audits ALTER COLUMN id SET DEFAULT nextval('public.sentiment_evaluation_audits_id_seq'::regclass);


--
-- Name: sentiment_results id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sentiment_results ALTER COLUMN id SET DEFAULT nextval('public.sentiment_results_id_seq'::regclass);


--
-- Name: signal_selection id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.signal_selection ALTER COLUMN id SET DEFAULT nextval('public.signal_selection_id_seq'::regclass);


--
-- Name: signals id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.signals ALTER COLUMN id SET DEFAULT nextval('public.signals_id_seq'::regclass);


--
-- Name: stocks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stocks ALTER COLUMN id SET DEFAULT nextval('public.stocks_id_seq'::regclass);


--
-- Name: strategy_config id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.strategy_config ALTER COLUMN id SET DEFAULT nextval('public.strategy_config_id_seq'::regclass);


--
-- Name: synthesis_evaluations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.synthesis_evaluations ALTER COLUMN id SET DEFAULT nextval('public.synthesis_evaluations_id_seq'::regclass);


--
-- Name: trades id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trades ALTER COLUMN id SET DEFAULT nextval('public.trades_id_seq'::regclass);


--
-- Name: universe_snapshots id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.universe_snapshots ALTER COLUMN id SET DEFAULT nextval('public.universe_snapshots_id_seq'::regclass);


--
-- Name: watchlist id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.watchlist ALTER COLUMN id SET DEFAULT nextval('public.watchlist_id_seq'::regclass);


--
-- Name: app_settings app_settings_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_settings
    ADD CONSTRAINT app_settings_key_key UNIQUE (key);


--
-- Name: app_settings app_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_settings
    ADD CONSTRAINT app_settings_pkey PRIMARY KEY (id);


--
-- Name: backtest_result backtest_result_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.backtest_result
    ADD CONSTRAINT backtest_result_pkey PRIMARY KEY (id);


--
-- Name: candidate_history_eligibility candidate_history_eligibility_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.candidate_history_eligibility
    ADD CONSTRAINT candidate_history_eligibility_pkey PRIMARY KEY (symbol);


--
-- Name: candidate_scan_results candidate_scan_results_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.candidate_scan_results
    ADD CONSTRAINT candidate_scan_results_pkey PRIMARY KEY (id);


--
-- Name: candidate_scan_runs candidate_scan_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.candidate_scan_runs
    ADD CONSTRAINT candidate_scan_runs_pkey PRIMARY KEY (id);


--
-- Name: candidate_scan_runs candidate_scan_runs_run_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.candidate_scan_runs
    ADD CONSTRAINT candidate_scan_runs_run_id_key UNIQUE (run_id);


--
-- Name: corporate_actions corporate_actions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.corporate_actions
    ADD CONSTRAINT corporate_actions_pkey PRIMARY KEY (id);


--
-- Name: daily_loss_circuit_breaker_state daily_loss_circuit_breaker_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daily_loss_circuit_breaker_state
    ADD CONSTRAINT daily_loss_circuit_breaker_state_pkey PRIMARY KEY (id);


--
-- Name: flyway_schema_history flyway_schema_history_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flyway_schema_history
    ADD CONSTRAINT flyway_schema_history_pk PRIMARY KEY (installed_rank);


--
-- Name: fyers_symbol_master fyers_symbol_master_fy_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fyers_symbol_master
    ADD CONSTRAINT fyers_symbol_master_fy_token_key UNIQUE (fy_token);


--
-- Name: fyers_symbol_master fyers_symbol_master_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fyers_symbol_master
    ADD CONSTRAINT fyers_symbol_master_pkey PRIMARY KEY (id);


--
-- Name: gate_effectiveness_audit gate_effectiveness_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.gate_effectiveness_audit
    ADD CONSTRAINT gate_effectiveness_audit_pkey PRIMARY KEY (id);


--
-- Name: sentiment_results idx_sentiment_symbol_date; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sentiment_results
    ADD CONSTRAINT idx_sentiment_symbol_date UNIQUE (symbol, date);


--
-- Name: stocks idx_stocks_symbol; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stocks
    ADD CONSTRAINT idx_stocks_symbol UNIQUE (symbol);


--
-- Name: watchlist idx_watchlist_symbol; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.watchlist
    ADD CONSTRAINT idx_watchlist_symbol UNIQUE (symbol);


--
-- Name: intraday_paper_sessions intraday_paper_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.intraday_paper_sessions
    ADD CONSTRAINT intraday_paper_sessions_pkey PRIMARY KEY (run_id);


--
-- Name: job_run_stages job_run_stages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_run_stages
    ADD CONSTRAINT job_run_stages_pkey PRIMARY KEY (id);


--
-- Name: job_runs job_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_runs
    ADD CONSTRAINT job_runs_pkey PRIMARY KEY (id);


--
-- Name: job_runs job_runs_run_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_runs
    ADD CONSTRAINT job_runs_run_id_key UNIQUE (run_id);


--
-- Name: kill_switch kill_switch_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kill_switch
    ADD CONSTRAINT kill_switch_pkey PRIMARY KEY (id);


--
-- Name: llm_analysis_audit llm_analysis_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_analysis_audit
    ADD CONSTRAINT llm_analysis_audit_pkey PRIMARY KEY (id);


--
-- Name: llm_analysis_result llm_analysis_result_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_analysis_result
    ADD CONSTRAINT llm_analysis_result_pkey PRIMARY KEY (id);


--
-- Name: news_articles news_articles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.news_articles
    ADD CONSTRAINT news_articles_pkey PRIMARY KEY (id);


--
-- Name: news_items news_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.news_items
    ADD CONSTRAINT news_items_pkey PRIMARY KEY (id);


--
-- Name: nse_calendar_coverage nse_calendar_coverage_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.nse_calendar_coverage
    ADD CONSTRAINT nse_calendar_coverage_pkey PRIMARY KEY (calendar_year);


--
-- Name: nse_holidays nse_holidays_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.nse_holidays
    ADD CONSTRAINT nse_holidays_pkey PRIMARY KEY (id);


--
-- Name: ohlcv_candles ohlcv_candles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ohlcv_candles
    ADD CONSTRAINT ohlcv_candles_pkey PRIMARY KEY (id);


--
-- Name: paper_trading_orders paper_trading_orders_order_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.paper_trading_orders
    ADD CONSTRAINT paper_trading_orders_order_id_key UNIQUE (order_id);


--
-- Name: paper_trading_orders paper_trading_orders_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.paper_trading_orders
    ADD CONSTRAINT paper_trading_orders_pkey PRIMARY KEY (id);


--
-- Name: paper_trading_portfolio paper_trading_portfolio_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.paper_trading_portfolio
    ADD CONSTRAINT paper_trading_portfolio_pkey PRIMARY KEY (id);


--
-- Name: paper_trading_portfolio_snapshots paper_trading_portfolio_snapshots_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.paper_trading_portfolio_snapshots
    ADD CONSTRAINT paper_trading_portfolio_snapshots_pkey PRIMARY KEY (id);


--
-- Name: pdf_extractions pdf_extractions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pdf_extractions
    ADD CONSTRAINT pdf_extractions_pkey PRIMARY KEY (id);


--
-- Name: positions positions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.positions
    ADD CONSTRAINT positions_pkey PRIMARY KEY (id);


--
-- Name: price_band price_band_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.price_band
    ADD CONSTRAINT price_band_pkey PRIMARY KEY (id);


--
-- Name: reconciliation_audits reconciliation_audits_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reconciliation_audits
    ADD CONSTRAINT reconciliation_audits_pkey PRIMARY KEY (id);


--
-- Name: sentiment_accuracy sentiment_accuracy_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sentiment_accuracy
    ADD CONSTRAINT sentiment_accuracy_pkey PRIMARY KEY (id);


--
-- Name: sentiment_classification_log sentiment_classification_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sentiment_classification_log
    ADD CONSTRAINT sentiment_classification_log_pkey PRIMARY KEY (id);


--
-- Name: sentiment_evaluation_audits sentiment_evaluation_audits_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sentiment_evaluation_audits
    ADD CONSTRAINT sentiment_evaluation_audits_pkey PRIMARY KEY (id);


--
-- Name: sentiment_results sentiment_results_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sentiment_results
    ADD CONSTRAINT sentiment_results_pkey PRIMARY KEY (id);


--
-- Name: signal_selection signal_selection_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.signal_selection
    ADD CONSTRAINT signal_selection_pkey PRIMARY KEY (id);


--
-- Name: signals signals_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.signals
    ADD CONSTRAINT signals_pkey PRIMARY KEY (id);


--
-- Name: stocks stocks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stocks
    ADD CONSTRAINT stocks_pkey PRIMARY KEY (id);


--
-- Name: stocks stocks_symbol_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stocks
    ADD CONSTRAINT stocks_symbol_key UNIQUE (symbol);


--
-- Name: strategy_config strategy_config_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.strategy_config
    ADD CONSTRAINT strategy_config_pkey PRIMARY KEY (id);


--
-- Name: synthesis_evaluations synthesis_evaluations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.synthesis_evaluations
    ADD CONSTRAINT synthesis_evaluations_pkey PRIMARY KEY (id);


--
-- Name: trade_labels trade_labels_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trade_labels
    ADD CONSTRAINT trade_labels_pkey PRIMARY KEY (id);


--
-- Name: trades trades_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trades
    ADD CONSTRAINT trades_pkey PRIMARY KEY (id);


--
-- Name: universe_snapshots universe_snapshots_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.universe_snapshots
    ADD CONSTRAINT universe_snapshots_pkey PRIMARY KEY (id);


--
-- Name: backtest_result uq_backtest_result_symbol_date; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.backtest_result
    ADD CONSTRAINT uq_backtest_result_symbol_date UNIQUE (symbol, run_date);


--
-- Name: corporate_actions uq_corporate_action_symbol_date_type; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.corporate_actions
    ADD CONSTRAINT uq_corporate_action_symbol_date_type UNIQUE (symbol, effective_date, action_type);


--
-- Name: gate_effectiveness_audit uq_gate_effectiveness_symbol_date_gate; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.gate_effectiveness_audit
    ADD CONSTRAINT uq_gate_effectiveness_symbol_date_gate UNIQUE (symbol, signal_date, gate_name, strategy);


--
-- Name: gate_effectiveness_audit uq_gate_effectiveness_symbol_date_gate_strategy; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.gate_effectiveness_audit
    ADD CONSTRAINT uq_gate_effectiveness_symbol_date_gate_strategy UNIQUE (symbol, signal_date, gate_name, strategy);


--
-- Name: llm_analysis_result uq_llm_analysis_result_symbol_date; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_analysis_result
    ADD CONSTRAINT uq_llm_analysis_result_symbol_date UNIQUE (symbol, analysis_date);


--
-- Name: nse_holidays uq_nse_holiday_date; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.nse_holidays
    ADD CONSTRAINT uq_nse_holiday_date UNIQUE (holiday_date);


--
-- Name: ohlcv_candles uq_ohlcv_candles_symbol_timeframe_date_bar_time; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ohlcv_candles
    ADD CONSTRAINT uq_ohlcv_candles_symbol_timeframe_date_bar_time UNIQUE (symbol, timeframe, date, bar_time);


--
-- Name: paper_trading_portfolio uq_paper_trading_portfolio_portfolio_id; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.paper_trading_portfolio
    ADD CONSTRAINT uq_paper_trading_portfolio_portfolio_id UNIQUE (portfolio_id);


--
-- Name: price_band uq_price_band_symbol_date; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.price_band
    ADD CONSTRAINT uq_price_band_symbol_date UNIQUE (symbol, band_date);


--
-- Name: signal_selection uq_signal_selection_symbol_date; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.signal_selection
    ADD CONSTRAINT uq_signal_selection_symbol_date UNIQUE (symbol, selection_date);


--
-- Name: strategy_config uq_strategy_config_variant_version; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.strategy_config
    ADD CONSTRAINT uq_strategy_config_variant_version UNIQUE (variant_id, version);


--
-- Name: synthesis_evaluations uq_synthesis_evaluation_symbol_date; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.synthesis_evaluations
    ADD CONSTRAINT uq_synthesis_evaluation_symbol_date UNIQUE (symbol, analysis_date);


--
-- Name: universe_snapshots uq_universe_snapshot_symbol_date; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.universe_snapshots
    ADD CONSTRAINT uq_universe_snapshot_symbol_date UNIQUE (symbol, snapshot_date);


--
-- Name: watchlist watchlist_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.watchlist
    ADD CONSTRAINT watchlist_pkey PRIMARY KEY (id);


--
-- Name: watchlist watchlist_symbol_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.watchlist
    ADD CONSTRAINT watchlist_symbol_key UNIQUE (symbol);


--
-- Name: flyway_schema_history_s_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX flyway_schema_history_s_idx ON public.flyway_schema_history USING btree (success);


--
-- Name: idx_app_settings_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_settings_key ON public.app_settings USING btree (key);


--
-- Name: idx_backtest_result_symbol_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_backtest_result_symbol_date ON public.backtest_result USING btree (symbol, run_date);


--
-- Name: idx_candidate_scan_strategy_outcomes_result; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_candidate_scan_strategy_outcomes_result ON public.candidate_scan_strategy_outcomes USING btree (candidate_scan_result_id);


--
-- Name: idx_corporate_actions_symbol_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_corporate_actions_symbol_date ON public.corporate_actions USING btree (symbol, effective_date);


--
-- Name: idx_gate_effectiveness_gate_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_gate_effectiveness_gate_date ON public.gate_effectiveness_audit USING btree (gate_name, signal_date);


--
-- Name: idx_job_run_stages_run; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_job_run_stages_run ON public.job_run_stages USING btree (run_id);


--
-- Name: idx_job_run_stages_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_job_run_stages_status ON public.job_run_stages USING btree (status);


--
-- Name: idx_job_run_stages_symbol_stage; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_job_run_stages_symbol_stage ON public.job_run_stages USING btree (symbol, stage_name);


--
-- Name: idx_job_runs_started_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_job_runs_started_at ON public.job_runs USING btree (started_at DESC);


--
-- Name: idx_job_runs_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_job_runs_status ON public.job_runs USING btree (status);


--
-- Name: idx_llm_analysis_audit_provider_model; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_llm_analysis_audit_provider_model ON public.llm_analysis_audit USING btree (provider, model_version);


--
-- Name: idx_llm_analysis_audit_started_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_llm_analysis_audit_started_at ON public.llm_analysis_audit USING btree (started_at);


--
-- Name: idx_llm_analysis_audit_symbol_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_llm_analysis_audit_symbol_date ON public.llm_analysis_audit USING btree (symbol, analysis_date);


--
-- Name: idx_llm_analysis_result_run_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_llm_analysis_result_run_id ON public.llm_analysis_result USING btree (run_id);


--
-- Name: idx_news_articles_published; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_news_articles_published ON public.news_articles USING btree (published_at);


--
-- Name: idx_news_articles_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_news_articles_source ON public.news_articles USING btree (source);


--
-- Name: idx_news_articles_symbol; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_news_articles_symbol ON public.news_articles USING btree (symbol);


--
-- Name: idx_news_articles_symbol_published_first_seen; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_news_articles_symbol_published_first_seen ON public.news_articles USING btree (symbol, published_at, first_seen_at);


--
-- Name: idx_news_headline; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_news_headline ON public.news_items USING btree (lower(headline));


--
-- Name: idx_news_symbol_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_news_symbol_date ON public.news_items USING btree (symbol, published_at);


--
-- Name: idx_nse_holidays_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_nse_holidays_date ON public.nse_holidays USING btree (holiday_date);


--
-- Name: idx_ohlcv_candles_symbol_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ohlcv_candles_symbol_date ON public.ohlcv_candles USING btree (symbol, date);


--
-- Name: idx_ohlcv_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ohlcv_date ON public.ohlcv_candles USING btree (date);


--
-- Name: idx_ohlcv_symbol_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ohlcv_symbol_date ON public.ohlcv_candles USING btree (symbol, date);


--
-- Name: idx_ohlcv_timeframe_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ohlcv_timeframe_date ON public.ohlcv_candles USING btree (timeframe, date);


--
-- Name: idx_paper_trading_orders_portfolio_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_paper_trading_orders_portfolio_id ON public.paper_trading_orders USING btree (portfolio_id);


--
-- Name: idx_pdf_symbol_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pdf_symbol_date ON public.pdf_extractions USING btree (symbol, extraction_date);


--
-- Name: idx_positions_portfolio_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_positions_portfolio_id ON public.positions USING btree (portfolio_id);


--
-- Name: idx_positions_signal_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_positions_signal_id ON public.positions USING btree (signal_id);


--
-- Name: idx_positions_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_positions_status ON public.positions USING btree (status);


--
-- Name: idx_positions_symbol; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_positions_symbol ON public.positions USING btree (symbol);


--
-- Name: idx_price_band_symbol_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_price_band_symbol_date ON public.price_band USING btree (symbol, band_date);


--
-- Name: idx_pt_orders_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pt_orders_created_at ON public.paper_trading_orders USING btree (created_at);


--
-- Name: idx_pt_orders_signal_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pt_orders_signal_id ON public.paper_trading_orders USING btree (signal_id);


--
-- Name: idx_pt_orders_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pt_orders_status ON public.paper_trading_orders USING btree (status);


--
-- Name: idx_pt_orders_symbol; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pt_orders_symbol ON public.paper_trading_orders USING btree (symbol);


--
-- Name: idx_pt_snapshots_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pt_snapshots_time ON public.paper_trading_portfolio_snapshots USING btree (snapshot_time);


--
-- Name: idx_reconciliation_audits_symbol_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reconciliation_audits_symbol_created ON public.reconciliation_audits USING btree (symbol, created_at DESC);


--
-- Name: idx_sentiment_accuracy_evaluated; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sentiment_accuracy_evaluated ON public.sentiment_accuracy USING btree (evaluated_at) WHERE (evaluated_at IS NOT NULL);


--
-- Name: idx_sentiment_accuracy_label; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sentiment_accuracy_label ON public.sentiment_accuracy USING btree (ground_truth_label);


--
-- Name: idx_sentiment_accuracy_regime; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sentiment_accuracy_regime ON public.sentiment_accuracy USING btree (market_regime);


--
-- Name: idx_sentiment_accuracy_symbol_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sentiment_accuracy_symbol_date ON public.sentiment_accuracy USING btree (symbol, analysis_date);


--
-- Name: idx_sentiment_classification_log_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sentiment_classification_log_created ON public.sentiment_classification_log USING btree (created_at);


--
-- Name: idx_sentiment_classification_log_model; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sentiment_classification_log_model ON public.sentiment_classification_log USING btree (model_used);


--
-- Name: idx_sentiment_classification_log_symbol_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sentiment_classification_log_symbol_date ON public.sentiment_classification_log USING btree (symbol, analysis_date);


--
-- Name: idx_sentiment_results_audit_request_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sentiment_results_audit_request_id ON public.sentiment_results USING btree (audit_request_id);


--
-- Name: idx_sentiment_results_model; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sentiment_results_model ON public.sentiment_results USING btree (model_version);


--
-- Name: idx_sentiment_results_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sentiment_results_source ON public.sentiment_results USING btree (source);


--
-- Name: idx_sentiment_results_symbol_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sentiment_results_symbol_date ON public.sentiment_results USING btree (symbol, date);


--
-- Name: idx_signal_selection_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_signal_selection_date ON public.signal_selection USING btree (selection_date);


--
-- Name: idx_signal_selection_winner; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_signal_selection_winner ON public.signal_selection USING btree (winner_variant_id, selection_date);


--
-- Name: idx_signals_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_signals_date ON public.signals USING btree (date);


--
-- Name: idx_signals_processed_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_signals_processed_date ON public.signals USING btree (processed, date);


--
-- Name: idx_signals_strategy_version; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_signals_strategy_version ON public.signals USING btree (strategy, strategy_version);


--
-- Name: idx_signals_symbol_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_signals_symbol_date ON public.signals USING btree (symbol, date);


--
-- Name: idx_signals_symbol_date_strategy; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_signals_symbol_date_strategy ON public.signals USING btree (symbol, date, strategy);


--
-- Name: idx_stocks_exchange; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_stocks_exchange ON public.stocks USING btree (exchange);


--
-- Name: idx_stocks_sector; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_stocks_sector ON public.stocks USING btree (sector);


--
-- Name: idx_synthesis_evaluation_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_synthesis_evaluation_date ON public.synthesis_evaluations USING btree (analysis_date);


--
-- Name: idx_synthesis_evaluation_outcome; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_synthesis_evaluation_outcome ON public.synthesis_evaluations USING btree (outcome_measured_at);


--
-- Name: idx_trade_labels_position_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trade_labels_position_id ON public.trade_labels USING btree (position_id);


--
-- Name: idx_trade_labels_trade_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trade_labels_trade_id ON public.trade_labels USING btree (trade_id);


--
-- Name: idx_trades_entry_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trades_entry_date ON public.trades USING btree (entry_date);


--
-- Name: idx_trades_position_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trades_position_id ON public.trades USING btree (position_id);


--
-- Name: idx_trades_symbol; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trades_symbol ON public.trades USING btree (symbol);


--
-- Name: idx_universe_snapshots_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_universe_snapshots_date ON public.universe_snapshots USING btree (snapshot_date, symbol);


--
-- Name: idx_watchlist_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_watchlist_active ON public.watchlist USING btree (is_active);


--
-- Name: ix_candidate_scan_results_qualified; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_candidate_scan_results_qualified ON public.candidate_scan_results USING btree (run_id, qualified);


--
-- Name: ix_candidate_scan_results_run_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_candidate_scan_results_run_id ON public.candidate_scan_results USING btree (run_id);


--
-- Name: ix_fyers_symbol_master_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_fyers_symbol_master_name ON public.fyers_symbol_master USING btree (lower(name) text_pattern_ops);


--
-- Name: ix_fyers_symbol_master_trading; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_fyers_symbol_master_trading ON public.fyers_symbol_master USING btree (trading_symbol);


--
-- Name: ix_intraday_paper_sessions_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_intraday_paper_sessions_created_at ON public.intraday_paper_sessions USING btree (created_at DESC);


--
-- Name: ix_job_runs_candidate_scan_run_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_job_runs_candidate_scan_run_id ON public.job_runs USING btree (candidate_scan_run_id);


--
-- Name: ux_fyers_symbol_master_symbol; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX ux_fyers_symbol_master_symbol ON public.fyers_symbol_master USING btree (fyers_symbol);


--
-- Name: ux_news_articles_symbol_article_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX ux_news_articles_symbol_article_key ON public.news_articles USING btree (symbol, article_key) WHERE (article_key IS NOT NULL);


--
-- Name: ux_strategy_config_champion; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX ux_strategy_config_champion ON public.strategy_config USING btree ((1)) WHERE (is_current AND ((mode)::text = 'CHAMPION'::text));


--
-- Name: ux_strategy_config_current; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX ux_strategy_config_current ON public.strategy_config USING btree (variant_id) WHERE is_current;


--
-- Name: positions update_positions_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_positions_updated_at BEFORE UPDATE ON public.positions FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: sentiment_results update_sentiment_results_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_sentiment_results_updated_at BEFORE UPDATE ON public.sentiment_results FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: signals update_signals_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_signals_updated_at BEFORE UPDATE ON public.signals FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: stocks update_stocks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_stocks_updated_at BEFORE UPDATE ON public.stocks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: trade_labels update_trade_labels_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_trade_labels_updated_at BEFORE UPDATE ON public.trade_labels FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: trades update_trades_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_trades_updated_at BEFORE UPDATE ON public.trades FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: candidate_scan_strategy_outcomes candidate_scan_strategy_outcomes_candidate_scan_result_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.candidate_scan_strategy_outcomes
    ADD CONSTRAINT candidate_scan_strategy_outcomes_candidate_scan_result_id_fkey FOREIGN KEY (candidate_scan_result_id) REFERENCES public.candidate_scan_results(id) ON DELETE CASCADE;


--
-- Name: trade_labels fk4eegadsq0r26prbtdlw9oqp6x; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trade_labels
    ADD CONSTRAINT fk4eegadsq0r26prbtdlw9oqp6x FOREIGN KEY (position_id) REFERENCES public.positions(id);


--
-- Name: candidate_scan_results fk_candidate_scan_result_run; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.candidate_scan_results
    ADD CONSTRAINT fk_candidate_scan_result_run FOREIGN KEY (run_id) REFERENCES public.candidate_scan_runs(run_id);


--
-- Name: positions fk_positions_stock; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.positions
    ADD CONSTRAINT fk_positions_stock FOREIGN KEY (symbol) REFERENCES public.stocks(symbol) ON DELETE CASCADE;


--
-- Name: signals fk_signals_stock; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.signals
    ADD CONSTRAINT fk_signals_stock FOREIGN KEY (symbol) REFERENCES public.stocks(symbol) ON DELETE CASCADE;


--
-- Name: trades fk_trades_position; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trades
    ADD CONSTRAINT fk_trades_position FOREIGN KEY (position_id) REFERENCES public.positions(id) ON DELETE SET NULL;


--
-- Name: trades fk_trades_stock; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trades
    ADD CONSTRAINT fk_trades_stock FOREIGN KEY (symbol) REFERENCES public.stocks(symbol) ON DELETE CASCADE;


--
-- Name: job_run_stages job_run_stages_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_run_stages
    ADD CONSTRAINT job_run_stages_run_id_fkey FOREIGN KEY (run_id) REFERENCES public.job_runs(run_id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict gBPHXGmUNfL0bpg4hk0VpTJa4cnVeCBgT47hQQ8BeAo1sYa8ZzAT8l8CDLgUTDd
