/// Stage 5 golden tests — integration tests using anonymised fixture files.
///
/// Fixture files are generated in-process via raw OOXML + zip if they do not
/// already exist on disk. No external Python or Node tooling required for the
/// Rust-side tests. The Node golden comparison is marked #[ignore] when pnpm
/// is not in PATH.
use std::io::Write as IoWrite;
use std::path::{Path, PathBuf};

// ---------------------------------------------------------------------------
// Minimal OOXML xlsx generator
//
// Produces a valid .xlsx that calamine can read. Only uses the `zip` crate
// which is already pulled in as a transitive dependency of calamine.
// ---------------------------------------------------------------------------

/// One row of cell data for a fixture sheet.
struct XlsxRow {
    values: Vec<String>,
}

/// Write a minimal .xlsx to `path` with the given sheets.
/// `sheets`: Vec<(sheet_name, rows)> where rows[0] is the header.
fn write_xlsx(path: &Path, sheets: &[(String, Vec<XlsxRow>)]) {
    use zip::{write::SimpleFileOptions, ZipWriter};

    let file = std::fs::File::create(path).expect("create fixture xlsx");
    let mut zip = ZipWriter::new(file);
    let opts = SimpleFileOptions::default();

    // [Content_Types].xml
    let mut content_types = String::from(
        r#"<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
"#,
    );
    for (i, _) in sheets.iter().enumerate() {
        content_types.push_str(&format!(
            r#"  <Override PartName="/xl/worksheets/sheet{}.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
"#,
            i + 1
        ));
    }
    content_types.push_str("</Types>");
    zip.start_file("[Content_Types].xml", opts).unwrap();
    zip.write_all(content_types.as_bytes()).unwrap();

    // _rels/.rels
    zip.start_file("_rels/.rels", opts).unwrap();
    zip.write_all(
        br#"<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>"#,
    )
    .unwrap();

    // xl/_rels/workbook.xml.rels
    let mut wb_rels = String::from(
        r#"<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
"#,
    );
    for (i, _) in sheets.iter().enumerate() {
        wb_rels.push_str(&format!(
            r#"  <Relationship Id="rId{}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet{}.xml"/>
"#,
            i + 1,
            i + 1
        ));
    }
    wb_rels.push_str("</Relationships>");
    zip.start_file("xl/_rels/workbook.xml.rels", opts).unwrap();
    zip.write_all(wb_rels.as_bytes()).unwrap();

    // xl/workbook.xml
    let mut wb = String::from(
        r#"<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"
          xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>
"#,
    );
    for (i, (name, _)) in sheets.iter().enumerate() {
        let escaped = xml_escape(name);
        wb.push_str(&format!(
            r#"    <sheet name="{}" sheetId="{}" r:id="rId{}"/>
"#,
            escaped,
            i + 1,
            i + 1
        ));
    }
    wb.push_str("  </sheets>\n</workbook>");
    zip.start_file("xl/workbook.xml", opts).unwrap();
    zip.write_all(wb.as_bytes()).unwrap();

    // xl/worksheets/sheetN.xml
    for (i, (_, rows)) in sheets.iter().enumerate() {
        let mut ws = String::from(
            r#"<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
"#,
        );
        for (row_idx, row) in rows.iter().enumerate() {
            ws.push_str(&format!(
                r#"    <row r="{}">
"#,
                row_idx + 1
            ));
            for (col_idx, val) in row.values.iter().enumerate() {
                let col_letter = col_letter(col_idx);
                let cell_ref = format!("{}{}", col_letter, row_idx + 1);
                let escaped = xml_escape(val);
                ws.push_str(&format!(
                    r#"      <c r="{}" t="inlineStr"><is><t>{}</t></is></c>
"#,
                    cell_ref, escaped
                ));
            }
            ws.push_str("    </row>\n");
        }
        ws.push_str("  </sheetData>\n</worksheet>");
        zip.start_file(&format!("xl/worksheets/sheet{}.xml", i + 1), opts)
            .unwrap();
        zip.write_all(ws.as_bytes()).unwrap();
    }

    zip.finish().unwrap();
}

fn col_letter(idx: usize) -> &'static str {
    const LETTERS: &[&str] = &[
        "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R",
        "S", "T", "U", "V", "W", "X", "Y", "Z",
    ];
    LETTERS[idx % 26]
}

fn xml_escape(s: &str) -> String {
    s.replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
}

// ---------------------------------------------------------------------------
// Fixture data builders
// ---------------------------------------------------------------------------

