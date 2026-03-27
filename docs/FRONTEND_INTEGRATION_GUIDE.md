# Letpai Frontend - Implementation Guide for AI Agents

**Version**: 1.0.0  
**Base URL**: `http://localhost:3000/api/v1`  
**Swagger**: `docs/swagger.yaml`

---

## Quick Reference

### Authentication

**User Flow:**
1. Register → OTP sent to WhatsApp
2. Verify OTP → Get JWT token
3. Use token in `Authorization: Bearer {token}` header

**Admin Flow:**
1. Initiate login → OTP sent
2. Verify OTP → Get JWT with role (`admin` or `super_admin`)
3. Super admins can manage other admins

### Core Features

**Contacts**: CRUD operations,- **Sessions**: Create bill split → Add participants → Add items → Calculate → Send notifications
- **Payments**: Public payment page → Upload proof → Creator approves/rejects
- **WhatsApp**: OTP delivery, payment notifications,- **Admin**: User management, WhatsApp config, gateway status

### API Response Format

```json
// Success
{
  "success": true,
  "data": {...},
  "message": "Optional"
}

// Error
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  }
}
```

### Common Error Codes

| Code | Status | Meaning |
|------|--------|---------|
| AUTH_001 | 401 | Invalid token |
| AUTH_002 | 401 | Not authenticated |
| AUTH_003 | 403 | Admin access required |
| AUTH_004 | 403 | Super admin required |
| BAD_REQUEST | 400 | Invalid input |
| NOT_FOUND | 404 | Resource not found |
| CONFLICT | 409 | Duplicate resource |

### Rate Limits

- Auth endpoints: 5 req/min
- Admin endpoints: 20 req/min
- General API: 100 req/min

Headers: `X-RateLimit-Remaining` shows remaining requests

---

## Feature Flows

### 1. User Registration & Login

```
Register (phone + password + name)
  ↓
Backend sends OTP to WhatsApp
  ↓
User enters OTP
  ↓
Verify OTP → Get JWT token
  ↓
Store token, redirect to dashboard
```

**Login Alternative:**
```
Login (phone + password)
  ↓
Get JWT token directly
```

### 2. Bill Splitting Session

```
Create Session (title, currency, split method)
  ↓
Add Participants (from contacts or group)
  ↓
Add Bill Items (description, amount)
  ↓
Calculate Splits (backend distributes amounts)
  ↓
Send Notifications (WhatsApp messages to participants)
  ↓
Track Payments (participants upload proof)
  ↓
Complete Session (all payments verified)
```

**Session States:**
- `draft` → `pending` → `in_progress` → `completed`
- Any state can transition to `cancelled`

### 3. Payment Flow

```
Participant receives WhatsApp notification
  ↓
Opens public payment page (no auth required)
  ↓
Views amount due and bank details
  ↓
Transfers money via bank
  ↓
Uploads payment proof (screenshot)
  ↓
Creator reviews proof
  ↓
Approves or Rejects
  ↓
Participant notified via WhatsApp
```

**Payment States:**
- `pending` → `pending_review` → `approved` or `rejected`
- If rejected, participant can resubmit

### 4. Admin Management

```
Super Admin logs in (OTP)
  ↓
Views all admins
  ↓
Creates new admin (phone + name + role)
  ↓
New admin receives OTP
  ↓
New admin sets password
  ↓
New admin can now log in
```

**Admin Roles:**
- `admin`: Can manage own profile, view WhatsApp status
- `super_admin`: Can do everything + manage other admins

---

## UI/UX Vision

### Design Principles

1. **Simplicity First**: Clean interface, minimal cognitive load
2. **Trust & Security**: Professional appearance for financial transactions
3. **Mobile-First**: Works seamlessly on phones (primary use case)
4. **WhatsApp Native**: Feels like an extension of WhatsApp experience

### Light Mode Theme (Primary)

