ALTER TABLE files
ADD CONSTRAINT file_name_unique UNIQUE (name);