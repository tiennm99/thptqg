package dev.miti99.thptqg2017;

import dev.miti99.thptqg2017.entity.Student;
import java.io.FileInputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.text.SimpleDateFormat;
import java.util.regex.Pattern;
import lombok.extern.slf4j.Slf4j;
import org.apache.poi.xssf.usermodel.XSSFWorkbook;
import org.hibernate.cfg.Configuration;

@Slf4j
public class Converter {
  private static final SimpleDateFormat dateFormat = new SimpleDateFormat("dd/MM/yyyy");
  private static final String excelFolder = "src/main/resources/raw";
  private static final Pattern toanPattern = Pattern.compile("Toán:\\s*(\\d*.\\d*)");
  private static final Pattern nguVanPattern = Pattern.compile("Ngữ văn:\\s*(\\d*.\\d*)");
  private static final Pattern vatLiPattern = Pattern.compile("Vật lí:\\s*(\\d*.\\d*)");
  private static final Pattern hoaHocPattern = Pattern.compile("Hóa học:\\s*(\\d*.\\d*)");
  private static final Pattern sinhHocPattern = Pattern.compile("Sinh học:\\s*(\\d*.\\d*)");
  private static final Pattern khtnPattern = Pattern.compile("KHTN:\\s*(\\d*.\\d*)");
  private static final Pattern lichSuPattern = Pattern.compile("Lịch sử:\\s*(\\d*.\\d*)");
  private static final Pattern diaLiPattern = Pattern.compile("Địa lí:\\s*(\\d*.\\d*)");
  private static final Pattern gdcdPattern = Pattern.compile("GDCD:\\s*(\\d*.\\d*)");
  private static final Pattern khxhPattern = Pattern.compile("KHXH:\\s*(\\d*.\\d*)");
  private static final Pattern tiengAnhPattern = Pattern.compile("Tiếng Anh:\\s*(\\d*.\\d*)");

  public static void main(String[] args) {
    var configuration = new Configuration();
    configuration.configure("hibernate.cfg.xml");

    try (var sessionFactory = configuration.buildSessionFactory();
        var session = sessionFactory.openSession();
        var paths = Files.walk(Path.of(excelFolder))) {
      var transaction = session.beginTransaction();

      paths
          .filter(Files::isRegularFile)
          .forEach(
              file -> {
                try (var workbook = new XSSFWorkbook(new FileInputStream(file.toFile()))) {
                  var sheet = workbook.getSheetAt(0);

                  for (var row : sheet) {
                    try {
                      // Một số file lỗi nên không chắc có header, không skip
                      // if (row.getRowNum() == 0) {
                      //   return;
                      // }
                      var student = new Student();

                      var hoTen = row.getCell(0).getStringCellValue();
                      student.setHoTen(hoTen);

                      var ngaySinh = row.getCell(1).getStringCellValue();
                      student.setNgaySinh(dateFormat.parse(ngaySinh));

                      var soBaoDanh = row.getCell(2).getStringCellValue();
                      student.setSoBaoDanh(soBaoDanh);

                      var diemThi = row.getCell(3).getStringCellValue();

                      var toanMatcher = toanPattern.matcher(diemThi);
                      if (toanMatcher.find()) {
                        var toan = Double.parseDouble(toanMatcher.group(1));
                        student.setToan(toan);
                      }

                      var nguVanMatcher = nguVanPattern.matcher(diemThi);
                      if (nguVanMatcher.find()) {
                        var nguVan = Double.parseDouble(nguVanMatcher.group(1));
                        student.setNguVan(nguVan);
                      }

                      var vatLiMatcher = vatLiPattern.matcher(diemThi);
                      if (vatLiMatcher.find()) {
                        var vatLi = Double.parseDouble(vatLiMatcher.group(1));
                        student.setVatLy(vatLi);
                      }

                      var hoaHocMatcher = hoaHocPattern.matcher(diemThi);
                      if (hoaHocMatcher.find()) {
                        var hoaHoc = Double.parseDouble(hoaHocMatcher.group(1));
                        student.setHoaHoc(hoaHoc);
                      }

                      var sinhHocMatcher = sinhHocPattern.matcher(diemThi);
                      if (sinhHocMatcher.find()) {
                        var sinhHoc = Double.parseDouble(sinhHocMatcher.group(1));
                        student.setSinhHoc(sinhHoc);
                      }

                      var khtnMatcher = khtnPattern.matcher(diemThi);
                      if (khtnMatcher.find()) {
                        var khtn = Double.parseDouble(khtnMatcher.group(1));
                        student.setKhtn(khtn);
                      }

                      var lichSuMatcher = lichSuPattern.matcher(diemThi);
                      if (lichSuMatcher.find()) {
                        var lichSu = Double.parseDouble(lichSuMatcher.group(1));
                        student.setLichSu(lichSu);
                      }

                      var diaLiMatcher = diaLiPattern.matcher(diemThi);
                      if (diaLiMatcher.find()) {
                        var diaLi = Double.parseDouble(diaLiMatcher.group(1));
                        student.setDiaLy(diaLi);
                      }

                      var gdcdMatcher = gdcdPattern.matcher(diemThi);
                      if (gdcdMatcher.find()) {
                        var gdcd = Double.parseDouble(gdcdMatcher.group(1));
                        student.setGdcd(gdcd);
                      }

                      var khxhMatcher = khxhPattern.matcher(diemThi);
                      if (khxhMatcher.find()) {
                        var khxh = Double.parseDouble(khxhMatcher.group(1));
                        student.setKhxh(khxh);
                      }

                      var tiengAnhMatcher = tiengAnhPattern.matcher(diemThi);
                      if (tiengAnhMatcher.find()) {
                        var tiengAnh = Double.parseDouble(tiengAnhMatcher.group(1));
                        student.setTiengAnh(tiengAnh);
                      }

                      session.merge(student);
                    } catch (Exception e) {
                      log.error("Error", e);
                    }
                  }

                } catch (Exception e) {
                  log.error("Error", e);
                }
              });
      transaction.commit();
    } catch (Exception e) {
      log.error("Error", e);
    }
  }
}
