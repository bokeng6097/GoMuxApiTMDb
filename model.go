// model.go

package main

import (
	"database/sql"
	"fmt"
)

type photo struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Filename    string `json:"filename"`
	OriLink     string `json:"ori_link"`
}

func (p *photo) getPhoto(db *sql.DB) error {
	statement := fmt.Sprintf("SELECT id, title, description, filename, ori_link FROM photos WHERE id=%d", p.ID)
	return db.QueryRow(statement).Scan(&p.ID, &p.Title, &p.Description, &p.Filename, &p.OriLink)
}

func (p *photo) createPhoto(db *sql.DB) error {
	statement := fmt.Sprintf("INSERT INTO photos(title, description, filename, ori_link) VALUES ('%s', '%s', '%s', '%s')", p.Title, p.Description, p.Filename, p.OriLink)
	_, err := db.Exec(statement)

	if err != nil {
		return err
	}

	// TOUNDERSTAND
	err = db.QueryRow("SELECT LAST_INSERT_ID()").Scan(&p.ID)

	if err != nil {
		return err
	}

	return nil
}

func (p *photo) updatePhoto(db *sql.DB) error {
	statement := fmt.Sprintf("UPDATE photos SET title='%s', description='%s', filename='%s', ori_link='%s' WHERE id=%d", p.Title, p.Description, p.Filename, p.OriLink, p.ID)
	_, err := db.Exec(statement)
	return err
}

func (p *photo) deletePhoto(db *sql.DB) error {
	statement := fmt.Sprintf("DELETE FROM photos WHERE id=%d", p.ID)
	_, err := db.Exec(statement)
	return err
}

func getPhotos(db *sql.DB) ([]photo, error) {
	statement := fmt.Sprintf("SELECT id, title, description, filename, ori_link FROM photos")
	rows, err := db.Query(statement)

	if err != nil {

	}

	defer rows.Close()

	photos := []photo{}

	for rows.Next() {
		var p photo
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Filename, &p.OriLink); err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}

	return photos, nil
}
