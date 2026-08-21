package store

import "gorm.io/gorm"

func EnsureUserFTS(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	for _, tbl := range []string{"admin_user", "web_user"} {
		if err := ensureAccountFTS(db, tbl); err != nil {
			return err
		}
	}
	return nil
}

func ensureAccountFTS(db *gorm.DB, tbl string) error {
	fts := tbl + "_fts"
	var existing int64
	if err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", fts).Scan(&existing).Error; err != nil {
		return err
	}
	stmts := []string{
		`CREATE VIRTUAL TABLE IF NOT EXISTS ` + fts + ` USING fts5(username, nickname, email, phone, content='` + tbl + `', content_rowid='id', prefix='2 3 4')`,
		`CREATE TRIGGER IF NOT EXISTS ` + tbl + `_fts_ai AFTER INSERT ON ` + tbl + ` BEGIN
			INSERT INTO ` + fts + `(rowid, username, nickname, email, phone)
			VALUES (new.id, new.username, new.nickname, new.email, new.phone);
		END`,
		`CREATE TRIGGER IF NOT EXISTS ` + tbl + `_fts_ad AFTER DELETE ON ` + tbl + ` BEGIN
			INSERT INTO ` + fts + `(` + fts + `, rowid, username, nickname, email, phone)
			VALUES ('delete', old.id, old.username, old.nickname, old.email, old.phone);
		END`,
		`CREATE TRIGGER IF NOT EXISTS ` + tbl + `_fts_au AFTER UPDATE ON ` + tbl + ` BEGIN
			INSERT INTO ` + fts + `(` + fts + `, rowid, username, nickname, email, phone)
			VALUES ('delete', old.id, old.username, old.nickname, old.email, old.phone);
			INSERT INTO ` + fts + `(rowid, username, nickname, email, phone)
			VALUES (new.id, new.username, new.nickname, new.email, new.phone);
		END`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	if existing > 0 {
		return nil
	}
	return db.Exec(`INSERT INTO ` + fts + `(` + fts + `) VALUES('rebuild')`).Error
}
