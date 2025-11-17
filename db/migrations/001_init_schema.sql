CREATE TABLE IF NOT EXISTS labs (
    id           TEXT PRIMARY KEY,
    title        TEXT NOT NULL,
    type         TEXT NOT NULL,
    instructions TEXT NOT NULL,
    initial_code TEXT NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS workspaces (
    id         TEXT PRIMARY KEY,
    lab_id     TEXT NOT NULL,
    user_code  TEXT NOT NULL,
    state      BLOB,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (lab_id) REFERENCES labs (id)
);

INSERT OR IGNORE INTO labs (id, title, type, instructions, initial_code) VALUES
('lab-tf-01', 'Terraform: Criar um S3 Bucket', 'terraform', 'O seu objetivo é criar um S3 bucket...', 'resource "aws_s3_bucket" "meu_bucket" {\n  bucket = "meu-bucket-de-lab-12345"\n}');

INSERT OR IGNORE INTO workspaces (id, lab_id, user_code) VALUES
('ws-tf-01', 'lab-tf-01', 'resource "aws_s3_bucket" "meu_bucket" {\n  bucket = "meu-bucket-de-lab-12345"\n}');

ALTER TABLE workspaces ADD COLUMN status TEXT NOT NULL DEFAULT 'in_progress'