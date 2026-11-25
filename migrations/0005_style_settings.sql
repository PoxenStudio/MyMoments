-- 样式自定义：家庭默认与个人样式均为可整体覆盖的 JSON。
ALTER TABLE household_settings ADD COLUMN style TEXT;
ALTER TABLE user_preferences ADD COLUMN style TEXT;
