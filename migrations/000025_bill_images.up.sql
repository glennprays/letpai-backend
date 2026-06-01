-- Bill image attachments for session receipts/bills.
-- Each row is one image belonging to a session (host-uploaded).
CREATE TABLE bill_images (
    bill_image_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id    UUID NOT NULL REFERENCES sessions(session_id) ON DELETE CASCADE,
    image_url     TEXT NOT NULL,         -- S3 object key (e.g. bill-images/filename_1234.jpg)
    thumbnail_url TEXT,                  -- reserved for future thumbnail generation
    file_name     TEXT NOT NULL,
    file_format   VARCHAR(10) NOT NULL,  -- jpeg, png, webp
    file_size     BIGINT NOT NULL DEFAULT 0,
    ordinal       INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    TIMESTAMP              -- soft delete
);

CREATE INDEX idx_bill_images_session ON bill_images(session_id);
CREATE INDEX idx_bill_images_created ON bill_images(created_at ASC);
