package dev.miti99.thptqg2017.entity;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.Id;
import jakarta.persistence.Table;
import jakarta.persistence.Temporal;
import jakarta.persistence.TemporalType;
import java.util.Date;
import lombok.Data;

@Entity
@Table(name = "Student")
@Data
public class Student {
  @Id
  @Column(name = "SO_BAO_DANH")
  private String soBaoDanh;

  @Column(name = "HO_TEN")
  private String hoTen;

  @Column(name = "NGAY_SINH")
  @Temporal(TemporalType.DATE)
  private Date ngaySinh;

  @Column(name = "TOAN")
  private double toan;

  @Column(name = "NGU_VAN")
  private double nguVan;

  @Column(name = "VAT_LY")
  private double vatLy;

  @Column(name = "HOA_HOC")
  private double hoaHoc;

  @Column(name = "SINH_HOC")
  private double sinhHoc;

  @Column(name = "KHTN")
  private double khtn;

  @Column(name = "LICH_SU")
  private double lichSu;

  @Column(name = "DIA_LY")
  private double diaLy;

  @Column(name = "GDCD")
  private double gdcd;

  @Column(name = "KHXH")
  private double khxh;

  @Column(name = "TIENG_ANH")
  private double tiengAnh;
}
