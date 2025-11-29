package repository

import (
	"context"
	"database/sql"
	"lab-devops/internal/domain"
	"lab-devops/internal/service"
	"log"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type sqlRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(dbPath string, migrationScriptPath string) (service.WorkspaceRepository, error) {
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	script, err := os.ReadFile(migrationScriptPath)
	if err != nil {
		return nil, err
	}

	if _, err = db.Exec(string(script)); err != nil {
		return nil, err
	}

	log.Println("✅ Base de dados SQLite conectada e migrações aplicadas.")
	return &sqlRepository{db: db}, nil

}
func (r *sqlRepository) GetLabByID(ctx context.Context, labID string) (*domain.Lab, error) {
	query := `SELECT id, title, type, instructions, initial_code, created_at, 
	                 track_id, lab_order, COALESCE(validation_code, '')
	          FROM labs WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, labID)

	var lab domain.Lab
	err := row.Scan(
		&lab.ID,
		&lab.Title,
		&lab.Type,
		&lab.Instructions,
		&lab.InitialCode,
		&lab.CreatedAt,
		&lab.TrackID,  
		&lab.LabOrder,
		&lab.ValidationCode,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &lab, nil
}


func (r *sqlRepository) GetWorkspaceByLabID(ctx context.Context, labID string) (*domain.Workspace, error) {
	// CORREÇÃO: Adicionado 'status' no SELECT
	query := `SELECT id, lab_id, user_code, state, updated_at, status 
	          FROM workspaces WHERE lab_id = ?`

	row := r.db.QueryRowContext(ctx, query, labID)

	var ws domain.Workspace
	err := row.Scan(
		&ws.ID,
		&ws.LabID,
		&ws.UserCode,
		&ws.State,
		&ws.UpdatedAt,
		&ws.Status, 
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Nenhum workspace encontrado
		}
		return nil, err
	}
	return &ws, nil
}

// UpdateWorkspaceState atualiza o ficheiro .tfstate (blob).
func (r *sqlRepository) UpdateWorkspaceState(ctx context.Context, workspaceID string, state []byte) error {
	query := `UPDATE workspaces SET state = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, state, workspaceID)
	return err
}

// GetWorkspaceState obtém o .tfstate atual.
func (r *sqlRepository) GetWorkspaceState(ctx context.Context, workspaceID string) ([]byte, error) {
	query := `SELECT state FROM workspaces WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, workspaceID)

	var state []byte
	if err := row.Scan(&state); err != nil {
		return nil, err
	}
	return state, nil
}

// --- Métodos por implementar (para completar a interface) ---

func (r *sqlRepository) ListLabs(ctx context.Context) ([]*domain.Lab, error) {
	query := `SELECT id, title, type, instructions, initial_code, created_at,
	                 track_id, lab_order, COALESCE(validation_code, '')
	          FROM labs ORDER BY lab_order ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var labs []*domain.Lab
	for rows.Next() {
		var lab domain.Lab
		if err := rows.Scan(
			&lab.ID,
			&lab.Title,
			&lab.Type,
			&lab.Instructions,
			&lab.InitialCode,
			&lab.CreatedAt,
			&lab.TrackID,  // Novo
			&lab.LabOrder, // Novo
			&lab.ValidationCode,	
		); err != nil {
			return nil, err
		}
		labs = append(labs, &lab)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return labs, nil
}

func (r *sqlRepository) UpdateWorkspaceCode(ctx context.Context, workspaceID string, code string) error {
	query := `UPDATE workspaces SET user_code = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, code, workspaceID)
	return err
}
func (r *sqlRepository) CreateWorkspace(ctx context.Context, labID string) (*domain.Workspace, error) {
	lab, err := r.GetLabByID(ctx, labID)
	if err != nil {
		return nil, err
	}
	if lab == nil {
		return nil, sql.ErrNoRows // Ou um erro customizado "lab not found"
	}

	newWorkspaceID := uuid.New().String()
	insertQuery := `INSERT INTO workspaces (id, lab_id, user_code) VALUES (?, ?, ?)`

	_, err = r.db.ExecContext(ctx, insertQuery, newWorkspaceID, labID, lab.InitialCode)
	if err != nil {
		return nil, err
	}

	selectQuery := `SELECT id, lab_id, user_code, state, updated_at, status FROM workspaces WHERE id = ?`
	row := r.db.QueryRowContext(ctx, selectQuery, newWorkspaceID)

	var ws domain.Workspace
	if err := row.Scan(
		&ws.ID,
		&ws.LabID,
		&ws.UserCode,
		&ws.State,
		&ws.UpdatedAt,
		&ws.Status, 
	); err != nil {
		return nil, err
	}

	return &ws, nil
}

func (r *sqlRepository) CreateLab(ctx context.Context, lab *domain.Lab) error {
    query := `
        INSERT INTO labs (id, title, type, instructions, initial_code, track_id, lab_order, validation_code)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
    _, err := r.db.ExecContext(ctx, query, 
        lab.ID, 
        lab.Title, 
        lab.Type, 
        lab.Instructions, 
        lab.InitialCode,
		lab.TrackID,
		lab.LabOrder,
		lab.ValidationCode,
    )
    return err
}

func (r *sqlRepository) CleanLab(ctx context.Context, labId string) error {
	query := `
		DELETE FROM labs WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query, labId)
	return err
}

func (r *sqlRepository) UpdateWorkspaceStatus(ctx context.Context, workspaceId string, status string) error {
	query := `
		UPDATE workspaces SET status = ? WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, status, workspaceId)
	return err
}

