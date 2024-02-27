// Copyright observIQ, Inc.
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

package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestInitDatabaseConn(t *testing.T) {
	testCases := []struct {
		desc            string
		ctx             context.Context
		role            string
		database        string
		setExpectations func(m sqlmock.Sqlmock, role, database string)
		expectedErr     error
	}{
		{
			desc:     "pass",
			ctx:      context.Background(),
			role:     "role",
			database: "db",
			setExpectations: func(m sqlmock.Sqlmock, role, database string) {
				m.ExpectExec(fmt.Sprintf(`USE ROLE "%s";`, role)).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(nil)
				m.ExpectExec(fmt.Sprintf(`CREATE DATABASE IF NOT EXISTS "%s";`, database)).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(nil)
				m.ExpectExec(fmt.Sprintf(`USE DATABASE "%s";`, database)).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(nil)
			},
		},
		{
			desc:     "fail USE ROLE stmt",
			ctx:      context.Background(),
			role:     "role",
			database: "",
			setExpectations: func(m sqlmock.Sqlmock, role, _ string) {
				m.ExpectExec(fmt.Sprintf(`USE ROLE "%s";`, role)).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(fmt.Errorf("fail"))
			},
			expectedErr: fmt.Errorf("failed to call 'USE ROLE \"role\";': fail"),
		},
		{
			desc:     "fail CREATE DATABASE stmt",
			ctx:      context.Background(),
			role:     "role",
			database: "db",
			setExpectations: func(m sqlmock.Sqlmock, role, database string) {
				m.ExpectExec(fmt.Sprintf(`USE ROLE "%s";`, role)).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(nil)
				m.ExpectExec(fmt.Sprintf(`CREATE DATABASE IF NOT EXISTS "%s";`, database)).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(fmt.Errorf("fail"))
			},
			expectedErr: fmt.Errorf("failed to create new database: fail"),
		},
		{
			desc:     "fail USE DATABASE stmt",
			ctx:      context.Background(),
			role:     "role",
			database: "db",
			setExpectations: func(m sqlmock.Sqlmock, role, database string) {
				m.ExpectExec(fmt.Sprintf(`USE ROLE "%s";`, role)).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(nil)
				m.ExpectExec(fmt.Sprintf(`CREATE DATABASE IF NOT EXISTS "%s";`, database)).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(nil)
				m.ExpectExec(fmt.Sprintf(`USE DATABASE "%s"`, database)).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(fmt.Errorf("fail"))
			},
			expectedErr: fmt.Errorf("failed to call 'USE DATABASE \"db\";': fail"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			db, mock := NewMock(t)
			defer db.Close()

			tc.setExpectations(mock, tc.role, tc.database)

			s := &Snowflake{db: db, warehouse: "", database: tc.database}
			err := s.InitDatabaseConn(tc.ctx, tc.role)

			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCreateSchema(t *testing.T) {
	testCases := []struct {
		desc            string
		ctx             context.Context
		schema          string
		database        string
		setExpectations func(m sqlmock.Sqlmock, schema, database string)
		expectedErr     error
	}{
		{
			desc:     "pass",
			ctx:      context.Background(),
			schema:   "schema",
			database: "db",
			setExpectations: func(m sqlmock.Sqlmock, schema, database string) {
				m.ExpectExec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"."%s";`, database, schema)).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(nil)
				m.ExpectExec(fmt.Sprintf(`USE SCHEMA "%s"."%s";`, database, schema)).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(nil)
			},
		},
		{
			desc:     "fail CREATE SCHEMA stmt",
			ctx:      context.Background(),
			schema:   "schema",
			database: "db",
			setExpectations: func(m sqlmock.Sqlmock, schema, database string) {
				m.ExpectExec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"."%s";`, database, schema)).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(fmt.Errorf("fail"))
			},
			expectedErr: fmt.Errorf("failed to create schema 'schema': fail"),
		},
		{
			desc:     "fail USE SCHEMA stmt",
			ctx:      context.Background(),
			schema:   "schema",
			database: "db",
			setExpectations: func(m sqlmock.Sqlmock, schema, database string) {
				m.ExpectExec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"."%s";`, database, schema)).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(nil)
				m.ExpectExec(fmt.Sprintf(`USE SCHEMA "%s"."%s";`, database, schema)).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(fmt.Errorf("fail"))
			},
			expectedErr: fmt.Errorf("failed to call 'USE SCHEMA \"schema\";': fail"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			db, mock := NewMock(t)
			defer db.Close()

			tc.setExpectations(mock, tc.schema, tc.database)

			s := &Snowflake{db: db, warehouse: "", database: tc.database}
			err := s.CreateSchema(tc.ctx, tc.schema)

			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCreateTable(t *testing.T) {
	sql := `CREATE TABLE "test";`
	testCases := []struct {
		desc            string
		ctx             context.Context
		setExpectations func(m sqlmock.Sqlmock, sql string)
		expectedErr     error
	}{
		{
			desc: "pass",
			ctx:  context.Background(),
			setExpectations: func(m sqlmock.Sqlmock, sql string) {
				m.ExpectExec(sql).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(nil)
			},
		},
		{
			desc: "fail CRATE TABLE stmt",
			ctx:  context.Background(),
			setExpectations: func(m sqlmock.Sqlmock, sql string) {
				m.ExpectExec(sql).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(fmt.Errorf("fail"))
			},
			expectedErr: fmt.Errorf("failed to create table: fail"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			db, mock := NewMock(t)
			defer db.Close()

			tc.setExpectations(mock, sql)

			s := &Snowflake{db: db}
			err := s.CreateTable(tc.ctx, sql)

			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestBatchInsert(t *testing.T) {
	testCases := []struct {
		desc            string
		ctx             context.Context
		data            []map[string]any
		setExpectations func(m sqlmock.Sqlmock, insert string)
		expectedErr     error
	}{
		{
			desc: "pass",
			ctx:  context.Background(),
			data: []map[string]any{
				{
					"map1key1": "1",
					"map1key2": "2",
				},
			},
			setExpectations: func(m sqlmock.Sqlmock, insert string) {
				m.ExpectBegin().WillReturnError(nil)
				m.ExpectExec(`USE WAREHOUSE "wh";`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(nil)
				m.ExpectExec(insert).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(nil)
				m.ExpectCommit()
			},
		},
		{
			desc: "fail BeginTxx",
			ctx:  context.Background(),
			data: []map[string]any{},
			setExpectations: func(m sqlmock.Sqlmock, _ string) {
				m.ExpectBegin().WillReturnError(fmt.Errorf("fail"))
			},
			expectedErr: fmt.Errorf("failed to create transaction: fail"),
		},
		{
			desc: "fail USE WAREHOUSE stmt",
			ctx:  context.Background(),
			data: []map[string]any{},
			setExpectations: func(m sqlmock.Sqlmock, _ string) {
				m.ExpectBegin().WillReturnError(nil)
				m.ExpectExec(`USE WAREHOUSE "wh";`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(fmt.Errorf("fail"))
				m.ExpectRollback()
			},
			expectedErr: fmt.Errorf("failed to call 'USE WAREHOUSE \"wh\";': fail"),
		},
		{
			desc: "fail insert stmt",
			ctx:  context.Background(),
			data: []map[string]any{
				{
					"map1key1": "1",
					"map1key2": "2",
				},
			},
			setExpectations: func(m sqlmock.Sqlmock, insert string) {
				m.ExpectBegin().WillReturnError(nil)
				m.ExpectExec(`USE WAREHOUSE "wh";`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(nil)
				m.ExpectExec(insert).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0)).WillReturnError(fmt.Errorf("fail"))
				m.ExpectRollback()
			},
			expectedErr: fmt.Errorf("failed to execute batch insert: fail"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			db, mock := NewMock(t)
			defer db.Close()

			tc.setExpectations(mock, "insert")

			s := &Snowflake{db: db, warehouse: "wh", database: ""}
			err := s.BatchInsert(tc.ctx, tc.data, "insert")

			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestClose(t *testing.T) {
	// nil db case
	s := &Snowflake{}
	require.NoError(t, s.Close())

	// mock returns err
	db, mock := NewMock(t)
	mock.ExpectClose().WillReturnError(fmt.Errorf("fail"))
	s.db = db
	require.ErrorContains(t, s.Close(), "fail")
}

func NewMock(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	return sqlxDB, mock
}
