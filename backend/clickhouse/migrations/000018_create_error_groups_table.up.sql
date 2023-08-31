CREATE TABLE IF NOT EXISTS error_groups (
    ProjectID Int32,
    CreatedAt DateTime64(6),
    ID Int64,
    Event String,
    Status LowCardinality(String),
    Type LowCardinality(String),
) ENGINE = ReplacingMergeTree
ORDER BY (
        ProjectID,
        CreatedAt,
        ID
    );