// One-off audit: detect content-identical Excel files via md5
import crypto from "crypto";
import fs from "fs";
import path from "path";

const dirs = [
  "D:/tiennm99/thptqg2017/data/raw",
  "D:/tiennm99/thptqg2017/data/raw/update",
];

const byHash = {};
for (const d of dirs) {
  for (const f of fs.readdirSync(d)) {
    const full = path.join(d, f);
    if (!fs.statSync(full).isFile() || !f.endsWith(".xlsx")) continue;
    const h = crypto.createHash("md5").update(fs.readFileSync(full)).digest("hex");
    (byHash[h] ||= []).push(full);
  }
}

const total = Object.values(byHash).reduce((s, a) => s + a.length, 0);
const dupes = Object.entries(byHash).filter(([, a]) => a.length > 1);
console.log(`Total files: ${total}`);
console.log(`Unique by md5: ${Object.keys(byHash).length}`);
console.log(`Duplicate groups: ${dupes.length}`);
for (const [h, a] of dupes) {
  console.log(`  ${h.slice(0, 12)}:`);
  a.forEach((p) => console.log(`    ${p}`));
}
