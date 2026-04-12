# Phase 02 — Excel Parser + DB Builder

## Context Links
- [plan.md](./plan.md)
- [phase-01](./phase-01-project-scaffolding.md) — prerequisite
- [Converter.java](../../src/main/java/dev/miti99/thptqg2017/Converter.java) — source parsing logic to port

## Overview
- **Priority**: P1 (produces the .db file needed by Phase 03)
- **Status**: Pending
- **Description**: Node.js script that reads ~119 Excel files, regex-parses scores, writes SQLite .db

## Key Insights

From Converter.java analysis:
- 4 columns per row: hoTen (0), ngaySinh (1), soBaoDanh (2), diemThi (3)
- Column 3 is a text blob with scores like `"Toán: 8.5 Ngữ văn: 7.0 Vật lí: 6.25 ..."`
- 11 subject patterns extracted via regex; not all subjects present for every student
- Some files have no header row (comment in Java: "Một số file lỗi nên không chắc có header")
- `session.merge()` = upsert behavior; update folder files should overwrite raw folder data
- ngaySinh format: `dd/MM/yyyy`

**Format edge cases to handle:**
- Row 0 might be header OR data — detect by checking if cell 2 (soBaoDanh) looks like an ID pattern
- Some cells may be numeric instead of string (Excel auto-detection)
- Duplicate .xlsx files exist: `10_LamDong_GNFT (1).xls.xlsx` and `10_LamDong_GNFT.xls.xlsx`

## Requirements

### Functional
- Parse all .xlsx from `src/main/resources/raw/` and `src/main/resources/raw/(update)/`
- Process `raw/` first, then `(update)/` so updates overwrite via INSERT OR REPLACE
- Extract all 11 subject scores via regex (matching Java patterns exactly)
- Store ngaySinh as text string (no date conversion)
- Output `web/public/thptqg2017.db`

### Non-Functional
- Complete in < 60 seconds on modern hardware
- Log progress: file count, row count, error count
- Skip malformed rows gracefully (log, don't crash)

## Architecture

```
scripts/build-database.js
  ├── reads: src/main/resources/raw/**/*.xlsx
  ├── reads: src/main/resources/raw/(update)/**/*.xlsx
  ├── uses: xlsx (SheetJS) for Excel parsing
  ├── uses: better-sqlite3 for fast SQLite writes
  └── outputs: web/public/thptqg2017.db
```

### Data Flow Per File

```
.xlsx file
  → xlsx.readFile()
  → sheet_to_json({ header: 1, raw: false })  // array of arrays, all strings
  → for each row:
      → validate: row[2] matches soBaoDanh pattern (skip headers/junk)
      → extract: hoTen, ngaySinh, soBaoDanh from columns 0-2
      → regex match: diemThi (column 3) against 11 subject patterns
      → INSERT OR REPLACE into student table
```

## Related Code Files

**Create:**
- `scripts/build-database.js` — main parser script (~120 lines)

**Read (reference only):**
- `src/main/java/dev/miti99/thptqg2017/Converter.java`

**No modifications to existing files.**

## Implementation Steps

### 1. Create score-parsing utility

Port the 11 regex patterns from Converter.java:

```javascript
const SCORE_PATTERNS = {
  toan:      /Toán:\s*(\d*\.\d*)/,
  ngu_van:   /Ngữ văn:\s*(\d*\.\d*)/,
  vat_ly:    /Vật lí:\s*(\d*\.\d*)/,
  hoa_hoc:   /Hóa học:\s*(\d*\.\d*)/,
  sinh_hoc:  /Sinh học:\s*(\d*\.\d*)/,
  khtn:      /KHTN:\s*(\d*\.\d*)/,
  lich_su:   /Lịch sử:\s*(\d*\.\d*)/,
  dia_ly:    /Địa lí:\s*(\d*\.\d*)/,
  gdcd:      /GDCD:\s*(\d*\.\d*)/,
  khxh:      /KHXH:\s*(\d*\.\d*)/,
  tieng_anh: /Tiếng Anh:\s*(\d*\.\d*)/,
};
```

### 2. Create database schema

```javascript
db.exec(`
  CREATE TABLE IF NOT EXISTS student (
    so_bao_danh TEXT PRIMARY KEY,
    ho_ten      TEXT NOT NULL,
    ngay_sinh   TEXT,
    toan        REAL, ngu_van REAL, vat_ly REAL,
    hoa_hoc     REAL, sinh_hoc REAL, khtn REAL,
    lich_su     REAL, dia_ly REAL, gdcd REAL,
    khxh        REAL, tieng_anh REAL
  );
`);
```

### 3. Build the main parsing loop

```
for each folder in [raw/, raw/(update)/]:
  for each .xlsx file:
    workbook = xlsx.readFile(filePath)
    sheet = workbook.Sheets[workbook.SheetNames[0]]
    rows = xlsx.utils.sheet_to_json(sheet, { header: 1, raw: false })
    for each row:
      if row[2] doesn't look like a valid soBaoDanh → skip (header detection)
      parse scores from row[3]
      INSERT OR REPLACE
```

### 4. Header detection heuristic

```javascript
// soBaoDanh is typically a numeric string like "02000001"
function isDataRow(row) {
  return row[2] && /^\d{6,}$/.test(String(row[2]).trim());
}
```

### 5. Wrap inserts in a transaction

```javascript
const insert = db.prepare(`INSERT OR REPLACE INTO student (...) VALUES (...)`);
const insertMany = db.transaction((rows) => {
  for (const row of rows) insert.run(row);
});
```

### 6. Add indexes after all inserts

```javascript
db.exec('CREATE INDEX IF NOT EXISTS idx_ho_ten ON student(ho_ten)');
```

### 7. Add summary logging

```
console.log(`Processed ${fileCount} files, ${rowCount} rows, ${errorCount} errors`);
```

### 8. Ensure output directory exists

```javascript
fs.mkdirSync('web/public', { recursive: true });
```

## Todo List

- [ ] Create `scripts/build-database.js`
- [ ] Port all 11 regex patterns from Converter.java
- [ ] Implement header-detection heuristic
- [ ] Process raw/ folder first, then (update)/ folder
- [ ] Wrap all inserts in single transaction for speed
- [ ] Add index on ho_ten after insert
- [ ] Log file count, row count, error count
- [ ] Run script and verify row count matches Java output
- [ ] Verify .db file size is reasonable (< 60MB expected)

## Success Criteria

1. `node scripts/build-database.js` completes without unhandled errors
2. Output .db contains same number of unique students as existing database.sqlite
3. Spot-check: query 5 random soBaoDanh values → scores match between old and new DB
4. Script completes in < 60 seconds
5. Skipped/errored rows are logged with file name and row number

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| xlsx library misreads Vietnamese characters | Low | High | Use `{ raw: false }` to get string values; test with known file |
| Regex patterns don't match all score formats | Medium | Medium | Test against sample rows; add `\d+\.?\d*` fallback if needed |
| better-sqlite3 won't install on Windows | Low | Medium | Prebuild binaries exist; fallback: use sql.js for build too |
| Memory pressure with 119 files open | Low | Low | Process files sequentially, one at a time |
| Duplicate students from overlapping files | Medium | Low | INSERT OR REPLACE handles this; (update) processed last wins |

## Security Considerations
- Public exam data, no PII concerns beyond names (already public)
- No network access needed during build
- Output .db is read-only in production

## Next Steps
- Phase 03 consumes the `web/public/thptqg2017.db` file produced here
- Verify row count against existing database before proceeding
