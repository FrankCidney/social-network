ALTER TABLE group_members RENAME TO group_members_old;

CREATE TABLE group_members (
    group_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, user_id),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CHECK (status IN ('invited', 'requested', 'accepted', 'declined'))
);

INSERT INTO group_members (group_id, user_id, status, created_at)
SELECT group_id, user_id, status, created_at
FROM group_members_old;

DROP TABLE group_members_old;