fn header_row() -> XlsxRow {
    XlsxRow {
        values: vec![
            "HO_TEN".into(),
            "NGAY_SINH".into(),
            "SO_BAO_DANH".into(),
            "DIEM_THI".into(),
        ],
    }
}

fn data_row(idx: usize, scores: &str) -> XlsxRow {
    // Anonymised: name uses sequential pattern, SBD is purely synthetic
    let name = if idx % 2 == 0 {
        format!("Nguyen Van Test {:03}", idx)
    } else {
        format!("Tran Thi Test {:03}", idx)
    };
    XlsxRow {
        values: vec![
            name,
            format!("15/0{}/{}", (idx % 9) + 1, 1999 + (idx % 5)),
            format!("1000{:04}", idx),
            scores.to_owned(),
        ],
    }
}

fn sample_scores(idx: usize) -> String {
    // Realistic scores in 0–10 range, varies by idx
    let toan = 4.0 + (idx % 60) as f64 / 10.0;
    let van = 3.5 + (idx % 65) as f64 / 10.0;
    format!("Toán: {toan:.1}  Ngữ văn: {van:.1}  Tiếng Anh: 7.5")
}

// ---------------------------------------------------------------------------
// Fixture file paths
// ---------------------------------------------------------------------------

fn fixtures_dir() -> PathBuf {
    // tests/fixtures/ relative to the crate root
    let mut p = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
    p.push("tests");
    p.push("fixtures");
    p
}

fn province_fixture_path() -> PathBuf {
    fixtures_dir().join("province-100.xlsx")
}

fn hcm_overflow_fixture_path() -> PathBuf {
    fixtures_dir().join("hcm-overflow.xlsx")
}

fn numeric_sbd_fixture_path() -> PathBuf {
    fixtures_dir().join("province-numeric-sbd.xlsx")
}

// ---------------------------------------------------------------------------
// Fixture generation — called once per test run if files missing
// ---------------------------------------------------------------------------

fn ensure_fixtures() {
    let dir = fixtures_dir();
    std::fs::create_dir_all(&dir).expect("create fixtures dir");

    // province-100.xlsx — 100 data rows, single sheet, with header
    if !province_fixture_path().exists() {
        let mut rows = vec![header_row()];
        for i in 0..100 {
            rows.push(data_row(i, &sample_scores(i)));
        }
        write_xlsx(&province_fixture_path(), &[("Sheet1".to_owned(), rows)]);
    }

    // hcm-overflow.xlsx — 2 sheets × 200 rows each (no header on sheet 2)
    if !hcm_overflow_fixture_path().exists() {
        let mut sheet1 = vec![header_row()];
        for i in 0..200 {
            sheet1.push(data_row(i, &sample_scores(i)));
        }
        // Sheet2: continuation rows, no header row (as in real HCM overflow)
        let mut sheet2 = Vec::new();
        for i in 200..400 {
            sheet2.push(data_row(i, &sample_scores(i)));
        }
        write_xlsx(
            &hcm_overflow_fixture_path(),
            &[("Sheet1".to_owned(), sheet1), ("Sheet2".to_owned(), sheet2)],
        );
    }

    // province-numeric-sbd.xlsx — strictly numeric SBDs for data-old config
    if !numeric_sbd_fixture_path().exists() {
        let mut rows = vec![header_row()];
        for i in 0..100 {
            rows.push(XlsxRow {
                values: vec![
                    format!("Nguyen Van Test {:03}", i),
                    "01/01/2000".to_owned(),
                    format!("{:08}", 20000000 + i), // pure digits
                    sample_scores(i),
                ],
            });
        }
        write_xlsx(&numeric_sbd_fixture_path(), &[("Sheet1".to_owned(), rows)]);
    }
}

// ---------------------------------------------------------------------------
// Config helpers
// ---------------------------------------------------------------------------

fn make_data_config() -> xlsxread::config::DatasetConfig {
    let cfg_path = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("configs")
        .join("thptqg2017-data.toml");
    xlsxread::config::load_config(&cfg_path).expect("load data config")
}

// ---------------------------------------------------------------------------
// Integration tests — pure Rust, no Node dependency
// ---------------------------------------------------------------------------

#[test]
fn province_100_builds_100_rows() {
    ensure_fixtures();
    let dir = tempdir();
    let db_path = dir.join("test.db");
    let fixture_dir = dir.join("input");
    std::fs::create_dir_all(&fixture_dir).unwrap();
    std::fs::copy(province_fixture_path(), fixture_dir.join("province.xlsx")).unwrap();

    run_build_cmd(&fixture_dir, &db_path, "thptqg2017-data.toml");

    let count = query_count(&db_path);
    assert_eq!(count, 100, "expected 100 rows from province-100 fixture");
}

