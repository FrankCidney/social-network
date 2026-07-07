-- Followers: confirmed relationships only
CREATE TABLE IF NOT EXISTS followers (
    follower_id TEXT NOT NULL,
    followed_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (follower_id, followed_id),
    FOREIGN KEY (follower_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (followed_id) REFERENCES users(id) ON DELETE CASCADE,
    CHECK (follower_id != followed_id)
);

CREATE INDEX IF NOT EXISTS idx_followers_followed ON followers(followed_id);
