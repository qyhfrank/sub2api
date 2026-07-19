package repository

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

func TestSetModelRateLimitDeadlineOrdering(t *testing.T) {
	resetAt := time.Date(2026, 7, 19, 14, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		existing   any
		wantUpdate bool
	}{
		{name: "later offset deadline is retained", existing: "2026-07-19T13:00:00-08:00", wantUpdate: false},
		{name: "malformed deadline is replaced", existing: "9999", wantUpdate: true},
		{name: "missing deadline is replaced", existing: nil, wantUpdate: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })

			driver := entsql.OpenDB(dialect.Postgres, db)
			client := dbent.NewClient(dbent.Driver(driver))
			t.Cleanup(func() { _ = client.Close() })
			repo := newAccountRepositoryWithSQL(client, db, nil)

			mock.ExpectBegin()
			mock.ExpectQuery(`(?s)SELECT .*rate_limit_reset_at.*FOR NO KEY UPDATE`).
				WithArgs("gemini-2.5-pro", int64(42)).
				WillReturnRows(sqlmock.NewRows([]string{"rate_limit_reset_at"}).AddRow(tt.existing))
			if tt.wantUpdate {
				mock.ExpectExec(`(?s)UPDATE accounts SET.*\$2::jsonb.*WHERE id = \$3`).
					WithArgs("gemini-2.5-pro", sqlmock.AnyArg(), int64(42)).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO scheduler_outbox")).
					WithArgs(service.SchedulerOutboxEventAccountChanged, int64(42), nil, nil, sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			} else {
				mock.ExpectCommit()
			}

			require.NoError(t, repo.SetModelRateLimit(context.Background(), 42, "gemini-2.5-pro", resetAt))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

type modelRateLimitSchedulerCacheGuard struct {
	service.SchedulerCache
}

func TestSetModelRateLimitTransactionBoundClientDefersCacheRefresh(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	driver := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(driver))
	t.Cleanup(func() { _ = client.Close() })

	mock.ExpectBegin()
	tx, err := client.Tx(context.Background())
	require.NoError(t, err)
	repo := newAccountRepositoryWithSQL(tx.Client(), db, &modelRateLimitSchedulerCacheGuard{})

	mock.ExpectQuery(`(?s)SELECT .*rate_limit_reset_at.*FOR NO KEY UPDATE`).
		WithArgs("gemini-2.5-pro", int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"rate_limit_reset_at"}).AddRow(nil))
	mock.ExpectExec(`(?s)UPDATE accounts SET.*\$2::jsonb.*WHERE id = \$3`).
		WithArgs("gemini-2.5-pro", sqlmock.AnyArg(), int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO scheduler_outbox")).
		WithArgs(service.SchedulerOutboxEventAccountChanged, int64(42), nil, nil, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.SetModelRateLimit(context.Background(), 42, "gemini-2.5-pro", time.Now().Add(time.Minute)))
	mock.ExpectRollback()
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListModelAvailabilityCandidates_GroupQueryIgnoresTransientState(t *testing.T) {
	var capturedSQL string
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(captureEntQueryMatcher{actual: &capturedSQL}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	driver := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(driver))
	t.Cleanup(func() { _ = client.Close() })
	repo := newAccountRepositoryWithSQL(client, db, nil)

	mock.ExpectQuery("model availability candidates").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	groupID := int64(42)
	accounts, err := repo.ListModelAvailabilityCandidates(
		context.Background(),
		&groupID,
		[]string{service.PlatformAnthropic},
		false,
	)
	require.NoError(t, err)
	require.Empty(t, accounts)
	require.NoError(t, mock.ExpectationsWereMet())

	normalized := normalizeSQLWhitespace(capturedSQL)
	_, whereClause, found := strings.Cut(normalized, " WHERE ")
	require.True(t, found, "expected WHERE clause in query: %s", normalized)
	whereClause, _, _ = strings.Cut(whereClause, " ORDER BY ")
	for _, configuredPredicate := range []string{"group_id", "status", "schedulable", "platform"} {
		require.Contains(t, whereClause, configuredPredicate)
	}
	for _, transientPredicate := range []string{
		"rate_limit_reset_at",
		"overload_until",
		"temp_unschedulable_until",
		"expires_at",
		"auto_pause_on_expired",
	} {
		require.NotContains(t, whereClause, transientPredicate, "configured-state diagnosis must not filter transient predicate %q", transientPredicate)
	}
}
