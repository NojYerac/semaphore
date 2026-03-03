package db_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/nojyerac/go-lib/audit"
	"github.com/nojyerac/go-lib/auth"
	"github.com/nojyerac/go-lib/db"
	"github.com/nojyerac/go-lib/log"
	"github.com/nojyerac/semaphore/data"
	. "github.com/nojyerac/semaphore/data/db"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	featureFlagSelectSQLRegex = `SELECT f\.id, f\.name, .* FROM feature_flags`
	mockUUID                  = "123e4567-e89b-12d3-a456-426614174000"
)

var _ = Describe("Db", func() {
	var (
		conn         db.Database
		opCtx, ctx   context.Context
		cancel       context.CancelFunc
		dataSource   *DataSource
		auditLog     audit.AuditLogger
		auditEntries = &bytes.Buffer{}
		mockTime     = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	)
	var _, sqlMock, err = sqlmock.NewWithDSN("testDB")
	if err != nil {
		panic(err)
	}
	AfterEach(func() {
		auditEntries.Reset()
	})
	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		config := db.NewConfiguration()
		config.Driver = "sqlmock"
		config.DBConnStr = "testDB"
		l := log.NewLogger(log.NewConfiguration(), log.WithOutput(GinkgoWriter))
		opCtx = auth.WithClaims(log.WithLogger(ctx, l), &auth.Claims{Subject: uuid.New().String()})

		conn = db.NewDatabase(config, db.WithLogger(l))
		Expect(conn.Open(ctx)).To(Succeed())

		sqlMock.ExpectExec("CREATE TABLE IF NOT EXISTS feature_flags").WillReturnResult(sqlmock.NewResult(0, 0))
		auditLog, err = audit.NewAuditLogger(
			&audit.Configuration{AuditLoggerType: "stdout"},
			audit.WithOutput(auditEntries),
			audit.WithTimeNow(func() time.Time { return mockTime }),
		)
		Expect(err).ToNot(HaveOccurred())
		dataSource, err = NewDataSource(ctx, conn, WithAuditLogger(auditLog))
		Expect(err).ToNot(HaveOccurred())
		Expect(dataSource).ToNot(BeNil())
	})
	AfterEach(func() {
		Expect(sqlMock.ExpectationsWereMet()).To(Succeed())
		cancel()
	})
	Describe("DataSource", func() {
		Context("GetFlags", func() {
			It("returns empty list when no flags exist", func() {
				columns := []string{"id", "name", "description", "enabled", "created_at", "updated_at", "strategies"}
				sqlMock.ExpectQuery(featureFlagSelectSQLRegex).
					WillReturnRows(sqlmock.NewRows(columns))
				flags, err := dataSource.GetFlags(opCtx)
				Expect(err).ToNot(HaveOccurred())
				Expect(flags).To(BeEmpty())
			})
			It("returns list of flags", func() {
				columns := []string{"id", "name", "description", "enabled", "created_at", "updated_at", "strategies"}
				sqlMock.ExpectQuery(featureFlagSelectSQLRegex).
					WillReturnRows(sqlmock.NewRows(columns).
						AddRow(mockUUID, "test-flag", "A test flag", true, mockTime, mockTime, []byte("[]")))
				flags, err := dataSource.GetFlags(opCtx)
				Expect(err).ToNot(HaveOccurred())
				Expect(flags).To(HaveLen(1))
				Expect(flags[0].ID).To(Equal(mockUUID))
				Expect(flags[0].Name).To(Equal("test-flag"))
				Expect(flags[0].Description).To(Equal("A test flag"))
				Expect(flags[0].Enabled).To(BeTrue())
			})
			It("returns error on query failure", func() {
				sqlMock.ExpectQuery(featureFlagSelectSQLRegex).
					WillReturnError(fmt.Errorf("query error"))
				_, err := dataSource.GetFlags(opCtx)
				Expect(err).To(MatchError(ContainSubstring("query error")))
			})
			It("returns error on scan failure", func() {
				columns := []string{"id", "name", "description", "enabled", "created_at", "updated_at", "strategies"}
				sqlMock.ExpectQuery(featureFlagSelectSQLRegex).
					WillReturnRows(sqlmock.NewRows(columns).
						AddRow(mockUUID, "test-flag", "A test flag", true, "2024-01-01 00:00:00", mockTime, []byte("[]")))
				_, err := dataSource.GetFlags(opCtx)
				Expect(err).To(MatchError(ContainSubstring("sql: Scan error on column index 4")))
			})
		})
		Context("GetFlagByID", func() {
			It("returns flag by ID", func() {
				columns := []string{"id", "name", "description", "enabled", "created_at", "updated_at", "strategies"}
				sqlMock.ExpectQuery(featureFlagSelectSQLRegex + `.* WHERE f\.id = \$1`).
					WithArgs(mockUUID).
					WillReturnRows(sqlmock.NewRows(columns).
						AddRow(mockUUID, "test-flag", "A test flag", true, mockTime, mockTime, []byte("[]")))
				flag, err := dataSource.GetFlagByID(opCtx, mockUUID)
				Expect(err).ToNot(HaveOccurred())
				Expect(flag).ToNot(BeNil())
				Expect(flag.ID).To(Equal(mockUUID))
			})
		})
		Context("CreateFlag", func() {
			It("creates a new flag", func() {
				newFlag := &data.FeatureFlag{Name: "new-flag", Description: "A new flag", Enabled: false}
				sqlMock.ExpectBegin()
				sqlMock.ExpectExec(`INSERT INTO feature_flags`).
					WithArgs(sqlmock.AnyArg(), "new-flag", "A new flag", false).
					WillReturnResult(sqlmock.NewResult(1, 1))
				sqlMock.ExpectCommit()
				id, err := dataSource.CreateFlag(opCtx, newFlag)
				Expect(err).ToNot(HaveOccurred())
				Expect(id).ToNot(BeEmpty())
				Expect(auditEntries.String()).To(And(
					ContainSubstring(`"action": "flag.created"`),
					ContainSubstring(`"new_value": "`+id+`"`),
				))
			})
		})
		Context("UpdateFlag", func() {
			When("flag has no strategies", func() {
				It("updates an existing flag", func() {
					columns := []string{"id", "name", "description", "enabled", "created_at", "updated_at", "strategies"}
					updateFlag := &data.FeatureFlag{ID: mockUUID, Name: "updated-flag", Description: "An updated flag", Enabled: true}
					sqlMock.ExpectBegin()
					sqlMock.ExpectQuery(featureFlagSelectSQLRegex + `.* WHERE f\.id = \$1`).
						WithArgs(mockUUID).
						WillReturnRows(sqlmock.NewRows(columns).
							AddRow(mockUUID, "test-flag", "A test flag", true, mockTime, mockTime, []byte("[]")))
					sqlMock.ExpectExec(`UPDATE feature_flags`).
						WithArgs("updated-flag", "An updated flag", true, mockUUID).
						WillReturnResult(sqlmock.NewResult(1, 1))
					sqlMock.ExpectExec(`DELETE FROM strategies`).
						WithArgs(mockUUID).
						WillReturnResult(sqlmock.NewResult(1, 1))
					sqlMock.ExpectCommit()
					err := dataSource.UpdateFlag(opCtx, updateFlag)
					Expect(err).ToNot(HaveOccurred())
					Expect(auditEntries.String()).To(And(
						ContainSubstring(`"action": "flag.updated"`),
					))
				})
			})
			When("flag has strategies", func() {
				It("updates an existing flag with strategies", func() {
					columns := []string{"id", "name", "description", "enabled", "created_at", "updated_at", "strategies"}
					sqlMock.ExpectBegin()
					sqlMock.ExpectQuery(featureFlagSelectSQLRegex + `.* WHERE f\.id = \$1`).
						WithArgs(mockUUID).
						WillReturnRows(sqlmock.NewRows(columns).
							AddRow(mockUUID, "test-flag", "A test flag", true, mockTime, mockTime, []byte("[]")))
					sqlMock.ExpectExec(`UPDATE feature_flags`).
						WithArgs("updated-flag", "An updated flag", true, mockUUID).
						WillReturnResult(sqlmock.NewResult(1, 1))
					sqlMock.ExpectExec(`DELETE FROM strategies`).
						WithArgs(mockUUID).
						WillReturnResult(sqlmock.NewResult(1, 1))
					sqlMock.ExpectExec(`INSERT INTO strategies`).
						WithArgs(mockUUID, "user_targeting", []byte(`{"user_ids":["user1"]}`)).
						WillReturnResult(sqlmock.NewResult(1, 1))
					sqlMock.ExpectCommit()
					err := dataSource.UpdateFlag(opCtx, &data.FeatureFlag{
						ID:          mockUUID,
						Name:        "updated-flag",
						Description: "An updated flag",
						Enabled:     true,
						Strategies: data.Strategies{
							data.Strategy{
								Type:    "user_targeting",
								Payload: json.RawMessage(`{"user_ids":["user1"]}`),
							},
						},
					})
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
		Context("DeleteFlag", func() {
			It("deletes an existing flag", func() {
				columns := []string{"id", "name", "description", "enabled", "created_at", "updated_at", "strategies"}
				sqlMock.ExpectBegin()
				sqlMock.ExpectQuery(featureFlagSelectSQLRegex + `.* WHERE f\.id = \$1`).
					WithArgs(mockUUID).
					WillReturnRows(sqlmock.NewRows(columns).
						AddRow(mockUUID, "test-flag", "A test flag", true, mockTime, mockTime, []byte("[]")))
				sqlMock.ExpectExec(`DELETE FROM feature_flags`).
					WithArgs(mockUUID).
					WillReturnResult(sqlmock.NewResult(1, 1))
				sqlMock.ExpectCommit()
				err := dataSource.DeleteFlag(opCtx, mockUUID)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
