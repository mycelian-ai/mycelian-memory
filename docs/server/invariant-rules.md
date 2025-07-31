# 🔒 Invariant Rules - Never Mutate for Convenience

## ⚠️ CRITICAL RULE: Never Mutate Invariants to Get Incremental Changes Working

This document establishes the **most important rule** for maintaining system integrity:

**NEVER modify system invariants to make incremental feature changes work. Invariants and implementation are separate concepts.**

## 🛡️ What This Means

### ✅ **Correct Approach**
- **Invariants are contracts** - they define what the system guarantees
- **Implementation changes to satisfy invariants** - not the other way around
- **Tests fail when invariants are violated** - this is the system working correctly
- **Fix the implementation**, not the test or invariant

### ❌ **Forbidden Approach**
- Relaxing invariants because they're "too strict" for a feature
- Commenting out invariant tests to get builds to pass
- Modifying invariant validation to accommodate edge cases
- Adding "temporary" bypasses that become permanent

## 🔍 **Blackbox Invariant Testing**

Our invariant tests are **blackbox tests** that treat the service as an external system:

```go
// ✅ This is how invariants are tested
func TestMemoryEntryImmutabilityInvariant(t *testing.T) {
    // Create memory and entry via public API
    entry := createEntryViaAPI(...)
    
    // Correct the entry via public API
    correctionEntry := correctEntryViaAPI(...)
    
    // 🔒 INVARIANT: Cannot correct already corrected entries
    // This MUST fail - if it doesn't, the system is broken
    response := attemptSecondCorrection(...)
    assert.Equal(t, http.StatusBadRequest, response.StatusCode)
    assert.Contains(t, response.Error, "IMMUTABILITY_VIOLATION")
}
```

## 🎯 **Implementation Strategy**

### **When Invariant Tests Fail**

1. **DO NOT modify the invariant** - it's telling you the truth
2. **Analyze WHY the invariant is violated** - what business rule is being broken?
3. **Fix the implementation** to respect the invariant
4. **If the business rule is wrong**, discuss with product/stakeholders
5. **Only then** consider if the invariant itself needs updating

### **Separation of Concerns**

```
┌─────────────────────────────────────────────────────────────┐
│                     BUSINESS INVARIANTS                     │
│              (Never change for convenience)                 │
├─────────────────────────────────────────────────────────────┤
│                    IMPLEMENTATION LAYER                     │
│                 (Change to satisfy invariants)              │
├─────────────────────────────────────────────────────────────┤
│                      DATABASE LAYER                        │
│                 (Change to support implementation)          │
└─────────────────────────────────────────────────────────────┘
```

## 📋 **Practical Examples**

### ✅ **Good: Implementation Changes**
```go
// Invariant test fails because correction allows multiple corrections
// SOLUTION: Add transactional check in CorrectMemoryEntry

func (s *SpannerStorage) CorrectMemoryEntry(...) {
    return s.client.ReadWriteTransaction(ctx, func(...) error {
        // Check if already corrected
        if originalEntry.CorrectionTime != nil {
            return fmt.Errorf("IMMUTABILITY_VIOLATION: already corrected")
        }
        // ... rest of implementation
    })
}
```

### ❌ **Bad: Invariant Mutation**
```go
// DON'T DO THIS - relaxing invariant for convenience
func TestMemoryEntryImmutabilityInvariant(t *testing.T) {
    // ... correction logic ...
    
    // ❌ WRONG: Allowing second correction "for now"
    response := attemptSecondCorrection(...)
    // assert.Equal(t, http.StatusBadRequest, response.StatusCode) // COMMENTED OUT!
    // "TODO: Fix this later" - NEVER HAPPENS
}
```

## 🔧 **Development Workflow**

### **Feature Development Process**
1. **Write feature implementation**
2. **Run invariant tests** (they should pass)
3. **If invariant tests fail**:
   - ❌ Don't disable the test
   - ❌ Don't modify the invariant  
   - ✅ Fix your implementation
   - ✅ Discuss with team if business rule unclear

### **Code Review Checklist**
- [ ] Are all invariant tests still passing?
- [ ] Have any invariants been modified? (Requires special justification)
- [ ] Are there any TODOs about "fixing invariants later"? (Block merge)
- [ ] Does the implementation respect all business rules?

## 🏗️ **System Architecture Benefits**

### **Why This Rule Works**
1. **Prevents technical debt** - no shortcuts that become permanent
2. **Maintains data integrity** - business rules are always enforced
3. **Enables confident refactoring** - invariants catch regressions
4. **Improves system reliability** - fewer edge cases and bugs
5. **Better team communication** - invariant violations force discussions

### **Long-term Impact**
- **System becomes more reliable over time** (not more fragile)
- **Business rules are clearly documented and enforced**
- **New team members understand system constraints immediately**
- **Debugging is easier** - invariant violations point to exact problems

## 🚨 **Emergency Procedures**

### **If Production Issue Requires Invariant Bypass**
1. **Create incident ticket** with business justification
2. **Implement bypass with expiration date** and monitoring
3. **Plan permanent fix** within sprint cycle  
4. **Document technical debt** and track resolution
5. **Never make bypasses permanent**

## 🎉 **Success Metrics**

The system is healthy when:
- ✅ All invariant tests pass consistently
- ✅ Feature development respects existing invariants
- ✅ New features add invariants (don't remove them)
- ✅ Team discussions focus on business logic, not workarounds
- ✅ Production incidents decrease over time

---

**Remember: Invariants are your friend. They prevent bugs before they happen. Respect them, and they'll protect your system's integrity.** 