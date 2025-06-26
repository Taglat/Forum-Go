--  Рекомендованный порядок:
-- ИМЯ ТИП [NOT NULL] [DEFAULT ...] [UNIQUE] [PRIMARY KEY] [CHECK ...] [REFERENCES ...]

PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password BLOB NOT NULL, -- BLOB (от англ. Binary Large OBject) — это тип данных в базах данных, предназначенный для хранения больших объемов бинарной информации
    created DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    expires DATETIME NOT NULL,
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires);

CREATE TABLE IF NOT EXISTS posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id);
CREATE INDEX IF NOT EXISTS idx_posts_created ON posts(created);

CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,
    created DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Связь многие-ко-многим между постами и категориями
CREATE TABLE IF NOT EXISTS post_categories (
    post_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    PRIMARY KEY (post_id, category_id),
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_post_categories_post_id ON post_categories(post_id);
CREATE INDEX IF NOT EXISTS idx_post_categories_category_id ON post_categories(category_id);

INSERT OR IGNORE INTO categories (name, slug, description) VALUES 
    ('General Discussion', 'general', 'General tea talk, news, and off-topic chats'),
    ('Tea Types', 'tea-types', 'Discuss various types of tea: green, black, oolong, pu-erh, etc.'),
    ('Brewing Methods', 'brewing', 'Talk about steeping techniques, temperature, teaware, and rituals'),
    ('Origins & Regions', 'origins', 'Tea by origin: China, Japan, India, Taiwan, and more'),
    ('Reviews & Recommendations', 'reviews', 'Tea reviews, tasting notes, and suggestions'),
    ('Teaware & Accessories', 'teaware', 'Teapots, gaiwans, cups, filters, and other tools'),
    ('Tea & Health', 'health', 'Health benefits, risks, and wellness tips related to tea');
