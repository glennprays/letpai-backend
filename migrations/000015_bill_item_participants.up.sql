-- Per-bill participant selection. An empty set of rows for a given
-- bill_item_id means "applies to everyone in the session" (legacy
-- behavior). A non-empty set means the bill is shared only among the
-- listed participants.
CREATE TABLE bill_item_participants (
    bill_item_id   UUID NOT NULL REFERENCES bill_items(bill_item_id)            ON DELETE CASCADE,
    participant_id UUID NOT NULL REFERENCES session_participants(participant_id) ON DELETE CASCADE,
    PRIMARY KEY (bill_item_id, participant_id)
);

CREATE INDEX bill_item_participants_pid_idx ON bill_item_participants(participant_id);
