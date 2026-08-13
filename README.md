# thptqg

Tra cứu điểm thi THPT Quốc gia — exam-score lookup for Vietnam's national high
school graduation exam. Client-side SQL (sql.js) over a SQLite database built
from the ministry's raw `.xls` score files by the Rust `xlsxread` CLI.

Served as one site at **[tiennm99.github.io/thptqg](https://tiennm99.github.io/thptqg/)**.

| Directory | Year | Size | Site |
| --- | --- | --- | --- |
| [`2016/`](./2016/) | THPT QG 2016 — 877,461 candidates | ~50 MB raw data | [/2016/](https://tiennm99.github.io/thptqg/2016/) |
| [`2017/`](./2017/) | THPT QG 2017 — ~861,000 candidates | ~117 MB raw data | [/2017/](https://tiennm99.github.io/thptqg/2017/) (+ [old](https://tiennm99.github.io/thptqg/2017/old/), [old2](https://tiennm99.github.io/thptqg/2017/old2/) generations) |

Each year was a standalone repository (`thptqg2016`, `thptqg2017`), merged here
with full history. Each keeps its own `tools/xlsxread` copy because the raw
data formats differ per year.

The `Deploy to GitHub Pages` workflow builds both years' databases and site
bundles and publishes them behind a root index.
