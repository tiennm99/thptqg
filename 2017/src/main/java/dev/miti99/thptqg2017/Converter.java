package dev.miti99.thptqg2017;

import dev.miti99.thptqg2017.entity.Student;
import java.text.SimpleDateFormat;
import lombok.extern.slf4j.Slf4j;
import org.hibernate.cfg.Configuration;

@Slf4j
public class Converter {
  private static final SimpleDateFormat dateFormat = new SimpleDateFormat("dd/MM/yyyy");

  public static void main(String[] args) {
    var configuration = new Configuration();
    configuration.configure("hibernate.cfg.xml");

    try (var sessionFactory = configuration.buildSessionFactory();
        var session = sessionFactory.openSession()) {
      var transaction = session.beginTransaction();

      var newStudent = new Student();
      newStudent.setSoBaoDanh(12345);
      newStudent.setHoTen("John Doe");
      newStudent.setNgaySinh(dateFormat.parse("01/01/1999"));

      session.merge(newStudent);

      transaction.commit();
    } catch (Exception e) {
      log.error("Exception occurred", e);
    }
  }
}
