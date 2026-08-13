package sources

// source2016 fetches the 2016 dataset: one spreadsheet per exam cluster
// (cụm thi), 4 .xls and 115 .xlsx.
//
// # Provenance
//
// The list below was recovered from the aggregator article that originally
// published these files:
//
//	cong-bo-diem-thi-thptqg-2016-toan-bo-120-cum-thi-da-co-diem.html
//
// The site that first carried it (dtntbacgiang.edu.vn) no longer resolves, so
// the list was read from the Internet Archive's copy. It is verified, not
// guessed: all 119 filenames match data/2016/ exactly in both directions, which
// TestSource2016MatchesFilesOnDisk asserts.
//
// # Filenames are load-bearing — do not "tidy" them
//
// Dest is the server-assigned name, a 32-hex content hash followed by the
// cluster slug and a millisecond timestamp. It is kept verbatim because
// go-parser sorts input files bytewise and inserts last-wins, so filenames
// decide which row survives a duplicate exam number. That is live for 2016, not
// hypothetical: 877,464 source rows collapse to 877,461, so three rows' contents
// depend on this sort order. The hashes in
// go-parser/testdata/reader-fidelity-hashes.tsv are keyed by full path as well,
// and they are frozen — they were produced by the Rust reader, which no longer
// exists.
//
// # The host is not verified
//
// Only the paths are known to be right; the host that still serves them is not.
// The original is gone and the Internet Archive captured the article but none of
// the spreadsheets, so this could not be confirmed by fetching. baseURL points
// at a mirror of the same article that is still online. If it serves these files
// from a different directory, uploadPath is the single constant to change.
const (
	baseURL    = "https://dtnt.bacninh.edu.vn"
	uploadPath = "/upload/s/20180102/"
)

var source2016 = Source{
	ID:      "2016",
	Summary: "119 exam-cluster files (4 .xls + 115 .xlsx)",
	Files: func() ([]File, error) {
		out := make([]File, 0, len(clusters2016))
		for _, c := range clusters2016 {
			out = append(out, File{
				Name: c.name,
				URL:  baseURL + uploadPath + c.file,
				Dest: c.file,
			})
		}
		return out, nil
	},
}

type cluster struct{ name, file string }

