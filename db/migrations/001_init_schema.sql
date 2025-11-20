/* 1. Tabela de Trilhas (NOVA) */
CREATE TABLE IF NOT EXISTS tracks (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

/* 2. Tabela de Labs (ATUALIZADA com track_id e lab_order) */
CREATE TABLE IF NOT EXISTS labs (
    id           TEXT PRIMARY KEY,
    title        TEXT NOT NULL,
    type         TEXT NOT NULL,
    instructions TEXT NOT NULL,
    initial_code TEXT NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    track_id     TEXT,
    lab_order    INTEGER,
    
    FOREIGN KEY (track_id) REFERENCES tracks(id)
);

/* 3. Tabela de Workspaces (ATUALIZADA com o campo status) */
CREATE TABLE IF NOT EXISTS workspaces (
    id         TEXT PRIMARY KEY,
    lab_id     TEXT NOT NULL,
    user_code  TEXT NOT NULL,
    state      BLOB,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status     TEXT NOT NULL DEFAULT 'in_progress', /* 'in_progress' ou 'completed' */
    FOREIGN KEY (lab_id) REFERENCES labs (id)
);

/* --- SEED DATA (Dados Iniciais) --- */

/* Criar uma Trilha Básica */
INSERT OR IGNORE INTO tracks (id, title, description) VALUES
('track-devops-101', 'DevOps 101: O Início', 'Aprenda os fundamentos de Linux, Terraform e Ansible.');

/* Lab 1: Linux */
INSERT OR IGNORE INTO labs (id, title, type, instructions, initial_code, track_id, lab_order) VALUES
('lab-linux-01', 'Linux: Olá Mundo', 'linux', 'Crie um ficheiro chamado "ola.txt" em /tmp.', '#!/bin/sh\n\n# Escreva o seu script aqui', 'track-devops-101', 1);

/* Lab 2: Terraform */
INSERT OR IGNORE INTO labs (id, title, type, instructions, initial_code, track_id, lab_order) VALUES
('lab-tf-01', 'Terraform: Criar um S3 Bucket', 'terraform', 'Crie um recurso aws_s3_bucket.', 'resource "aws_s3_bucket" "b" {\n  bucket = "meu-bucket"\n}', 'track-devops-101', 2);

/* Lab 3: Ansible */
INSERT OR IGNORE INTO labs (id, title, type, instructions, initial_code, track_id, lab_order) VALUES
('lab-ansible-01', 'Ansible: Ping Local', 'ansible', 'Use o módulo ping.', '- hosts: localhost\n  tasks:\n', 'track-devops-101', 3);