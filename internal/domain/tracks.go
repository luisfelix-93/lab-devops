package domain

import "time"

type Track struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`


	Labs        []*Lab     `json:"labs"`
}