# API Reference Documentation

## Table of Contents

1. [Authentication](#authentication)
2. [Invoices](#invoices)
3. [Suppliers](#suppliers)
4. [Contacts](#contacts)
5. [Areas](#areas)
6. [Workflows](#workflows)
7. [Routing Rules](#routing-rules)
8. [Approvals](#approvals)
9. [Purchase Orders](#purchase-orders)
10. [Error Handling](#error-handling)

---

## Authentication

### OAuth Login Flow

#### Initiate Login

```http
POST /auth/login
Content-Type: application/json

{
  "provider": "GOOGLE|MICROSOFT",
  "tenant_ruc": "20123456789" // optional
}
```

**Response:**

```json
{
  "auth_url": "https://accounts.google.com/o/oauth2/auth?...",
  "state": "random-state-token"
}
```

#### OAuth Callback

```http
GET /auth/callback/:provider?code=xxx&state=xxx
```

**Response:**

```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    /* UserDetailsDTO */
  },
  "tenant": {
    /* TenantDetailsDTO */
  }
}
```

#### Refresh Token

```http
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGc..."
}
```

#### Logout

```http
POST /auth/logout
Authorization: Bearer <token>
```

#### Get Current User

```http
GET /auth/me
Authorization: Bearer <token>
```

---

## Invoices

**Base Path:** `/api/tenant/:tenantId/invoices`

### Create Invoice

```http
POST /api/tenant/:tenantId/invoices
Authorization: Bearer <token>
Content-Type: application/json

{
  "vendor_id": "vendor_123",
  "assigned_area_id": "area_456",
  "invoice_number": "F001-00001234",
  "issue_date": "2025-10-01T00:00:00Z",
  "due_date": "2025-10-31T00:00:00Z",
  "currency": "PEN",
  "subtotal_amount": 100.00,
  "tax_amount": 18.00,
  "total_amount": 118.00
}
```

**Status:** `201 Created`

### Get Invoice by ID

```http
GET /api/tenant/:tenantId/invoices/:invoiceId
Authorization: Bearer <token>
```

### List Invoices

```http
GET /api/tenant/:tenantId/invoices?status=PENDING_APPROVAL&vendor_id=xxx&area_id=xxx
Authorization: Bearer <token>
```

**Query Parameters:**

- `status` - Invoice status filter
- `vendor_id` - Filter by vendor
- `area_id` - Filter by area
- `date_from` - Start date (YYYY-MM-DD)
- `date_to` - End date (YYYY-MM-DD)
- `min_amount` - Minimum amount
- `max_amount` - Maximum amount
- `invoice_number` - Invoice number search

### Get Invoices by Status

```http
GET /api/tenant/:tenantId/invoices/status/:status
Authorization: Bearer <token>
```

**Statuses:**

- `PENDING_VENDOR_VERIFICATION`
- `PENDING_APPROVAL`
- `APPROVED`
- `REJECTED`
- `IN_DISPUTE`
- `CONTABILIZED`

### Get Pending Approval Invoices

```http
GET /api/tenant/:tenantId/invoices/pending-approval
Authorization: Bearer <token>
```

### Get Invoices by Vendor

```http
GET /api/tenant/:tenantId/vendors/:vendorId/invoices
Authorization: Bearer <token>
```

### Get Invoices by Area

```http
GET /api/tenant/:tenantId/areas/:areaId/invoices
Authorization: Bearer <token>
```

### Update Invoice

```http
PUT /api/tenant/:tenantId/invoices/:invoiceId
Authorization: Bearer <token>
Content-Type: application/json

{
  "assigned_area_id": "area_789",
  "due_date": "2025-11-15T00:00:00Z",
  "subtotal_amount": 120.00,
  "tax_amount": 21.60,
  "total_amount": 141.60
}
```

### Approve Invoice

```http
POST /api/tenant/:tenantId/invoices/:invoiceId/approval/approve
Authorization: Bearer <token>
Content-Type: application/json

{
  "comments": "Approved by finance team"
}
```

### Reject Invoice

```http
POST /api/tenant/:tenantId/invoices/:invoiceId/approval/reject
Authorization: Bearer <token>
Content-Type: application/json

{
  "reason": "Missing required documentation for approval"
}
```

### Send to Dispute

```http
POST /api/tenant/:tenantId/invoices/:invoiceId/approval/dispute
Authorization: Bearer <token>
Content-Type: application/json

{
  "reason": "Amounts do not match purchase order"
}
```

### Mark as Contabilized

```http
POST /api/tenant/:tenantId/invoices/:invoiceId/contabilize
Authorization: Bearer <token>
```

### Validate with SIRE

```http
POST /api/tenant/:tenantId/invoices/:invoiceId/validate/sire
Authorization: Bearer <token>
```

### Three-Way Match

```http
POST /api/tenant/:tenantId/invoices/:invoiceId/validate/three-way-match
Authorization: Bearer <token>
```

### Delete Invoice

```http
DELETE /api/tenant/:tenantId/invoices/:invoiceId
Authorization: Bearer <token>
```

### SIRE Synchronization

#### Sync Invoices from SIRE

```http
POST /api/tenant/:tenantId/invoices/sire/sync
Authorization: Bearer <token>
Content-Type: application/json

{
  "date_from": "2025-09-01T00:00:00Z",
  "date_to": "2025-09-30T23:59:59Z"
}
```

**Response:**

```json
{
  "start_time": "2025-10-05T10:00:00Z",
  "end_time": "2025-10-05T10:15:00Z",
  "duration": "15m0s",
  "periods_synced": 1,
  "total_invoices": 150,
  "new_invoices": 142,
  "skipped": 5,
  "failed": 3,
  "results": [
    {
      "period": "202509",
      "total_invoices": 150,
      "new_invoices": 142,
      "skipped_invoices": 5,
      "failed_invoices": 3,
      "errors": []
    }
  ],
  "success": true
}
```

#### Get Sync Status

```http
GET /api/tenant/:tenantId/invoices/sire/sync/status
Authorization: Bearer <token>
```

---

## Suppliers

**Base Path:** `/api/tenant/:tenantId/suppliers`

### Create Supplier

```http
POST /api/tenant/:tenantId/suppliers
Authorization: Bearer <token>
Content-Type: application/json

{
  "ruc": "20123456789",
  "business_name": "Empresa SAC",
  "trade_name": "Empresa",
  "address": "Av. Principal 123, Lima",
  "phone": "+51 1 1234567",
  "email": "contacto@empresa.com",
  "website": "https://empresa.com",
  "supplier_type": "GOODS|SERVICES|BOTH",
  "tax_category": "RUC",
  "payment_terms": 30,
  "credit_limit": 50000.00
}
```

**Status:** `201 Created`

### Get Supplier by ID

```http
GET /api/tenant/:tenantId/suppliers/:supplierId
Authorization: Bearer <token>
```

### Get Supplier by RUC

```http
GET /api/tenant/:tenantId/suppliers/ruc/:ruc
Authorization: Bearer <token>
```

### List Suppliers

```http
GET /api/tenant/:tenantId/suppliers?status=ACTIVE&supplier_type=GOODS
Authorization: Bearer <token>
```

**Query Parameters:**

- `status` - `PENDING_VERIFICATION|ACTIVE|SUSPENDED|BLACKLISTED`
- `supplier_type` - `GOODS|SERVICES|BOTH`
- `is_verified` - `true|false`
- `ruc` - RUC search
- `business_name` - Name search

### Get Active Suppliers

```http
GET /api/tenant/:tenantId/suppliers/active
Authorization: Bearer <token>
```

### Get Pending Verification

```http
GET /api/tenant/:tenantId/suppliers/pending
Authorization: Bearer <token>
```

### Search Suppliers

```http
POST /api/tenant/:tenantId/suppliers/search
Authorization: Bearer <token>
Content-Type: application/json

{
  "query": "empresa",
  "status": "ACTIVE",
  "supplier_type": "GOODS",
  "is_verified": true,
  "page": 1,
  "per_page": 20
}
```

### Update Supplier

```http
PUT /api/tenant/:tenantId/suppliers/:supplierId
Authorization: Bearer <token>
Content-Type: application/json

{
  "trade_name": "New Trade Name",
  "phone": "+51 1 7654321",
  "email": "new@empresa.com"
}
```

### Verify Supplier

```http
POST /api/tenant/:tenantId/suppliers/:supplierId/verify
Authorization: Bearer <token>
Content-Type: application/json

{
  "comments": "Verified documents and legal status"
}
```

### Suspend Supplier

```http
POST /api/tenant/:tenantId/suppliers/:supplierId/suspend
Authorization: Bearer <token>
Content-Type: application/json

{
  "reason": "Payment issues - multiple overdue invoices"
}
```

### Blacklist Supplier

```http
POST /api/tenant/:tenantId/suppliers/:supplierId/blacklist
Authorization: Bearer <token>
Content-Type: application/json

{
  "reason": "Fraudulent activity detected"
}
```

### Activate Supplier

```http
POST /api/tenant/:tenantId/suppliers/:supplierId/activate
Authorization: Bearer <token>
```

### Delete Supplier

```http
DELETE /api/tenant/:tenantId/suppliers/:supplierId
Authorization: Bearer <token>
```

### Get Supplier Stats

```http
GET /api/tenant/:tenantId/suppliers/:supplierId/stats
Authorization: Bearer <token>
```

**Response:**

```json
{
  "supplier_id": "supplier_123",
  "business_name": "Empresa SAC",
  "status": "ACTIVE",
  "is_active": true,
  "is_verified": true,
  "total_orders": 45,
  "total_invoices": 120,
  "total_amount": 150000.0,
  "average_order_days": 15,
  "credit_limit": 50000.0,
  "credit_used": 12000.0,
  "last_order_date": "2025-09-28T00:00:00Z",
  "created_at": "2024-01-15T00:00:00Z"
}
```

### Get Supplier Health Check

```http
GET /api/tenant/:tenantId/suppliers/:supplierId/health
Authorization: Bearer <token>
```

### Get Supplier Contacts

```http
GET /api/tenant/:tenantId/suppliers/:supplierId/contacts
Authorization: Bearer <token>
```

### Bulk Operations

#### Bulk Verify

```http
POST /api/tenant/:tenantId/suppliers/bulk/verify
Authorization: Bearer <token>
Content-Type: application/json

{
  "supplier_ids": ["supplier_1", "supplier_2"],
  "comments": "Bulk verification completed"
}
```

#### Bulk Operation

```http
POST /api/tenant/:tenantId/suppliers/bulk/operation
Authorization: Bearer <token>
Content-Type: application/json

{
  "supplier_ids": ["supplier_1", "supplier_2"],
  "operation": "activate|suspend|blacklist",
  "reason": "Reason for bulk operation"
}
```

---

## Contacts

**Base Path:** `/api/tenant/:tenantId/suppliers/:supplierId/contacts`

### Create Contact

```http
POST /api/tenant/:tenantId/suppliers/:supplierId/contacts
Authorization: Bearer <token>
Content-Type: application/json

{
  "supplier_id": "supplier_123",
  "type": "PRIMARY|ACCOUNTING|PURCHASES|GENERAL|TECHNICAL|LEGAL|COMMERCIAL",
  "name": "Juan Pérez",
  "position": "Gerente de Ventas",
  "department": "Ventas",
  "email": "juan.perez@empresa.com",
  "phone": "+51 1 1234567",
  "mobile": "+51 999 123456",
  "extension": "101",
  "address": "Oficina Principal",
  "notes": "Contact notes",
  "is_primary": true
}
```

**Status:** `201 Created`

### Get Contact by ID

```http
GET /api/tenant/:tenantId/suppliers/:supplierId/contacts/:contactId
Authorization: Bearer <token>
```

### List Contacts

```http
GET /api/tenant/:tenantId/suppliers/:supplierId/contacts
Authorization: Bearer <token>
```

### Get Primary Contact

```http
GET /api/tenant/:tenantId/suppliers/:supplierId/contacts/primary
Authorization: Bearer <token>
```

### Get Contacts by Type

```http
GET /api/tenant/:tenantId/suppliers/:supplierId/contacts/type/:type
Authorization: Bearer <token>
```

**Types:** `PRIMARY`, `ACCOUNTING`, `PURCHASES`, `GENERAL`, `TECHNICAL`, `LEGAL`, `COMMERCIAL`

### Search Contacts

```http
POST /api/tenant/:tenantId/suppliers/:supplierId/contacts/search
Authorization: Bearer <token>
Content-Type: application/json

{
  "query": "juan",
  "type": "PRIMARY",
  "status": "ACTIVE",
  "is_active": true,
  "department": "Ventas",
  "page": 1,
  "per_page": 20
}
```

### Update Contact

```http
PUT /api/tenant/:tenantId/suppliers/:supplierId/contacts/:contactId
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Juan Carlos Pérez",
  "email": "jc.perez@empresa.com",
  "phone": "+51 1 7654321"
}
```

### Set as Primary

```http
POST /api/tenant/:tenantId/suppliers/:supplierId/contacts/:contactId/primary
Authorization: Bearer <token>
Content-Type: application/json

{
  "contact_id": "contact_123",
  "supplier_id": "supplier_456"
}
```

### Add Note

```http
POST /api/tenant/:tenantId/suppliers/:supplierId/contacts/:contactId/notes
Authorization: Bearer <token>
Content-Type: application/json

{
  "note": "Met during trade show, very responsive"
}
```

### Update Type

```http
PUT /api/tenant/:tenantId/suppliers/:supplierId/contacts/:contactId/type
Authorization: Bearer <token>
Content-Type: application/json

{
  "type": "ACCOUNTING"
}
```

### Activate Contact

```http
POST /api/tenant/:tenantId/suppliers/:supplierId/contacts/:contactId/activate
Authorization: Bearer <token>
```

### Deactivate Contact

```http
POST /api/tenant/:tenantId/suppliers/:supplierId/contacts/:contactId/deactivate
Authorization: Bearer <token>
```

### Block Contact

```http
POST /api/tenant/:tenantId/suppliers/:supplierId/contacts/:contactId/block
Authorization: Bearer <token>
Content-Type: application/json

{
  "reason": "No longer with company"
}
```

### Delete Contact

```http
DELETE /api/tenant/:tenantId/suppliers/:supplierId/contacts/:contactId
Authorization: Bearer <token>
```

### Get Contact Stats

```http
GET /api/tenant/:tenantId/suppliers/:supplierId/contacts/:contactId/stats
Authorization: Bearer <token>
```

### Get Contacts by Type Stats

```http
GET /api/tenant/:tenantId/suppliers/:supplierId/contacts/stats
Authorization: Bearer <token>
```

### Validate Contact Info

```http
POST /api/tenant/:tenantId/contacts/validate
Authorization: Bearer <token>
Content-Type: application/json

{
  "email": "test@empresa.com",
  "phone": "+51 1 1234567",
  "mobile": "+51 999 123456"
}
```

**Response:**

```json
{
  "is_valid": true,
  "issues": [],
  "email_valid": true,
  "phone_valid": true,
  "score": 95.5
}
```

### Bulk Operations

```http
POST /api/tenant/:tenantId/suppliers/:supplierId/contacts/bulk/operation
Authorization: Bearer <token>
Content-Type: application/json

{
  "contact_ids": ["contact_1", "contact_2"],
  "operation": "activate|deactivate|block|delete",
  "reason": "Bulk operation reason"
}
```

---

## Areas

**Base Path:** `/api/tenant/:tenantId/areas`

### Create Area

```http
POST /api/tenant/:tenantId/areas
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Finanzas",
  "code": "FIN",
  "description": "Departamento de Finanzas",
  "parent_id": "area_parent_123",
  "manager_id": "user_456",
  "budget": 100000.00,
  "cost_center": "CC-001"
}
```

**Status:** `201 Created`

### Get Area by ID

```http
GET /api/tenant/:tenantId/areas/:id
Authorization: Bearer <token>
```

### Get Area by Code

```http
GET /api/tenant/:tenantId/areas/code/:code
Authorization: Bearer <token>
```

### List Areas

```http
GET /api/tenant/:tenantId/areas?status=ACTIVE&parent_id=xxx
Authorization: Bearer <token>
```

**Query Parameters:**

- `status` - `ACTIVE|INACTIVE|ARCHIVED`
- `parent_id` - Filter by parent area (use "null" or "root" for root areas)
- `manager_id` - Filter by manager
- `code` - Area code
- `name` - Area name
- `cost_center` - Cost center

### Get Active Areas

```http
GET /api/tenant/:tenantId/areas/active
Authorization: Bearer <token>
```

### Get Root Areas

```http
GET /api/tenant/:tenantId/areas/root
Authorization: Bearer <token>
```

### Get Sub-areas

```http
GET /api/tenant/:tenantId/areas/:id/subareas
Authorization: Bearer <token>
```

### Get Areas by Manager

```http
GET /api/tenant/:tenantId/areas/manager/:managerId
Authorization: Bearer <token>
```

### Search Areas

```http
POST /api/tenant/:tenantId/areas/search
Authorization: Bearer <token>
Content-Type: application/json

{
  "query": "finanzas",
  "status": "ACTIVE",
  "parent_id": "area_parent_123",
  "manager_id": "user_456",
  "code": "FIN",
  "name": "Finanzas",
  "cost_center": "CC-001",
  "page": 1,
  "per_page": 20
}
```

### Update Area

```http
PUT /api/tenant/:tenantId/areas/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Finanzas y Contabilidad",
  "description": "Updated description",
  "status": "ACTIVE"
}
```

### Set Manager

```http
PUT /api/tenant/:tenantId/areas/:id/manager
Authorization: Bearer <token>
Content-Type: application/json

{
  "manager_id": "user_789"
}
```

### Remove Manager

```http
DELETE /api/tenant/:tenantId/areas/:id/manager
Authorization: Bearer <token>
```

### Set Budget

```http
PUT /api/tenant/:tenantId/areas/:id/budget
Authorization: Bearer <token>
Content-Type: application/json

{
  "budget": 150000.00
}
```

### Activate Area

```http
PUT /api/tenant/:tenantId/areas/:id/activate
Authorization: Bearer <token>
```

### Deactivate Area

```http
PUT /api/tenant/:tenantId/areas/:id/deactivate
Authorization: Bearer <token>
```

### Archive Area

```http
PUT /api/tenant/:tenantId/areas/:id/archive
Authorization: Bearer <token>
```

### Move Area

```http
PUT /api/tenant/:tenantId/areas/:id/move
Authorization: Bearer <token>
Content-Type: application/json

{
  "new_parent_id": "area_new_parent_456"
}
```

### Delete Area

```http
DELETE /api/tenant/:tenantId/areas/:id
Authorization: Bearer <token>
```

### Get Area Hierarchy

```http
GET /api/tenant/:tenantId/areas/:id/hierarchy
Authorization: Bearer <token>
```

**Response:**

```json
{
  "area": {
    /* Area entity */
  },
  "sub_areas": [
    {
      "area": {
        /* Sub-area entity */
      },
      "sub_areas": [],
      "level": 1
    }
  ],
  "level": 0
}
```

### Get Area Stats

```http
GET /api/tenant/:tenantId/areas/:id/stats
Authorization: Bearer <token>
```

**Response:**

```json
{
  "area_id": "area_123",
  "name": "Finanzas",
  "code": "FIN",
  "status": "ACTIVE",
  "is_active": true,
  "sub_areas_count": 3,
  "users_count": 12,
  "budget": 100000.0,
  "budget_used": 45000.0,
  "budget_percent": 45.0,
  "created_at": "2024-01-15T00:00:00Z"
}
```

### Bulk Operations

#### Bulk Activate

```http
PUT /api/tenant/:tenantId/areas/bulk/activate
Authorization: Bearer <token>
Content-Type: application/json

{
  "area_ids": ["area_1", "area_2"]
}
```

#### Bulk Deactivate

```http
PUT /api/tenant/:tenantId/areas/bulk/deactivate
Authorization: Bearer <token>
Content-Type: application/json

{
  "area_ids": ["area_1", "area_2"]
}
```

#### Bulk Archive

```http
PUT /api/tenant/:tenantId/areas/bulk/archive
Authorization: Bearer <token>
Content-Type: application/json

{
  "area_ids": ["area_1", "area_2"]
}
```

---

## Workflows

Workflows define multi-level approval processes for invoices.

**Workflow Types:**

- `SIMPLE` - Single approval level
- `MULTI_LEVEL` - Sequential multi-level approval
- `CONDITIONAL` - Approval based on conditions (amount, vendor, etc.)
- `PARALLEL` - Multiple approvals in parallel

**Step Types:**

- `APPROVAL` - Requires approval
- `NOTIFICATION` - Send notification only
- `AUTOMATIC` - Automatic action

### Workflow Structure

```json
{
  "id": "workflow_123",
  "tenant_id": "tenant_456",
  "name": "Standard Invoice Approval",
  "description": "Standard 3-level approval for invoices over $10,000",
  "type": "MULTI_LEVEL",
  "area_id": "area_finance",
  "is_active": true,
  "steps": [
    {
      "step_number": 1,
      "name": "Manager Approval",
      "type": "APPROVAL",
      "approver_role_id": "role_manager",
      "approver_user_id": null,
      "require_all": false,
      "timeout_hours": 48,
      "escalation_role_id": "role_director",
      "conditions": {
        "min_amount": 0,
        "max_amount": 50000
      }
    },
    {
      "step_number": 2,
      "name": "Finance Director Approval",
      "type": "APPROVAL",
      "approver_role_id": "role_finance_director",
      "require_all": true,
      "timeout_hours": 24
    }
  ],
  "created_at": "2024-01-15T00:00:00Z",
  "updated_at": "2025-09-01T00:00:00Z"
}
```

---

## Routing Rules

Routing rules automatically assign invoices to the correct area based on conditions.

**Rule Types:**

- `SUPPLIER` - Route by specific supplier
- `SUPPLIER_GROUP` - Route by supplier group
- `AMOUNT` - Route by amount threshold
- `CATEGORY` - Route by expense category
- `DEFAULT` - Default routing rule

### Rule Structure

```json
{
  "id": "rule_123",
  "tenant_id": "tenant_456",
  "name": "IT Vendors to IT Department",
  "description": "Route all IT vendor invoices to IT area",
  "type": "SUPPLIER_GROUP",
  "priority": 10,
  "is_active": true,
  "conditions": [
    {
      "field": "vendor_id",
      "operator": "in",
      "value": ["vendor_1", "vendor_2", "vendor_3"]
    }
  ],
  "target_area_id": "area_it",
  "workflow_id": "workflow_it_approval",
  "created_at": "2024-01-15T00:00:00Z",
  "updated_at": "2025-09-01T00:00:00Z"
}
```

**Condition Operators:**

- `eq` - Equals
- `gt` - Greater than
- `lt` - Less than
- `in` - In array
- `contains` - String contains
- `between` - Value between min and max

**Example Conditions:**

```json
// Amount-based routing
{
  "field": "total_amount",
  "operator": "between",
  "value": {
    "min": 10000,
    "max": 50000,
    "currency": "PEN"
  }
}

// Vendor-based routing
{
  "field": "vendor_id",
  "operator": "eq",
  "value": "vendor_123"
}

// Category-based routing
{
  "field": "category",
  "operator": "contains",
  "value": "IT"
}
```

### Routing Priority

Rules are evaluated in priority order (lower number = higher priority). The first matching rule is applied.

---

## Approvals

Approvals track the approval process for invoices through workflow steps.

### Approval Status

- `PENDING` - Awaiting approval
- `APPROVED` - Approved
- `REJECTED` - Rejected
- `CANCELED` - Canceled
- `TIMED_OUT` - Expired due to timeout

### Approval Structure

```json
{
  "id": "approval_123",
  "invoice_id": "invoice_456",
  "tenant_id": "tenant_789",
  "workflow_id": "workflow_101",
  "current_step": 2,
  "total_steps": 3,
  "status": "PENDING",
  "approver_role_id": "role_director",
  "approver_user_id": "user_123",
  "approved_by_id": null,
  "comments": "",
  "requested_at": "2025-10-05T10:00:00Z",
  "responded_at": null,
  "timeout_at": "2025-10-07T10:00:00Z",
  "created_at": "2025-10-05T10:00:00Z",
  "updated_at": "2025-10-05T10:00:00Z"
}
```

### Approval Workflow

1. **Invoice Created** → Routing engine determines target area
2. **Area Assigned** → Workflow assigned based on area/amount/rules
3. **Approval Created** → First step approval request created
4. **Step 1 Approved** → Next step approval created
5. **All Steps Approved** → Invoice marked as approved
6. **Any Step Rejected** → Invoice marked as rejected

---

## Purchase Orders

**Base Path:** `/api/tenant/:tenantId/purchase-orders` (not implemented in provided code, but referenced)

Purchase orders are linked to invoices for three-way matching validation.

### Three-Way Match Process

The system validates:

1. **Purchase Order** - Ordered quantities and prices
2. **Goods Receipt** - Received quantities
3. **Invoice** - Billed quantities and prices

**Match Criteria:**

- Quantities match (within tolerance)
- Prices match (within tolerance)
- Vendor matches
- Product codes match

---

## Error Handling

### Error Response Format

```json
{
  "code": "INVOICE_NOT_FOUND",
  "type": "NOT_FOUND",
  "message": "Factura no encontrada",
  "http_status": 404,
  "details": {
    "invoice_id": "invoice_123",
    "tenant_id": "tenant_456"
  },
  "timestamp": "2025-10-05T10:00:00Z"
}
```

### Error Types

| Type            | Description                         | HTTP Status Range |
| --------------- | ----------------------------------- | ----------------- |
| `VALIDATION`    | Invalid input data                  | 400               |
| `AUTHORIZATION` | Authentication/permission issues    | 401, 403          |
| `NOT_FOUND`     | Resource not found                  | 404               |
| `CONFLICT`      | Resource conflict (duplicate, etc.) | 409               |
| `BUSINESS`      | Business rule violation             | 400, 412, 428     |
| `EXTERNAL`      | External service error              | 502, 503          |
| `INTERNAL`      | Internal server error               | 500               |
| `TIMEOUT`       | Request timeout                     | 408, 504          |

### Common Error Codes

#### Invoice Errors

- `INVOICE_NOT_FOUND` - Invoice not found
- `INVOICE_ALREADY_EXISTS` - Duplicate invoice
- `INVALID_INVOICE_STATUS` - Invalid status for operation
- `REJECTION_REASON_REQUIRED` - Rejection reason required
- `INVOICE_NOT_APPROVED` - Invoice not approved
- `SIRE_VALIDATION_REQUIRED` - SIRE validation required
- `DUPLICATE_INVOICE` - Duplicate invoice detected
- `THREE_WAY_MATCH_FAILED` - Three-way match validation failed

#### Supplier Errors

- `SUPPLIER_NOT_FOUND` - Supplier not found
- `SUPPLIER_ALREADY_EXISTS` - Duplicate supplier
- `SUPPLIER_NOT_VERIFIED` - Supplier not verified
- `SUPPLIER_SUSPENDED` - Supplier suspended
- `SUPPLIER_BLACKLISTED` - Supplier blacklisted
- `INVALID_RUC` - Invalid RUC number
- `CREDIT_LIMIT_EXCEEDED` - Credit limit exceeded
- `SUPPLIER_IN_USE` - Cannot delete (in use)

#### Contact Errors

- `CONTACT_NOT_FOUND` - Contact not found
- `PRIMARY_CONTACT_EXISTS` - Primary contact already exists
- `INVALID_CONTACT_TYPE` - Invalid contact type
- `CONTACT_INFO_REQUIRED` - Email or phone required
- `CONTACT_BLOCKED` - Contact is blocked
- `DUPLICATE_CONTACT` - Duplicate contact
- `INVALID_EMAIL` - Invalid email format
- `INVALID_PHONE` - Invalid phone format

#### Area Errors

- `AREA_NOT_FOUND` - Area not found
- `AREA_ALREADY_EXISTS` - Duplicate area
- `AREA_IN_USE` - Cannot delete (in use)
- `AREA_INACTIVE` - Area is inactive
- `AREA_ARCHIVED` - Area is archived
- `INVALID_CODE` - Invalid area code
- `CIRCULAR_REFERENCE` - Circular reference in hierarchy
- `PARENT_NOT_FOUND` - Parent area not found

#### Workflow/Routing Errors

- `RULE_NOT_FOUND` - Routing rule not found
- `NO_MATCH` - No routing rule matched
- `WORKFLOW_NOT_FOUND` - Workflow not found
- `STEP_NOT_FOUND` - Workflow step not found
- `WORKFLOW_NOT_COMPLETE` - Workflow not complete
- `APPROVAL_NOT_FOUND` - Approval not found
- `ALREADY_RESPONDED` - Approval already responded
- `TIMED_OUT` - Approval timed out

#### Auth Errors

- `INVALID_REFRESH_TOKEN` - Invalid refresh token
- `EXPIRED_REFRESH_TOKEN` - Expired refresh token
- `INVALID_OAUTH_PROVIDER` - Invalid OAuth provider
- `OAUTH_AUTHORIZATION_FAILED` - OAuth authorization failed
- `INVALID_STATE` - Invalid OAuth state
- `TOKEN_GENERATION_FAILED` - Token generation failed
- `TOKEN_VALIDATION_FAILED` - Token validation failed

---

## Rate Limits

- **Standard requests:** 1,000 requests per hour per tenant
- **SIRE sync:** 10 requests per hour per tenant
- **Bulk operations:** 100 requests per hour per tenant

---

## Webhooks

_(Not implemented in provided code, but commonly needed)_

Webhooks can be configured to receive real-time notifications for:

- Invoice status changes
- Approval actions
- SIRE sync completion
- Supplier verification
- Workflow completion

---

## Pagination

List endpoints support pagination via query parameters:

```http
GET /api/tenant/:tenantId/invoices?page=1&per_page=50
```

**Response includes pagination metadata:**

```json
{
  "items": [
    /* ... */
  ],
  "total": 150,
  "page": 1,
  "per_page": 50,
  "total_pages": 3
}
```

---

## Authentication Headers

All authenticated endpoints require:

```http
Authorization: Bearer <access_token>
```

Or via cookie:

```
Cookie: access_token=<token>
```

---

## Date/Time Formats

- **Dates:** ISO 8601 format `YYYY-MM-DD`
- **Date-Times:** ISO 8601 with timezone `2025-10-05T10:00:00Z`
- **Timezone:** UTC by default (server: `America/Lima`)

---

## Currency

- **Default:** PEN (Peruvian Sol)
- **Supported:** PEN, USD
- **Format:** Decimal with 2 decimal places (e.g., `1234.56`)

---

## Multi-tenancy

All API endpoints are tenant-scoped via the `:tenantId` path parameter. Users can only access resources within their assigned tenant(s).

---

This documentation covers the main API endpoints available in your system. For specific field validations and detailed entity structures, refer to the individual entity documentation.
