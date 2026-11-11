# SQL Dialect Unsupported Features

## Overview

This document records SQL constructs and DB-specific features that are not yet supported or are only partially supported in the TEI SQL dialect abstraction layer. These are tracked here as issues for future implementation.

## PostgreSQL Unsupported Features

### 1. ON CONFLICT / UPSERT
- **Status**: Not supported
- **Issue**: No generic representation for `INSERT ... ON CONFLICT ... DO UPDATE/DO NOTHING`
- **Impact**: Applications that need upsert behavior must use native SQL realizations
- **Workaround**: Use `ContractQuerySQLRealization` with native PostgreSQL SQL

### 2. RETURNING clause limitations
- **Status**: Partially supported (generated but not executed)
- **Issue**: While SQL generation supports `RETURNING`, not all target databases support it
- **Impact**: SQLite < 3.35.0 does not support `RETURNING`
- **Workaround**: Dialect adapters should conditionally include `RETURNING` based on version

### 3. Window functions
- **Status**: Not supported
- **Issue**: No representation for `OVER (PARTITION BY ... ORDER BY ...)`
- **Impact**: Analytics queries requiring window functions must use native SQL
- **Workaround**: Use `EmbeddedSQL` with dialect-specific source

### 4. CTEs (WITH RECURSIVE)
- **Status**: Not supported
- **Issue**: No representation for Common Table Expressions
- **Impact**: Hierarchical or recursive queries must use native SQL
- **Workaround**: Use native SQL realizations

### 5. Array operations
- **Status**: Not supported
- **Issue**: No representation for PostgreSQL array types or operators (`ANY`, `ALL`, `ARRAY[...]`)
- **Impact**: Array-valued attributes cannot be queried through the common contract
- **Workaround**: Use native SQL

### 6. JSONB operators
- **Status**: Not supported
- **Issue**: No representation for JSONB operators (`->`, `->>`, `@>`, `?`, etc.)
- **Impact**: JSON queries must use native SQL
- **Workaround**: Use native SQL realizations

### 7. DISTINCT ON
- **Status**: Not supported
- **Issue**: PostgreSQL-specific `SELECT DISTINCT ON (...)` has no generic equivalent
- **Impact**: Queries needing distinct-on behavior must use native SQL
- **Workaround**: Use native PostgreSQL SQL

### 8. TABLESAMPLE
- **Status**: Not supported
- **Issue**: No representation for `TABLESAMPLE` clauses
- **Impact**: Statistical sampling queries must use native SQL
- **Workaround**: Use native SQL

### 9. IS DISTINCT FROM / IS NOT DISTINCT FROM
- **Status**: Not supported
- **Issue**: These operators treat NULL as a comparable value, unlike `=` and `!=`
- **Impact**: NULL-safe comparisons must use native SQL or `IS NULL`/`IS NOT NULL` combinations
- **Workaround**: Use `IS NULL`/`IS NOT NULL` or native SQL

### 10. Advanced index features
- **Status**: Partially supported
- **Issue**: No representation for expression indexes, partial indexes with complex predicates
- **Impact**: Advanced PostgreSQL index features require provider-native authoring
- **Note**: TEI already has `ProviderExpressionMapping` for provider-native index expressions

## SQLite Unsupported Features

### 1. RETURNING clause version dependency
- **Status**: Partially supported
- **Issue**: SQLite added `RETURNING` in version 3.35.0 (2021-03-12)
- **Impact**: Older SQLite versions will fail on `RETURNING` clauses
- **Workaround**: Dialect should detect SQLite version and omit `RETURNING` if unsupported

### 2. Stored procedures
- **Status**: Not supported
- **Issue**: SQLite does not support stored procedures or functions
- **Impact**: No impact on common contract (stored procedures are out of scope)
- **Note**: This is a SQLite limitation, not a TEI limitation

