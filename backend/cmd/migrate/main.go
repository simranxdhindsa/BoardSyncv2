package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"asana-youtrack-sync/database"
)

// This tool migrates data from the JSON file-based database to PostgreSQL

func main() {
	log.Println("=" + string("=")[0:70] + "=")
	log.Println("  BoardSync Data Migration Tool")
	log.Println("  JSON Files → PostgreSQL")
	log.Println("=" + string("=")[0:70] + "=")
	log.Println()

	// Check if DATABASE_URL is set
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("❌ DATABASE_URL environment variable not set!")
	}

	// Get data directory path
	dataDir := "./sync_app.db_data"
	if len(os.Args) > 1 {
		dataDir = os.Args[1]
	}

	log.Printf("📂 Data directory: %s\n", dataDir)
	log.Printf("🔗 Database: %s...\n", dbURL[:30]+"...")
	log.Println()

	// Check if data directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		log.Fatal("❌ Data directory does not exist:", dataDir)
	}

	// Initialize PostgreSQL
	log.Println("📡 Connecting to PostgreSQL...")
	pgDB, err := database.InitPostgres()
	if err != nil {
		log.Fatal("❌ Failed to connect to PostgreSQL:", err)
	}
	defer pgDB.Close()

	log.Println("✅ Connected to PostgreSQL")
	log.Println()

	// Start migration
	log.Println("🚀 Starting migration...")
	log.Println()

	// Migrate users
	if err := migrateUsers(dataDir, pgDB); err != nil {
		log.Printf("⚠️  Warning: Failed to migrate users: %v\n", err)
	}

	// Migrate settings
	if err := migrateSettings(dataDir, pgDB); err != nil {
		log.Printf("⚠️  Warning: Failed to migrate settings: %v\n", err)
	}

	// Migrate ticket mappings
	if err := migrateTicketMappings(dataDir, pgDB); err != nil {
		log.Printf("⚠️  Warning: Failed to migrate ticket mappings: %v\n", err)
	}

	// Migrate ignored tickets
	if err := migrateIgnoredTickets(dataDir, pgDB); err != nil {
		log.Printf("⚠️  Warning: Failed to migrate ignored tickets: %v\n", err)
	}

	// Migrate sync operations
	if err := migrateSyncOperations(dataDir, pgDB); err != nil {
		log.Printf("⚠️  Warning: Failed to migrate sync operations: %v\n", err)
	}

	// Migrate snapshots
	if err := migrateSnapshots(dataDir, pgDB); err != nil {
		log.Printf("⚠️  Warning: Failed to migrate snapshots: %v\n", err)
	}

	// Migrate audit logs
	if err := migrateAuditLogs(dataDir, pgDB); err != nil {
		log.Printf("⚠️  Warning: Failed to migrate audit logs: %v\n", err)
	}

	log.Println()
	log.Println("=" + string("=")[0:70] + "=")
	log.Println("✅ Migration completed!")
	log.Println("=" + string("=")[0:70] + "=")
	log.Println()
	log.Println("Next steps:")
	log.Println("1. Verify data in PostgreSQL")
	log.Println("2. Update backend to use PostgreSQL (set DATABASE_URL)")
	log.Println("3. Deploy updated backend to Render")
	log.Println("4. Backup old JSON files (keep them safe!)")
}

func migrateUsers(dataDir string, pgDB *database.PostgresDB) error {
	log.Println("👥 Migrating users...")

	filePath := filepath.Join(dataDir, "users.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("   ℹ️  No users.json found, skipping")
			return nil
		}
		return err
	}

	var users map[int]*database.User
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("failed to unmarshal users: %w", err)
	}

	count := 0
	for _, user := range users {
		_, err := pgDB.CreateUser(user.Username, user.Email, user.PasswordHash)
		if err != nil {
			log.Printf("   ⚠️  Failed to migrate user %s: %v\n", user.Email, err)
			continue
		}
		count++
	}

	log.Printf("   ✅ Migrated %d users\n", count)
	return nil
}

func migrateSettings(dataDir string, pgDB *database.PostgresDB) error {
	log.Println("⚙️  Migrating user settings...")

	filePath := filepath.Join(dataDir, "settings.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("   ℹ️  No settings.json found, skipping")
			return nil
		}
		return err
	}

	var settings map[int]*database.UserSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	count := 0
	for _, s := range settings {
		// Delete default settings first (created automatically with user)
		existingSettings, err := pgDB.GetUserSettings(s.UserID)
		if err == nil && existingSettings != nil {
			// Update existing settings instead
			if err := pgDB.UpdateUserSettings(s); err != nil {
				log.Printf("   ⚠️  Failed to update settings for user %d: %v\n", s.UserID, err)
				continue
			}
		}
		count++
	}

	log.Printf("   ✅ Migrated %d settings\n", count)
	return nil
}

