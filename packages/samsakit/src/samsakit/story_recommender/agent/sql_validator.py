"""Deterministic SQL validator - Layer 1 of SQL safety."""

import re
from dataclasses import dataclass

from .schema_registry import (
    FORBIDDEN_TABLES,
    TABLES_REQUIRING_USER_FILTER,
    is_column_allowed,
    is_table_allowed,
)


@dataclass
class ValidationResult:
    """Result of SQL validation."""

    valid: bool
    error: str | None = None
    warning: str | None = None


# Forbidden SQL keywords (any write operations)
FORBIDDEN_KEYWORDS = [
    r"\bINSERT\b",
    r"\bUPDATE\b",
    r"\bDELETE\b",
    r"\bDROP\b",
    r"\bTRUNCATE\b",
    r"\bALTER\b",
    r"\bCREATE\b",
    r"\bGRANT\b",
    r"\bREVOKE\b",
    r"\bEXECUTE\b",
    r"\bCALL\b",
]

# SQL injection patterns
INJECTION_PATTERNS = [
    r"--",  # Single line comment
    r"/\*",  # Multi-line comment start
    r"\*/",  # Multi-line comment end
    r"xp_",  # xp_ commands
    r"information_schema",
    r"pg_",
    r"pg_catalog",
]

# Regex patterns compiled
_FORBIDDEN_KW_PATTERN = re.compile("|".join(FORBIDDEN_KEYWORDS), re.IGNORECASE)
_INJECTION_PATTERN = re.compile("|".join(INJECTION_PATTERNS), re.IGNORECASE)
_SELECT_PATTERN = re.compile(r"^\s*SELECT\b", re.IGNORECASE)
_FROM_PATTERN = re.compile(r"\bFROM\b", re.IGNORECASE)
_WHERE_PATTERN = re.compile(r"\bWHERE\b", re.IGNORECASE)
_USER_ID_PATTERN = re.compile(r"\buser_id\s*=\s*['\"]?\$\d+['\"]?", re.IGNORECASE)


class DeterministicSQLChecker:
    """Deterministic SQL validation - no LLM needed."""

    def __init__(self):
        self.max_limit = 100

    def validate(self, sql: str, user_id_param: str = "$1") -> ValidationResult:
        """
        Validate SQL query deterministically.

        Args:
            sql: The SQL query to validate
            user_id_param: The parameter placeholder for user_id (e.g., "$1")
        """
        if not sql or not sql.strip():
            return ValidationResult(valid=False, error="Empty SQL query")

        # Check 1: Must be SELECT only
        if not _SELECT_PATTERN.match(sql):
            return ValidationResult(valid=False, error="Only SELECT statements allowed")

        # Check 2: No forbidden keywords (write operations)
        if _FORBIDDEN_KW_PATTERN.search(sql):
            return ValidationResult(valid=False, error="Forbidden SQL keywords detected")

        # Check 3: No injection patterns
        if _INJECTION_PATTERN.search(sql):
            return ValidationResult(valid=False, error="Potential SQL injection detected")

        # Check 4: All tables must exist in registry
        tables = self._extract_tables(sql)
        for table in tables:
            if table.lower() in FORBIDDEN_TABLES:
                return ValidationResult(valid=False, error=f"Forbidden table '{table}' referenced")
            if not is_table_allowed(table):
                return ValidationResult(valid=False, error=f"Table '{table}' not in allowed schema")

        # Check 5: Tables requiring user_id must have it in WHERE
        for table in tables:
            if table.lower() in TABLES_REQUIRING_USER_FILTER:
                if not self._has_user_filter(sql, table, user_id_param):
                    return ValidationResult(
                        valid=False,
                        error=f"Table '{table}' requires user_id filter in WHERE clause",
                    )

        # Check 6: LIMIT must not exceed max
        limit_match = re.search(r"\bLIMIT\s+(\d+)", sql, re.IGNORECASE)
        if limit_match:
            limit = int(limit_match.group(1))
            if limit > self.max_limit:
                return ValidationResult(
                    valid=False,
                    error=f"LIMIT cannot exceed {self.max_limit}",
                )

        # Check 7: All columns must be allowed
        columns = self._extract_columns(sql, tables)
        for table, column in columns:
            if not is_column_allowed(table, column):
                return ValidationResult(
                    valid=False,
                    error=f"Column '{column}' not allowed in table '{table}'",
                )

        return ValidationResult(valid=True)

    def _extract_tables(self, sql: str) -> list[str]:
        """Extract table names from SQL."""
        tables = []
        # Match FROM and JOIN clauses
        pattern = r"(?:FROM|JOIN)\s+([a-zA-Z_][a-zA-Z0-9_]*)"
        for match in re.finditer(pattern, sql, re.IGNORECASE):
            tables.append(match.group(1))
        return tables

    def _extract_columns(self, sql: str, tables: list[str]) -> list[tuple[str, str]]:
        """Extract (table, column) pairs from SQL."""
        columns = []
        # Match table.column or just column
        pattern = r"(?:([a-zA-Z_][a-zA-Z0-9_]*)\.)?([a-zA-Z_][a-zA-Z0-9_]*)"
        for match in re.finditer(pattern, sql):
            table = match.group(1)
            column = match.group(2)
            if table:
                columns.append((table, column))
            elif tables:
                # Assume first table if no table specified
                columns.append((tables[0], column))
        return columns

    def _has_user_filter(self, sql: str, table: str, user_id_param: str) -> bool:
        """Check if SQL has user_id filter for the given table."""
        # Look for WHERE clause with user_id
        if not _WHERE_PATTERN.search(sql):
            return False

        # Check for user_id = $1 or user_id = :user_id
        user_filter_pattern = rf"\b{table}\.user_id\s*=\s*{re.escape(user_id_param)}"
        if re.search(user_filter_pattern, sql, re.IGNORECASE):
            return True

        # Check for JOIN with user_id
        join_pattern = rf"JOIN\s+{table}\s+ON\s+.+\.user_id\s*="
        if re.search(join_pattern, sql, re.IGNORECASE):
            return True

        return False


# Singleton instance
sql_checker = DeterministicSQLChecker()
