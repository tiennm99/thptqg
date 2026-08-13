package sources

import (
	"regexp"
	"strings"
)

// article is the page the links were taken from. The CDN rejects requests that
// do not carry it as a Referer.
const article = "https://baotintuc.vn/tuyen-sinh/tra-cuu-diem-thi-thpt-2017-cua-63-tinh-thanh-pho-tren-baotintucvn-20170706073512672.htm"

// browserUA is required too: the CDN 403s an unrecognised User-Agent.
const browserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"

// source2017 fetches the 2017 dataset from the baotintuc.vn CDN: one .xls per
// province.
//
// This is the only dataset that can be re-fetched from its original source.
// The CDN filenames are inconsistent (Angiang.xls, 1BaRiaVungTau.xls,
// 23HaiPhong.xls, gia-lai.xls), so local names are derived from the province
// name instead — which is what produced the data/2017/ files currently on disk,
// and what any re-crawl must keep producing.
var source2017 = Source{
	ID:      "2017",
	Summary: "63 province .xls files from the baotintuc.vn CDN",
	Headers: map[string]string{
		"User-Agent": browserUA,
		"Referer":    article,
	},
	Files: func() ([]File, error) {
		out := make([]File, 0, len(provinces2017))
		for _, p := range provinces2017 {
			out = append(out, File{
				Name: p.name,
				URL:  p.url,
				Dest: slug(p.name) + ".xls",
			})
		}
		return out, nil
	},
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// slug converts a province name to its local filename stem: lowercase, with
// every run of non-alphanumeric characters collapsed to a single hyphen.
//
//	"Ba Ria - Vung Tau" -> "ba-ria-vung-tau"
//
// The names are already unaccented ASCII, so no transliteration is involved.
func slug(name string) string {
	return strings.Trim(nonAlphanumeric.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

type province struct{ name, url string }

var provinces2017 = []province{
	{"An Giang", "https://cdnmedia.baotintuc.vn/2017/07/06/17/57/Angiang.xls"},
	{"Bac Lieu", "https://cdnmedia.baotintuc.vn/2017/07/06/17/59/Baclieu.xls"},
	{"Ba Ria - Vung Tau", "https://cdnmedia.baotintuc.vn/2017/07/06/08/17/1BaRiaVungTau.xls"},
	{"Bac Giang", "https://cdnmedia.baotintuc.vn/2017/07/06/09/33/BacGiang.xls"},
	{"Bac Kan", "https://cdnmedia.baotintuc.vn/2017/07/06/08/27/BacKan.xls"},
	{"Bac Ninh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/33/BacNinh.xls"},
	{"Ben Tre", "https://cdnmedia.baotintuc.vn/2017/07/06/09/34/BenTre.xls"},
	{"Binh Duong", "https://cdnmedia.baotintuc.vn/2017/07/06/09/34/BinhDuong.xls"},
	{"Binh Thuan", "https://cdnmedia.baotintuc.vn/2017/07/06/09/35/BinhThuan.xls"},
	{"Binh Phuoc", "https://cdnmedia.baotintuc.vn/2017/07/06/09/09/BinhPhuoc.xls"},
	{"Binh Dinh", "https://cdnmedia.baotintuc.vn/2017/07/06/13/05/Binhdinh.xls"},
	{"Ca Mau", "https://cdnmedia.baotintuc.vn/2017/07/06/13/05/Camau.xls"},
	{"Cao Bang", "https://cdnmedia.baotintuc.vn/2017/07/06/18/02/Caobang.xls"},
	{"Can Tho", "https://cdnmedia.baotintuc.vn/2017/07/06/13/07/Cantho.xls"},
	{"Da Nang", "https://cdnmedia.baotintuc.vn/2017/07/06/18/03/Danang.xls"},
	{"Dak Nong", "https://cdnmedia.baotintuc.vn/2017/07/06/09/35/DakNong.xls"},
	{"Dak Lak", "https://cdnmedia.baotintuc.vn/2017/07/06/18/02/Daklak.xls"},
	{"Dong Nai", "https://cdnmedia.baotintuc.vn/2017/07/06/13/08/dongnai.xls"},
	{"Dong Thap", "https://cdnmedia.baotintuc.vn/2017/07/06/17/59/Dongthap.xls"},
	{"Dien Bien", "https://cdnmedia.baotintuc.vn/2017/07/06/09/08/DienBien.xls"},
	{"Gia Lai", "https://cdnmedia.baotintuc.vn/2017/07/06/13/09/Gia-Lai.xls"},
	{"Ha Giang", "https://cdnmedia.baotintuc.vn/2017/07/06/18/18/Hagiang.xls"},
	{"Ha Noi", "https://cdnmedia.baotintuc.vn/2017/07/07/08/16/HaNoi.xls"},
	{"Ha Nam", "https://cdnmedia.baotintuc.vn/2017/07/06/09/07/Hanam.xls"},
	{"Ha Tinh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/36/HaTinh.xls"},
	{"Hai Phong", "https://cdnmedia.baotintuc.vn/2017/07/06/08/18/23HaiPhong.xls"},
	{"Hai Duong", "https://cdnmedia.baotintuc.vn/2017/07/06/09/36/HaiDuong.xls"},
	{"Hau Giang", "https://cdnmedia.baotintuc.vn/2017/07/06/18/00/Haugiang.xls"},
	{"Ho Chi Minh", "https://cdnmedia.baotintuc.vn/2017/07/06/08/26/HCM.xls"},
	{"Hoa Binh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/06/HoaBinh.xls"},
	{"Hung Yen", "https://cdnmedia.baotintuc.vn/2017/07/06/08/25/HungYen.xls"},
	{"Khanh Hoa", "https://cdnmedia.baotintuc.vn/2017/07/06/18/04/Khanhhoa.xls"},
	{"Kien Giang", "https://cdnmedia.baotintuc.vn/2017/07/06/08/28/KienGiang.xls"},
	{"Kon Tum", "https://cdnmedia.baotintuc.vn/2017/07/06/18/05/KonTum.xls"},
	{"Nam Dinh", "https://cdnmedia.baotintuc.vn/2017/07/06/07/55/13NamDinh.xls"},
	{"Nghe An", "https://cdnmedia.baotintuc.vn/2017/07/06/07/55/14NgheAn.xls"},
	{"Ninh Binh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/37/NinhBinh.xls"},
	{"Ninh Thuan", "https://cdnmedia.baotintuc.vn/2017/07/06/18/00/Ninhthuan.xls"},
	{"Lao Cai", "https://cdnmedia.baotintuc.vn/2017/07/06/08/52/11LaoCai.xls"},
	{"Lai Chau", "https://cdnmedia.baotintuc.vn/2017/07/06/13/33/LaiChau.xls"},
	{"Lang Son", "https://cdnmedia.baotintuc.vn/2017/07/06/18/05/Langson.xls"},
	{"Lam Dong", "https://cdnmedia.baotintuc.vn/2017/07/06/09/00/LamDong.xls"},
	{"Long An", "https://cdnmedia.baotintuc.vn/2017/07/06/08/53/12LongAn.xls"},
	{"Quang Binh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/23/QuangBinh.xls"},
	{"Quang Nam", "https://cdnmedia.baotintuc.vn/2017/07/06/09/24/QuangNam.xls"},
	{"Quang Ninh", "https://cdnmedia.baotintuc.vn/2017/07/06/18/06/Quangninh.xls"},
	{"Quang Ngai", "https://cdnmedia.baotintuc.vn/2017/07/06/09/24/QuangNgai.xls"},
	{"Quang Tri", "https://cdnmedia.baotintuc.vn/2017/07/06/09/25/QuangTri.xls"},
	{"Phu Tho", "https://cdnmedia.baotintuc.vn/2017/07/06/09/23/PhuTho.xls"},
	{"Phu Yen", "https://cdnmedia.baotintuc.vn/2017/07/06/09/38/PhuYen.xls"},
	{"Son La", "https://cdnmedia.baotintuc.vn/2017/07/06/18/01/Sonla.xls"},
	{"Soc Trang", "https://cdnmedia.baotintuc.vn/2017/07/06/18/06/Soctrang.xls"},
	{"Tay Ninh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/26/TayNinh.xls"},
	{"Thai Binh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/09/ThaiBinh.xls"},
	{"Thai Nguyen", "https://cdnmedia.baotintuc.vn/2017/07/06/13/35/ThaiNguyen.xls"},
	{"Thanh Hoa", "https://cdnmedia.baotintuc.vn/2017/07/06/13/10/thanhoa.xls"},
	{"Tra Vinh", "https://cdnmedia.baotintuc.vn/2017/07/06/09/38/TraVinh.xls"},
	{"Thua Thien Hue", "https://cdnmedia.baotintuc.vn/2017/07/06/13/10/thuathienhue.xls"},
	{"Tien Giang", "https://cdnmedia.baotintuc.vn/2017/07/06/13/11/tiengiang.xls"},
	{"Tuyen Quang", "https://cdnmedia.baotintuc.vn/2017/07/06/09/39/TuyenQuang.xls"},
	{"Vinh Phuc", "https://cdnmedia.baotintuc.vn/2017/07/06/09/40/VinhPhuc.xls"},
	{"Vinh Long", "https://cdnmedia.baotintuc.vn/2017/07/06/13/13/vinhlong.xls"},
	{"Yen Bai", "https://cdnmedia.baotintuc.vn/2017/07/06/18/07/Yenbai.xls"},
}
