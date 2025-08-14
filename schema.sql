PRAGMA journal_mode = WAL;

CREATE TABLE users (
    uid INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE games (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    white_uid INTEGER NOT NULL,
    black_uid INTEGER NOT NULL,
    result TEXT CHECK (Result IN ('white', 'black', 'draw')) NOT NULL,
    -- PGN of moves
    moves TEXT NOT NULL,  
    finished_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
