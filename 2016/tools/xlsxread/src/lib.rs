/// Public library interface for integration tests.
/// The binary entry point is src/main.rs; this file re-exports the internal
/// modules so tests/golden.rs can call them without going through the CLI.
pub mod audit;
pub mod config;
pub mod error;
pub mod format_detect_2016;
pub mod reader;
pub mod transform;
pub mod writer;
