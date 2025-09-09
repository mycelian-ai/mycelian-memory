#!/usr/bin/env python3
"""
Delete all memories and related data within a vault (by ID or title).

Usage:
    python delete_vault_memories.py <vault_id_or_title> \
        [--by-title] [--actor-id ACTOR] \
        [--pg-dsn "postgres://user:pass@host:5432/db?sslmode=disable"] \
        [--delete-vault] [--yes]

    # or rely on environment (MEMORY_SERVER_POSTGRES_DSN preferred; fallback MEMORY_BACKEND_POSTGRES_DSN)
    MEMORY_SERVER_POSTGRES_DSN=postgres://... python delete_vault_memories.py <vault_id_or_title> [--by-title] [--delete-vault] [--yes]

Examples:
    # Preview what will be deleted (safe)
    python delete_vault_memories.py 97db1a27-695b-4bf3-bbd1-a00c6d4501de --pg-dsn postgres://...

    # Delete by title for current dev env (11544 host port in local/dev)
    python delete_vault_memories.py github-hackathon --by-title --pg-dsn "postgres://memory:memory@localhost:11544/memory?sslmode=disable" --yes

    # Delete everything including the vault itself
    python delete_vault_memories.py 97db1a27-695b-4bf3-bbd1-a00c6d4501de --pg-dsn postgres://... --delete-vault --yes
"""

import argparse
import os
import sys
from typing import Dict, Any, List, Tuple, Optional

try:
    import psycopg
except Exception:
    psycopg = None


class VaultMemoryDeleter:
    """Postgres deleter for vault data using DSN connection string."""

    def __init__(self, pg_dsn: str):
        if not psycopg:
            raise RuntimeError("psycopg is required for Postgres operations. Install psycopg[binary] >=3.2")
        if not pg_dsn:
            raise ValueError("Postgres DSN is required")
        self.pg_dsn = pg_dsn

    def get_vault_info(self, actor_id: str, vault_id: str) -> Optional[Dict[str, Any]]:
        """Return vault info and aggregate counts, filtered by actor_id + vault_id for safety."""
        with psycopg.connect(self.pg_dsn) as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "SELECT title, description, creation_time FROM vaults WHERE actor_id = %s AND vault_id = %s",
                    (actor_id, vault_id),
                )
                row = cur.fetchone()
                if not row:
                    return None
                title, description, creation_time = row

                cur.execute("SELECT COUNT(*) FROM memories WHERE actor_id = %s AND vault_id = %s", (actor_id, vault_id))
                memory_count = cur.fetchone()[0]
                cur.execute("SELECT COUNT(*) FROM memory_entries WHERE actor_id = %s AND vault_id = %s", (actor_id, vault_id))
                entry_count = cur.fetchone()[0]
                cur.execute("SELECT COUNT(*) FROM memory_contexts WHERE actor_id = %s AND vault_id = %s", (actor_id, vault_id))
                context_count = cur.fetchone()[0]

                return {
                    'vault': {
                        'ActorId': actor_id,
                        'VaultId': vault_id,
                        'Title': title,
                        'Description': description,
                        'CreationTime': creation_time,
                    },
                    'memory_count': memory_count,
                    'entry_count': entry_count,
                    'context_count': context_count,
                }

    def get_memories_list(self, actor_id: str, vault_id: str) -> List[Tuple[str, str, Optional[str]]]:
        with psycopg.connect(self.pg_dsn) as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "SELECT title, memory_type, description FROM memories WHERE actor_id = %s AND vault_id = %s ORDER BY title",
                    (actor_id, vault_id),
                )
                return list(cur.fetchall())

    def delete_vault_memories(self, actor_id: str, vault_id: str, delete_vault: bool = False) -> Dict[str, int]:
        """Delete in dependency order, scoped by (actor_id, vault_id). Also cleans up outbox rows that reference the vault."""
        with psycopg.connect(self.pg_dsn) as conn:
            with conn.cursor() as cur:
                # 0. Opportunistically clean outbox records whose payload references this vault
                try:
                    cur.execute("DELETE FROM outbox WHERE payload::text ILIKE %s", (f'%"vaultId":"{vault_id}"%',))
                except Exception:
                    # outbox table may not exist in older schemas; ignore errors
                    pass

                # 1. Delete entries
                cur.execute("DELETE FROM memory_entries WHERE actor_id = %s AND vault_id = %s", (actor_id, vault_id))
                entries_deleted = cur.rowcount or 0
                # 2. Delete contexts
                cur.execute("DELETE FROM memory_contexts WHERE actor_id = %s AND vault_id = %s", (actor_id, vault_id))
                contexts_deleted = cur.rowcount or 0
                # 3. Delete memories
                cur.execute("DELETE FROM memories WHERE actor_id = %s AND vault_id = %s", (actor_id, vault_id))
                memories_deleted = cur.rowcount or 0
                # 4. Optionally delete vault
                vault_deleted = 0
                if delete_vault:
                    cur.execute("DELETE FROM vaults WHERE actor_id = %s AND vault_id = %s", (actor_id, vault_id))
                    vault_deleted = cur.rowcount or 0
            conn.commit()
        return {
            'entries_deleted': entries_deleted,
            'contexts_deleted': contexts_deleted,
            'memories_deleted': memories_deleted,
            'vault_deleted': vault_deleted,
        }

    def resolve_vault(self, vault_id_or_title: str, by_title: bool, actor_id_hint: Optional[str]) -> Optional[Tuple[str, str, str]]:
        """Resolve to (actor_id, vault_id, title). If multiple matches by title and no actor hint, return None after printing matches."""
        with psycopg.connect(self.pg_dsn) as conn:
            with conn.cursor() as cur:
                if not by_title:
                    # Resolve by vault_id
                    cur.execute("SELECT actor_id, title FROM vaults WHERE vault_id = %s", (vault_id_or_title,))
                    row = cur.fetchone()
                    if not row:
                        return None
                    actor_id, title = row
                    return actor_id, vault_id_or_title, title
                # Resolve by title
                if actor_id_hint:
                    cur.execute("SELECT actor_id, vault_id, title FROM vaults WHERE actor_id = %s AND title = %s", (actor_id_hint, vault_id_or_title))
                    row = cur.fetchone()
                    if not row:
                        return None
                    a, v, t = row
                    return a, v, t
                # No actor hint: check for uniqueness
                cur.execute("SELECT actor_id, vault_id, title FROM vaults WHERE title = %s ORDER BY creation_time DESC", (vault_id_or_title,))
                rows = cur.fetchall()
                if not rows:
                    return None
                if len(rows) == 1:
                    a, v, t = rows[0]
                    return a, v, t
                # Ambiguous
                print("Multiple vaults found with this title; please re-run with --actor-id to disambiguate:\n")
                for a, v, t in rows:
                    print(f"  actor_id={a} vault_id={v} title={t}")
                return None


