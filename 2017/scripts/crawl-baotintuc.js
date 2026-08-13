// Crawl Excel score files for all 63 provinces from baotintuc.vn article:
// https://baotintuc.vn/tuyen-sinh/tra-cuu-diem-thi-thpt-2017-cua-63-tinh-thanh-pho-tren-baotintucvn-20170706073512672.htm
// Saves to data/<ascii-kebab-name>.xls (63 files). Idempotent: skips already-downloaded files.
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const OUT_DIR = path.join(__dirname, "..", "data");

const LINKS = [
  ["An Giang", "https://cdnmedia.baotintuc.vn/2017/07/06/17/57/Angiang.xls"],
  ["Bac Lieu", "https://cdnmedia.baotintuc.vn/2017/07/06/17/59/Baclieu.xls"],
  ["Ba Ria - Vung Tau", "https://cdnmedia.baotintuc.vn/2017/07/06/08/17/1BaRiaVungTau.xls"],
  ["Bac Giang", "https://cdnmedia.baotintuc.vn/2017/07/06/09/33/BacGiang.xls"],
  ["Bac Kan", "https://cdnmedia.baotintuc.vn/2017/07/06/08/27/BacKan.xls"],
  ["Bac Ninh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/33/BacNinh.xls"],
  ["Ben Tre", "https://cdnmedia.baotintuc.vn/2017/07/06/09/34/BenTre.xls"],
  ["Binh Duong", "https://cdnmedia.baotintuc.vn/2017/07/06/09/34/BinhDuong.xls"],
  ["Binh Thuan", "https://cdnmedia.baotintuc.vn/2017/07/06/09/35/BinhThuan.xls"],
  ["Binh Phuoc", "https://cdnmedia.baotintuc.vn/2017/07/06/09/09/BinhPhuoc.xls"],
  ["Binh Dinh", "https://cdnmedia.baotintuc.vn/2017/07/06/13/05/Binhdinh.xls"],
  ["Ca Mau", "https://cdnmedia.baotintuc.vn/2017/07/06/13/05/Camau.xls"],
  ["Cao Bang", "https://cdnmedia.baotintuc.vn/2017/07/06/18/02/Caobang.xls"],
  ["Can Tho", "https://cdnmedia.baotintuc.vn/2017/07/06/13/07/Cantho.xls"],
  ["Da Nang", "https://cdnmedia.baotintuc.vn/2017/07/06/18/03/Danang.xls"],
  ["Dak Nong", "https://cdnmedia.baotintuc.vn/2017/07/06/09/35/DakNong.xls"],
  ["Dak Lak", "https://cdnmedia.baotintuc.vn/2017/07/06/18/02/Daklak.xls"],
  ["Dong Nai", "https://cdnmedia.baotintuc.vn/2017/07/06/13/08/dongnai.xls"],
  ["Dong Thap", "https://cdnmedia.baotintuc.vn/2017/07/06/17/59/Dongthap.xls"],
  ["Dien Bien", "https://cdnmedia.baotintuc.vn/2017/07/06/09/08/DienBien.xls"],
  ["Gia Lai", "https://cdnmedia.baotintuc.vn/2017/07/06/13/09/Gia-Lai.xls"],
  ["Ha Giang", "https://cdnmedia.baotintuc.vn/2017/07/06/18/18/Hagiang.xls"],
  ["Ha Noi", "https://cdnmedia.baotintuc.vn/2017/07/07/08/16/HaNoi.xls"],
  ["Ha Nam", "https://cdnmedia.baotintuc.vn/2017/07/06/09/07/Hanam.xls"],
  ["Ha Tinh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/36/HaTinh.xls"],
  ["Hai Phong", "https://cdnmedia.baotintuc.vn/2017/07/06/08/18/23HaiPhong.xls"],
  ["Hai Duong", "https://cdnmedia.baotintuc.vn/2017/07/06/09/36/HaiDuong.xls"],
  ["Hau Giang", "https://cdnmedia.baotintuc.vn/2017/07/06/18/00/Haugiang.xls"],
  ["Ho Chi Minh", "https://cdnmedia.baotintuc.vn/2017/07/06/08/26/HCM.xls"],
  ["Hoa Binh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/06/HoaBinh.xls"],
  ["Hung Yen", "https://cdnmedia.baotintuc.vn/2017/07/06/08/25/HungYen.xls"],
  ["Khanh Hoa", "https://cdnmedia.baotintuc.vn/2017/07/06/18/04/Khanhhoa.xls"],
  ["Kien Giang", "https://cdnmedia.baotintuc.vn/2017/07/06/08/28/KienGiang.xls"],
  ["Kon Tum", "https://cdnmedia.baotintuc.vn/2017/07/06/18/05/KonTum.xls"],
  ["Nam Dinh", "https://cdnmedia.baotintuc.vn/2017/07/06/07/55/13NamDinh.xls"],
  ["Nghe An", "https://cdnmedia.baotintuc.vn/2017/07/06/07/55/14NgheAn.xls"],
  ["Ninh Binh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/37/NinhBinh.xls"],
  ["Ninh Thuan", "https://cdnmedia.baotintuc.vn/2017/07/06/18/00/Ninhthuan.xls"],
  ["Lao Cai", "https://cdnmedia.baotintuc.vn/2017/07/06/08/52/11LaoCai.xls"],
  ["Lai Chau", "https://cdnmedia.baotintuc.vn/2017/07/06/13/33/LaiChau.xls"],
  ["Lang Son", "https://cdnmedia.baotintuc.vn/2017/07/06/18/05/Langson.xls"],
  ["Lam Dong", "https://cdnmedia.baotintuc.vn/2017/07/06/09/00/LamDong.xls"],
  ["Long An", "https://cdnmedia.baotintuc.vn/2017/07/06/08/53/12LongAn.xls"],
  ["Quang Binh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/23/QuangBinh.xls"],
  ["Quang Nam", "https://cdnmedia.baotintuc.vn/2017/07/06/09/24/QuangNam.xls"],
  ["Quang Ninh", "https://cdnmedia.baotintuc.vn/2017/07/06/18/06/Quangninh.xls"],
  ["Quang Ngai", "https://cdnmedia.baotintuc.vn/2017/07/06/09/24/QuangNgai.xls"],
  ["Quang Tri", "https://cdnmedia.baotintuc.vn/2017/07/06/09/25/QuangTri.xls"],
  ["Phu Tho", "https://cdnmedia.baotintuc.vn/2017/07/06/09/23/PhuTho.xls"],
  ["Phu Yen", "https://cdnmedia.baotintuc.vn/2017/07/06/09/38/PhuYen.xls"],
  ["Son La", "https://cdnmedia.baotintuc.vn/2017/07/06/18/01/Sonla.xls"],
  ["Soc Trang", "https://cdnmedia.baotintuc.vn/2017/07/06/18/06/Soctrang.xls"],
  ["Tay Ninh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/26/TayNinh.xls"],
  ["Thai Binh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/09/ThaiBinh.xls"],
  ["Thai Nguyen", "https://cdnmedia.baotintuc.vn/2017/07/06/13/35/ThaiNguyen.xls"],
  ["Thanh Hoa", "https://cdnmedia.baotintuc.vn/2017/07/06/13/10/thanhoa.xls"],
  ["Tra Vinh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/38/TraVinh.xls"],
  ["Thua Thien Hue", "https://cdnmedia.baotintuc.vn/2017/07/06/13/10/thuathienhue.xls"],
  ["Tien Giang", "https://cdnmedia.baotintuc.vn/2017/07/06/13/11/tiengiang.xls"],
  ["Tuyen Quang", "https://cdnmedia.baotintuc.vn/2017/07/06/09/39/TuyenQuang.xls"],
  ["Vinh Phuc", "https://cdnmedia.baotintuc.vn/2017/07/06/09/40/VinhPhuc.xls"],
  ["Vinh Long", "https://cdnmedia.baotintuc.vn/2017/07/06/13/13/vinhlong.xls"],
  ["Yen Bai", "https://cdnmedia.baotintuc.vn/2017/07/06/18/07/Yenbai.xls"],
];

