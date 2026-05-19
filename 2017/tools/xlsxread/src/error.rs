use thiserror::Error;

#[derive(Debug, Error)]
pub enum BuildError {
    #[error("I/O error for {path}: {source}")]
    Io {
        path: String,
        #[source]
        source: std::io::Error,
    },

    #[error("Calamine error for {path}: {source}")]
    Calamine {
        path: String,
        #[source]
        source: calamine::Error,
    },

    #[error("SQLite error: {0}")]
    Sqlite(#[from] rusqlite::Error),

    #[error("Config parse error: {0}")]
    Config(#[from] toml::de::Error),

    #[error("Regex compile error for pattern '{pattern}': {source}")]
    Regex {
        pattern: String,
        #[source]
        source: regex::Error,
    },

    #[error("Schema has no sheets in file: {0}")]
    NoSheets(String),
}
