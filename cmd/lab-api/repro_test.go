package main

import (
	"context"
	"database/sql"
	"fmt"
	"lab-devops/internal/repository"
	"lab-devops/internal/service"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestReproListTracks(t *testing.T) {
	// 1. Setup Temp DB
	dbFile := "test_repro.db"
	os.Remove(dbFile)
	defer os.Remove(dbFile)

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}

	// 2. Create Tables
	schema := `
	CREATE TABLE IF NOT EXISTS tracks (
		id          TEXT PRIMARY KEY,
		title       TEXT NOT NULL,
		description TEXT,
		created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS labs (
		id              TEXT PRIMARY KEY,
		title           TEXT NOT NULL,
		type            TEXT NOT NULL,
		instructions    TEXT NOT NULL,
		initial_code    TEXT NOT NULL,
		validation_code TEXT, 
		created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		track_id        TEXT,
		lab_order       INTEGER,
		FOREIGN KEY (track_id) REFERENCES tracks(id)
	);
	`
	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// 3. Insert Data
	_, err = db.Exec(`INSERT INTO tracks (id, title, description) VALUES ('track1', 'Track 1', 'Desc 1')`)
	if err != nil {
		t.Fatalf("Failed to insert track: %v", err)
	}
	_, err = db.Exec(`INSERT INTO labs (id, title, type, instructions, initial_code, track_id, lab_order, validation_code) 
		VALUES ('lab1', 'Lab 1', 'type1', 'inst', 'code', 'track1', 1, 'valid')`)
	if err != nil {
		t.Fatalf("Failed to insert lab: %v", err)
	}

	// 4. Initialize Service
	// We can't use NewSQLiteRepository easily because it expects a migration file path.
	// So we'll construct the repo manually or use a dummy path.
	// But NewSQLiteRepository is in internal/repository, so we can't access sqlRepository struct directly if it's private.
	// Wait, sqlRepository is private. NewSQLiteRepository returns the interface.

	// Let's create a dummy migration file.
	migFile := "dummy_migration.sql"
	os.WriteFile(migFile, []byte(schema), 0644)
	defer os.Remove(migFile)

	repo, err := repository.NewSQLiteRepository(dbFile, migFile)
	if err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}

	svc := service.NewLabService(repo, nil) // executor can be nil for ListTracks

	// 5. Call ListTracks
	tracks, err := svc.ListTracks(context.Background())
	if err != nil {
		t.Fatalf("ListTracks failed: %v", err)
	}

	if len(tracks) != 1 {
		t.Errorf("Expected 1 track, got %d", len(tracks))
	}
	if len(tracks[0].Labs) != 1 {
		t.Errorf("Expected 1 lab, got %d", len(tracks[0].Labs))
	}

	fmt.Println("Test passed!")
}
