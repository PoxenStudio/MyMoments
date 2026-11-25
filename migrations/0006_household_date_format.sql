-- 时间格式改为家庭级全局设置。
ALTER TABLE household_settings ADD COLUMN date_format TEXT NOT NULL DEFAULT '24h';
