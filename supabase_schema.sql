-- PostgreSQL schema for Supabase migration
-- Converted from SQLite schema

-- Users table (needs to be created first)
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE,
    password VARCHAR(255),
    display_name VARCHAR(255),
    role INTEGER DEFAULT 1,
    status INTEGER DEFAULT 1,
    email VARCHAR(255),
    github_id VARCHAR(255),
    wechat_id VARCHAR(255),
    lark_id VARCHAR(255),
    access_token VARCHAR(255),
    quota INTEGER DEFAULT 0,
    used_quota INTEGER DEFAULT 0,
    request_count INTEGER DEFAULT 0,
    group VARCHAR(255) DEFAULT 'default',
    affiliation VARCHAR(255),
    inviter_id INTEGER,
    created_time BIGINT,
    deleted_at TIMESTAMP
);

-- Channels table
CREATE TABLE IF NOT EXISTS channels (
    id SERIAL PRIMARY KEY,
    type INTEGER DEFAULT 0,
    key TEXT NOT NULL,
    open_ai_organization TEXT,
    test_model TEXT,
    status INTEGER DEFAULT 1,
    name TEXT,
    weight INTEGER DEFAULT 0,
    created_time BIGINT,
    test_time BIGINT,
    response_time BIGINT,
    base_url TEXT DEFAULT '',
    other TEXT,
    balance REAL,
    balance_updated_time BIGINT,
    models TEXT,
    group_name VARCHAR(64) DEFAULT 'default',
    used_quota INTEGER DEFAULT 0,
    model_mapping TEXT,
    status_code_mapping VARCHAR(1024) DEFAULT '',
    priority INTEGER DEFAULT 0,
    auto_ban INTEGER DEFAULT 1,
    other_info TEXT,
    tag TEXT,
    setting TEXT,
    param_override TEXT,
    header_override TEXT,
    remark VARCHAR(255),
    channel_info JSONB,
    settings TEXT
);

-- Tokens table
CREATE TABLE IF NOT EXISTS tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER,
    key CHAR(48),
    status INTEGER DEFAULT 1,
    name TEXT,
    created_time BIGINT,
    accessed_time BIGINT,
    expired_time BIGINT DEFAULT -1,
    remain_quota INTEGER DEFAULT 0,
    unlimited_quota BOOLEAN,
    model_limits_enabled BOOLEAN,
    model_limits VARCHAR(1024) DEFAULT '',
    allow_ips TEXT DEFAULT '',
    used_quota INTEGER DEFAULT 0,
    group_name TEXT DEFAULT '',
    deleted_at TIMESTAMP
);

-- Options table
CREATE TABLE IF NOT EXISTS options (
    key TEXT PRIMARY KEY,
    value TEXT
);

-- Redemptions table
CREATE TABLE IF NOT EXISTS redemptions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER,
    key CHAR(32),
    status INTEGER DEFAULT 1,
    name TEXT,
    quota INTEGER DEFAULT 100,
    created_time BIGINT,
    redeemed_time BIGINT,
    used_user_id INTEGER,
    deleted_at TIMESTAMP,
    expired_time BIGINT
);

-- Abilities table
CREATE TABLE IF NOT EXISTS abilities (
    group_name VARCHAR(64),
    model VARCHAR(255),
    channel_id INTEGER,
    enabled BOOLEAN,
    priority INTEGER DEFAULT 0,
    weight INTEGER DEFAULT 0,
    tag TEXT,
    PRIMARY KEY (group_name, model, channel_id)
);

-- Logs table
CREATE TABLE IF NOT EXISTS logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER,
    created_at BIGINT,
    type INTEGER,
    content TEXT,
    username TEXT DEFAULT '',
    token_name TEXT DEFAULT '',
    model_name TEXT DEFAULT '',
    quota INTEGER DEFAULT 0,
    prompt_tokens INTEGER DEFAULT 0,
    completion_tokens INTEGER DEFAULT 0,
    use_time INTEGER DEFAULT 0,
    is_stream BOOLEAN,
    channel_id INTEGER,
    channel_name TEXT,
    token_id INTEGER DEFAULT 0,
    group_name TEXT,
    ip TEXT DEFAULT '',
    other TEXT
);

-- Midjourneys table
CREATE TABLE IF NOT EXISTS midjourneys (
    id SERIAL PRIMARY KEY,
    code INTEGER,
    user_id INTEGER,
    action VARCHAR(40),
    mj_id TEXT,
    prompt TEXT,
    prompt_en TEXT,
    description TEXT,
    state TEXT,
    submit_time BIGINT,
    start_time BIGINT,
    finish_time BIGINT,
    image_url TEXT,
    video_url TEXT,
    video_urls TEXT,
    status VARCHAR(20),
    progress VARCHAR(30),
    fail_reason TEXT,
    channel_id INTEGER,
    quota INTEGER,
    buttons TEXT,
    properties TEXT
);

-- Quota_data table
CREATE TABLE IF NOT EXISTS quota_data (
    id SERIAL PRIMARY KEY,
    user_id INTEGER,
    username TEXT DEFAULT '',
    model_name TEXT DEFAULT '',
    created_at BIGINT,
    token_used INTEGER DEFAULT 0,
    count INTEGER DEFAULT 0,
    quota INTEGER DEFAULT 0
);