**Brand Color**: Sky Blue (#0ea5e9) - trustworthy, modern, friendly

**Visual Style:**
- Clean white backgrounds
- Subtle shadows and depth
- Ample white space
- Clear visual hierarchy
- WhatsApp green for WhatsApp-related actions

**Financial Colors:**
- Green for money received/income
- Red for money owed/expenses
- Neutral for pending states

### Landing Page

**Hero Section:**
- Clear value proposition: "Split bills effortlessly with friends"
- WhatsApp integration prominently featured
- Social proof (user count, bills split)
- Single CTA: "Get Started Free"

**Features Section:**
- Create sessions in seconds
- Automatic WhatsApp notifications
- Track payments with proof upload
- Works for everyone (no app needed for participants)

**Trust Signals:**
- Security badges
- User testimonials
- Clear pricing (free for basic use)

### Core Screens

**1. Dashboard**
- Quick stats: Total owed to you, Total you owe, Active sessions
- Recent sessions list
- Quick action: "New Session" button

**2. Session Detail**
- Session info (title, total, status)
- Participants list with amounts
- Bill items breakdown
- Actions: Send reminders, Mark complete

**3. Contacts**
- Search and filter
- List view with quick actions
- Groups for organization
- Import from phone (future)

**4. Payment Page (Public)**
- Large, clear amount display
- Bank details prominently shown
- Simple upload interface
- Status indicator

**5. Admin Panel**
- Dashboard with WhatsApp status
- Admin management table
- QR code display for WhatsApp pairing

### Mobile Experience

**Bottom Navigation:**
- Home (Dashboard)
- Sessions
- Contacts
- Profile

**Key Interactions:**
- Swipe to delete
- Pull to refresh
- Tap to action
- Long press for options

### WhatsApp Integration Feel

- Messages match WhatsApp's tone
- Green accent color for WhatsApp actions
- Seamless transition between app and WhatsApp
- Notifications feel native

---

## Technical Notes

### State Management

**What to Store:**
- Auth token (secure storage)
- User profile (context)
- Contacts cache (React Query)
- Active sessions (React Query)

**What NOT to Store:**
- Session details (fetch on demand)
- Payment details (public, no auth)
- Admin data (admin-only context)

### Real-time Updates

**Polling Strategies:**
- Payment status: Poll every 5 seconds when on payment page
- WhatsApp connection: Poll every 3 seconds during pairing
- Session updates: Poll every 10 seconds when viewing session

**When to Poll:**
- User is actively viewing the page
- Status is in a pending state
- After user action (e.g., sent notification)

### Error Handling

**User-Friendly Messages:**
- Network errors: "Check your connection"
- Auth errors: Redirect to login
- Validation errors: Show inline
- Server errors: "Something went wrong, please try again"

**Recovery:**
- Auto-retry for network errors (with backoff)
- Token refresh on 401
- Graceful degradation for non-critical features

### Performance

**Optimistic Updates:**
- Marking notification as sent
- Updating contact names
- Changing session status

**Background Sync:**
- Contact list refresh
- Session status updates
- Payment status checks

---

## Implementation Priorities

### Phase 1: Core Auth & Sessions
1. Login/Register screens
2. Dashboard
3. Create session flow
4. Basic session management

### Phase 2: Payments & Contacts
1. Public payment page
2. Payment proof upload
3. Contact management
4. Payment approval flow

### Phase 3: Notifications & Polish
1. WhatsApp notification triggers
2. Reminder system
3. Session completion flow
4. Error handling & edge cases

### Phase 4: Admin (Optional)
1. Admin authentication
2. WhatsApp configuration
3. Admin management (super admin)

---

## Questions to Explore

1. **Offline Support**: What should work without internet?
2. **Deep Linking**: Handle WhatsApp deep links to specific sessions?
3. **Push Notifications**: Beyond WhatsApp, browser push notifications?
4. **Multi-currency**: Display formatting for different currencies?
5. **Accessibility**: Screen reader support, keyboard navigation?

---

**End of Guide**

This document provides vision and direction. Specific implementation details (colors, components, exact APIs) should be referenced from `docs/swagger.yaml` and decided by the implementing developer based on this guidance.
