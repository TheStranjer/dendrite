// Copyright 2018 New Vector Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sqlite3

import (
	"context"
	"database/sql"

	"github.com/matrix-org/dendrite/common"
	"github.com/matrix-org/dendrite/syncapi/storage/tables"
)

const backwardExtremitiesSchema = `
-- Stores output room events received from the roomserver.
CREATE TABLE IF NOT EXISTS syncapi_backward_extremities (
	-- The 'room_id' key for the event.
	room_id TEXT NOT NULL,
	-- The event ID for the last known event. This is the backwards extremity.
	event_id TEXT NOT NULL,
	-- The prev_events for the last known event. This is used to update extremities.
	prev_event_id TEXT NOT NULL,
	PRIMARY KEY(room_id, event_id, prev_event_id)
);
`

const insertBackwardExtremitySQL = "" +
	"INSERT INTO syncapi_backward_extremities (room_id, event_id, prev_event_id)" +
	" VALUES ($1, $2, $3)" +
	" ON CONFLICT (room_id, event_id, prev_event_id) DO NOTHING"

const selectBackwardExtremitiesForRoomSQL = "" +
	"SELECT DISTINCT event_id FROM syncapi_backward_extremities WHERE room_id = $1"

const deleteBackwardExtremitySQL = "" +
	"DELETE FROM syncapi_backward_extremities WHERE room_id = $1 AND prev_event_id = $2"

type backwardExtremitiesStatements struct {
	insertBackwardExtremityStmt          *sql.Stmt
	selectBackwardExtremitiesForRoomStmt *sql.Stmt
	deleteBackwardExtremityStmt          *sql.Stmt
}

func NewSqliteBackwardsExtremitiesTable(db *sql.DB) (tables.BackwardsExtremities, error) {
	s := &backwardExtremitiesStatements{}
	_, err := db.Exec(backwardExtremitiesSchema)
	if err != nil {
		return nil, err
	}
	if s.insertBackwardExtremityStmt, err = db.Prepare(insertBackwardExtremitySQL); err != nil {
		return nil, err
	}
	if s.selectBackwardExtremitiesForRoomStmt, err = db.Prepare(selectBackwardExtremitiesForRoomSQL); err != nil {
		return nil, err
	}
	if s.deleteBackwardExtremityStmt, err = db.Prepare(deleteBackwardExtremitySQL); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *backwardExtremitiesStatements) InsertsBackwardExtremity(
	ctx context.Context, txn *sql.Tx, roomID, eventID string, prevEventID string,
) (err error) {
	_, err = txn.Stmt(s.insertBackwardExtremityStmt).ExecContext(ctx, roomID, eventID, prevEventID)
	return
}

func (s *backwardExtremitiesStatements) SelectBackwardExtremitiesForRoom(
	ctx context.Context, roomID string,
) (eventIDs []string, err error) {
	rows, err := s.selectBackwardExtremitiesForRoomStmt.QueryContext(ctx, roomID)
	if err != nil {
		return
	}
	defer common.CloseAndLogIfError(ctx, rows, "selectBackwardExtremitiesForRoom: rows.close() failed")

	for rows.Next() {
		var eID string
		if err = rows.Scan(&eID); err != nil {
			return
		}

		eventIDs = append(eventIDs, eID)
	}

	return eventIDs, rows.Err()
}

func (s *backwardExtremitiesStatements) DeleteBackwardExtremity(
	ctx context.Context, txn *sql.Tx, roomID, knownEventID string,
) (err error) {
	_, err = txn.Stmt(s.deleteBackwardExtremityStmt).ExecContext(ctx, roomID, knownEventID)
	return
}
