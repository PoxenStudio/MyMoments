ALTER TABLE tasks ADD COLUMN reminder_offsets TEXT;
ALTER TABLE task_completions ADD COLUMN next_task_id INTEGER;