function slug(name) {
  return name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
}

async function downloadOne(name, url) {
  const outPath = path.join(OUT_DIR, `${slug(name)}.xls`);
  if (fs.existsSync(outPath) && fs.statSync(outPath).size > 0) {
    return { name, url, outPath, status: "skip", size: fs.statSync(outPath).size };
  }
  const res = await fetch(url, {
    headers: {
      "User-Agent":
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36",
      Referer:
        "https://baotintuc.vn/tuyen-sinh/tra-cuu-diem-thi-thpt-2017-cua-63-tinh-thanh-pho-tren-baotintucvn-20170706073512672.htm",
    },
  });
  if (!res.ok) {
    return { name, url, outPath, status: "fail", httpStatus: res.status };
  }
  const buf = Buffer.from(await res.arrayBuffer());
  fs.writeFileSync(outPath, buf);
  return { name, url, outPath, status: "ok", size: buf.length };
}

async function main() {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  console.log(`Downloading ${LINKS.length} files to ${OUT_DIR}...`);

  const CONCURRENCY = 6;
  const results = [];
  let i = 0;
  async function worker() {
    while (i < LINKS.length) {
      const idx = i++;
      const [name, url] = LINKS[idx];
      try {
        const r = await downloadOne(name, url);
        results.push(r);
        const tag = r.status === "ok" ? "✓" : r.status === "skip" ? "·" : "✗";
        const sz = r.size ? `${(r.size / 1024).toFixed(0)} KB` : "";
        console.log(`  ${tag} [${idx + 1}/${LINKS.length}] ${name.padEnd(20)} ${sz} ${r.status === "fail" ? "HTTP " + r.httpStatus : ""}`);
      } catch (err) {
        console.log(`  ✗ [${idx + 1}/${LINKS.length}] ${name}: ${err.message}`);
        results.push({ name, url, status: "error", message: err.message });
      }
    }
  }
  await Promise.all(Array.from({ length: CONCURRENCY }, worker));

  const ok = results.filter((r) => r.status === "ok").length;
  const skip = results.filter((r) => r.status === "skip").length;
  const fail = results.filter((r) => r.status === "fail" || r.status === "error");
  console.log(`\nDone. ok=${ok} skip=${skip} fail=${fail.length}`);
  if (fail.length) {
    console.log("Failed:", fail.map((f) => f.name).join(", "));
    process.exit(1);
  }
}

main();
