/// CLI argument structs via clap derive.
use std::path::PathBuf;

use clap::{Parser, Subcommand};

#[derive(Parser)]
#[command(
    name = "xlsxread",
    version,
    about = "Read .xls/.xlsx files and build SQLite databases for thptqg datasets"
)]
pub struct Cli {
    #[command(subcommand)]
    pub cmd: Cmd,
}

#[derive(Subcommand)]
pub enum Cmd {
    /// Read input spreadsheets and write a SQLite database
    Build {
        /// Path to the dataset TOML config file
        #[arg(long)]
        schema: PathBuf,

        /// Directory containing the .xls / .xlsx source files
        #[arg(long)]
        input: PathBuf,

        /// Output SQLite database path
        #[arg(long)]
        output: PathBuf,
    },

    /// Audit: compare distinct SBD count from xlsx files vs DB row count
    Audit {
        /// Path to the dataset TOML config file
        #[arg(long)]
        schema: PathBuf,

        /// Directory containing the .xlsx source files
        #[arg(long)]
        input: PathBuf,

        /// SQLite database to compare against
        #[arg(long)]
        db: PathBuf,
    },
}
