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
		status VARCHAR(50) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'under_review', 'approved', 'rejected', 'published', 'recommended_for_publication')),
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

	// Create notifications table
	createNotificationsTable := `
	CREATE TABLE IF NOT EXISTS notifications (
		id SERIAL PRIMARY KEY,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		message TEXT NOT NULL,
		paper_id UUID REFERENCES papers(id) ON DELETE CASCADE,
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

	// Add file_url to papers
	addFileUrlToPapers := `ALTER TABLE papers ADD COLUMN IF NOT EXISTS file_url TEXT;`

	// Add paper_id to notifications
	addPaperIdToNotifications := `ALTER TABLE notifications ADD COLUMN IF NOT EXISTS paper_id UUID REFERENCES papers(id) ON DELETE CASCADE;`

	// Add type to papers
	addTypeToPapers := `ALTER TABLE papers ADD COLUMN IF NOT EXISTS type VARCHAR(50) DEFAULT 'research';`

	// Add verification columns to users
	addVerificationColumns := `
		ALTER TABLE users ADD COLUMN IF NOT EXISTS is_verified BOOLEAN DEFAULT FALSE;
		ALTER TABLE users ADD COLUMN IF NOT EXISTS verification_code VARCHAR(6);
	`

	// Add category to events
	addCategoryToEvents := `ALTER TABLE events ADD COLUMN IF NOT EXISTS category VARCHAR(100) DEFAULT 'Other';`

	// Add status to events
	addStatusToEvents := `ALTER TABLE events ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'draft';`

	// Add Author Profile columns to users
	addAuthorProfileColumns := `
		ALTER TABLE users ADD COLUMN IF NOT EXISTS academic_year VARCHAR(50);
		ALTER TABLE users ADD COLUMN IF NOT EXISTS author_type VARCHAR(50);
		ALTER TABLE users ADD COLUMN IF NOT EXISTS author_category VARCHAR(50);
		ALTER TABLE users ADD COLUMN IF NOT EXISTS academic_rank VARCHAR(50);
		ALTER TABLE users ADD COLUMN IF NOT EXISTS qualification VARCHAR(50);
		ALTER TABLE users ADD COLUMN IF NOT EXISTS employment_type VARCHAR(50);
		ALTER TABLE users ADD COLUMN IF NOT EXISTS gender VARCHAR(20);
	`

	createNewsTable := `
	CREATE TABLE IF NOT EXISTS news (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		title VARCHAR(500) NOT NULL,
		summary TEXT NOT NULL,
		content TEXT NOT NULL,
		category VARCHAR(100) NOT NULL,
		status VARCHAR(50) DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
		editor_id UUID REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`

	// Add Editor Submission columns to papers
	addEditorSubmissionColumns := `
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS institution_code VARCHAR(50);
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS publication_id VARCHAR(50);
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS publication_isced_band VARCHAR(50);
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS publication_title_amharic TEXT;
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS publication_date TIMESTAMP WITH TIME ZONE;
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS publication_type VARCHAR(50);
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS journal_type VARCHAR(50);
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS journal_name VARCHAR(255);
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS indigenous_knowledge BOOLEAN DEFAULT FALSE;
	`

	// Add Research Project columns to papers
	addResearchProjectColumns := `
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS fiscal_year VARCHAR(50);
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS allocated_budget NUMERIC DEFAULT 0;
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS external_budget NUMERIC DEFAULT 0;
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS nrf_fund NUMERIC DEFAULT 0;
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS research_type VARCHAR(100);
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS completion_status VARCHAR(50);
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS female_researchers INTEGER DEFAULT 0;
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS male_researchers INTEGER DEFAULT 0;
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS outside_female_researchers INTEGER DEFAULT 0;
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS outside_male_researchers INTEGER DEFAULT 0;
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS benefited_industry VARCHAR(255);
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS ethical_clearance VARCHAR(50);
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS pi_name VARCHAR(255);
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS pi_gender VARCHAR(50);
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS co_investigators TEXT;
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS produced_prototype VARCHAR(50);
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS hetril_collaboration VARCHAR(50);
		ALTER TABLE papers ADD COLUMN IF NOT EXISTS submitted_to_incubator VARCHAR(50);
	`

	// Update paper status check constraint
	updatePaperStatusConstraint := `
		DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.constraint_column_usage WHERE table_name = 'papers' AND constraint_name = 'papers_status_check') THEN
				ALTER TABLE papers DROP CONSTRAINT papers_status_check;
			END IF;
			ALTER TABLE papers ADD CONSTRAINT papers_status_check CHECK (status IN ('draft', 'submitted', 'under_review', 'approved', 'rejected', 'published', 'recommended_for_publication'));
		END $$;
	`

	// Add date_of_birth to users
	addDateOfBirthToUsers := `
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'date_of_birth') THEN
				ALTER TABLE users ADD COLUMN date_of_birth TEXT DEFAULT '';
			END IF;
		END $$;
	`

	// Create likes table
	createLikesTable := `
	CREATE TABLE IF NOT EXISTS likes (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		post_type VARCHAR(10) NOT NULL CHECK (post_type IN ('news', 'event')),
		post_id UUID NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		UNIQUE(user_id, post_type, post_id)
	);`

	// Create comments table
	createCommentsTable := `
	CREATE TABLE IF NOT EXISTS comments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		post_type VARCHAR(10) NOT NULL CHECK (post_type IN ('news', 'event')),
		post_id UUID NOT NULL,
		content TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`

	// Create shares table
	createSharesTable := `
	CREATE TABLE IF NOT EXISTS shares (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		post_type VARCHAR(10) NOT NULL CHECK (post_type IN ('news', 'event')),
		post_id UUID NOT NULL,
		message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`

	// Create indexes for interactions
	createInteractionIndexes := `
		CREATE INDEX IF NOT EXISTS idx_likes_post ON likes(post_type, post_id);
		CREATE INDEX IF NOT EXISTS idx_comments_post ON comments(post_type, post_id);
		CREATE INDEX IF NOT EXISTS idx_shares_post ON shares(post_type, post_id);
		CREATE INDEX IF NOT EXISTS idx_likes_user ON likes(user_id);
		CREATE INDEX IF NOT EXISTS idx_comments_user ON comments(user_id);
	`

	// Add detailed rating columns to reviews
	addReviewRatingColumns := `
		ALTER TABLE reviews ADD COLUMN IF NOT EXISTS problem_statement INTEGER DEFAULT 0;
		ALTER TABLE reviews ADD COLUMN IF NOT EXISTS literature_review INTEGER DEFAULT 0;
		ALTER TABLE reviews ADD COLUMN IF NOT EXISTS methodology INTEGER DEFAULT 0;
		ALTER TABLE reviews ADD COLUMN IF NOT EXISTS results INTEGER DEFAULT 0;
		ALTER TABLE reviews ADD COLUMN IF NOT EXISTS conclusion INTEGER DEFAULT 0;
		ALTER TABLE reviews ADD COLUMN IF NOT EXISTS originality INTEGER DEFAULT 0;
		ALTER TABLE reviews ADD COLUMN IF NOT EXISTS clarity_organization INTEGER DEFAULT 0;
		ALTER TABLE reviews ADD COLUMN IF NOT EXISTS contribution_knowledge INTEGER DEFAULT 0;
		ALTER TABLE reviews ADD COLUMN IF NOT EXISTS technical_quality INTEGER DEFAULT 0;
		
		-- Drop old rating constraint if it exists
		DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.constraint_column_usage WHERE table_name = 'reviews' AND constraint_name = 'reviews_rating_check') THEN
				ALTER TABLE reviews DROP CONSTRAINT reviews_rating_check;
			END IF;
		END $$;
	`

	// Add image and video columns to news
	addMediaToNews := `
		ALTER TABLE news ADD COLUMN IF NOT EXISTS image_url TEXT;
		ALTER TABLE news ADD COLUMN IF NOT EXISTS video_url TEXT;
	`

	// Add image and video columns to events
	addMediaToEvents := `
		ALTER TABLE events ADD COLUMN IF NOT EXISTS image_url TEXT;
		ALTER TABLE events ADD COLUMN IF NOT EXISTS video_url TEXT;
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
		createNotificationsTable,
		addFileUrlToPapers,
		addPaperIdToNotifications,
		addTypeToPapers,
		addVerificationColumns,
		addVerificationColumns,
		addCategoryToEvents,
		addStatusToEvents,
		addAuthorProfileColumns,
		createNewsTable,
		addEditorSubmissionColumns,
		addResearchProjectColumns,
		updatePaperStatusConstraint,
		addDateOfBirthToUsers,
		createLikesTable,
		createCommentsTable,
		createSharesTable,
		createInteractionIndexes,
		addReviewRatingColumns,
		addMediaToNews,
		addMediaToEvents,
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