func migrateTicketMappings(dataDir string, pgDB *database.PostgresDB) error {
	log.Println("🎫 Migrating ticket mappings...")

	filePath := filepath.Join(dataDir, "ticket_mappings.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("   ℹ️  No ticket_mappings.json found, skipping")
			return nil
		}
		return err
	}

	var mappings map[int]*database.TicketMapping
	if err := json.Unmarshal(data, &mappings); err != nil {
		return fmt.Errorf("failed to unmarshal mappings: %w", err)
	}

	count := 0
	for _, m := range mappings {
		_, err := pgDB.CreateTicketMapping(
			m.UserID,
			m.AsanaProjectID,
			m.AsanaTaskID,
			m.YouTrackProjectID,
			m.YouTrackIssueID,
		)
		if err != nil {
			log.Printf("   ⚠️  Failed to migrate mapping %d: %v\n", m.ID, err)
			continue
		}
		count++
	}

	log.Printf("   ✅ Migrated %d ticket mappings\n", count)
	return nil
}

func migrateIgnoredTickets(dataDir string, pgDB *database.PostgresDB) error {
	log.Println("🚫 Migrating ignored tickets...")

	filePath := filepath.Join(dataDir, "ignored_tickets.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("   ℹ️  No ignored_tickets.json found, skipping")
			return nil
		}
		return err
	}

	var ignored map[int]*database.IgnoredTicket
	if err := json.Unmarshal(data, &ignored); err != nil {
		return fmt.Errorf("failed to unmarshal ignored tickets: %w", err)
	}

	count := 0
	for _, i := range ignored {
		err := pgDB.AddIgnoredTicket(i.UserID, i.AsanaProjectID, i.TicketID, i.IgnoreType)
		if err != nil {
			log.Printf("   ⚠️  Failed to migrate ignored ticket %s: %v\n", i.TicketID, err)
			continue
		}
		count++
	}

	log.Printf("   ✅ Migrated %d ignored tickets\n", count)
	return nil
}

func migrateSyncOperations(dataDir string, pgDB *database.PostgresDB) error {
	log.Println("🔄 Migrating sync operations...")

	filePath := filepath.Join(dataDir, "sync_operations.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("   ℹ️  No sync_operations.json found, skipping")
			return nil
		}
		return err
	}

	var operations map[int]*database.SyncOperation
	if err := json.Unmarshal(data, &operations); err != nil {
		return fmt.Errorf("failed to unmarshal operations: %w", err)
	}

	count := 0
	for _, op := range operations {
		_, err := pgDB.CreateSyncOperation(op.UserID, op.OperationType, op.OperationData)
		if err != nil {
			log.Printf("   ⚠️  Failed to migrate operation %d: %v\n", op.ID, err)
			continue
		}
		count++
	}

	log.Printf("   ✅ Migrated %d sync operations\n", count)
	return nil
}

func migrateSnapshots(dataDir string, pgDB *database.PostgresDB) error {
	log.Println("📸 Migrating rollback snapshots...")

	filePath := filepath.Join(dataDir, "rollback_snapshots.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("   ℹ️  No rollback_snapshots.json found, skipping")
			return nil
		}
		return err
	}

	var snapshots map[int]*database.RollbackSnapshot
	if err := json.Unmarshal(data, &snapshots); err != nil {
		return fmt.Errorf("failed to unmarshal snapshots: %w", err)
	}

	count := 0
	for _, s := range snapshots {
		_, err := pgDB.CreateRollbackSnapshot(s.OperationID, s.UserID, s.SnapshotData)
		if err != nil {
			log.Printf("   ⚠️  Failed to migrate snapshot %d: %v\n", s.ID, err)
			continue
		}
		count++
	}

	log.Printf("   ✅ Migrated %d snapshots\n", count)
	return nil
}

func migrateAuditLogs(dataDir string, pgDB *database.PostgresDB) error {
	log.Println("📋 Migrating audit logs...")

	filePath := filepath.Join(dataDir, "audit_logs.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("   ℹ️  No audit_logs.json found, skipping")
			return nil
		}
		return err
	}

	var logs map[int]*database.AuditLogEntry
	if err := json.Unmarshal(data, &logs); err != nil {
		return fmt.Errorf("failed to unmarshal audit logs: %w", err)
	}

	count := 0
	for _, entry := range logs {
		err := pgDB.CreateAuditLog(entry)
		if err != nil {
			log.Printf("   ⚠️  Failed to migrate audit log %d: %v\n", entry.ID, err)
			continue
		}
		count++
	}

	log.Printf("   ✅ Migrated %d audit logs\n", count)
	return nil
}
