# Database Migrations

This directory contains database migration files for the backend services.

## Migration Files

Migration files follow the naming convention: `{version}_{description}.{direction}.sql`

- `version`: 6-digit zero-padded number (e.g., 000001, 000002)
- `description`: Brief description of what the migration does
- `direction`: Either `up` or `down`

## Available Migrations

- `000001_init_schema.up.sql` / `000001_init_schema.down.sql`: Initial database schema
- `000002_add_indexes.up.sql` / `000002_add_indexes.down.sql`: Performance indexes
- `000003_long_text_support.up.sql` / `000003_long_text_support.down.sql`: Long text RAG support

## Usage

### Build Migration Tool

```bash
go build -o migrate ./cmd/migrate
```

### Run Migrations

```bash
# Run all pending migrations
./migrate -action=up

# Rollback last migration
./migrate -action=down

# Check current version
./migrate -action=version

# Check migration status
./migrate -action=status

# Migrate to specific version
./migrate -action=goto -version=2
```

### Environment Variables

Set the following environment variables:

- `DATABASE_URL`: PostgreSQL connection string
- `CONFIG_FILE`: Path to config file (optional)

## Creating New Migrations

### Manual Creation

1. Create up migration file: `00000X_description.up.sql`
2. Create down migration file: `00000X_description.down.sql`

### Using migrate CLI (if installed)

```bash
migrate create -ext sql -dir migrations -seq add_new_feature
```

## Migration Best Practices

1. **Always test migrations** on a copy of production data
2. **Make migrations idempotent** - they should be safe to run multiple times
3. **Include both up and down migrations** for reversibility
4. **Use transactions** for complex migrations when possible
5. **Test rollback** functionality before deploying to production

## Troubleshooting

### Dirty Database State

If migrations are in a "dirty" state (partially applied):

```bash
# Check current state
./migrate -action=status

# Force version (use with caution!)
./migrate -action=force -version=X
```

### Migration Conflicts

If you encounter migration conflicts:

1. Check the migration files for syntax errors
2. Ensure column/table names are correct
3. Verify foreign key relationships
4. Test on a development database first

## Integration with Application

The migration system is designed to be separate from application startup. Migrations should be run manually or as part of your deployment pipeline, not automatically during application startup.

This approach provides better control and prevents accidental schema changes in production environments.
