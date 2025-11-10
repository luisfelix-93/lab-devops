package repository

import "database/sql"

type sqlRepository struct {
	db *sql.DB
}