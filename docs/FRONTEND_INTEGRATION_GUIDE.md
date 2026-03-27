# Letpai Backend Integration Guide

**Version**: 1.0.0  
**Base URL**: `http://localhost:3000/api/v1`  
**Documentation**: See `docs/swagger.yaml` for complete API specs

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Authentication](#authentication)
3. [Contact Management](#contact-management)
4. [Bill Splitting Sessions](#bill-splitting-sessions)
5. [Payment Processing](#payment-processing)
6. [WhatsApp Integration](#whatsapp-integration)
7. [Admin Module](#admin-module)
8. [Error Handling](#error-handling)
9. [Rate Limiting](#rate-limiting)
10. [Common Patterns](#common-patterns)
11. [UI/UX Design Guidelines](#uiux-design-guidelines)

---

## Quick Start

### Environment Setup
```bash
# Backend runs on
http://localhost:3000/api/v1

# Swagger UI (development)
http://localhost:8080
```

### API Response Format

**Success Response:**
```json
{
  "success": true,
  "data": { ... },
  "message": "Optional message"
}
```

**Error Response:**
```json
{
  "success": false,
  "error": {
    "code": "AUTH_001",
    "message": "Invalid credentials"
  }
}
```

### Authentication Header
```javascript
headers: {
  'Authorization': `Bearer ${token}`,
  'Content-Type': 'application/json'
}
```

---

## Authentication

### User Authentication Flow

#### 1. Register New User

```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "whatsapp_number": "+6281234567890",
  "password": "SecurePass123!",
  "full_name": "John Doe"
}
```

**Response (201):**
```json
{
  "success": true,
  "message": "OTP sent to your WhatsApp",
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Frontend Flow:**
1. User enters WhatsApp number, password, and name
2. Backend sends OTP to WhatsApp number
3. Store `user_id` temporarily for verification
4. Redirect to OTP verification screen

#### 2. Verify OTP

```http
POST /api/v1/auth/verify-otp
Content-Type: application/json

{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "otp_code": "123456"
}
```

**Response (200):**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "whatsapp_number": "+6281234567890",
  "full_name": "John Doe",
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

**Frontend Flow:**
1. User enters 6-digit OTP
2. On success, store `token` in secure storage (localStorage/sessionStorage)
3. Store user info in context/state
4. Redirect to dashboard

#### 3. Login

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "whatsapp_number": "+6281234567890",
  "password": "SecurePass123!"
}
```

**Response (200):**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "whatsapp_number": "+6281234567890",
  "full_name": "John Doe",
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

#### 4. Update Profile

```http
PATCH /api/v1/auth/profile
Authorization: Bearer {token}
Content-Type: application/json

{
  "full_name": "John Updated",
  "avatar_url": "https://cdn.example.com/avatar.jpg"
}
```

#### 5. Logout

```http
POST /api/v1/auth/logout
Authorization: Bearer {token}
```

### Token Management

**Token Lifecycle:**
- Tokens expire based on `JWT_EXPIRY_HOURS` config
- Check `expires_at` field from login response
- Implement token refresh or re-login when expired

**Frontend Implementation:**
```javascript
// Store token
localStorage.setItem('token', response.token);
localStorage.setItem('expires_at', response.expires_at);

// Check token validity
const isTokenValid = () => {
  const expiresAt = localStorage.getItem('expires_at');
  return expiresAt && new Date(expiresAt) > new Date();
};

// Add to all authenticated requests
const getAuthHeaders = () => ({
  'Authorization': `Bearer ${localStorage.getItem('token')}`,
  'Content-Type': 'application/json'
});
```

---

## Contact Management

### Contact Data Structure

```typescript
interface Contact {
  contact_id: string;
  whatsapp_number: string;
  full_name: string;
  group_id?: string;
  is_favorite: boolean;
  created_at: string;
  updated_at: string;
}
```

### CRUD Operations

#### 1. Get All Contacts

```http
GET /api/v1/contacts?search=john&sort_by=name&sort_order=asc&page=1&limit=20
Authorization: Bearer {token}
```

**Query Parameters:**
- `search`: Search by name
- `group_id`: Filter by group
- `is_favorite`: Filter favorites (boolean)
- `sort_by`: `name` | `created_at` | `whatsapp_number`
- `sort_order`: `asc` | `desc`
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20)

**Response (200):**
```json
{
  "contacts": [
    {
      "contact_id": "...",
      "whatsapp_number": "+6281234567890",
      "full_name": "John Doe",
      "is_favorite": false,
      "group": {
        "group_id": "...",
        "name": "Friends"
      }
    }
  ],
  "total": 50,
  "page": 1,
  "limit": 20
}
```

#### 2. Create Contact

```http
POST /api/v1/contacts
Authorization: Bearer {token}
Content-Type: application/json

{
  "whatsapp_number": "+6281234567890",
  "full_name": "John Doe",
  "group_id": "optional-group-id",
  "is_favorite": false
}
```

#### 3. Update Contact

```http
PUT /api/v1/contacts/{contact_id}
Authorization: Bearer {token}
Content-Type: application/json

{
  "full_name": "John Updated",
  "is_favorite": true
}
```

#### 4. Delete Contact

```http
DELETE /api/v1/contacts/{contact_id}
Authorization: Bearer {token}
```

### Bulk Operations

```http
POST /api/v1/contacts/bulk
Authorization: Bearer {token}
Content-Type: application/json

{
  "action": "delete", // or "add_to_group", "remove_from_group"
  "contact_ids": ["id1", "id2", "id3"],
  "group_id": "target-group-id" // for group operations
}
```

### Import from Contact Groups

```http
POST /api/v1/contacts/import
Authorization: Bearer {token}
Content-Type: application/json

{
  "group_id": "source-group-id"
}
```

### Contact Groups

#### Get Groups
```http
GET /api/v1/contact-groups
Authorization: Bearer {token}
```

#### Create Group
```http
POST /api/v1/contact-groups
Authorization: Bearer {token}

{
  "name": "Office Friends",
  "description": "Friends from office"
}
```

---

## Bill Splitting Sessions

### Session Data Structure

```typescript
interface Session {
  session_id: string;
  user_id: string;
  title: string;
  description?: string;
  status: 'draft' | 'pending' | 'in_progress' | 'completed' | 'cancelled';
  total_amount: number;
  currency: string;
  split_method: 'equal' | 'custom' | 'percentage';
  created_at: string;
  updated_at: string;
  participants: Participant[];
  bill_items: BillItem[];
}
```

### Session Flow

```
1. Create Session (draft)
   ↓
2. Add Participants
   ↓
3. Add Bill Items
   ↓
4. Calculate Splits
   ↓
5. Send Notifications
   ↓
6. Track Payments
   ↓
7. Complete Session
```

### 1. Create Session

```http
POST /api/v1/sessions
Authorization: Bearer {token}
Content-Type: application/json

{
  "title": "Dinner at Restaurant",
  "description": "Monthly team dinner",
  "currency": "IDR",
  "split_method": "equal"
}
```

**Response (201):**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Dinner at Restaurant",
  "status": "draft",
  "total_amount": 0,
  "currency": "IDR"
}
```

### 2. Add Participants

```http
POST /api/v1/sessions/{session_id}/participants
Authorization: Bearer {token}
Content-Type: application/json

{
  "participants": [
    {
      "contact_id": "contact-uuid",
      "custom_amount": 50000, // for custom split
      "percentage": 25 // for percentage split
    }
  ]
}
```

**Or import from group:**
```http
POST /api/v1/sessions/{session_id}/participants/import
Authorization: Bearer {token}

{
  "group_id": "group-uuid"
}
```

### 3. Add Bill Items

```http
POST /api/v1/sessions/{session_id}/bills
Authorization: Bearer {token}
Content-Type: application/json

{
  "description": "Main course",
  "amount": 250000,
  "quantity": 1,
  "participant_ids": ["id1", "id2"] // optional split
}
```

### 4. Calculate Splits

```http
PUT /api/v1/sessions/{session_id}/calculate-splits
Authorization: Bearer {token}
```

**Response (200):**
```json
{
  "session_id": "...",
  "total_amount": 500000,
  "participants": [
    {
      "participant_id": "...",
      "contact_name": "John Doe",
      "amount_due": 125000,
      "amount_paid": 0,
      "payment_status": "pending"
    }
  ]
}
```

### 5. Send Notifications

```http
POST /api/v1/sessions/{session_id}/send-notifications
Authorization: Bearer {token}

{
  "message": "Please pay your share for dinner"
}
```

### 6. Update Session Status

```http
PUT /api/v1/sessions/{session_id}
Authorization: Bearer {token}

{
  "status": "in_progress"
}
```

### Session Status Transitions

```
draft → pending → in_progress → completed
  ↓        ↓          ↓
  └────────┴──────────┴→ cancelled
```

---

## Payment Processing

### Payment Flow

```
1. Participant receives WhatsApp notification
   ↓
2. Participant opens public payment page
   ↓
3. Participant submits payment proof
   ↓
4. Creator reviews payment proof
   ↓
5. Creator approves/rejects payment
   ↓
6. Notification sent to participant
```

### 1. Get Public Payment Page

```http
GET /api/v1/payments/{participant_id}/public
```

**Response (200):**
```json
{
  "session_title": "Dinner at Restaurant",
  "participant_name": "John Doe",
  "amount_due": 125000,
  "currency": "IDR",
  "payment_status": "pending",
  "bank_details": {
    "bank_name": "BCA",
    "account_number": "1234567890",
    "account_name": "John Creator"
  }
}
```

### 2. Submit Payment Proof

```http
POST /api/v1/payments/{participant_id}/submit
Content-Type: multipart/form-data

{
  "proof_image": <binary>,
  "notes": "Paid via BCA mobile"
}
```

**Response (201):**
```json
{
  "proof_id": "proof-uuid",
  "status": "pending_review",
  "message": "Payment proof submitted successfully"
}
```

### 3. Approve Payment

```http
POST /api/v1/payments/{proof_id}/approve
Authorization: Bearer {token}

{
  "notes": "Payment verified"
}
```

### 4. Reject Payment

```http
POST /api/v1/payments/{proof_id}/reject
Authorization: Bearer {token}

{
  "reason": "Invalid proof",
  "notes": "Please upload clearer image"
}
```

### 5. Bulk Approve/Reject

```http
POST /api/v1/payments/bulk-approve
Authorization: Bearer {token}

{
  "proof_ids": ["id1", "id2", "id3"]
}
```

### Payment Status

- `pending`: No payment submitted
- `pending_review`: Payment proof submitted, awaiting approval
- `approved`: Payment accepted
- `rejected`: Payment rejected, needs resubmission

---

## WhatsApp Integration

### How WhatsApp Integration Works

```
Backend sends notification
        ↓
WhatsApp Gateway receives request
        ↓
Gateway sends to participant's WhatsApp
        ↓
Participant receives message
        ↓
(Optional) Webhook receives delivery status
```

### Supported Message Types

1. **OTP Messages**: For authentication
2. **Payment Notifications**: Bill split requests
3. **Reminders**: Payment reminders
4. **Custom Messages**: Session notifications

### Notification Endpoints

#### Send Session Notifications

```http
POST /api/v1/sessions/{session_id}/send-notifications
Authorization: Bearer {token}

{
  "message": "Custom message",
  "participant_ids": ["id1", "id2"] // optional, all if omitted
}
```

#### Send Reminder

```http
POST /api/v1/participants/{participant_id}/reminder
Authorization: Bearer {token}

{
  "message": "Please pay your share"
}
```

#### Bulk Reminder

```http
POST /api/v1/sessions/{session_id}/bulk-reminder
Authorization: Bearer {token}

{
  "message": "Reminder: Payment due tomorrow"
}
```

---

## Admin Module

### Admin vs User Authentication

**User Authentication:**
- WhatsApp number + password OR OTP
- Standard JWT token (no role)

**Admin Authentication:**
- WhatsApp number + OTP only (no password initially)
- JWT token with role (`admin` or `super_admin`)

### Admin Authentication Flow

#### 1. Initiate Login

```http
POST /api/v1/admin/auth/initiate
Content-Type: application/json

{
  "whatsapp_number": "+6281234567890"
}
```

**Response (200):**
```json
{
  "session_id": "verification-uuid",
  "message": "OTP sent to your WhatsApp"
}
```

#### 2. Verify OTP

```http
POST /api/v1/admin/auth/verify-otp
Content-Type: application/json

{
  "session_id": "verification-uuid",
  "otp_code": "123456"
}
```

**Response (200):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

#### 3. Setup Password (First-time Admins)

```http
PUT /api/v1/admin/profile/setup-password
Authorization: Bearer {admin-token}

{
  "password": "SecureAdminPass123!"
}
```

### Admin Roles

**Admin Role:**
- View own profile
- Update own password
- View WhatsApp gateway status
- Generate QR codes
- Logout

**Super Admin Role:**
- All admin capabilities
- List all admins
- Create new admins
- Update any admin
- Delete admins
- Manage WhatsApp configuration

### Admin Management Endpoints

#### Get All Admins (Super Admin Only)

```http
GET /api/v1/admin/admins
Authorization: Bearer {super-admin-token}
```

#### Create Admin (Super Admin Only)

```http
POST /api/v1/admin/admins
Authorization: Bearer {super-admin-token}

{
  "whatsapp_number": "+6281234567890",
  "full_name": "New Admin",
  "role": "admin" // or "super_admin"
}
```

#### Update Admin (Super Admin Only)

```http
PUT /api/v1/admin/admins/{admin_id}
Authorization: Bearer {super-admin-token}

{
  "full_name": "Updated Name",
  "role": "super_admin",
  "is_active": true
}
```

#### Delete Admin (Super Admin Only)

```http
DELETE /api/v1/admin/admins/{admin_id}
Authorization: Bearer {super-admin-token}
```

### WhatsApp Configuration

#### Get Gateway Status

```http
GET /api/v1/admin/status
Authorization: Bearer {admin-token}
```

**Response (200):**
```json
{
  "is_connected": true,
  "gateway_token_valid": true,
  "phone_number": "+6281234567890",
  "last_connected_at": "2024-01-01T10:00:00Z",
  "qr_code": "base64-string-if-pending",
  "qr_code_expires_at": "2024-01-01T10:05:00Z"
}
```

#### Generate QR Code

```http
POST /api/v1/admin/qr-code
Authorization: Bearer {admin-token}

{
  "phone_number": "+6281234567890"
}
```

**Response (200):**
```json
{
  "qr_code": "base64-encoded-image",
  "qr_code_expires_at": "2024-01-01T10:05:00Z"
}
```

**Frontend Flow:**
1. Display QR code image (base64)
2. Show countdown timer (5 minutes expiry)
3. Poll `/admin/status` endpoint
4. When `is_connected` becomes true, redirect to dashboard

#### Update WhatsApp Config

```http
PUT /api/v1/admin/config
Authorization: Bearer {admin-token}

{
  "phone_number": "+6281234567890",
  "token": "gateway-api-token"
}
```

---

## Error Handling

### Error Response Format

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message"
  }
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `AUTH_001` | 401 | Missing/invalid token |
| `AUTH_002` | 401 | Authentication required |
| `AUTH_003` | 403 | Admin access required |
| `AUTH_004` | 403 | Super admin access required |
| `BAD_REQUEST` | 400 | Invalid request data |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource conflict |
| `INTERNAL_FAILURE` | 500 | Server error |

### Frontend Error Handling

```javascript
const handleApiError = (error) => {
  if (!error.response) {
    // Network error
    return 'Network error. Please check your connection.';
  }

  const { status, data } = error.response;

  switch (status) {
    case 401:
      // Clear token and redirect to login
      localStorage.removeItem('token');
      window.location.href = '/login';
      break;
    case 403:
      return 'You do not have permission to perform this action.';
    case 404:
      return 'Resource not found.';
    case 429:
      return 'Too many requests. Please wait and try again.';
    case 500:
      return 'Server error. Please try again later.';
    default:
      return data?.error?.message || 'An error occurred.';
  }
};
```

---

## Rate Limiting

### Rate Limit Headers

Every response includes:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640000000
```

### Rate Limit Error (429)

```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Please wait before retrying.",
    "retry_after": 60
  }
}
```

### Frontend Rate Limit Handling

```javascript
const checkRateLimit = (response) => {
  const remaining = response.headers['x-ratelimit-remaining'];
  if (remaining && parseInt(remaining) < 10) {
    console.warn('Rate limit approaching:', remaining, 'requests remaining');
  }
};

const handleRateLimit = (error) => {
  if (error.response?.status === 429) {
    const retryAfter = error.response.data.error.retry_after || 60;
    console.log(`Rate limited. Retry after ${retryAfter} seconds`);
    // Implement exponential backoff
    return new Promise(resolve => {
      setTimeout(resolve, retryAfter * 1000);
    });
  }
};
```

### Rate Limits by Endpoint Type

- **Authentication**: 5 requests per minute
- **Admin endpoints**: 20 requests per minute
- **General API**: 100 requests per minute

---

## Common Patterns

### 1. Pagination

```javascript
const fetchContacts = async (page = 1, limit = 20) => {
  const response = await fetch(
    `/api/v1/contacts?page=${page}&limit=${limit}`,
    { headers: getAuthHeaders() }
  );
  
  const data = await response.json();
  
  return {
    items: data.contacts,
    total: data.total,
    hasMore: (page * limit) < data.total
  };
};
```

### 2. File Upload

```javascript
const uploadPaymentProof = async (participantId, file) => {
  const formData = new FormData();
  formData.append('proof_image', file);
  formData.append('notes', 'Payment via mobile banking');

  const response = await fetch(
    `/api/v1/payments/${participantId}/submit`,
    {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${getToken()}`
        // Don't set Content-Type, let browser set it
      },
      body: formData
    }
  );

  return response.json();
};
```

### 3. Real-time Status Updates

```javascript
// Polling pattern for payment status
const pollPaymentStatus = async (participantId) => {
  const interval = setInterval(async () => {
    const response = await fetch(
      `/api/v1/payments/${participantId}/public`
    );
    const data = await response.json();
    
    if (data.payment_status !== 'pending_review') {
      clearInterval(interval);
      // Update UI
    }
  }, 5000); // Poll every 5 seconds
};

// Polling pattern for WhatsApp connection
const pollWhatsAppStatus = async () => {
  const interval = setInterval(async () => {
    const response = await fetch(
      '/api/v1/admin/status',
      { headers: getAuthHeaders() }
    );
    const data = await response.json();
    
    if (data.is_connected) {
      clearInterval(interval);
      // Redirect to dashboard
    } else if (data.qr_code) {
      // Update QR code display
    }
  }, 3000); // Poll every 3 seconds
};
```

### 4. Optimistic Updates

```javascript
const updateContact = async (contactId, updates) => {
  // Optimistically update UI
  dispatch(updateContactLocal({ id: contactId, ...updates }));
  
  try {
    const response = await fetch(
      `/api/v1/contacts/${contactId}`,
      {
        method: 'PUT',
        headers: getAuthHeaders(),
        body: JSON.stringify(updates)
      }
    );
    
    if (!response.ok) throw new Error('Update failed');
    
    const data = await response.json();
    // Confirm update with server data
    dispatch(updateContactSuccess(data));
  } catch (error) {
    // Revert on error
    dispatch(updateContactFailure(contactId));
  }
};
```

### 5. Token Refresh Pattern

```javascript
let isRefreshing = false;
let refreshSubscribers = [];

const refreshToken = async () => {
  if (isRefreshing) {
    return new Promise(resolve => {
      refreshSubscribers.push(resolve);
    });
  }

  isRefreshing = true;
  
  try {
    const response = await fetch('/api/v1/auth/refresh', {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${getToken()}` }
    });
    
    const data = await response.json();
    setToken(data.token);
    
    refreshSubscribers.forEach(resolve => resolve(data.token));
    refreshSubscribers = [];
    
    return data.token;
  } finally {
    isRefreshing = false;
  }
};
```

---

## Environment Configuration

### Development
```
API_BASE_URL=http://localhost:3000/api/v1
SWAGGER_UI=http://localhost:8080
```

### Production
```
API_BASE_URL=https://api.letpai.com/api/v1
```

---

## Testing Checklist

### User Flow
- [ ] Register with WhatsApp number
- [ ] Receive and verify OTP
- [ ] Login with password
- [ ] Update profile
- [ ] Logout

### Contact Flow
- [ ] Create contact
- [ ] List contacts with pagination
- [ ] Search contacts
- [ ] Update contact
- [ ] Delete contact
- [ ] Create contact group
- [ ] Import contacts from group

### Session Flow
- [ ] Create session
- [ ] Add participants
- [ ] Add bill items
- [ ] Calculate splits
- [ ] Send notifications
- [ ] Update session status
- [ ] Complete session

### Payment Flow
- [ ] View public payment page
- [ ] Upload payment proof
- [ ] Approve payment
- [ ] Reject payment
- [ ] Bulk approve payments

### Admin Flow
- [ ] Initiate admin login
- [ ] Verify admin OTP
- [ ] Setup admin password
- [ ] View admin profile
- [ ] Get WhatsApp status
- [ ] Generate QR code
- [ ] Create new admin (super admin)
- [ ] Update admin (super admin)
- [ ] Delete admin (super admin)

---

## UI/UX Design Guidelines

### Design Philosophy

**Core Values:**
- **Simplicity**: Clean, minimal interface with clear hierarchy
- **Trust**: Professional appearance that builds confidence in financial transactions
- **Efficiency**: Quick actions with minimal friction
- **Accessibility**: WCAG 2.1 AA compliant, works for everyone

### Light Mode Theme (Primary)

#### Color Palette

```css
/* Primary Colors */
--primary-50: #f0f9ff;     /* Lightest blue */
--primary-100: #e0f2fe;
--primary-200: #bae6fd;
--primary-300: #7dd3fc;
--primary-400: #38bdf8;
--primary-500: #0ea5e9;    /* Main brand color - Sky Blue */
--primary-600: #0284c7;    /* Primary actions */
--primary-700: #0369a1;
--primary-800: #075985;
--primary-900: #0c4a6e;    /* Darkest blue */

/* Neutral Colors (Light Mode) */
--neutral-50: #fafafa;     /* Background */
--neutral-100: #f5f5f5;    /* Card background */
--neutral-200: #e5e5e5;    /* Borders */
--neutral-300: #d4d4d4;
--neutral-400: #a3a3a3;
--neutral-500: #737373;    /* Secondary text */
--neutral-600: #525252;
--neutral-700: #404040;    /* Primary text */
--neutral-800: #262626;
--neutral-900: #171717;    /* Headings */

/* Semantic Colors */
--success-500: #22c55e;    /* Green - Success states */
--success-100: #dcfce7;
--warning-500: #f59e0b;    /* Amber - Warnings */
--warning-100: #fef3c7;
--error-500: #ef4444;      /* Red - Errors */
--error-100: #fee2e2;
--info-500: #3b82f6;       /* Blue - Information */
--info-100: #dbeafe;

/* WhatsApp Theme */
--whatsapp-500: #25d366;   /* WhatsApp green */
--whatsapp-600: #128c7e;

/* Financial Colors */
--income-500: #10b981;     /* Money received */
--expense-500: #f43f5e;    /* Money owed */
```

#### Background & Surface

```css
/* Light Mode Backgrounds */
--bg-primary: #ffffff;     /* Main background */
--bg-secondary: #fafafa;   /* Page background */
--bg-tertiary: #f5f5f5;    /* Card background */
--bg-hover: #f0f0f0;       /* Hover states */

/* Shadows */
--shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.05);
--shadow-md: 0 4px 6px rgba(0, 0, 0, 0.07);
--shadow-lg: 0 10px 15px rgba(0, 0, 0, 0.1);
--shadow-xl: 0 20px 25px rgba(0, 0, 0, 0.15);
```

### Typography

```css
/* Font Stack */
font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 
             'Helvetica Neue', Arial, sans-serif;

/* Font Sizes */
--text-xs: 0.75rem;      /* 12px */
--text-sm: 0.875rem;     /* 14px */
--text-base: 1rem;       /* 16px */
--text-lg: 1.125rem;     /* 18px */
--text-xl: 1.25rem;      /* 20px */
--text-2xl: 1.5rem;      /* 24px */
--text-3xl: 1.875rem;    /* 30px */
--text-4xl: 2.25rem;     /* 36px */

/* Line Heights */
--leading-tight: 1.25;
--leading-normal: 1.5;
--leading-relaxed: 1.625;

/* Font Weights */
--font-normal: 400;
--font-medium: 500;
--font-semibold: 600;
--font-bold: 700;
```

### Layout Guidelines

#### Spacing System
```css
/* 8px base unit */
--space-1: 0.25rem;  /* 4px */
--space-2: 0.5rem;   /* 8px */
--space-3: 0.75rem;  /* 12px */
--space-4: 1rem;     /* 16px */
--space-5: 1.25rem;  /* 20px */
--space-6: 1.5rem;   /* 24px */
--space-8: 2rem;     /* 32px */
--space-10: 2.5rem;  /* 40px */
--space-12: 3rem;    /* 48px */
--space-16: 4rem;    /* 64px */
```

#### Border Radius
```css
--radius-sm: 0.25rem;   /* 4px - Small elements */
--radius-md: 0.5rem;    /* 8px - Buttons, inputs */
--radius-lg: 0.75rem;   /* 12px - Cards */
--radius-xl: 1rem;      /* 16px - Large cards */
--radius-2xl: 1.5rem;   /* 24px - Hero sections */
--radius-full: 9999px;  /* Circular elements */
```

#### Breakpoints
```css
/* Mobile First */
--bp-sm: 640px;    /* Small devices */
--bp-md: 768px;    /* Tablets */
--bp-lg: 1024px;   /* Laptops */
--bp-xl: 1280px;   /* Desktops */
--bp-2xl: 1536px;  /* Large screens */
```

---

## Landing Page Design

### Hero Section

**Layout:** Full-width, centered content
**Background:** White with subtle gradient or pattern

```
┌─────────────────────────────────────────────────────┐
│  Logo                    Login  Sign Up              │
├─────────────────────────────────────────────────────┤
│                                                      │
│          Split Bills Effortlessly                   │
│                                                      │
│   The easiest way to split expenses with            │
│   friends, family, and colleagues via WhatsApp      │
│                                                      │
│   [Get Started - Free]  [Learn More]               │
│                                                      │
│   ┌──────────────────────────────────────────────┐ │
│   │   App Screenshot / Illustration               │ │
│   │   - Bill splitting interface                  │ │
│   │   - Contact list                              │ │
│   │   - Payment tracking                          │ │
│   └──────────────────────────────────────────────┘ │
│                                                      │
└─────────────────────────────────────────────────────┘
```

### Features Section

```
┌─────────────────────────────────────────────────────┐
│                   How It Works                       │
├─────────────────────────────────────────────────────┤
│  ┌─────────┐  ┌─────────┐  ┌─────────┐            │
│  │ 📱      │  │ 💰      │  │ ✅      │            │
│  │ Create  │  │ Track   │  │ Settle  │            │
│  │ Session │  │ Bills   │  │ Up      │            │
│  │         │  │         │  │         │            │
│  │ Add     │  │ See who │  │ Mark    │            │
│  │ expenses│  │ owes    │  │ as paid │            │
│  │ easily  │  │ what    │  │ with    │            │
│  │         │  │         │  │ proof   │            │
│  └─────────┘  └─────────┘  └─────────┘            │
└─────────────────────────────────────────────────────┘
```

### Benefits Section

**Key Messages:**
- **No More Awkward Conversations**: Automated WhatsApp reminders
- **Crystal Clear Tracking**: Know exactly who owes what
- **Quick Settlements**: Upload payment proof, get approved fast
- **Works for Everyone**: No app download needed for participants

### Social Proof Section

```
┌─────────────────────────────────────────────────────┐
│              Trusted by Thousands                    │
├─────────────────────────────────────────────────────┤
│                                                      │
│    💰 1M+       👥 10K+       ⚡ 50K+              │
│    Bills        Users         Sessions             │
│    Split        Worldwide     Completed            │
│                                                      │
│  ┌─────────────────────────────────────────────┐  │
│  │ "Finally, a bill splitting app that works!  │  │
│  │  No more spreadsheets or confusion."        │  │
│  │                                    - User A  │  │
│  └─────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

### CTA Section

```
┌─────────────────────────────────────────────────────┐
│                                                      │
│       Ready to Split Bills Stress-Free?            │
│                                                      │
│       [Get Started - It's Free]                    │
│                                                      │
│       No credit card required                      │
│                                                      │
└─────────────────────────────────────────────────────┘
```

### Footer

```
┌─────────────────────────────────────────────────────┐
│  Logo                                               │
│                                                      │
│  Product       Company        Support              │
│  - Features    - About        - Help Center        │
│  - Pricing     - Blog         - Contact            │
│  - API         - Careers      - Privacy            │
│                               - Terms              │
│                                                      │
│  © 2024 Letpai. All rights reserved.               │
│  Made with ❤️ for easier bill splitting            │
└─────────────────────────────────────────────────────┘
```

---

## Page Layouts

### 1. Dashboard Layout

```
┌─────────────────────────────────────────────────────────┐
│  Logo    [Dashboard] [Sessions] [Contacts]    👤 Profile │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌────────────────────────────────────────────────────┐│
│  │  Welcome back, John!                               ││
│  │  Here's your bill splitting overview               ││
│  └────────────────────────────────────────────────────┘│
│                                                          │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐      │
│  │ Total Owed  │ │ Total Owed  │ │ Active      │      │
│  │ to You      │ │ by You      │ │ Sessions    │      │
│  │             │ │             │ │             │      │
│  │  Rp 250.000 │ │ Rp 125.000  │ │ 3           │      │
│  └─────────────┘ └─────────────┘ └─────────────┘      │
│                                                          │
│  Recent Sessions                              [View All]│
│  ┌────────────────────────────────────────────────────┐│
│  │ 🍽️ Dinner at Restaurant         Rp 500.000       ││
│  │    4 participants • 2 pending                      ││
│  ├────────────────────────────────────────────────────┤│
│  │ 🎉 Team Lunch                   Rp 750.000        ││
│  │    6 participants • Completed                      ││
│  └────────────────────────────────────────────────────┘│
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 2. Session Detail Layout

```
┌─────────────────────────────────────────────────────────┐
│  ← Back to Sessions                                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  🍽️ Dinner at Restaurant                                │
│  Status: In Progress                                     │
│                                                          │
│  ┌────────────────────────────────────────────────────┐│
│  │ Summary                                            ││
│  │ Total Amount: Rp 500.000                           ││
│  │ Split Method: Equal                                ││
│  │ Participants: 4                                    ││
│  └────────────────────────────────────────────────────┘│
│                                                          │
│  Participants (4)                         [+ Add]       │
│  ┌────────────────────────────────────────────────────┐│
│  │ 👤 John Doe                Rp 125.000    Pending   ││
│  │ 👤 Jane Smith              Rp 125.000    Paid      ││
│  │ 👤 Bob Johnson             Rp 125.000    Pending   ││
│  │ 👤 Alice Brown (You)       Rp 125.000    Creator   ││
│  └────────────────────────────────────────────────────┘│
│                                                          │
│  Bill Items (3)                           [+ Add Item]  │
│  ┌────────────────────────────────────────────────────┐│
│  │ Main Course                           Rp 250.000  ││
│  │ Drinks                                Rp 150.000  ││
│  │ Dessert                               Rp 100.000  ││
│  ├────────────────────────────────────────────────────┤│
│  │ Total                                 Rp 500.000  ││
│  └────────────────────────────────────────────────────┘│
│                                                          │
│  [Calculate Splits] [Send Reminders] [Mark Complete]   │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 3. Payment Page (Public)

```
┌─────────────────────────────────────────────────────────┐
│                                                          │
│  💳 Payment for Dinner at Restaurant                    │
│                                                          │
│  Hi John!                                                │
│  Your share of the bill is:                             │
│                                                          │
│  ┌────────────────────────────────────────────────────┐│
│  │                                                     ││
│  │            Rp 125.000                              ││
│  │                                                     ││
│  └────────────────────────────────────────────────────┘│
│                                                          │
│  Bank Details                                            │
│  Bank: BCA                                               │
│  Account: 1234567890                                     │
│  Name: Alice Brown                                       │
│                                                          │
│  Upload Payment Proof                                    │
│  ┌────────────────────────────────────────────────────┐│
│  │                                                     ││
│  │         📷 Click or drag to upload                 ││
│  │            Screenshots accepted                    ││
│  │                                                     ││
│  └────────────────────────────────────────────────────┘│
│                                                          │
│  Notes (Optional)                                        │
│  ┌────────────────────────────────────────────────────┐│
│  │ e.g., "Paid via BCA mobile at 2:30 PM"            ││
│  └────────────────────────────────────────────────────┘│
│                                                          │
│  [Submit Payment Proof]                                  │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## Component Guidelines

### Buttons

```jsx
// Primary Button
<button className="bg-primary-600 text-white px-6 py-3 rounded-lg font-medium hover:bg-primary-700 transition-colors">
  Get Started
</button>

// Secondary Button
<button className="bg-neutral-100 text-neutral-700 px-6 py-3 rounded-lg font-medium hover:bg-neutral-200 transition-colors">
  Learn More
</button>

// WhatsApp Button
<button className="bg-whatsapp-500 text-white px-6 py-3 rounded-lg font-medium hover:bg-whatsapp-600">
  Send via WhatsApp
</button>

// Ghost Button
<button className="text-primary-600 px-6 py-3 rounded-lg font-medium hover:bg-primary-50">
  Cancel
</button>
```

### Cards

```jsx
// Standard Card
<div className="bg-white rounded-xl shadow-sm border border-neutral-200 p-6">
  <h3 className="text-lg font-semibold text-neutral-900">Card Title</h3>
  <p className="text-neutral-600 mt-2">Card content goes here</p>
</div>

// Interactive Card
<div className="bg-white rounded-xl shadow-sm border border-neutral-200 p-6 hover:shadow-md hover:border-primary-300 transition-all cursor-pointer">
  {/* Card content */}
</div>
```

### Forms

```jsx
// Input Field
<div className="space-y-2">
  <label className="text-sm font-medium text-neutral-700">
    WhatsApp Number
  </label>
  <input 
    type="tel"
    className="w-full px-4 py-3 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none transition-all"
    placeholder="+6281234567890"
  />
  <p className="text-xs text-neutral-500">
    Include country code (e.g., +62)
  </p>
</div>
```

### Badges/Status

```jsx
// Status Badges
<span className="px-3 py-1 text-xs font-medium rounded-full bg-warning-100 text-warning-700">
  Pending
</span>

<span className="px-3 py-1 text-xs font-medium rounded-full bg-success-100 text-success-700">
  Paid
</span>

<span className="px-3 py-1 text-xs font-medium rounded-full bg-error-100 text-error-700">
  Overdue
</span>

<span className="px-3 py-1 text-xs font-medium rounded-full bg-neutral-100 text-neutral-700">
  Draft
</span>
```

### Amount Display

```jsx
// Large Amount (Hero)
<div className="text-center">
  <p className="text-sm text-neutral-500 mb-1">Amount Due</p>
  <p className="text-5xl font-bold text-neutral-900">Rp 125.000</p>
</div>

// Inline Amount
<span className="text-lg font-semibold text-neutral-900">
  Rp 125.000
</span>

// Amount Owed (Red)
<span className="text-lg font-semibold text-expense-500">
  - Rp 250.000
</span>

// Amount to Receive (Green)
<span className="text-lg font-semibold text-income-500">
  + Rp 125.000
</span>
```

---

## Mobile-First Design

### Touch Targets
- Minimum 44px x 44px for all interactive elements
- Adequate spacing between clickable elements

### Mobile Navigation
```
┌─────────────────────┐
│  ☰  Letpai    👤    │  <- Hamburger menu
├─────────────────────┤
│                      │
│   [Content Area]     │
│                      │
│                      │
├─────────────────────┤
│ 🏠  📋  👥  ➕  ⚙️  │  <- Bottom navigation
└─────────────────────┘
```

### Responsive Breakpoints
```css
/* Mobile (default) */
.container { padding: 1rem; }

/* Tablet (md: 768px) */
@media (min-width: 768px) {
  .container { padding: 2rem; max-width: 768px; }
}

/* Desktop (lg: 1024px) */
@media (min-width: 1024px) {
  .container { padding: 2rem; max-width: 1024px; }
}

/* Large Desktop (xl: 1280px) */
@media (min-width: 1280px) {
  .container { padding: 3rem; max-width: 1280px; }
}
```

---

## Animation & Transitions

### Recommended Durations
```css
--duration-fast: 150ms;     /* Hover states */
--duration-normal: 300ms;   /* Modals, dropdowns */
--duration-slow: 500ms;     /* Page transitions */
```

### Easing Functions
```css
--ease-in: cubic-bezier(0.4, 0, 1, 1);
--ease-out: cubic-bezier(0, 0, 0.2, 1);
--ease-in-out: cubic-bezier(0.4, 0, 0.2, 1);
```

### Common Animations
```css
/* Fade In */
@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

/* Slide Up */
@keyframes slideUp {
  from { 
    opacity: 0; 
    transform: translateY(10px); 
  }
  to { 
    opacity: 1; 
    transform: translateY(0); 
  }
}

/* Scale In */
@keyframes scaleIn {
  from { 
    opacity: 0; 
    transform: scale(0.95); 
  }
  to { 
    opacity: 1; 
    transform: scale(1); 
  }
}
```

---

## Icon Library

**Recommended:** Heroicons or Lucide Icons

**Common Icons:**
- `Home` - Dashboard
- `ClipboardList` - Sessions
- `Users` - Contacts
- `Plus` - Create new
- `Check` - Confirm/Approve
- `X` - Cancel/Reject
- `Camera` - Upload proof
- `Send` - Send notifications
- `Bell` - Notifications
- `Settings` - Settings
- `User` - Profile
- `Logout` - Logout
- `WhatsApp` - WhatsApp actions

---

## Accessibility Checklist

- [ ] All images have alt text
- [ ] Color contrast ratio ≥ 4.5:1 (WCAG AA)
- [ ] Focus indicators visible
- [ ] Keyboard navigation works
- [ ] Screen reader compatible
- [ ] Form labels associated with inputs
- [ ] Error messages clear and helpful
- [ ] Loading states announced
- [ ] Touch targets ≥ 44px
- [ ] No color-only information

---

## Recommended Tech Stack

### Frontend Framework
- **React** (with Next.js for SSR/SSG)
- **Vue** (with Nuxt.js)
- **Svelte** (with SvelteKit)

### Styling
- **Tailwind CSS** - Utility-first (recommended)
- **CSS Modules** - Scoped styles
- **Styled Components** - CSS-in-JS

### State Management
- **React Query** / **TanStack Query** - Server state
- **Zustand** / **Jotai** - Client state
- **Context API** - Simple needs

### Form Handling
- **React Hook Form** - Performance
- **Zod** / **Yup** - Validation

### HTTP Client
- **Axios** - Feature-rich
- **Fetch API** - Native

### UI Components
- **Headless UI** - Unstyled components
- **Radix UI** - Accessible primitives
- **Shadcn/ui** - Pre-built components

---

## Support

For issues or questions:
- Check `docs/swagger.yaml` for complete API specs
- Review error codes in Error Handling section
- Contact backend team for API-related issues

---

**Last Updated**: 2024-01-01  
**Backend Version**: 1.0.0