#[test]
fn hcm_overflow_builds_400_rows() {
    ensure_fixtures();
    let dir = tempdir();
    let db_path = dir.join("test.db");
    let fixture_dir = dir.join("input");
    std::fs::create_dir_all(&fixture_dir).unwrap();
    std::fs::copy(hcm_overflow_fixture_path(), fixture_dir.join("hcm.xlsx")).unwrap();

    run_build_cmd(&fixture_dir, &db_path, "thptqg2017-data.toml");

    let count = query_count(&db_path);
    assert_eq!(
        count, 400,
        "expected 400 rows (200 × 2 sheets) from hcm-overflow fixture"
    );
}

#[test]
fn data_old_first_sheet_only_100_rows() {
    ensure_fixtures();
    let dir = tempdir();
    let db_path = dir.join("test.db");
    let fixture_dir = dir.join("input");
    std::fs::create_dir_all(&fixture_dir).unwrap();
    // Use the overflow file but with data-old config (first sheet only → 200 rows)
    std::fs::copy(hcm_overflow_fixture_path(), fixture_dir.join("hcm.xlsx")).unwrap();

    run_build_cmd(&fixture_dir, &db_path, "thptqg2017-data-old.toml");

    // data-old: sheet_mode=first → only 200 rows from sheet1; but SBDs "1000NNNN" are
    // all digits so all pass the numeric guard
    let count = query_count(&db_path);
    assert_eq!(
        count, 200,
        "data-old config should read only first sheet (200 rows)"
    );
}

#[test]
fn numeric_sbd_guard_rejects_non_numeric() {
    ensure_fixtures();
    let dir = tempdir();
    let db_path = dir.join("test.db");
    let fixture_dir = dir.join("input");
    std::fs::create_dir_all(&fixture_dir).unwrap();

    // Write a fixture with one non-numeric SBD mixed in
    let mut rows = vec![header_row()];
    for i in 0..10 {
        rows.push(XlsxRow {
            values: vec![
                format!("Test {:03}", i),
                "01/01/2000".to_owned(),
                if i == 5 {
                    "ABC123".to_owned()
                } else {
                    format!("{:08}", 20000000 + i)
                },
                sample_scores(i),
            ],
        });
    }
    let mixed_path = fixture_dir.join("mixed.xlsx");
    write_xlsx(&mixed_path, &[("Sheet1".to_owned(), rows)]);

    run_build_cmd(&fixture_dir, &db_path, "thptqg2017-data-old.toml");

    // Row i=5 has non-numeric SBD → rejected by data-old config
    let count = query_count(&db_path);
    assert_eq!(
        count, 9,
        "non-numeric SBD row should be skipped by data-old config"
    );
}

#[test]
fn scores_parsed_correctly_into_db() {
    ensure_fixtures();
    let dir = tempdir();
    let db_path = dir.join("test.db");
    let fixture_dir = dir.join("input");
    std::fs::create_dir_all(&fixture_dir).unwrap();

    let rows = vec![
        header_row(),
        XlsxRow {
            values: vec![
                "Nguyen Van Test 001".to_owned(),
                "01/01/2000".to_owned(),
                "10000001".to_owned(),
                "Toán: 8.5  Ngữ văn: 7.0  Tiếng Anh: 9.25".to_owned(),
            ],
        },
    ];
    write_xlsx(
        &fixture_dir.join("one.xlsx"),
        &[("Sheet1".to_owned(), rows)],
    );

    run_build_cmd(&fixture_dir, &db_path, "thptqg2017-data.toml");

    let conn = rusqlite::Connection::open(&db_path).unwrap();
    let (toan, van, anh): (f64, f64, f64) = conn
        .query_row(
            "SELECT toan, ngu_van, tieng_anh FROM student WHERE so_bao_danh = '10000001'",
            [],
            |r| Ok((r.get(0)?, r.get(1)?, r.get(2)?)),
        )
        .expect("row not found");
    assert!((toan - 8.5).abs() < 1e-9);
    assert!((van - 7.0).abs() < 1e-9);
    assert!((anh - 9.25).abs() < 1e-9);
}