### 3. Limited ALTER TABLE
- **Status**: Documented limitation
- **Issue**: SQLite has limited `ALTER TABLE` support (no `DROP COLUMN` before 3.35.0, no `RENAME COLUMN` before 3.25.0)
- **Impact**: Schema migrations must account for SQLite limitations
- **Workaround**: Use table rebuild pattern for complex ALTER operations

### 4. No native BOOLEAN type
- **Status**: Documented limitation
- **Issue**: SQLite uses INTEGER for booleans (0/1)
- **Impact**: Type mapping already handles this (maps to TEXT/INTEGER)
- **Note**: Current implementation maps to TEXT for simplicity

### 5. Limited window function support
- **Status**: Partially supported
- **Issue**: SQLite added window functions in 3.25.0, but some features may be limited
- **Impact**: Complex window functions may not work on older SQLite versions
- **Workaround**: Use native SQL with version checks

## General Limitations

### 1. No stored procedure support
- **Status**: Out of scope
- **Reason**: TEI's common contract focuses on SQL generation, not stored procedures
- **Impact**: Applications requiring stored procedures must use native SQL or provider-specific APIs

### 2. No full-text search
- **Status**: Out of scope
- **Reason**: FTS is DB-specific and not part of the common relational contract
- **Impact**: Full-text search must use provider-native features
- **Workaround**: Use `ProviderExpressionMapping` for FTS queries

### 3. No spatial/geometric types
- **Status**: Out of scope
- **Reason**: Spatial types are highly DB-specific
- **Impact**: Spatial queries must use native SQL

### 4. No recursive queries (without CTE)
- **Status**: Not supported
- **Issue**: Recursive queries require CTEs (`WITH RECURSIVE`)
- **Impact**: Hierarchical data queries must use native SQL or application-level recursion
- **Workaround**: Use native SQL realizations

### 5. No lateral joins
- **Status**: Not supported
- **Issue**: `LATERAL` joins are PostgreSQL-specific and have no generic equivalent
- **Impact**: Queries requiring lateral joins must use native SQL
- **Workaround**: Use native PostgreSQL SQL

### 6. No set operations
- **Status**: Not supported in common contract
- **Issue**: `UNION`, `INTERSECT`, `EXCEPT` are not represented in the common `SelectOptions`
- **Impact**: Set operations must use native SQL or separate queries
- **Workaround**: Use `EmbeddedSQL` with dialect-specific source

### 7. Parameter binding limitations
- **Status**: Known limitation
- **Issue**: Named parameters (`:name`) are converted to positional (`$1`, `$2`, `?`) but complex parameter expressions are not supported
- **Impact**: Dynamic parameter names or expressions cannot be used
- **Workaround**: Use native SQL with provider-specific parameter binding

## Compatibility Matrix

| Feature | PostgreSQL | SQLite | Generic Contract |
|---------|-----------|--------|-----------------|
| SELECT | ✅ | ✅ | ✅ |
| INSERT | ✅ | ✅ | ✅ |
| UPDATE | ✅ | ✅ | ✅ |
| DELETE | ✅ | ✅ | ✅ |
| JOINs | ✅ | ✅ | ✅ |
| WHERE conditions | ✅ | ✅ | ✅ |
| GROUP BY | ✅ | ✅ | ✅ |
| HAVING | ✅ | ✅ | ✅ |
| ORDER BY | ✅ | ✅ | ✅ |
| LIMIT/OFFSET | ✅ | ✅ | ✅ |
| RETURNING | ✅ | ⚠️ | ❌ |
| FOR UPDATE/SHARE | ✅ | ❌ | ❌ |
| CTEs | ❌ | ❌ | ❌ |
| Window functions | ❌ | ⚠️ | ❌ |
| Array types | ❌ | ❌ | ❌ |
| JSON operators | ❌ | ❌ | ❌ |
| Stored procedures | ❌ | ❌ | ❌ |

## Recommendations

1. **Short term**: Document these limitations in user-facing documentation
2. **Medium term**: Add version detection to dialect adapters for conditional feature support
3. **Long term**: Consider extending the common contract for high-demand features like CTEs and window functions
