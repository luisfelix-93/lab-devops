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

/* Exemplo de Lab com Validação (CKA) */
/* Note o validation_code: ele verifica se existe um pod running com a imagem certa */
