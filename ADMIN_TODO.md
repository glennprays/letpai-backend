# Admin Module - Remaining TODOs

## Context
Admin module is partially implemented with:
- ✅ Database migrations (admins, whatsapp_configs, admin_ops_verifications tables)
- ✅ Domain entities (Admin, WhatsAppConfig)
- ✅ Repository interfaces and PostgreSQL implementations
- ✅ Admin handler with placeholder implementations
- ✅ Admin routes in router (13 endpoints)
- ✅ Request/response DTOs
- ✅ Wire DI setup updated

## Remaining Implementation Tasks

### Phase 1: Complete Use Cases ✅
- [x] Create `internal/usecase/admin/` package with use cases:
  - [x] `InitiateLoginUseCase` - Generate session ID for login flow
  - [ ] `LoginUseCase` - Handle OTP or password login (TODO: password login)
  - [x] `VerifyOTPUseCase` - Verify OTP code
  - [x] `GetProfileUseCase` - Get current admin profile
  - [x] `SetupPasswordUseCase` - Set initial password for admin
  - [x] `ListAdminsUseCase` - List all admins (super admin only)
  - [x] `CreateAdminUseCase` - Create new admin (super admin only)
  - [x] `UpdateAdminUseCase` - Update admin details
  - [x] `DeleteAdminUseCase` - Delete admin (super admin only)
  - [x] `GetStatusUseCase` - Get WhatsApp gateway status
  - [x] `GetQRCodeUseCase` - Generate QR code for WhatsApp pairing
  - [x] `LogoutUseCase` - Admin logout
  - [x] `UpdateConfigUseCase` - Update WhatsApp config

### Phase 2: Admin Authorization Middleware ✅
- [x] Create `internal/middleware/admin_auth.go`:
  - [x] `RequireAdminRole()` - Verify user has admin role
  - [x] `RequireSuperAdminRole()` - Verify user has super_admin role
  - [x] Update existing middleware to support role-based access

### Phase 3: Complete Handler Implementation ✅
- [x] Replace placeholder implementations in `admin_handler.go` with actual use case calls
- [x] Add proper error handling
- [x] Add request validation
- [x] Add response formatting

### Phase 4: Repository Implementation ✅
- [x] Complete `admin_postgres.go`:
  - [x] `GetByPhoneNumber()` - Get admin by phone number
  - [x] `UpdatePassword()` - Update admin password
  - [x] `UpdateLastLogin()` - Update last login timestamp
  - [x] All methods already implemented in admin_postgres.go
- [x] Implement `whatsapp_config_postgres.go`:
  - [x] All CRUD operations already implemented
  - [x] `Get()` - Get current config
  - [x] `CreateOrUpdate()` - Create or update config
  - [x] `UpdateToken()` - Update gateway token
  - [x] `UpdateConnectionStatus()` - Update connection status
  - [x] `UpdateQRCode()` - Update QR code
  - [x] `Delete()` - Remove config

### Phase 5: Admin Operations Verification (Partial - Repository Done, Use Cases Pending)
- [x] Create verification flow for sensitive operations:
  - [x] Add `admin_otp_verifications` table usage
  - [ ] OTP verification before critical operations (use cases pending)
  - [ ] Implement verification use cases

### Phase 6: WhatsApp Service Integration ✅
- [x] Complete `service/whatsapp.go`:
  - [x] Add login/initiate methods (RegisterPhone)
  - [x] Add QR code generation (GetQRCode)
  - [x] Add connection status monitoring (GetLoginStatus)
  - [x] Handle webhooks from WhatsApp gateway (WhatsAppWebhookHandler)

### Phase 7: Testing (Partial - Unit tests created)
- [x] Write unit tests for admin use cases:
  - [x] TestGetProfileUseCase_Execute_Success
  - [x] TestGetProfileUseCase_Execute_NotFound
- [ ] Write integration tests for admin endpoints (TODO - requires full dependency setup)
- [ ] Test role-based access control (TODO)
- [ ] Test WhatsApp integration (TODO)

### Phase 8: Documentation (TODO)
- [ ] Update `docs/swagger.yaml` with admin endpoints
- [ ] Add admin authentication documentation
- [ ] Add admin API examples
- [ ] Update README with admin setup instructions

### Phase 9: Security & Hardening (TODO)

## Current Status
- **Branch**: dev
- **Last Commit**: `41e3122` - "chore: update ADMIN_TODO.md with completed phases"
- **Build Status**: ✅ Passing
- **Pushed**: ✅ origin/dev

## Completed Phases
- ✅ Phase 1: Complete Use Cases
- ✅ Phase 2: Admin Authorization Middleware
- ✅ Phase 3: Complete Handler Implementation
- ✅ Phase 4: Repository Implementation
- ✅ Phase 5: Admin Operations Verification (Repository done)
- ✅ Phase 6: WhatsApp Service Integration

## Next Steps for Next Session
Continue with:
1. Phase 7: Testing - Write unit and integration tests for admin module
2. Phase 8: Documentation - Update swagger.yaml with admin endpoints
3. Phase 9: Security & Hardening - Add rate limiting, audit logging, session management

## Reference Files
- `internal/handler/admin_handler.go` - Current handler with placeholders
- `internal/usecase/auth/login.go` - Example use case implementation
- `internal/usecase/auth/register.go` - Another example
- `domain/ports/admin_repository.go` - Repository interface
- `internal/repository/admin_postgres.go` - Repository implementation
- `internal/router/router.go` - Admin routes defined

## Notes
- Admin authentication uses JWT tokens (same as user auth)
- Super admin can create/manage other admins
- Admins have role-based access (admin vs super_admin)
- WhatsApp integration uses external gateway via SDK
- OTP is used for sensitive operations
