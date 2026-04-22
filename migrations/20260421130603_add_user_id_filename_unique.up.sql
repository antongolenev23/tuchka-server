ALTER TABLE files
ADD CONSTRAINT files_user_id_name_unique UNIQUE(user_id, name);