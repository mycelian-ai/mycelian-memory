# 🔒 Memory Backend System Invariants

## Overview
These are the **immutable business rules** that govern the memory backend system. These invariants must **never be violated** and are enforced through code validation, database constraints, and comprehensive testing.

## 🧠 Memory Entry Invariants

### Entry State Rules
- **Once corrected, entries are immutable forever** - No further modifications allowed
- **Only summaries can be updated** on active entries (content and metadata are immutable)
- **Fresh entries can be corrected exactly once** - One correction per entry maximum
- **Entry content (RawEntry) is immutable** after creation - Never changes
- **LastUpdateTime tracks summary changes** - Updated only when summary modified

### Entry Correction Rules
- **Corrections must stay within the same memory** - No cross-memory pathways
- **User manages cross-memory corrections externally** - System doesn't provide this
- **Cannot correct already corrected entries** - Prevents correction chains
- **Cannot correct deleted entries** - Deleted entries are immutable
- **Must provide correction reason** - Audit trail requirement
- **Correction creates new entry** - Original preserved with pointers

### Entry Update Rules
- **Summary is the only updatable field** - All other fields immutable
- **Content (RawEntry) never changes** - Immutable after creation
- **Metadata updates forbidden** - Business decision for simplicity
- **No updates on corrected entries** - Corrected = immutable
- **No updates on deleted entries** - Deleted = immutable
- **LastUpdateTime set to UTC** when summary changes

### Entry Deletion Rules
- **Soft delete sets DeletionScheduledTime** - Doesn't physically remove
- **Deleted entries filtered from all lists** - Immediate consistency
- **Deleting already deleted is idempotent** - No error, just noop
- **Cannot modify deleted entries** - Deleted = read-only

## 🗂️ Memory Container Invariants

### Memory Lifecycle Rules
- **Memory deletion is soft delete** - Sets DeletionScheduledTime
- **Deleted memories don't appear in lists** - Immediate filtering
- **Memory deletion cascades to entries** - All entries marked deleted
- **Memory metadata can be updated** - Title, description changeable
- **Memory type is immutable** - Cannot change after creation

### Memory Access Rules
- **Users can only access own memories** - Strict user boundary
- **No shared memories between users** - Single ownership model
- **All memory operations require userID** - Must be in request path
- **Memory operations filtered by userID** - Automatic isolation

## 👤 User System Invariants

### User Identity Rules
- **Email must be unique across active users** - Business identifier
- **Email reusable after user deletion** - Allows account recreation
- **UserID is immutable UUID4** - Never changes after creation
- **DisplayName is optional** - Can be null or updated
- **TimeZone can be updated** - User preference

### User Access Rules
- **Users access only their own data** - Complete data isolation
- **No cross-user data access** - Enforced at all levels
- **All operations require valid userID** - Must exist in system
- **Suspended users cannot create/modify** - Read-only access
- **Only ACTIVE users can perform full operations**

### User Lifecycle Rules
- **New users default to ACTIVE status** - Ready to use immediately
- **User status: ACTIVE, SUSPENDED, DELETED** - Three valid states
- **ACTIVE users have full access** - All operations allowed
- **SUSPENDED users are read-only** - Cannot modify anything
- **DELETED users have no access** - Cannot perform any operations

### User Data Rules
- **User deletion cascades completely** - All memories and entries deleted
- **Deleted user data not recoverable** - One-way operation
- **LastActiveTime updated on operations** - Tracks user activity
- **CreationTime is immutable** - Set once, never changed
- **All user timestamps in UTC** - No timezone complications

### User Validation Rules
- **Email format validation required** - RFC 5321 compliant
- **TimeZone must be valid string** - Standard timezone identifiers
- **UserID must be valid UUID4** - Format validation enforced
- **DisplayName has length limits** - Reasonable size constraints

## ⏰ Timestamp Invariants

### Universal UTC Rules
- **All timestamps stored in UTC** - No timezone confusion
- **All application logs in UTC** - Consistent logging
- **CreationTime, LastUpdateTime, CorrectionTime in UTC** - System consistency
- **DeletionScheduledTime in UTC** - Cleanup scheduling
- **Database commit timestamps are UTC** - Spanner consistency

### Timestamp Immutability
- **CreationTime never changes** - Immutable record of creation
- **CorrectionTime set once** - When correction happens
- **LastUpdateTime tracks latest summary change** - Summary modification tracking
- **DeletionScheduledTime set once** - When deletion scheduled

## 🔍 Query & Consistency Invariants

### Immediate Consistency Rules
- **List operations filter deleted immediately** - No deleted items in lists
- **State validation is transactional** - ACID compliance
- **User boundary enforcement immediate** - Access control instantaneous
- **CRUD operations have immediate consistency** - Real-time data integrity

### Eventual Consistency Rules
- **Vector search via CDC is eventually consistent** - Async processing
- **Deleted entries eventually disappear from search** - CDC propagation delay
- **Search results may briefly include deleted items** - Until CDC processes
- **Fresh corrections immediately in lists** - But search catches up later

## 🛡️ Transaction Invariants

### Atomicity Rules
- **All multi-table operations use transactions** - ACID compliance
- **State checks must be within transactions** - Consistency guarantee
- **Race conditions prevented by isolation** - Concurrent safety
- **Failed operations leave no partial changes** - All-or-nothing

### Validation Rules
- **Entry mutability checked before modification** - Immutability enforcement
- **User existence validated before operations** - Foreign key integrity
- **Email uniqueness checked transactionally** - Prevent duplicates
- **State transitions are atomic** - No partial state changes

## 🔐 Security Invariants

### Data Isolation Rules
- **Complete user data isolation** - No data leakage between users
- **Memory operations auto-filtered by userID** - Built-in security
- **Search results respect user boundaries** - Even in async operations
- **No system-wide data access** - Always user-scoped

### Access Control Rules
- **All operations require valid userID** - Authentication prerequisite
- **Suspended users limited to read operations** - Controlled access
- **Deleted users have zero access** - Complete lockout
- **No privilege escalation possible** - Users cannot access others' data

## 🎯 Business Rule Invariants

### Immutability Core Rules
- **Memory entries are immutable facts** - Represents things that happened
- **Corrections preserve original facts** - Audit trail maintained
- **Only summaries updatable for search optimization** - Limited mutability
- **Deletion is logical, not physical** - Data preservation for compliance

### Audit Trail Rules
- **Every correction creates audit trail** - Original + correction preserved
- **Correction reasons are mandatory** - Accountability requirement
- **All timestamps preserve chronological order** - Time-based ordering
- **User actions are traceable** - LastActiveTime tracking

---

## 🚨 Critical Implementation Notes

### Enforcement Mechanisms
- **Database constraints** - Basic data integrity
- **Application-level validation** - Business rule enforcement
- **Transaction boundaries** - Consistency guarantees
- **Comprehensive test coverage** - Invariant verification

### Violation Prevention
- **Code-level guards** - Runtime validation
- **Test-driven invariants** - Automated verification
- **File-level protection markers** - Development guidance
- **Review requirements** - Human oversight

### Monitoring & Alerts
- **Invariant violations logged** - Security monitoring
- **Performance impact tracking** - System health
- **Data integrity checks** - Regular validation
- **User behavior monitoring** - Audit compliance

---

**🔒 Remember: These invariants are SYSTEM CRITICAL. Violations can compromise data integrity, user trust, and system reliability.** 