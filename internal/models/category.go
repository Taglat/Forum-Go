package models

import "time"

type Category struct {
	ID          int
	Name        string
	Slug        string
	Description string
	Created     time.Time
}