-- Tasks table
CREATE TABLE IF NOT EXISTS tasks (
    id SERIAL PRIMARY KEY,
    created_at BIGINT,
    updated_at BIGINT,
    task_id VARCHAR(50),
    platform VARCHAR(30),
    user_id INTEGER,
    channel_id INTEGER,
    quota INTEGER,
    action VARCHAR(40),
    status VARCHAR(20),
    fail_reason TEXT,
    submit_time BIGINT,
    start_time BIGINT,
    finish_time BIGINT,
    progress VARCHAR(20),
    properties JSONB,
    data JSONB
);

-- Top_ups table
CREATE TABLE IF NOT EXISTS top_ups (
    id SERIAL PRIMARY KEY,
    user_id INTEGER,
    amount INTEGER,
    created_time BIGINT,
    status INTEGER DEFAULT 1
);

-- Two_fas table
CREATE TABLE IF NOT EXISTS two_fas (
    id SERIAL PRIMARY KEY,
    user_id INTEGER,
    secret VARCHAR(255),
    enabled BOOLEAN DEFAULT FALSE,
    created_time BIGINT
);

-- Two_fa_backup_codes table
CREATE TABLE IF NOT EXISTS two_fa_backup_codes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER,
    code VARCHAR(255),
    used BOOLEAN DEFAULT FALSE,
    created_time BIGINT
);

-- Vendors table
CREATE TABLE IF NOT EXISTS vendors (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255),
    description TEXT,
    created_time BIGINT
);

-- Models table
CREATE TABLE IF NOT EXISTS models (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255),
    vendor_id INTEGER,
    description TEXT,
    created_time BIGINT
);

-- Setups table
CREATE TABLE IF NOT EXISTS setups (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255),
    value TEXT,
    created_time BIGINT
);

-- Prefill_groups table
CREATE TABLE IF NOT EXISTS prefill_groups (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255),
    description TEXT,
    created_time BIGINT
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_channels_tag ON channels(tag);
CREATE INDEX IF NOT EXISTS idx_channels_name ON channels(name);
CREATE INDEX IF NOT EXISTS idx_tokens_name ON tokens(name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tokens_key ON tokens(key);
CREATE INDEX IF NOT EXISTS idx_tokens_user_id ON tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_tokens_deleted_at ON tokens(deleted_at);
CREATE INDEX IF NOT EXISTS idx_redemptions_deleted_at ON redemptions(deleted_at);
CREATE INDEX IF NOT EXISTS idx_redemptions_name ON redemptions(name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_redemptions_key ON redemptions(key);
CREATE INDEX IF NOT EXISTS idx_abilities_tag ON abilities(tag);
CREATE INDEX IF NOT EXISTS idx_abilities_weight ON abilities(weight);
CREATE INDEX IF NOT EXISTS idx_abilities_priority ON abilities(priority);
CREATE INDEX IF NOT EXISTS idx_abilities_channel_id ON abilities(channel_id);
CREATE INDEX IF NOT EXISTS idx_logs_group_name ON logs(group_name);
CREATE INDEX IF NOT EXISTS idx_logs_token_id ON logs(token_id);
CREATE INDEX IF NOT EXISTS idx_logs_channel_id ON logs(channel_id);
CREATE INDEX IF NOT EXISTS idx_logs_model_name ON logs(model_name);
CREATE INDEX IF NOT EXISTS index_username_model_name ON logs(model_name, username);
CREATE INDEX IF NOT EXISTS idx_created_at_type ON logs(created_at, type);
CREATE INDEX IF NOT EXISTS idx_logs_user_id ON logs(user_id);
CREATE INDEX IF NOT EXISTS idx_logs_ip ON logs(ip);
CREATE INDEX IF NOT EXISTS idx_logs_token_name ON logs(token_name);
CREATE INDEX IF NOT EXISTS idx_logs_username ON logs(username);
CREATE INDEX IF NOT EXISTS idx_created_at_id ON logs(id, created_at);
CREATE INDEX IF NOT EXISTS idx_midjourneys_mj_id ON midjourneys(mj_id);
CREATE INDEX IF NOT EXISTS idx_midjourneys_action ON midjourneys(action);
CREATE INDEX IF NOT EXISTS idx_midjourneys_user_id ON midjourneys(user_id);
CREATE INDEX IF NOT EXISTS idx_midjourneys_progress ON midjourneys(progress);
CREATE INDEX IF NOT EXISTS idx_midjourneys_status ON midjourneys(status);
CREATE INDEX IF NOT EXISTS idx_midjourneys_finish_time ON midjourneys(finish_time);
CREATE INDEX IF NOT EXISTS idx_midjourneys_start_time ON midjourneys(start_time);
CREATE INDEX IF NOT EXISTS idx_midjourneys_submit_time ON midjourneys(submit_time);
CREATE INDEX IF NOT EXISTS idx_qdt_created_at ON quota_data(created_at);
CREATE INDEX IF NOT EXISTS idx_qdt_model_user_name ON quota_data(model_name, username);
CREATE INDEX IF NOT EXISTS idx_quota_data_user_id ON quota_data(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_progress ON tasks(progress);
CREATE INDEX IF NOT EXISTS idx_tasks_finish_time ON tasks(finish_time);
CREATE INDEX IF NOT EXISTS idx_tasks_start_time ON tasks(start_time);
CREATE INDEX IF NOT EXISTS idx_tasks_submit_time ON tasks(submit_time);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_platform ON tasks(platform);