package domain

import "time"

type Lab struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Type         string    `json:"type"`
	Instructions string    `json:"instructions"`
	InitialCode  string    `json:"initial_code"`
	CreatedAt    time.Time `json:"created_at"`

	TrackID      string    `json:"track_id"`
	LabOrder     int       `json:"lab_order"`
	ValidationCode string  `json:"-"`
}