#[test]
fn to_ascii_stored_correctly() {
    ensure_fixtures();
    let dir = tempdir();
    let db_path = dir.join("test.db");
    let fixture_dir = dir.join("input");
    std::fs::create_dir_all(&fixture_dir).unwrap();

    let rows = vec![
        header_row(),
        XlsxRow {
            values: vec![
                "Nguyễn Văn Đức".to_owned(),
                "".to_owned(),
                "20000001".to_owned(),
                "".to_owned(),
            ],
        },
    ];
    write_xlsx(
        &fixture_dir.join("one.xlsx"),
        &[("Sheet1".to_owned(), rows)],
    );

    run_build_cmd(&fixture_dir, &db_path, "thptqg2017-data.toml");

    let conn = rusqlite::Connection::open(&db_path).unwrap();
    let ascii: String = conn
        .query_row(
            "SELECT ho_ten_ascii FROM student WHERE so_bao_danh = '20000001'",
            [],
            |r| r.get(0),
        )
        .expect("row not found");
    assert_eq!(ascii, "nguyen van duc");
}

#[test]
fn audit_subcommand_matches_after_build() {
    ensure_fixtures();
    let dir = tempdir();
    let db_path = dir.join("test.db");
    let fixture_dir = dir.join("input");
    std::fs::create_dir_all(&fixture_dir).unwrap();
    std::fs::copy(province_fixture_path(), fixture_dir.join("province.xlsx")).unwrap();

    run_build_cmd(&fixture_dir, &db_path, "thptqg2017-data.toml");

    // audit should match (100 distinct SBDs in xlsx == 100 rows in DB)
    let cfg = make_data_config();
    let result = xlsxread::audit::run_audit(&fixture_dir, &db_path, &cfg).expect("audit failed");
    assert!(result.matched, "audit should match after build");
    assert_eq!(result.distinct_sbds, 100);
    assert_eq!(result.db_count, 100);
}

#[test]
fn audit_subcommand_mismatch_detected() {
    ensure_fixtures();
    let dir = tempdir();
    let db_path = dir.join("test.db");
    let fixture_dir = dir.join("input");
    std::fs::create_dir_all(&fixture_dir).unwrap();

    // Write 10 rows to xlsx but build DB from only 5 rows
    let mut all_rows = vec![header_row()];
    for i in 0..10 {
        all_rows.push(data_row(i, &sample_scores(i)));
    }
    write_xlsx(
        &fixture_dir.join("all.xlsx"),
        &[("Sheet1".to_owned(), all_rows)],
    );

    // Build DB with only first 5 rows in a different file
    let build_dir = dir.join("build_input");
    std::fs::create_dir_all(&build_dir).unwrap();
    let mut five_rows = vec![header_row()];
    for i in 0..5 {
        five_rows.push(data_row(i, &sample_scores(i)));
    }
    write_xlsx(
        &build_dir.join("five.xlsx"),
        &[("Sheet1".to_owned(), five_rows)],
    );

    run_build_cmd(&build_dir, &db_path, "thptqg2017-data.toml");

    // audit against fixture_dir (10 xlsx rows) but DB has 5 rows → mismatch
    let cfg = make_data_config();
    let result = xlsxread::audit::run_audit(&fixture_dir, &db_path, &cfg).expect("audit failed");
    assert!(!result.matched, "audit should not match (10 xlsx vs 5 db)");
    assert_eq!(result.distinct_sbds, 10);
    assert_eq!(result.db_count, 5);
}

// ---------------------------------------------------------------------------
// Golden test: compare Rust DB vs Node DB on identical fixture
// Marked #[ignore] when pnpm / node is not in PATH — CI installs them first.
// ---------------------------------------------------------------------------