def print_vault_info(vault_info: Dict[str, Any]) -> None:
    """Print formatted vault information."""
    vault = vault_info['vault']
    print(f"\n📁 Vault Information:")
    print(f"   Actor: {vault['ActorId']}")
    print(f"   ID: {vault['VaultId']}")
    print(f"   Title: {vault['Title']}")
    print(f"   Description: {vault['Description'] or 'No description'}")
    print(f"   Created: {vault['CreationTime']}")

    print(f"\n📊 Contents to be deleted:")
    print(f"   • {vault_info['memory_count']:,} memories")
    print(f"   • {vault_info['entry_count']:,} memory entries")
    print(f"   • {vault_info['context_count']:,} memory contexts")


def print_memories_list(memories: List[Tuple[str, str, Optional[str]]]) -> None:
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
        'vault',
        help='Vault ID (UUID) or title (use --by-title to resolve by title)'
    )



    parser.add_argument(
        '--pg-dsn',
        default=(
            os.getenv('MEMORY_SERVER_POSTGRES_DSN', '')
            or os.getenv('MEMORY_BACKEND_POSTGRES_DSN', '')
        ),
        help='Postgres DSN (e.g., postgres://user:pass@host:5432/db?sslmode=disable). Defaults to MEMORY_SERVER_POSTGRES_DSN, fallback MEMORY_BACKEND_POSTGRES_DSN'
    )

    parser.add_argument(
        '--by-title',
        action='store_true',
        help='Treat the positional vault argument as a vault title (requires unique title or --actor-id)'
    )

    parser.add_argument(
        '--actor-id',
        default=os.getenv('MEMORY_ACTOR_ID', ''),
        help='Actor ID to scope the vault lookup when using --by-title'
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
        # Initialize Postgres deleter
        pg_dsn = args.pg_dsn or os.getenv('MEMORY_SERVER_POSTGRES_DSN') or os.getenv('MEMORY_BACKEND_POSTGRES_DSN') or ''
        if not pg_dsn:
            # Provide a sensible local/dev fallback to avoid surprises during quick operations
            pg_dsn = 'postgres://memory:memory@localhost:11544/memory?sslmode=disable'
        deleter_obj = VaultMemoryDeleter(pg_dsn)

        # Resolve (actor_id, vault_id, title)
        resolved = deleter_obj.resolve_vault(args.vault, args.by_title, args.actor_id or None)
        if not resolved:
            msg = f"Vault not found or ambiguous: {args.vault}"
            if args.by_title and not args.actor_id:
                msg += " (tip: provide --actor-id)"
            print(f"❌ {msg}")
            sys.exit(1)
        actor_id, vault_id, title = resolved

        # Get vault info
        vault_info = deleter_obj.get_vault_info(actor_id, vault_id)
        if not vault_info:
            print(f"❌ Vault not found: {args.vault}")
            sys.exit(1)

        # Show what will be deleted
        print_vault_info(vault_info)

        # Get and show memories list
        memories = deleter_obj.get_memories_list(actor_id, vault_id)
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
        results = deleter_obj.delete_vault_memories(actor_id, vault_id, args.delete_vault)

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