// clusters2016 is in the article's publication order. That order is incidental —
// go-parser re-sorts by filename — but it is kept as published so the list can
// be diffed against the source article.
var clusters2016 = []cluster{
	{"Trường Đại học Bách khoa Hà Nội", "b34c777942ca8a4de91f23a35cce1c6bdhbachkhoahn-1468901491486.xlsx"},
	{"Trường Đại học Sư phạm Hà Nội", "76f19098881e9c79f66ac4f1b42a8a55dhsuphamhanoi-1468901144203.xlsx"},
	{"Trường Đại học Thuỷ lợi * Cơ sở 1 ở phía Bắc", "4a3fa95a83fe9b3b2fafab309323fd22dhthuyloi-1468901840919.xlsx"},
	{"Học viện Kỹ thuật Quân sự * Cơ sở 1 ở phía Bắc (Quân đội)", "4334ea476a2b8196d54ba40a341831echvkythuatquansu-1468920342994.xlsx"},
	{"Trường Đại học Lâm nghiệp", "c5ecc58a731fd1a54b7e6cc3bc99449ddhlamnghiep-1468939300790.xlsx"},
	{"Trường Đại học Bách khoa - Đại học Quốc gia Thành phố Hồ Chí Minh", "357c6aaf59aeaa6c11c3e83c595d38cfdhbachkhoa-tphcm-1468939678829.xlsx"},
	{"Trường Đại học Khoa học tự nhiên - Đại học Quốc gia Thành phố Hồ Chí Minh", "64e44466ffeff9a587260795644805d0dhkhtunhien-dhqgtphcm-1468921015146.xlsx"},
	{"Trường Đại học Khoa học xã hội và Nhân văn - Đại học Quốc gia Thành phố Hồ Chí Minh", "5fd13ae5287cbf0149bd733b507e0608dhkhxhnv-dhqgtphcm-1468920630193.xlsx"},
	{"Trường Đại học Sư phạm Tp.HCM", "a995de3f7a7055cf10b2f002f0e194f7dhsptphcm-1468920792291.xlsx"},
	{"Trường Đại học Hàng Hải", "c6ef16bc9f75ce0fd16c1229afad4e71dhhanghai-1468906457479.xlsx"},
	{"Trường Đại học Sư phạm - Đại học Thái Nguyên", "ee09c723da4e86cdbe26213413450af2dhsupham-dhthainguyen-1468939061872.xlsx"},
	{"Trường Đại học Kỹ thuật công nghiệp - Đại học Thái Nguyên", "8af8f1b29b6a40e26ccb4a2e19e8eededhkythuatcongnghiep-dhthainguyen-1468901919568.xlsx"},
	{"Trường Đại học Nông lâm - Đại học Thái Nguyên", "53e5e20ebd74d84741513b703fbd13bfdhnonglam-dhthainguyen-1468939023637.xlsx"},
	{"Học viện Ngân hàng", "f2846894bd5533e02b122a1e3b4333f6hvnganhang-1468939472850.xlsx"},
	{"Trường Đại học Luật Hà Nội", "9e0e564ae801e347bd7649da8b89ad90dhluathn-1468920377978.xlsx"},
	{"Trường Đại học Tân Trào", "20db5eaace927596f1806f151649ee6edhtantrao-1468901296040.xlsx"},
	{"Trường Đại học Xây dựng Hà Nội", "7b5d74564ab9f6e0e137f6d161fab6f0dhxaydung-1468940561682.xlsx"},
	{"Trường Đại học Khoa học - Đại học Thái Nguyên", "c2cfaaf287fb1e7120894e545771b027dhkhoahoc-dhthainguyen-1468900365545.xlsx"},
	{"Đại học Thái Nguyên", "3daa6ee33fdafadc3c112c18bff18c2ddhthainguyen-1468940268065.xlsx"},
	{"Học viện Tài chính", "96c95c85c5549a06f2e92ffcacf8ff21hvtaichinh-1468934470924.xlsx"},
	{"Trường Đại học Tây Bắc", "74e6cd0dc78a3fb7dcf56ae57283b9b9dhtaybac-1468940302052.xlsx"},
	{"Trường Đại học Hùng Vương", "778ab074ed6a95d209fa901fd5d68843dhhungvuong-1468901964127.xlsx"},
	{"Trường Đại học Sư phạm Hà Nội 2", "a11e707e6ed5fe4d29a8e32380d7558cdhsphanoi-2-1468902006284.xlsx"},
	{"Trường Đại học Ngoại thương * Cơ sở 1 ở phía Bắc", "ec84067268f47d523b1e80b493248377dhngoaithuong-1468920480753.xlsx"},
	{"Trường Đại học Kinh tế Quốc dân", "8f948c2a5ab49a3c5fd03b6ea04887a7dhkinhtequocdanfix-1468902083782.xlsx"},
	{"Trường Đại học Giao thông Vận tải", "51bb2f9b6e9f565c2d705606fd56958bdhgiaothongvantai-1468939107463.xlsx"},
	{"Học viện Nông Nghiệp Việt Nam", "5aede4951e7af8c1e1056c747a553386hvnongnghiepvn-1468900610577.xlsx"},
	{"Trường Đại học Sư phạm Kỹ thuật Hưng Yên", "f85f04bb2bbd84457b83ac846728a2a6dhspkythuathungyen-1468920667091.xlsx"},
	{"Trường Đại học Hải Phòng", "deca197a916a8f633f6894b4e13c3ca4dhhaiphong-1468901178368.xlsx"},
	{"Trường Đại học Thương mại", "b2552e20c9ff4b237244dfbf2e278a47dhthuongmai-1468902876264.xlsx"},
	{"Trường Đại học Công nghiệp Hà Nội", "82338a4a93ea1e0cde26e0061066ca8adhcongnghiephanoi-1468920154757.xlsx"},
	{"Y Dược Thái Bình", "22e8617886412abd90ae3d33cc6d7db4dhyduocthaibinh-1468941020145.xlsx"},
	{"Trường Đại học Mỏ Địa chất", "c174631d90303276ac63ff38b31157bbdhmodiachat-1468900697349.xlsx"},
	{"Trường Đại học Hồng Đức", "bd857add0cfe92c1c247469ff2104527dh-hong-duc-1468900427119.xlsx"},
	{"Trường Đại học Vinh", "64d2635efd30a0d0db4f7c30b9735261dhvinh-1468940230027.xlsx"},
	{"Trường ĐH Sư phạm - ĐH Huế", "ab2f259ef55cb08b3a436d3f15d42cc7dhsupham-dhhue-1468934431828.xlsx"},
	{"Trường ĐH Khoa học - ĐH Huế", "87ffaf3197ba20f0919f98a0b3f33914dhkhoahoc-dhhue-1468938729472.xlsx"},
	{"Trường ĐH Kinh tế - ĐH Huế", "8916da7b3c4d58cffc1dd29dc7a8aabfdhkinhte-dhhue-1468938560205.xlsx"},
	{"Đại học Huế", "919276a0348634459f8937dfcd6c7129dhhue-1468938792115.xlsx"},
	{"ĐH Đà Nẵng", "b42fd58ead4c54279d7df5f53160a851dhdanang-1468920187203.xlsx"},
	{"Trường ĐH Bách khoa - ĐH Đà Nẵng", "43213f61074068d9177685eb6c3f2d3bdhbachkhoa-dhdanang-1468938494437.xlsx"},
	{"Trường ĐH Sư phạm - ĐH Đà Nẵng", "54bbcf865c137d58cf62af15733bd376dhsupham-dhdanang-1468938526558.xlsx"},
	{"Trường Đại học Quy Nhơn", "70bf8cd2b6e87523219e55149a8524c5dhquynhon-1468938878328.xlsx"},
	{"Trường ĐH Xây dựng miền Trung", "4e9bb192d56c9fa92b3e0ec13d8236c8dhxaydungmientrung-1468940978346.xlsx"},
	{"Trường ĐH Nông Lâm TPHCM", "b2a739584a8861502573d1c6372fd6e9dhnonglamtphcm-1468939555326.xlsx"},
	{"Trường ĐH Ngoại ngữ - ĐH Đà Nẵng", "9ba098a7913ad837a66e173b3fc41fd7dhngoaingu-dhdanang-1468902750424.xlsx"},
	{"Trường ĐH Tây Nguyên", "216ee4e385dbc236767be899b068bf98dhtaynguyen-1468940490240.xlsx"},
	{"Trường ĐH Tài chính Marketing", "af6828f837a80e03d77a65f3d310ec60dh-tai-chinh-marketing-1468924711596.xlsx"},
	{"Trường ĐH Nha Trang", "6375fd13557bee08d20f1f3e46dac130dhnhatrang-coso1-1468901350425.xlsx"},
	{"Trường ĐH Giao thông vận tải TP.HCM", "c5c28959fcd0b4e39dd9fd538db5448adhgtvttphcm-1468939150821.xlsx"},
	{"Trường ĐH Sư phạm Kĩ thuật TP.HCM", "514877c1cfebdc9c2251f651b5f02c7edhspkythuattphcm-1468906568655.xlsx"},
	{"Trường ĐH Đà Lạt", "2b33640b0dd154490f2809d96be81797dhdalat-1468902130616.xlsx"},
	{"Trường ĐH Kinh tế TP.HCM", "f0aa5a19c2c8c1fde8211341f0e98e26dhkinhtetphcm-1468939229043.xlsx"},
	{"Trường ĐH Kinh tế - Luật - ĐHQG TP.HCM", "ff877788e43bd84f0119dae026996892dhkinhteluat-dhqgtphcm-1468939860894.xlsx"},
	{"Trường ĐH Công nghiệp Thực phẩm TP.HCM", "167116af6c8a096c2a9307edfa115f77dhcnthucpham-tphcm-1468910578187.xls"},
	{"Trường ĐH Công nghiệp TP.HCM", "487c7287c1c9735722b31d877942becddhcongnghieptphcm-1468983373303.xlsx"},
	{"Trường ĐH Sài Gòn", "d75282bff0a4d8cd1421c3fb74f789c3dhsaigon-1468900867561.xlsx"},
	{"Trường ĐH Đồng Tháp", "388fcdfdac18ef6eee6125a2a015714ddhdongthap-1468934545359.xlsx"},
	{"Trường Đại học An Giang", "fc511852659150f6305f6f624445300ddhangiang-1468940189321.xlsx"},
	{"Trường ĐH Tôn Đức Thắng", "9d78f6dce7965fb4b7a337279920d4b1dhtonducthang-1468902168623.xlsx"},
	{"Trường ĐH Tiền Giang", "bacd1b2133b37d0ea9fbf46445265036dhtiengiang-1468940391597.xlsx"},
	{"Trường ĐH Cần Thơ", "b35a3f26a162bec3740e2da5639327afdhcantho-1468907131146.xls"},
	{"Trường ĐH Cần Thơ - Hậu Giang", "7f1cbe7f35d1f539ca817a0ab8bd00c2dhcantho-haugiang-1468907167921.xls"},
	{"Trường ĐH Luật TP.HCM", "d652d24e8bd4fe6a48d634b116f37c27dhluattphcm-1468939398928.xlsx"},
	{"Trường ĐH Sư phạm Kỹ thuật Vĩnh Long", "4c2fd846ed2a0670ba6d1200d1d68e1dspkythuatvinhlong-1468902640438.xlsx"},
	{"Trường ĐH Trà Vinh", "7418e83ff9d07b146216a3beafa927bbdhtravinh-1468902716174.xlsx"},
	{"Trường ĐH Ngân hàng TP.HCM", "86a692bebf9b0b7c58adb9876b9befd5dhnganhangtphcm-1468902244566.xlsx"},
	{"Trường ĐH Cần Thơ - Bạc Liêu", "5ca241aa1de8d25a3e6ff6db75234025dhcantho-baclieu-1468907071460.xls"},
	{"Trường ĐH Kiên Giang", "cdc912664d7217c75eb9b67c6ef3797adhkiengiang-1468901222446.xlsx"},
	{"Trường ĐH Y Dược Cần Thơ", "023718c7d3cf7ace3a7116fabb12bd9cdhyduoccantho-1468920829104.xlsx"},
	{"Sở GDĐT Hà Nội", "1c66db0265bf45df237b37a7bcaeaef3hanoi-1468932790746.xlsx"},
	{"Sở GDĐT Hà Giang", "178814c33c8b85a77d0fc295fa6d2be1hagiang-1468932847230.xlsx"},
	{"Sở GDĐT Cao Bằng", "ddb631f833af894200ef10cd408d2ef0caobang-1468932877769.xlsx"},
	{"Sở GDĐT Lai Châu", "a3aa106104f114c6426deb2d5e5da5fdlaichau-1468932954079.xlsx"},
	{"Sở GDĐT Lào Cai", "d035f470c13adbf5785797c6ccdcb204laocai-1468919805337.xlsx"},
	{"xem TẠI ĐÂY ) 07 Sở GDĐT Tuyên Quang", "d5ecd2574b5ab36c62dffad435262cb8tuyenquang-1468902296807.xlsx"},
	{"Sở GDĐT Lạng Sơn", "7afd5d7fdff7c170a6342cd8f9474b22langson-1468932984587.xlsx"},
	{"Sở GDĐT Bắc Kạn", "17843bccde10f362c2b9600bc49cd3f8backan-1468933016076.xlsx"},
	{"Sở GDĐT Thái Nguyên", "5fb3a1b66aff077ee3da482be08cd391thainguyen-1468902797020.xlsx"},
	{"Sở GDĐT Yên Bái", "97aa741a4c2b1276559eb021779daa32yenbai-1468933116055.xlsx"},
	{"Sở GDĐT Sơn La", "4b98fdd7300b86a94636e4d15394fc19sonla-1468899319124.xlsx"},
	{"Sở GDĐT Phú Thọ", "5465cca59495dcd9a48f689e2d38b85ephutho-1468902409682.xlsx"},
	{"xem TẠI ĐÂY ) 14 Sở GDĐT Vĩnh Phúc", "4e0ccee19eb64513d23a6dff96ff7f9cvinhphuc-1468919874814.xlsx"},
	{"Sở GDĐT Quảng Ninh", "a31b3446bae5a17c517348d8c38c5b7equangninh-1468919920197.xlsx"},
	{"Sở GDĐT Bắc Giang", "b7a0c1ecce7c8445b3a1021091a5e654bacgiang-1468983153932.xlsx"},
	{"Sở GDĐT Bắc Ninh", "7e0d0b6fb981d407cffdb79d724e8aecbacninh-1468919984998.xlsx"},
	{"Sở GDĐT Hải Dương", "7cfc91bd6a1f5ae9d7917e920f469b90haiduong-1468899640007.xlsx"},
	{"Sở GDĐT Hưng Yên", "92045214159cb0335b2836aa1c4bb25ahungyen-1468899732002.xlsx"},
	{"Sở GDĐT Hoà Bình", "505a876dfd1ea321291e29192aebe95ehoabinh-1468983220125.xlsx"},
	{"Sở GDĐT Hà Nam", "e693bc9de67acc916032c6a3b2fc9eedhanam-1468933146526.xlsx"},
	{"Sở GDĐT Nam Định", "dfa63c5b68a8f638379ca0e9c5a1c556namdinh-1468933234061.xlsx"},
	{"Sở GDĐT Thái Bình", "6960b6fa493e1365a183af461e945389thaibinh-1468920019273.xlsx"},
	{"xem TẠI ĐÂY ) 24 Sở GDĐT Ninh Bình", "0b583c9875443e65ffa449d5ca76fae8ninhbinh-1468899778557.xlsx"},
	{"xem TẠI ĐÂY ) 25 Sở GDĐT Thanh Hoá", "238adc19e80c0daef089a307af23edcethanhhoa-1468933304999.xlsx"},
	{"Sở GDĐT Nghệ An", "7f12b90225e91f9d00aac3bcdf9974e1nghean-1468920043121.xlsx"},
	{"Sở GDĐT Quảng Bình", "9c730c43bf4309171528c3853ed53b19quangbinh-1468933376032.xlsx"},
	{"Sở GDĐT Quảng Trị", "2208903e1fa12220d100f93e40d5ff3bquangtri-1468933440097.xlsx"},
	{"Sở GDĐT Thừa Thiên -Huế", "6badc7cfa258b59ad67e2b64f713b95fthuathienhue-1468983258438.xlsx"},
	{"Sở GDĐT Quảng Nam", "ec765c32190773f5a18cd8202a5eaf0dquangnam-1468933501467.xlsx"},
	{"Sở GDĐT Quảng Ngãi", "e80dc028d53fa49ee247391a0653dee5quangngai-1468933592400.xlsx"},
	{"Sở GDĐT Kon Tum", "cc57dc986acb525086b86e85df54949dkontum-1468933632113.xlsx"},
	{"Sở GDĐT Bình Định", "412e62265f35ca0078a0d94af5d1ed8abinhdinh-1468899884192.xlsx"},
	{"Sở GDĐT Gia Lai", "698b930ab7697a0672bbc39168024c9cgialai-1468933753098.xlsx"},
	{"Sở GDĐT Đắk Lắk", "2004a8d225fd87524e324d2d519e87a4daclac-1468899960505.xlsx"},
	{"Sở GDĐT Khánh Hoà", "4ad416ccf014bd5d01f415238566b210khanhhoa-1468933789968.xlsx"},
	{"Sở GDĐT Lâm Đồng", "2fa6d9a308bd11bc09f02d787eef2c15lamdong-1468933821082.xlsx"},
	{"Sở GDĐT Ninh Thuận", "15c85d901a73dd49f2bae71aadcdbb5cninhthuan-1468920071763.xlsx"},
	{"Sở GDĐT Đồng Nai", "538c9676b70f2509573245d58fba93bcdongnai-1468938241935.xlsx"},
	{"Sở GDĐT Đồng Tháp", "575df8a4df74e87abb8f55d807324374dongthap-1468933902699.xlsx"},
	{"Sở GDĐT Kiên Giang", "7435bb054e8e4ec34c173af1b6ad8402kiengiang-1468900147395.xlsx"},
	{"Sở GDĐT Cần Thơ", "5b16d7156a03bb7ed0618796bae87bc3cantho-1468933954714.xlsx"},
	{"Sở GDĐT Bến Tre", "74145979ee30186bf4e40ee3e6a3ac74bentre-1468934010040.xlsx"},
	{"Sở GDĐT Vĩnh Long", "7f4362ef567cdf74495368e0fc11072avinhlong-1468934067981.xlsx"},
	{"Sở GDĐT Trà Vinh", "08cdeb3636dcb2adaef829d62968274atravinh-1468902443859.xlsx"},
	{"Sở GDĐT Sóc Trăng", "a8ff80d9478747e36d21ec95cc9ecb8fsoctrang-1468902686409.xlsx"},
	{"Sở GDĐT Bạc Liêu", "18055d17e46854ecc6c8be585bcb6e5cbaclieu-1468934144150.xlsx"},
	{"Sở GDĐT Điện Biên", "3c0e56abd8c5334f4e1ff7df3ac23a15dienbien-1468920113820.xlsx"},
	{"Sở GDĐT Đăk Nông", "bf259a908ff8a8dcc067a761204459f4daknong-1468934173803.xlsx"},
	{"Sở GDĐT Hậu Giang", "964c4368131f6be6058fabb1caefba1chaugiang-1468934247324.xlsx"}}