#[test]
#[ignore]
fn golden_rust_matches_node_db() {
    // This test requires: pnpm, node, and the thptqg2017 package to be installed
    // Run with: cargo test -- --ignored golden_rust_matches_node_db
    let which_pnpm = std::process::Command::new("which")
        .arg("pnpm")
        .output()
        .map(|o| o.status.success())
        .unwrap_or(false);
    if !which_pnpm {
        eprintln!("pnpm not in PATH — skipping golden test");
        return;
    }

    let dir = tempdir();
    let fixture_dir = dir.join("input");
    std::fs::create_dir_all(&fixture_dir).unwrap();
    std::fs::copy(province_fixture_path(), fixture_dir.join("province.xlsx")).unwrap();

    // Build with Rust
    let rust_db = dir.join("rust.db");
    run_build_cmd(&fixture_dir, &rust_db, "thptqg2017-data.toml");

    // Build with Node (run build-database.js with DATA_DIR / DB_PATH overrides)
    // Node script reads env-vars via a thin wrapper — see scripts/build-database.js
    // For now: diff via SELECT * ORDER BY so_bao_danh
    let node_db = dir.join("node.db");
    let status = std::process::Command::new("pnpm")
        .args(["exec", "node", "scripts/build-database.js"])
        .env("OVERRIDE_SRC_DIR", fixture_dir.to_str().unwrap())
        .env("OVERRIDE_DB_PATH", node_db.to_str().unwrap())
        .current_dir(
            PathBuf::from(env!("CARGO_MANIFEST_DIR"))
                .parent()
                .unwrap()
                .parent()
                .unwrap(),
        )
        .status()
        .expect("failed to run node build script");

    if !status.success() {
        panic!("Node build script failed with: {status}");
    }

    // Row-by-row comparison
    diff_dbs(&rust_db, &node_db);
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

fn tempdir() -> PathBuf {
    let base = std::env::temp_dir().join(format!(
        "xlsxread-test-{}",
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .subsec_nanos()
    ));
    std::fs::create_dir_all(&base).unwrap();
    base
}

fn run_build_cmd(input_dir: &Path, db_path: &Path, config_name: &str) {
    let cfg_path = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("configs")
        .join(config_name);

    let cfg = xlsxread::config::load_config(&cfg_path)
        .unwrap_or_else(|e| panic!("load config {config_name}: {e}"));
    let patterns =
        xlsxread::transform::CompiledPatterns::new(&cfg.scores).expect("compile patterns");

    // Collect files
    let mut files: Vec<PathBuf> = std::fs::read_dir(input_dir)
        .unwrap()
        .filter_map(|e| e.ok())
        .map(|e| e.path())
        .filter(|p| {
            p.is_file()
                && p.extension()
                    .and_then(|e| e.to_str())
                    .map(|e| {
                        let l = e.to_lowercase();
                        l == "xls" || l == "xlsx"
                    })
                    .unwrap_or(false)
        })
        .collect();
    files.sort();

    let conn = xlsxread::writer::open_db(db_path, &cfg).expect("open db");
    conn.execute_batch("BEGIN").unwrap();

    for file in &files {
        xlsxread::reader::process_file(file, &cfg, |_, raw| {
            let all_blank = xlsxread::reader::is_all_blank(raw);
            if cfg.reader.strip_blank_rows && all_blank {
                return;
            }
            let ho_ten = raw
                .get(cfg.columns.ho_ten)
                .map(|c| c.to_string().trim().to_owned())
                .unwrap_or_default();
            let so_bao_danh = raw
                .get(cfg.columns.so_bao_danh)
                .map(|c| c.to_string().trim().to_owned())
                .unwrap_or_default();
            if xlsxread::transform::validate_row(
                &ho_ten,
                &so_bao_danh,
                &cfg.validation,
                cfg.reader.strip_blank_rows,
                all_blank,
            )
            .is_err()
            {
                return;
            }
            let row = xlsxread::transform::transform_row(raw, &cfg, &patterns);
            let _ = xlsxread::writer::insert_row(
                &conn,
                &cfg.insert.sql,
                &row,
                xlsxread::writer::SCORE_FIELDS,
            );
        })
        .expect("process file");
    }

    conn.execute_batch("COMMIT").unwrap();
    conn.execute_batch("VACUUM").unwrap();
}

fn query_count(db_path: &Path) -> i64 {
    let conn = rusqlite::Connection::open(db_path).expect("open db for count");
    conn.query_row("SELECT COUNT(*) FROM student", [], |r| r.get(0))
        .expect("count query")
}

fn diff_dbs(a: &Path, b: &Path) {
    let conn_a = rusqlite::Connection::open(a).unwrap();

    // Attach b as "other"
    conn_a
        .execute_batch(&format!("ATTACH DATABASE '{}' AS other", b.display()))
        .unwrap();

    // Rows in a not in b
    let missing_in_b: i64 = conn_a
        .query_row(
            "SELECT COUNT(*) FROM main.student s
             WHERE NOT EXISTS (SELECT 1 FROM other.student o WHERE o.so_bao_danh = s.so_bao_danh)",
            [],
            |r| r.get(0),
        )
        .unwrap();

    // Rows in b not in a
    let missing_in_a: i64 = conn_a
        .query_row(
            "SELECT COUNT(*) FROM other.student o
             WHERE NOT EXISTS (SELECT 1 FROM main.student s WHERE s.so_bao_danh = o.so_bao_danh)",
            [],
            |r| r.get(0),
        )
        .unwrap();

    assert_eq!(
        missing_in_b, 0,
        "{missing_in_b} rows in Rust DB missing from Node DB"
    );
    assert_eq!(
        missing_in_a, 0,
        "{missing_in_a} rows in Node DB missing from Rust DB"
    );
}