func (r *sqlRepository) ListTracks(ctx context.Context) ([]*domain.Track, error) {
	query := `SELECT id, title, description, created_at FROM tracks ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []*domain.Track
	for rows.Next() {
		var track domain.Track
		if err := rows.Scan(
			&track.ID,
			&track.Title,
			&track.Description,
			&track.CreatedAt,
		); err != nil {
			return nil, err
		}
		tracks = append(tracks, &track)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tracks, nil
}

func (r *sqlRepository) ListLabsByTrackID(ctx context.Context, trackID string) ([]*domain.Lab, error) {
	query := `SELECT id, title, type, instructions, initial_code, created_at,
	                 track_id, lab_order, COALESCE(validation_code, '')
	          FROM labs WHERE track_id = ? ORDER BY lab_order ASC`
	rows, err := r.db.QueryContext(ctx, query, trackID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labs []*domain.Lab
	for rows.Next() {
		var lab domain.Lab
		if err := rows.Scan(
			&lab.ID,
			&lab.Title,
			&lab.Type,
			&lab.Instructions,
			&lab.InitialCode,
			&lab.CreatedAt,
			&lab.TrackID,
			&lab.LabOrder,
			&lab.ValidationCode,
		); err != nil {
			return nil, err
		}
		labs = append(labs, &lab)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return labs, nil
}

func (r *sqlRepository) GetTrackByID(ctx context.Context, id string) (*domain.Track, error) {
	query := `SELECT id, title, description, created_at FROM tracks WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	var track domain.Track
	if err := row.Scan(
		&track.ID,
		&track.Title,
		&track.Description,
		&track.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &track, nil
}

func (r *sqlRepository) CreateTrack(ctx context.Context, track *domain.Track) error {
	query := `
	INSERT INTO tracks (id, title, description)
		VALUES (?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, 
		track.ID, 
		track.Title,
		track.Description,	
	)

	return err
}

func (r *sqlRepository) UpdateLab(ctx context.Context, lab *domain.Lab) error {
	query := `
		UPDATE labs SET title = ?, type = ?, instructions = ?, initial_code = ?, track_id = ?, lab_order = ?, validation_code = ? WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, 
		lab.Title,
		lab.Type,
		lab.Instructions,
		lab.InitialCode,
		lab.TrackID,
		lab.LabOrder,
		lab.ValidationCode,
		lab.ID,
	)
	return err
}

func (r* sqlRepository) DeleteLab(ctx context.Context, labID string) error {
	query := `DELETE FROM labs WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, labID)
	return err
}

func (r *sqlRepository) UpdateTrack(ctx context.Context, track *domain.Track) error {
	query := `
		UPDATE tracks SET title = ?, description = ? WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, 
		track.Title,
		track.Description,
		track.ID,	
	)
	return err
}

func (r *sqlRepository) DeleteTrack(ctx context.Context, trackID string) error {
	query := `DELETE FROM tracks WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, trackID)
	return err
}
