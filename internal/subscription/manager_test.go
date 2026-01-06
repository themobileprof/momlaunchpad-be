package subscription

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestManager_HasFeature(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		featureKey string
		setupMock  func(sqlmock.Sqlmock)
		want       bool
		wantErr    bool
	}{
		{
			name:       "feature available",
			userID:     "11111111-1111-1111-1111-111111111111",
			featureKey: "chat",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
				m.ExpectQuery(`SELECT EXISTS`).WithArgs("11111111-1111-1111-1111-111111111111", "chat").
					WillReturnRows(rows)
			},
			want: true,
		},
		{
			name:       "feature missing",
			userID:     "11111111-1111-1111-1111-111111111111",
			featureKey: "calendar",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
				m.ExpectQuery(`SELECT EXISTS`).WithArgs("11111111-1111-1111-1111-111111111111", "calendar").
					WillReturnRows(rows)
			},
			want: false,
		},
		{
			name:       "query error",
			userID:     "22222222-2222-2222-2222-222222222222",
			featureKey: "chat",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT EXISTS`).WithArgs("22222222-2222-2222-2222-222222222222", "chat").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New: %v", err)
			}
			defer db.Close()

			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			mgr := NewManager(db)
			got, err := mgr.HasFeature(context.Background(), tt.userID, tt.featureKey)
			if (err != nil) != tt.wantErr {
				t.Fatalf("HasFeature error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("HasFeature = %v, want %v", got, tt.want)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet expectations: %v", err)
			}
		})
	}
}

func TestManager_CheckQuota(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		featureKey string
		setupMock  func(sqlmock.Sqlmock)
		want       bool
		wantErr    bool
	}{
		{
			name:       "within quota - no usage yet",
			userID:     "user1",
			featureKey: "chat",
			setupMock: func(m sqlmock.Sqlmock) {
				// Query returns quota limit and current usage
				rows := sqlmock.NewRows([]string{"quota_limit", "quota_period", "usage_count"}).
					AddRow(100, "monthly", 0)
				m.ExpectQuery(`SELECT (.+) FROM subscriptions s`).
					WithArgs("user1", "chat").
					WillReturnRows(rows)
			},
			want: true,
		},
		{
			name:       "within quota - partial usage",
			userID:     "user1",
			featureKey: "chat",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"quota_limit", "quota_period", "usage_count"}).
					AddRow(100, "monthly", 50)
				m.ExpectQuery(`SELECT (.+) FROM subscriptions s`).
					WithArgs("user1", "chat").
					WillReturnRows(rows)
			},
			want: true,
		},
		{
			name:       "quota exceeded",
			userID:     "user1",
			featureKey: "chat",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"quota_limit", "quota_period", "usage_count"}).
					AddRow(100, "monthly", 100)
				m.ExpectQuery(`SELECT (.+) FROM subscriptions s`).
					WithArgs("user1", "chat").
					WillReturnRows(rows)
			},
			want: false,
		},
		{
			name:       "unlimited quota",
			userID:     "user1",
			featureKey: "chat",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"quota_limit", "quota_period", "usage_count"}).
					AddRow(nil, "unlimited", 1000)
				m.ExpectQuery(`SELECT (.+) FROM subscriptions s`).
					WithArgs("user1", "chat").
					WillReturnRows(rows)
			},
			want: true,
		},
		{
			name:       "no active subscription",
			userID:     "user1",
			featureKey: "chat",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT (.+) FROM subscriptions s`).
					WithArgs("user1", "chat").
					WillReturnError(sql.ErrNoRows)
			},
			want: false,
		},
		{
			name:       "database error",
			userID:     "user1",
			featureKey: "chat",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT (.+) FROM subscriptions s`).
					WithArgs("user1", "chat").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New: %v", err)
			}
			defer db.Close()

			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			mgr := NewManager(db)
			got, err := mgr.CheckQuota(context.Background(), tt.userID, tt.featureKey)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckQuota error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("CheckQuota = %v, want %v", got, tt.want)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet expectations: %v", err)
			}
		})
	}
}

func TestManager_IncrementUsage(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		featureKey string
		setupMock  func(sqlmock.Sqlmock)
		wantErr    bool
	}{
		{
			name:       "first usage in period",
			userID:     "user1",
			featureKey: "chat",
			setupMock: func(m sqlmock.Sqlmock) {
				// Query quota period
				rows := sqlmock.NewRows([]string{"quota_period"}).AddRow("monthly")
				m.ExpectQuery(`SELECT pf.quota_period`).
					WithArgs("user1", "chat").
					WillReturnRows(rows)

				// Insert or update usage (4 args: userID, featureKey, period_start, period_end)
				m.ExpectExec(`INSERT INTO feature_usage`).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name:       "increment existing usage",
			userID:     "user1",
			featureKey: "chat",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"quota_period"}).AddRow("daily")
				m.ExpectQuery(`SELECT pf.quota_period`).
					WithArgs("user1", "chat").
					WillReturnRows(rows)

				m.ExpectExec(`INSERT INTO feature_usage`).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name:       "no active subscription",
			userID:     "user1",
			featureKey: "chat",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT pf.quota_period`).
					WithArgs("user1", "chat").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name:       "database error",
			userID:     "user1",
			featureKey: "chat",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT pf.quota_period`).
					WithArgs("user1", "chat").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New: %v", err)
			}
			defer db.Close()

			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			mgr := NewManager(db)
			err = mgr.IncrementUsage(context.Background(), tt.userID, tt.featureKey)
			if (err != nil) != tt.wantErr {
				t.Fatalf("IncrementUsage error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet expectations: %v", err)
			}
		})
	}
}
