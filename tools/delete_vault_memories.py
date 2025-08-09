#!/usr/bin/env python3
"""
Delete all memories and related data within a vault by vault ID.

Supports Postgres (primary) and legacy SQLite (deprecated).

Usage:
    # Postgres (recommended)
    python delete_vault_memories.py <vault_id> --pg-dsn "postgres://user:pass@host:5432/db?sslmode=disable" [--delete-vault] [--yes]
    
    # or rely on environment (MEMORY_BACKEND_POSTGRES_DSN)
    MEMORY_BACKEND_POSTGRES_DSN=postgres://... python delete_vault_memories.py <vault_id> [--delete-vault] [--yes]

    # Legacy SQLite (deprecated)
    python delete_vault_memories.py <vault_id> --db-path /path/to/memory.db [--delete-vault] [--yes]

Examples:
    # Preview what will be deleted (safe)
    python delete_vault_memories.py 97db1a27-695b-4bf3-bbd1-a00c6d4501de --pg-dsn postgres://...
    
    # Delete memories but keep the vault
    python delete_vault_memories.py 97db1a27-695b-4bf3-bbd1-a00c6d4501de --pg-dsn postgres://... --yes
    
    # Delete everything including the vault itself
    python delete_vault_memories.py 97db1a27-695b-4bf3-bbd1-a00c6d4501de --pg-dsn postgres://... --delete-vault --yes
"""

import argparse
import os
import sqlite3
import sys
from pathlib import Path
from typing import Dict, Any, List, Tuple, Optional

try:
    import psycopg  
except Exception:  
    psycopg = None


class VaultMemoryDeleter:
    def __init__(self, db_path: str):
        self.db_path = Path(db_path)
        if not self.db_path.exists():
            raise FileNotFoundError(f"Database file not found: {db_path}")
    
    def get_vault_info(self, vault_id: str) -> Dict[str, Any]:
        """Get vault information and statistics."""
        with sqlite3.connect(self.db_path) as conn:
            conn.row_factory = sqlite3.Row
            cursor = conn.cursor()
            
            # Get vault info
            cursor.execute(
                "SELECT UserId, Title, Description, CreationTime FROM Vaults WHERE VaultId = ?",
                (vault_id,)
            )
            vault = cursor.fetchone()
            
            if not vault:
                return None
            
            # Get counts
            cursor.execute(
                "SELECT COUNT(*) as count FROM Memories WHERE VaultId = ?",
                (vault_id,)
            )
            memory_count = cursor.fetchone()['count']
            
            cursor.execute(
                "SELECT COUNT(*) as count FROM MemoryEntries WHERE VaultId = ?",
                (vault_id,)
            )
            entry_count = cursor.fetchone()['count']
            
            cursor.execute(
                "SELECT COUNT(*) as count FROM MemoryContexts WHERE VaultId = ?",
                (vault_id,)
            )
            context_count = cursor.fetchone()['count']
            
            return {
                'vault': dict(vault),
                'memory_count': memory_count,
                'entry_count': entry_count,
                'context_count': context_count
            }
    
    def get_memories_list(self, vault_id: str) -> List[Tuple[str, str, str]]:
        """Get list of memories in the vault."""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            cursor.execute(
                "SELECT Title, MemoryType, Description FROM Memories WHERE VaultId = ? ORDER BY Title",
                (vault_id,)
            )
            return cursor.fetchall()
    
    def delete_vault_memories(self, vault_id: str, delete_vault: bool = False) -> Dict[str, int]:
        """Delete all memories and related data from the vault."""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            
            # Delete in proper order to maintain referential integrity
            
            # 1. Delete memory entries
            cursor.execute("DELETE FROM MemoryEntries WHERE VaultId = ?", (vault_id,))
            entries_deleted = cursor.rowcount
            
            # 2. Delete memory contexts
            cursor.execute("DELETE FROM MemoryContexts WHERE VaultId = ?", (vault_id,))
            contexts_deleted = cursor.rowcount
            
            # 3. Delete memories
            cursor.execute("DELETE FROM Memories WHERE VaultId = ?", (vault_id,))
            memories_deleted = cursor.rowcount
            
            # 4. Optionally delete the vault itself
            vault_deleted = 0
            if delete_vault:
                cursor.execute("DELETE FROM Vaults WHERE VaultId = ?", (vault_id,))
                vault_deleted = cursor.rowcount
            
            conn.commit()
            
            return {
                'entries_deleted': entries_deleted,
                'contexts_deleted': contexts_deleted,
                'memories_deleted': memories_deleted,
                'vault_deleted': vault_deleted
            }


