# Task 2: Contact Notes & Avatar Fields

## Description
Add optional `notes` and `avatar_url` fields to Contact entity, create migration, and update usecases.

## Context
- Project: letpai-backend
- Location: /home/glenn/projects/letpai-backend
- Priority: P1 (Important)

## Requirements

### 1. Update Contact Entity
**File:** `domain/entity/contact.go`

Add these fields to the Contact struct:
```go
AvatarURL *string `json:"avatar_url,omitempty" db:"avatar_url"`
Notes     *string `json:"notes,omitempty" db:"notes"`
```

Place them after existing fields, before the "Joined fields" comment.

### 2. Update Contact Repository
**File:** `internal/repository/contact_postgres.go`

Update INSERT and UPDATE queries to include the new fields:
- In `Create()` method: Add avatar_url and notes to INSERT query
- In `Update()` method: Add avatar_url and notes to UPDATE query

### 3. Create Migration
**File:** `migrations/000010_add_contact_notes_and_avatar.up.sql`

```sql
ALTER TABLE contacts ADD COLUMN avatar_url TEXT;
ALTER TABLE contacts ADD COLUMN notes TEXT;
```

**File:** `migrations/000010_add_contact_notes_and_avatar.down.sql`

```sql
ALTER TABLE contacts DROP COLUMN IF EXISTS notes;
ALTER TABLE contacts DROP COLUMN IF EXISTS avatar_url;
```

### 4. Update Create Contact UseCase
**File:** `internal/usecase/contact/create_contact.go`

Update the request struct to include the new fields:
```go
type CreateContactRequest struct {
    Name           string  `json:"name" validate:"required"`
    WhatsAppNumber string  `json:"whatsapp_number" validate:"required"`
    GroupID        *string `json:"group_id,omitempty"`
    IsFavorite     bool    `json:"is_favorite"`
    AvatarURL      *string `json:"avatar_url,omitempty"`
    Notes          *string `json:"notes,omitempty"`
}
```

Update the Execute method to pass these fields to the entity.

### 5. Update Update Contact UseCase
**File:** `internal/usecase/contact/update_contact.go`

Update the request struct and Execute method similarly to include AvatarURL and Notes.

### 6. Update Get Contact Response
**File:** `internal/response/contact_response.go` (if exists)
Or update the usecase response to include the new fields.

## Testing
After implementation, verify:
1. Migration runs successfully
2. Can create contact with avatar_url and notes
3. Can update contact with avatar_url and notes
4. Get contact returns the new fields
5. Create and update still work when these fields are null

## Git Commit
Create a meaningful commit:
```
feat(contact): add notes and avatar_url fields to Contact entity

- Add AvatarURL and Notes fields to Contact entity
- Create migration 000010 for new columns
- Update ContactRepository queries
- Update CreateContact and UpdateContact usecases
- Update request/response DTOs
```

## References
- Database schema: /home/glenn/.openclaw/workspace/letpai-database-schema.md
- Features brief: /home/glenn/.openclaw/workspace/letpai-features-concise.md
- Development plan: /home/glenn/.openclaw/workspace/letpai-dev-plan.md
