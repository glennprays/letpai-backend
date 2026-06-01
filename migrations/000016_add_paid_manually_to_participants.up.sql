-- Track whether a participant's "paid" status was set by the host without
-- a proof upload (Mark as paid action). Used by submit_payment to reject
-- subsequent uploads that would otherwise quietly overwrite a manual close.
ALTER TABLE session_participants
    ADD COLUMN paid_manually BOOLEAN NOT NULL DEFAULT FALSE;
