package models

type Label struct {
	Model

	Value string

	Destinations []Destination `gorm:"many2many:destination_labels"`
	Grant        []Grant       `gorm:"many2many:grant_labels"`
}
