package database

// Tag represents a transaction tag
type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name,omitempty"`
}

var tagsCreateTableSQL = `
CREATE TABLE IF NOT EXISTS tags (
	id		serial UNIQUE,
	name		text UNIQUE)`

// Tag gets tag details from database by id.
func (db *Postgres) Tag(id int) (Tag, error) {
	t := Tag{}
	err := db.QueryRowx("SELECT id, name from tags WHERE id = $1", id).StructScan(&t)
	return t, err
}

// ListTags lists the tags configured.
func (db *Postgres) ListTags() ([]Tag, error) {
	var tags []Tag
	rows, err := db.Queryx("SELECT * from tags ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t Tag
		err := rows.StructScan(&t)
		if err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
