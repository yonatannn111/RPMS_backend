package database

import (
	"context"
	"fmt"
	"log"

	"rpms-backend/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Pool *pgxpool.Pool
}

func NewConnection(cfg *config.Config) (*Database, error) {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.GetDatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	log.Println("Successfully connected to database")
	return &Database{Pool: pool}, nil
}

func (db *Database) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

func (db *Database) GetDB() *pgxpool.Pool {
	return db.Pool
}

func RunMigrations(db *Database) error {
	ctx := context.Background()

	// Create users table
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		name VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL CHECK (role IN ('author', 'editor', 'admin', 'coordinator')),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`

	// Create papers table
	createPapersTable := `
	CREATE TABLE IF NOT EXISTS papers (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		title VARCHAR(500) NOT NULL,
		abstract TEXT,
		content TEXT,
		author_id UUID REFERENCES users(id) ON DELETE CASCADE,
		status VARCHAR(50) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'under_review', 'approved', 'rejected', 'published')),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`

	// Create reviews table
	createReviewsTable := `
	CREATE TABLE IF NOT EXISTS reviews (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		paper_id UUID REFERENCES papers(id) ON DELETE CASCADE,
		reviewer_id UUID REFERENCES users(id) ON DELETE CASCADE,
		rating INTEGER CHECK (rating >= 1 AND rating <= 5),
		comments TEXT,
		recommendation VARCHAR(50) CHECK (recommendation IN ('accept', 'minor_revision', 'major_revision', 'reject')),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		UNIQUE(paper_id, reviewer_id)
	);`

	// Create events table
	createEventsTable := `
	CREATE TABLE IF NOT EXISTS events (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		title VARCHAR(500) NOT NULL,
		description TEXT,
		date TIMESTAMP WITH TIME ZONE NOT NULL,
		location VARCHAR(255),
		coordinator_id UUID REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`

	// Create messages table
	createMessagesTable := `
	CREATE TABLE IF NOT EXISTS messages (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sender_id UUID REFERENCES users(id) ON DELETE CASCADE,
		receiver_id UUID REFERENCES users(id) ON DELETE CASCADE,
		content TEXT NOT NULL,
		attachment_url TEXT,
		attachment_name TEXT,
		attachment_type TEXT,
		attachment_size INTEGER,
		reply_to_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
		is_forwarded BOOLEAN DEFAULT FALSE,
		is_read BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`

	// Create indexes
	createIndexes := `
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_papers_author_id ON papers(author_id);
	CREATE INDEX IF NOT EXISTS idx_papers_status ON papers(status);
	CREATE INDEX IF NOT EXISTS idx_reviews_paper_id ON reviews(paper_id);
	CREATE INDEX IF NOT EXISTS idx_reviews_reviewer_id ON reviews(reviewer_id);
	CREATE INDEX IF NOT EXISTS idx_events_date ON events(date);
	CREATE INDEX IF NOT EXISTS idx_events_coordinator_id ON events(coordinator_id);
	CREATE INDEX IF NOT EXISTS idx_messages_sender_receiver ON messages(sender_id, receiver_id);
	CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);`

	// Add new columns to users table if they don't exist
	addAvatarColumn := `ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar VARCHAR(255) DEFAULT '';`
	addBioColumn := `ALTER TABLE users ADD COLUMN IF NOT EXISTS bio TEXT DEFAULT '';`
	addPreferencesColumn := `ALTER TABLE users ADD COLUMN IF NOT EXISTS preferences JSONB DEFAULT '{}';`

	// Alter events table date column to TIMESTAMP if it exists as DATE
	alterEventsDateColumn := `
		DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'events' AND column_name = 'date' AND data_type = 'date'
			) THEN
				ALTER TABLE events ALTER COLUMN date TYPE TIMESTAMP WITH TIME ZONE USING date::TIMESTAMP WITH TIME ZONE;
			END IF;
		END $$;
	`

	// Add new columns to messages table for attachments and replies
	addMessageAttachmentColumns := `
		ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachment_url TEXT;
		ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachment_name TEXT;
		ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachment_type TEXT;
		ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachment_size INTEGER;
		ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_message_id UUID REFERENCES messages(id) ON DELETE SET NULL;
		ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_forwarded BOOLEAN DEFAULT FALSE;
	`

	migrations := []string{
		createUsersTable,
		createPapersTable,
		createReviewsTable,
		createEventsTable,
		createMessagesTable,
		createIndexes,
		addAvatarColumn,
		addBioColumn,
		addPreferencesColumn,
		alterEventsDateColumn,
		addMessageAttachmentColumns,
	}

	for _, migration := range migrations {
		if _, err := db.Pool.Exec(ctx, migration); err != nil {
			return fmt.Errorf("failed to run migration: %w", err)
		}
	}

	log.Println("Database migrations completed successfully")
	return nil
}

func (db *Database) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return db.Pool.Begin(ctx)
}

func (db *Database) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.Pool.QueryRow(ctx, sql, args...)
}

func (db *Database) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return db.Pool.Query(ctx, sql, args...)
}

func (db *Database) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.Pool.Exec(ctx, sql, args...)
}
