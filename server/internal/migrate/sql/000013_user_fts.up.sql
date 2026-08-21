CREATE VIRTUAL TABLE IF NOT EXISTS admin_user_fts USING fts5(
  username, nickname, email, phone, content='admin_user', content_rowid='id', prefix='2 3 4'
);
CREATE VIRTUAL TABLE IF NOT EXISTS web_user_fts USING fts5(
  username, nickname, email, phone, content='web_user', content_rowid='id', prefix='2 3 4'
);

CREATE TRIGGER IF NOT EXISTS admin_user_fts_ai AFTER INSERT ON admin_user BEGIN
  INSERT INTO admin_user_fts(rowid, username, nickname, email, phone)
  VALUES (new.id, new.username, new.nickname, new.email, new.phone);
END;
CREATE TRIGGER IF NOT EXISTS admin_user_fts_ad AFTER DELETE ON admin_user BEGIN
  INSERT INTO admin_user_fts(admin_user_fts, rowid, username, nickname, email, phone)
  VALUES ('delete', old.id, old.username, old.nickname, old.email, old.phone);
END;
CREATE TRIGGER IF NOT EXISTS admin_user_fts_au AFTER UPDATE ON admin_user BEGIN
  INSERT INTO admin_user_fts(admin_user_fts, rowid, username, nickname, email, phone)
  VALUES ('delete', old.id, old.username, old.nickname, old.email, old.phone);
  INSERT INTO admin_user_fts(rowid, username, nickname, email, phone)
  VALUES (new.id, new.username, new.nickname, new.email, new.phone);
END;

CREATE TRIGGER IF NOT EXISTS web_user_fts_ai AFTER INSERT ON web_user BEGIN
  INSERT INTO web_user_fts(rowid, username, nickname, email, phone)
  VALUES (new.id, new.username, new.nickname, new.email, new.phone);
END;
CREATE TRIGGER IF NOT EXISTS web_user_fts_ad AFTER DELETE ON web_user BEGIN
  INSERT INTO web_user_fts(web_user_fts, rowid, username, nickname, email, phone)
  VALUES ('delete', old.id, old.username, old.nickname, old.email, old.phone);
END;
CREATE TRIGGER IF NOT EXISTS web_user_fts_au AFTER UPDATE ON web_user BEGIN
  INSERT INTO web_user_fts(web_user_fts, rowid, username, nickname, email, phone)
  VALUES ('delete', old.id, old.username, old.nickname, old.email, old.phone);
  INSERT INTO web_user_fts(rowid, username, nickname, email, phone)
  VALUES (new.id, new.username, new.nickname, new.email, new.phone);
END;

INSERT INTO admin_user_fts(admin_user_fts) VALUES('rebuild');
INSERT INTO web_user_fts(web_user_fts) VALUES('rebuild');
