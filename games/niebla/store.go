package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"golib"
)

// The local database. It is the schema a later server keeps, on one
// player's machine for now: players by identity, each with its saves,
// and a token column left empty until a server fills it. The whole
// colony is one JSON value in the saves table, the way the server would
// hold one authoritative region per player.

// db is the game's local database, nil when saving is off. resolvePlayer
// opens it before the first scene; the scenes reach it through saveBase
// and resumeState.
var db *store

// saveWarning says why saving is off, when it is, for the menu to show.
var saveWarning string

// regionSlot names the one save slot so far: the region as the player
// left it.
const regionSlot = "region"

// store is the local database.
type store struct {
	sql *sql.DB
}

// storePath returns where the database file goes: the player's settings
// folder, in the same place a GoLib game keeps its saves, so a debug
// build and a dist one share the base. Under golib shot it returns
// ":memory:", and nothing is ever written.
func storePath(getenv func(string) string, userDir func() (string, error)) (string, error) {
	if getenv("GOLIB_SHOT_DIR") != "" {
		return ":memory:", nil
	}
	settings, err := userDir()
	if err != nil {
		return "", fmt.Errorf("can't find the settings folder: %w", err)
	}
	if settings == "" {
		return "", errors.New("can't find the settings folder")
	}
	return filepath.Join(settings, "GoLib games", "niebla", "niebla.db"), nil
}

// openStore opens the database — making file and tables when they are
// missing. One connection at a time: the game writes alone, and a
// ":memory:" database lives only while its one connection does.
func openStore(path string) (*store, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("making the save folder: %w", err)
		}
	}
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	sqlDB.SetMaxOpenConns(1)
	st := &store{sql: sqlDB}
	if err := st.configure(); err != nil {
		st.close()
		return nil, err
	}
	if err := st.migrate(); err != nil {
		st.close()
		return nil, err
	}
	return st, nil
}

// configure sets the pragmas this game needs. The database has one
// connection, so they hold for every statement after it.
func (st *store) configure() error {
	for _, pragma := range []string{
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = 1",
	} {
		rows, err := st.sql.Query(pragma)
		if err != nil {
			return fmt.Errorf("setting up the database: %w", err)
		}
		rows.Close()
	}
	return nil
}

// migrate creates the tables the game needs, leaving any that exist.
// The columns beyond what the game uses today — the players' token, the
// saves' slot — are where the server grows into.
func (st *store) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS players (
	id         TEXT PRIMARY KEY, -- the player identity of identity.go; a server keys its players the same way
	source     TEXT NOT NULL,    -- 'machine' or 'random'
	created_at TEXT NOT NULL,    -- RFC 3339, UTC
	token      TEXT NOT NULL DEFAULT '' -- empty until a server hands one out
);
CREATE TABLE IF NOT EXISTS saves (
	player     TEXT NOT NULL REFERENCES players(id),
	slot       TEXT NOT NULL,
	state      TEXT NOT NULL, -- the whole State, as JSON
	ticks      INTEGER NOT NULL,
	updated_at TEXT NOT NULL, -- RFC 3339, UTC
	PRIMARY KEY (player, slot)
);
CREATE TABLE IF NOT EXISTS machine (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);`
	if _, err := st.sql.Exec(schema); err != nil {
		return fmt.Errorf("making the tables: %w", err)
	}
	return nil
}

// player returns this machine's identity, registering it the first
// time: the machine ID hashed, or a random one kept in the machine
// table for machines that won't say who they are.
func (st *store) player(machine string) (string, error) {
	id := playerIDOf(machine)
	source := "machine"
	if machine == "" {
		stored, err := st.machineValue("fallback player")
		if err != nil {
			return "", err
		}
		if stored != "" {
			return stored, nil
		}
		if id, err = randomID(); err != nil {
			return "", err
		}
		if err := st.setMachineValue("fallback player", id); err != nil {
			return "", err
		}
		source = "random"
	}
	_, err := st.sql.Exec(
		`INSERT INTO players (id, source, created_at) VALUES (?, ?, ?)
		 ON CONFLICT (id) DO NOTHING`,
		id, source, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return "", fmt.Errorf("registering the player: %w", err)
	}
	return id, nil
}

// saveState writes the whole colony over this player's save.
func (st *store) saveState(id string, s *State) error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("saving: state can't be written as JSON: %w", err)
	}
	_, err = st.sql.Exec(
		`INSERT INTO saves (player, slot, state, ticks, updated_at) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (player, slot) DO UPDATE SET
		   state = excluded.state, ticks = excluded.ticks, updated_at = excluded.updated_at`,
		id, regionSlot, string(data), s.Ticks, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("saving: %w", err)
	}
	return nil
}

// loadState reads this player's save, and says whether there was one.
func (st *store) loadState(id string) (*State, bool, error) {
	var data string
	err := st.sql.QueryRow(
		`SELECT state FROM saves WHERE player = ? AND slot = ?`, id, regionSlot,
	).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("loading: %w", err)
	}
	var s State
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		return nil, false, fmt.Errorf("loading: the save can't be read: %w", err)
	}
	return &s, true, nil
}

func (st *store) close() error {
	return st.sql.Close()
}

// machineValue reads one of the machine table's values, "" when the key
// isn't there.
func (st *store) machineValue(key string) (string, error) {
	var value string
	err := st.sql.QueryRow(`SELECT value FROM machine WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("reading %q from the database: %w", key, err)
	}
	return value, nil
}

func (st *store) setMachineValue(key, value string) error {
	_, err := st.sql.Exec(
		`INSERT INTO machine (key, value) VALUES (?, ?)
		 ON CONFLICT (key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	if err != nil {
		return fmt.Errorf("keeping %q in the database: %w", key, err)
	}
	return nil
}

// randomID makes an identity from nowhere, for machines that have no ID
// of their own. As long as a hashed machine ID, so everything that
// keys or shows identities treats both the same.
func randomID() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("making a player identity: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// saveBase writes the colony to the local database, when saving is on.
func saveBase(s *State) error {
	if db == nil {
		return errors.New("saving is off")
	}
	return db.saveState(player, s)
}

// resumeState returns the state the player comes back to: the one a
// golib shot --save seeded, when there is one — so shots can start deep
// in a game —, else the last one saved for this player. The second
// result says whether any was found.
func resumeState() (*State, bool, error) {
	var seeded State
	if found, err := golib.LoadData("state", &seeded); err != nil || found {
		return &seeded, found, err
	}
	if db == nil {
		return nil, false, nil
	}
	return db.loadState(player)
}