class PostgresVaultMemoryDeleter:
    """Postgres deleter for vault data using DSN connection string."""

    def __init__(self, pg_dsn: str):
        if not psycopg:
            raise RuntimeError("psycopg is required for Postgres operations. Install psycopg[binary] >=3.2")
        if not pg_dsn:
            raise ValueError("Postgres DSN is required")
        self.pg_dsn = pg_dsn

    def get_vault_info(self, vault_id: str) -> Optional[Dict[str, Any]]:
        with psycopg.connect(self.pg_dsn) as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "SELECT user_id, title, description, creation_time FROM vaults WHERE vault_id = %s",
                    (vault_id,)
                )
                row = cur.fetchone()
                if not row:
                    return None
                user_id, title, description, creation_time = row

                cur.execute("SELECT COUNT(*) FROM memories WHERE vault_id = %s", (vault_id,))
                memory_count = cur.fetchone()[0]
                cur.execute("SELECT COUNT(*) FROM memory_entries WHERE vault_id = %s", (vault_id,))
                entry_count = cur.fetchone()[0]
                cur.execute("SELECT COUNT(*) FROM memory_contexts WHERE vault_id = %s", (vault_id,))
                context_count = cur.fetchone()[0]

                return {
                    'vault': {
                        'UserId': user_id,
                        'Title': title,
                        'Description': description,
                        'CreationTime': creation_time,
                    },
                    'memory_count': memory_count,
                    'entry_count': entry_count,
                    'context_count': context_count,
                }

    def get_memories_list(self, vault_id: str) -> List[Tuple[str, str, Optional[str]]]:
        with psycopg.connect(self.pg_dsn) as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "SELECT title, memory_type, description FROM memories WHERE vault_id = %s ORDER BY title",
                    (vault_id,)
                )
                return list(cur.fetchall())

    def delete_vault_memories(self, vault_id: str, delete_vault: bool = False) -> Dict[str, int]:
        with psycopg.connect(self.pg_dsn) as conn:
            with conn.cursor() as cur:
                # 1. Delete entries
                cur.execute("DELETE FROM memory_entries WHERE vault_id = %s", (vault_id,))
                entries_deleted = cur.rowcount or 0
                # 2. Delete contexts
                cur.execute("DELETE FROM memory_contexts WHERE vault_id = %s", (vault_id,))
                contexts_deleted = cur.rowcount or 0
                # 3. Delete memories
                cur.execute("DELETE FROM memories WHERE vault_id = %s", (vault_id,))
                memories_deleted = cur.rowcount or 0
                # 4. Optionally delete vault
                vault_deleted = 0
                if delete_vault:
                    cur.execute("DELETE FROM vaults WHERE vault_id = %s", (vault_id,))
                    vault_deleted = cur.rowcount or 0
            conn.commit()
        return {
            'entries_deleted': entries_deleted,
            'contexts_deleted': contexts_deleted,
            'memories_deleted': memories_deleted,
            'vault_deleted': vault_deleted,
        }


def print_vault_info(vault_info: Dict[str, Any]) -> None:
    """Print formatted vault information."""
    vault = vault_info['vault']
    print(f"\n📁 Vault Information:")
    print(f"   ID: {vault_info.get('vault_id', 'N/A')}")
    print(f"   Title: {vault['Title']}")
    print(f"   Description: {vault['Description'] or 'No description'}")
    print(f"   Created: {vault['CreationTime']}")
    print(f"   User ID: {vault['UserId']}")
    
    print(f"\n📊 Contents to be deleted:")
    print(f"   • {vault_info['memory_count']:,} memories")
    print(f"   • {vault_info['entry_count']:,} memory entries")
    print(f"   • {vault_info['context_count']:,} memory contexts")


