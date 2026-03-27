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

### Phase 2: Admin Authorization Middleware
- [ ] Create `internal/middleware/admin_auth.go`:
  - [ ] `RequireAdminRole()` - Verify user has admin role
  - [ ] `RequireSuperAdminRole()` - Verify user has super_admin role
  - [ ] Update existing middleware to support role-based access

### Phase 3: Complete Handler Implementation ✅
- [x] Replace placeholder implementations in `admin_handler.go` with actual use case calls
- [x] Add proper error handling
- [x] Add request validation
- [x] Add response formatting

### Phase 4: Repository Implementation
- [ ] Complete `admin_postgres.go`:
  - [ ] `GetByPhoneNumber()` - Get admin by phone number
  - [ ] `UpdatePassword()` - Update admin password
  - [ ] `UpdateLastLogin()` - Update last login timestamp
  - [ ] `VerifyOTP()` - Verify OTP code for admin login
- [ ] Implement `whatsapp_config_postgres.go`:
  - [ ] `GetByPhoneNumber()` - Get config by phone number
  - [ ] Update `UpdateToken()` to return value
  - [ ] Complete all CRUD operations

### Phase 5: Admin Operations Verification
- [ ] Create verification flow for sensitive operations:
  - [ ] OTP verification before critical operations
  - [ ] Add `admin_ops_verifications` table usage
  - [ ] Implement verification use cases

### Phase 6: WhatsApp Service Integration
- [ ] Complete `service/whatsapp.go`:
  - [ ] Add login/initiate methods
  - [ ] Add QR code generation
  - [ ] Add connection status monitoring
  - [ ] Handle webhooks from WhatsApp gateway

### Phase 7: Testing
- [ ] Write unit tests for admin use cases
- [ ] Write integration tests for admin endpoints
- [ ] Test role-based access control
- [ ] Test WhatsApp integration

### Phase 8: Documentation
- [ ] Update `docs/swagger.yaml` with admin endpoints
- [ ] Add admin authentication documentation
- [ ] Add admin API examples
- [ ] Update README with admin setup instructions

### Phase 9: Security & Hardening
- [ ] Add rate limiting for admin endpoints
- [ ] Add audit logging for admin operations
- [ ] Add session management for admins
- [ ] Add password policies
- [ ] Add 2FA support

## Current Status
- **Branch**: dev
- **Last Commit**: `ae9cc99` - "feat: complete admin use cases and handler implementation"
- **Build Status**: ✅ Passing
- **Pushed**: ✅ origin/dev

## Completed Phases
- ✅ Phase 1: Complete Use Cases
- ✅ Phase 3: Complete Handler Implementation

## Next Steps for Next Session
1. Start with Phase 1 (Use Cases) - Create `internal/usecase/admin/` directory
2. Implement each use case following the pattern in `internal/usecase/auth/`
3. Update `admin_handler.go` to use real use cases instead of placeholders
4. Test the endpoints as you implement them

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
