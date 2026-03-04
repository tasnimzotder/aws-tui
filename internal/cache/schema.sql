CREATE TABLE IF NOT EXISTS resources (
    service     TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    region      TEXT NOT NULL,
    profile     TEXT NOT NULL,
    name        TEXT NOT NULL,
    data        TEXT NOT NULL,
    fetched_at  INTEGER NOT NULL,
    ttl_seconds INTEGER NOT NULL,
    PRIMARY KEY (service, resource_id, region, profile)
);

CREATE TABLE IF NOT EXISTS summaries (
    service     TEXT NOT NULL,
    region      TEXT NOT NULL,
    profile     TEXT NOT NULL,
    data        TEXT NOT NULL,
    fetched_at  INTEGER NOT NULL,
    ttl_seconds INTEGER NOT NULL,
    PRIMARY KEY (service, region, profile)
);

CREATE INDEX IF NOT EXISTS idx_resources_name ON resources(name);
CREATE INDEX IF NOT EXISTS idx_resources_freshness ON resources(fetched_at);