def print_memories_list(memories: List[Tuple[str, str, str]]) -> None:
    """Print list of memories that will be deleted."""
    if not memories:
        print("\n   No memories found in this vault.")
        return
    
    print(f"\n📝 Memories that will be deleted:")
    for title, memory_type, description in memories:
        print(f"   • {title} ({memory_type})")
        if description:
            print(f"     └─ {description[:80]}{'...' if len(description) > 80 else ''}")


def confirm_deletion(vault_info: Dict[str, Any], delete_vault: bool) -> bool:
    """Ask user for confirmation before deletion."""
    total_items = (vault_info['memory_count'] + 
                  vault_info['entry_count'] + 
                  vault_info['context_count'])
    
    if delete_vault:
        total_items += 1
    
    print(f"\n⚠️  WARNING: This will permanently delete {total_items} items!")
    
    if delete_vault:
        print("   This includes the vault itself - it will be completely removed.")
    else:
        print("   The vault will remain but will be empty.")
    
    response = input("\nType 'DELETE' to confirm (anything else cancels): ").strip()
    return response == 'DELETE'


def main():
    parser = argparse.ArgumentParser(
        description="Delete all memories within a vault by vault ID",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__
    )
    
    parser.add_argument(
        'vault_id',
        help='UUID of the vault to delete memories from'
    )
    
    parser.add_argument(
        '--db-path',
        default='',
        help='(Deprecated) Path to legacy SQLite database file if still present'
    )

    parser.add_argument(
        '--pg-dsn',
        default=os.getenv('MEMORY_BACKEND_POSTGRES_DSN', ''),
        help='Postgres DSN (e.g., postgres://user:pass@host:5432/db?sslmode=disable). Defaults to MEMORY_BACKEND_POSTGRES_DSN'
    )
    
    parser.add_argument(
        '--delete-vault',
        action='store_true',
        help='Also delete the vault itself (not just its memories)'
    )
    
    parser.add_argument(
        '--yes',
        action='store_true',
        help='Skip confirmation prompt (use with caution!)'
    )
    
    args = parser.parse_args()
    
    try:
        # Initialize deleter (prefer Postgres)
        deleter_obj = None
        if args.pg_dsn:
            deleter_obj = PostgresVaultMemoryDeleter(args.pg_dsn)
        elif args.db_path:
            deleter_obj = VaultMemoryDeleter(args.db_path)
        else:
            raise ValueError("Provide --pg-dsn (recommended) or --db-path (legacy SQLite)")
        
        # Get vault info
        vault_info = deleter_obj.get_vault_info(args.vault_id)
        if not vault_info:
            print(f"❌ Vault not found: {args.vault_id}")
            sys.exit(1)
        
        # Add vault_id to info for display
        vault_info['vault_id'] = args.vault_id
        
        # Show what will be deleted
        print_vault_info(vault_info)
        
        # Get and show memories list
        memories = deleter_obj.get_memories_list(args.vault_id)
        print_memories_list(memories)
        
        # Check if there's anything to delete
        total_items = (vault_info['memory_count'] + 
                      vault_info['entry_count'] + 
                      vault_info['context_count'])
        
        if total_items == 0 and not args.delete_vault:
            print("\n✅ Vault is already empty - nothing to delete.")
            sys.exit(0)
        
        # Confirm deletion
        if not args.yes:
            if not confirm_deletion(vault_info, args.delete_vault):
                print("\n❌ Deletion cancelled.")
                sys.exit(0)
        
        # Perform deletion
        print(f"\n🗑️  Deleting...")
        results = deleter_obj.delete_vault_memories(args.vault_id, args.delete_vault)
        
        # Show results
        print(f"\n✅ Deletion completed:")
        print(f"   • {results['entries_deleted']:,} memory entries deleted")
        print(f"   • {results['contexts_deleted']:,} memory contexts deleted")
        print(f"   • {results['memories_deleted']:,} memories deleted")
        
        if args.delete_vault:
            if results['vault_deleted'] > 0:
                print(f"   • Vault deleted")
            else:
                print(f"   • ⚠️  Vault was not found (may have been already deleted)")
        
        total_deleted = sum(results.values())
        print(f"\n🎯 Total items deleted: {total_deleted:,}")
        
    except Exception as e:
        print(f"❌ Error: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()