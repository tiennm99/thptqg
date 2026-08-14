/**
 * The shapes the app shares. `Student` mirrors the 22-column table in
 * parser/internal/schema/schema.go — every column is nullable except the three
 * the schema declares NOT NULL, so a typo in a column name fails the build
 * instead of rendering blank.
 */

export type SubjectKey =
  | "toan"
  | "ngu_van"
  | "vat_ly"
  | "hoa_hoc"
  | "sinh_hoc"
  | "khtn"
  | "lich_su"
  | "dia_ly"
  | "gdcd"
  | "khxh"
  | "tieng_anh"
  | "tieng_phap"
  | "tieng_nga"
  | "tieng_duc"
  | "tieng_nhat"
  | "tieng_trung";

export type IdentityKey = "ten_cum_thi" | "gioi_tinh";

export type Student = {
  so_bao_danh: string;
  ho_ten: string;
  ho_ten_ascii: string;
  ngay_sinh: string | null;
} & { [K in IdentityKey]: string | null } & { [K in SubjectKey]: number | null };

/** A dataset as the interface needs it: registry facts plus presentation. */
export type Dataset = {
  id: string;
  dbSizeMb: number;
  blurb: string;
  label: string;
  title: string;
  subtitle: string;
  source: string;
  examples: string[];
  presets: PresetGroup[];
};

export type PresetGroup = {
  category: string;
  queries: { label: string; sql: string }[];
};
