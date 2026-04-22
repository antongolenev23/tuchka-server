ALTER TABLE files 
ADD COLUMN user_id UUID NOT NULL;

ALTER TABLE files 
ADD CONSTRAINT fk_files_user 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_files_user_id_name 
ON files (user_id, name);