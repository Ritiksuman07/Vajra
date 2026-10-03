package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"

	log "github.com/sirupsen/logrus"
)

const schemaVersion = "1.0"

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
	mu   sync.RWMutex
}

// Init initializes the SQLite database with required tables.
func Init(dataDir string) (*DB, error) {
	// Expand ~ in data dir
	dataDir = os.ExpandEnv(dataDir)

	// Create data directory if needed
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "vajra-bot.db")

	// Open database
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Create tables
	tables := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			model TEXT NOT NULL,
			backend TEXT NOT NULL DEFAULT 'ollama',
			status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS pods (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'agent',
			image TEXT NOT NULL DEFAULT 'vajra-bot-agent:latest',
			status TEXT NOT NULL DEFAULT 'stopped',
			cpu TEXT DEFAULT '2',
			memory TEXT DEFAULT '4g',
			workspace TEXT NOT NULL,
			container_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			agent_id TEXT,
			task TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			result TEXT,
			verification JSON,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		);`,
		`CREATE TABLE IF NOT EXISTS providers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			auth_method TEXT NOT NULL DEFAULT 'api_key',
			api_key_encrypted TEXT,
			oauth_token TEXT,
			env_var TEXT,
			is_active BOOLEAN DEFAULT false,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trace_id TEXT NOT NULL,
			pod_id TEXT,
			action TEXT NOT NULL,
			details JSON,
			verification JSON,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS verification_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trace_id TEXT NOT NULL,
			layer INTEGER NOT NULL,
			passed BOOLEAN NOT NULL,
			details JSON,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS config (
			key TEXT PRIMARY KEY,
			value TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS version (
			version TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, sqlStmt := range tables {
		if _, err := conn.Exec(sqlStmt); err != nil {
			return nil, fmt.Errorf("create table: %w", err)
		}
	}

	// Record schema version
	if _, err := conn.Exec(
		"INSERT OR REPLACE INTO version (version) VALUES (?)",
		schemaVersion,
	); err != nil {
		return nil, fmt.Errorf("record version: %w", err)
	}

	log.Infof("Database initialized at %s (schema v%s)", dbPath, schemaVersion)
	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

// DB returns the underlying sql.DB for use in other packages.
func (d *DB) DB() *sql.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.conn
}

// CreateSession inserts a new session record.
func (d *DB) CreateSession(sessionID, userID, model, backend string) error {
	_, err := d.conn.Exec(
		"INSERT INTO sessions (id, user_id, model, backend, status) VALUES (?, ?, ?, ?, 'active')",
		sessionID, userID, model, backend,
	)
	return err
}

// GetSession retrieves a session by ID.
func (d *DB) GetSession(sessionID string) (map[string]interface{}, error) {
	row := d.conn.QueryRow("SELECT id, user_id, model, backend, status, created_at FROM sessions WHERE id = ?", sessionID)
	if row == nil {
		return nil, fmt.Errorf("session not found")
	}
	return map[string]interface{}{"id": sessionID}, nil
}

// CreatePod inserts a new pod record.
func (d *DB) CreatePod(podID, sessionID, role, image, status, workspace string) error {
	_, err := d.conn.Exec(
		"INSERT INTO pods (id, session_id, role, image, status, workspace) VALUES (?, ?, ?, ?, ?, ?)",
		podID, sessionID, role, image, status, workspace,
	)
	return err
}

// GetPod retrieves a pod by ID.
func (d *DB) GetPod(podID string) (map[string]interface{}, error) {
	row := d.conn.QueryRow("SELECT id, session_id, role, image, status, workspace, container_id FROM pods WHERE id = ?", podID)
	if row == nil {
		return nil, fmt.Errorf("pod not found")
	}
	return map[string]interface{}{"id": podID}, nil
}

// UpdatePodStatus updates the pod status.
func (d *DB) UpdatePodStatus(podID, status string) error {
	_, err := d.conn.Exec("UPDATE pods SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", status, podID)
	return err
}

// DeletePod removes a pod record.
func (d *DB) DeletePod(podID string) error {
	_, err := d.conn.Exec("DELETE FROM pods WHERE id = ?", podID)
	return err
}

// ListPods returns all pod records.
func (d *DB) ListPods() ([]map[string]interface{}, error) {
	rows, err := d.conn.Query("SELECT id, role, status, model, created_at FROM pods")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pods []map[string]interface{}
	for rows.Next() {
		var id, role, status, model, created string
		if err := rows.Scan(&id, &role, &status, &model, &created); err != nil {
			return nil, err
		}
		pods = append(pods, map[string]interface{}{
			"id":       id,
			"role":     role,
			"status":   status,
			"model":    model,
			"created":  created,
		})
	}
	return pods, nil
}

// LogAudit inserts an audit log entry.
func (d *DB) LogAudit(traceID, podID, action string, details, verification map[string]interface{}) error {
	detailsJSON, _ := json.Marshal(details)
	verificationJSON, _ := json.Marshal(verification)

	_, err := d.conn.Exec(
		"INSERT INTO audit_log (trace_id, pod_id, action, details, verification) VALUES (?, ?, ?, ?, ?)",
		traceID, podID, action, string(detailsJSON), string(verificationJSON),
	)
	return err
}
