/* 1. Tabela de Trilhas */
CREATE TABLE IF NOT EXISTS tracks (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

/* 2. Tabela de Labs (ATUALIZADA com validation_code) */
CREATE TABLE IF NOT EXISTS labs (
    id              TEXT PRIMARY KEY,
    title           TEXT NOT NULL,
    type            TEXT NOT NULL,
    instructions    TEXT NOT NULL,
    initial_code    TEXT NOT NULL,
    
    /* O código que o sistema roda para provar se o aluno acertou */
    validation_code TEXT, 
    
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    track_id        TEXT,
    lab_order       INTEGER,
    
    FOREIGN KEY (track_id) REFERENCES tracks(id)
);

/* 3. Tabela de Workspaces */
CREATE TABLE IF NOT EXISTS workspaces (
    id         TEXT PRIMARY KEY,
    lab_id     TEXT NOT NULL,
    user_code  TEXT NOT NULL,
    state      BLOB,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status     TEXT NOT NULL DEFAULT 'in_progress',
    FOREIGN KEY (lab_id) REFERENCES labs (id)
);

/* --- SEED DATA --- */

INSERT OR IGNORE INTO tracks (id, title, description) VALUES
('track-devops-101', 'DevOps 101: O Início', 'Aprenda os fundamentos.');

/* Exemplo de Lab com Validação (CKA) */
/* Note o validation_code: ele verifica se existe um pod running com a imagem certa */
INSERT OR IGNORE INTO labs (id, title, type, instructions, initial_code, validation_code, track_id, lab_order) VALUES
('lab-cka-01', 'CKA: Criar Pod Nginx', 'kubernetes', 
 'Crie um pod chamado "web" com imagem "nginx".', 
 '#!/bin/sh\nkubectl run web --image=nginx',
 '#!/bin/sh\n# Validação:\n# 1. Verifica se o pod existe\nkubectl get pod web > /dev/null 2>&1 || exit 1\n# 2. Verifica a imagem\nkubectl get pod web -o jsonpath="{.spec.containers[0].image}" | grep -q "nginx" || exit 1\necho "Pod web encontrado com nginx!"',
 'track-devops-101', 1);