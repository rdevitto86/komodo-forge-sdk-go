// LocalStack community does not support the RDS Data API (rds-data is Pro-only).
// Tests are component-only via SDK interface mocking. Integration coverage
// requires LocalStack Pro or a sandbox AWS account with an Aurora cluster
// that has the Data API enabled.

package rds

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

// fakeRDSData captures calls made to each SDK method so tests can assert on
// the inputs received, and returns pre-configured responses.
type fakeRDSData struct {
	executeIn   *rdsdata.ExecuteStatementInput
	executeOut  *rdsdata.ExecuteStatementOutput
	executeErr  error
	batchIn     *rdsdata.BatchExecuteStatementInput
	batchOut    *rdsdata.BatchExecuteStatementOutput
	batchErr    error
	beginOut    *rdsdata.BeginTransactionOutput
	beginErr    error
	commitIn    *rdsdata.CommitTransactionInput
	commitErr   error
	rollbackIn  *rdsdata.RollbackTransactionInput
	rollbackErr error
}

func (f *fakeRDSData) ExecuteStatement(_ context.Context, in *rdsdata.ExecuteStatementInput, _ ...func(*rdsdata.Options)) (*rdsdata.ExecuteStatementOutput, error) {
	f.executeIn = in
	if f.executeOut == nil {
		f.executeOut = &rdsdata.ExecuteStatementOutput{}
	}
	return f.executeOut, f.executeErr
}

func (f *fakeRDSData) BatchExecuteStatement(_ context.Context, in *rdsdata.BatchExecuteStatementInput, _ ...func(*rdsdata.Options)) (*rdsdata.BatchExecuteStatementOutput, error) {
	f.batchIn = in
	if f.batchOut == nil {
		f.batchOut = &rdsdata.BatchExecuteStatementOutput{}
	}
	return f.batchOut, f.batchErr
}

func (f *fakeRDSData) BeginTransaction(_ context.Context, _ *rdsdata.BeginTransactionInput, _ ...func(*rdsdata.Options)) (*rdsdata.BeginTransactionOutput, error) {
	if f.beginOut == nil {
		f.beginOut = &rdsdata.BeginTransactionOutput{TransactionId: aws.String("txn-123")}
	}
	return f.beginOut, f.beginErr
}

func (f *fakeRDSData) CommitTransaction(_ context.Context, in *rdsdata.CommitTransactionInput, _ ...func(*rdsdata.Options)) (*rdsdata.CommitTransactionOutput, error) {
	f.commitIn = in
	return &rdsdata.CommitTransactionOutput{}, f.commitErr
}

func (f *fakeRDSData) RollbackTransaction(_ context.Context, in *rdsdata.RollbackTransactionInput, _ ...func(*rdsdata.Options)) (*rdsdata.RollbackTransactionOutput, error) {
	f.rollbackIn = in
	return &rdsdata.RollbackTransactionOutput{}, f.rollbackErr
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func newTestClient(fake *fakeRDSData) *Client {
	return newWithAPI(fake, "arn:aws:rds:us-east-1:123456789012:cluster:test", "arn:aws:secretsmanager:us-east-1:123456789012:secret:test", "testdb")
}

// findParam returns the SqlParameter with the given name from a slice, or nil.
func findParam(params []types.SqlParameter, name string) *types.SqlParameter {
	for i := range params {
		if aws.ToString(params[i].Name) == name {
			return &params[i]
		}
	}
	return nil
}

// ── Unit Tests ────────────────────────────────────────────────────────────────

func TestNew_MissingRegion(t *testing.T) {
	_, err := New(context.Background(), Config{
		ResourceArn: "arn:aws:rds:us-east-1:123:cluster:x",
		SecretArn:   "arn:aws:secretsmanager:us-east-1:123:secret:x",
	})
	if err == nil {
		t.Fatal("expected error for missing region, got nil")
	}
}

func TestExecuteStatement_ParameterConversion(t *testing.T) {
	t.Parallel()

	fake := &fakeRDSData{}
	client := newTestClient(fake)

	_, err := client.ExecuteStatement(context.Background(), ExecuteStatementInput{
		SQL: "SELECT 1",
		Parameters: map[string]any{
			"strParam":   "hello",
			"intParam":   int(7),
			"int32Param": int32(8),
			"int64Param": int64(9),
			"f32Param":   float32(1.5),
			"f64Param":   float64(2.5),
			"boolParam":  true,
			"blobParam":  []byte{0xAB, 0xCD},
			"nullParam":  nil,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fake.executeIn == nil {
		t.Fatal("execute was not called")
	}
	params := fake.executeIn.Parameters

	cases := []struct {
		name      string
		checkType func(types.Field) bool
	}{
		{"strParam", func(f types.Field) bool { _, ok := f.(*types.FieldMemberStringValue); return ok }},
		{"intParam", func(f types.Field) bool { _, ok := f.(*types.FieldMemberLongValue); return ok }},
		{"int32Param", func(f types.Field) bool { _, ok := f.(*types.FieldMemberLongValue); return ok }},
		{"int64Param", func(f types.Field) bool { _, ok := f.(*types.FieldMemberLongValue); return ok }},
		{"f32Param", func(f types.Field) bool { _, ok := f.(*types.FieldMemberDoubleValue); return ok }},
		{"f64Param", func(f types.Field) bool { _, ok := f.(*types.FieldMemberDoubleValue); return ok }},
		{"boolParam", func(f types.Field) bool { _, ok := f.(*types.FieldMemberBooleanValue); return ok }},
		{"blobParam", func(f types.Field) bool { _, ok := f.(*types.FieldMemberBlobValue); return ok }},
		{"nullParam", func(f types.Field) bool { _, ok := f.(*types.FieldMemberIsNull); return ok }},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := findParam(params, tc.name)
			if p == nil {
				t.Fatalf("parameter %q not found in SDK call", tc.name)
			}
			if !tc.checkType(p.Value) {
				t.Fatalf("parameter %q has wrong Field type: %T", tc.name, p.Value)
			}
		})
	}
}

func TestExecuteStatement_RowDecoding(t *testing.T) {
	t.Parallel()

	fake := &fakeRDSData{
		executeOut: &rdsdata.ExecuteStatementOutput{
			ColumnMetadata: []types.ColumnMetadata{
				{Name: aws.String("id")},
				{Name: aws.String("name")},
				{Name: aws.String("score")},
				{Name: aws.String("active")},
			},
			Records: [][]types.Field{
				{
					&types.FieldMemberLongValue{Value: 1},
					&types.FieldMemberStringValue{Value: "alice"},
					&types.FieldMemberDoubleValue{Value: 9.8},
					&types.FieldMemberBooleanValue{Value: true},
				},
				{
					&types.FieldMemberLongValue{Value: 2},
					&types.FieldMemberStringValue{Value: "bob"},
					&types.FieldMemberIsNull{Value: true},
					&types.FieldMemberBooleanValue{Value: false},
				},
			},
			NumberOfRecordsUpdated: 2,
		},
	}
	client := newTestClient(fake)

	out, err := client.ExecuteStatement(context.Background(), ExecuteStatementInput{SQL: "SELECT id, name, score, active FROM users"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(out.Rows))
	}

	// Row 0
	row0 := out.Rows[0]
	if row0["id"] != int64(1) {
		t.Errorf("row0[id]: got %v (%T), want int64(1)", row0["id"], row0["id"])
	}
	if row0["name"] != "alice" {
		t.Errorf("row0[name]: got %v, want alice", row0["name"])
	}
	if row0["score"] != float64(9.8) {
		t.Errorf("row0[score]: got %v, want 9.8", row0["score"])
	}
	if row0["active"] != true {
		t.Errorf("row0[active]: got %v, want true", row0["active"])
	}

	// Row 1 — NULL score
	row1 := out.Rows[1]
	if row1["id"] != int64(2) {
		t.Errorf("row1[id]: got %v, want int64(2)", row1["id"])
	}
	if row1["score"] != nil {
		t.Errorf("row1[score]: got %v, want nil (NULL)", row1["score"])
	}

	if out.NumRecords != 2 {
		t.Errorf("NumRecords: got %d, want 2", out.NumRecords)
	}
}

func TestExecuteStatement_Validation(t *testing.T) {
	t.Parallel()

	fake := &fakeRDSData{}
	client := newTestClient(fake)

	_, err := client.ExecuteStatement(context.Background(), ExecuteStatementInput{SQL: ""})
	if err == nil {
		t.Fatal("expected error for empty SQL, got nil")
	}
	// Confirm the SDK was never called.
	if fake.executeIn != nil {
		t.Fatal("SDK ExecuteStatement was called despite empty SQL")
	}
}

func TestTransactionLifecycle(t *testing.T) {
	t.Parallel()

	fake := &fakeRDSData{
		beginOut: &rdsdata.BeginTransactionOutput{TransactionId: aws.String("txn-abc")},
	}
	client := newTestClient(fake)
	ctx := context.Background()

	// Begin
	txID, err := client.BeginTransaction(ctx)
	if err != nil {
		t.Fatalf("BeginTransaction error: %v", err)
	}
	if txID != "txn-abc" {
		t.Fatalf("BeginTransaction: got txID %q, want txn-abc", txID)
	}

	// Execute with transaction ID
	_, err = client.ExecuteStatement(ctx, ExecuteStatementInput{
		SQL:           "INSERT INTO t VALUES (:v)",
		Parameters:    map[string]any{"v": "data"},
		TransactionID: txID,
	})
	if err != nil {
		t.Fatalf("ExecuteStatement error: %v", err)
	}
	if aws.ToString(fake.executeIn.TransactionId) != "txn-abc" {
		t.Errorf("TransactionId not threaded through: got %q, want txn-abc",
			aws.ToString(fake.executeIn.TransactionId))
	}

	// Commit
	if err = client.CommitTransaction(ctx, txID); err != nil {
		t.Fatalf("CommitTransaction error: %v", err)
	}
	if aws.ToString(fake.commitIn.TransactionId) != "txn-abc" {
		t.Errorf("CommitTransaction TransactionId: got %q, want txn-abc",
			aws.ToString(fake.commitIn.TransactionId))
	}
}

func TestTransactionRollback(t *testing.T) {
	t.Parallel()

	fake := &fakeRDSData{
		beginOut: &rdsdata.BeginTransactionOutput{TransactionId: aws.String("txn-xyz")},
	}
	client := newTestClient(fake)
	ctx := context.Background()

	txID, err := client.BeginTransaction(ctx)
	if err != nil {
		t.Fatalf("BeginTransaction error: %v", err)
	}

	_, err = client.ExecuteStatement(ctx, ExecuteStatementInput{
		SQL:           "DELETE FROM t WHERE id = :id",
		Parameters:    map[string]any{"id": int64(99)},
		TransactionID: txID,
	})
	if err != nil {
		t.Fatalf("ExecuteStatement error: %v", err)
	}
	if aws.ToString(fake.executeIn.TransactionId) != "txn-xyz" {
		t.Errorf("TransactionId not threaded through: got %q, want txn-xyz",
			aws.ToString(fake.executeIn.TransactionId))
	}

	if err = client.RollbackTransaction(ctx, txID); err != nil {
		t.Fatalf("RollbackTransaction error: %v", err)
	}
	if aws.ToString(fake.rollbackIn.TransactionId) != "txn-xyz" {
		t.Errorf("RollbackTransaction TransactionId: got %q, want txn-xyz",
			aws.ToString(fake.rollbackIn.TransactionId))
	}
}

func TestBatchExecuteStatement(t *testing.T) {
	t.Parallel()

	fake := &fakeRDSData{
		batchOut: &rdsdata.BatchExecuteStatementOutput{
			UpdateResults: []types.UpdateResult{
				{GeneratedFields: []types.Field{&types.FieldMemberLongValue{Value: 10}}},
				{GeneratedFields: []types.Field{&types.FieldMemberLongValue{Value: 11}}},
				{GeneratedFields: []types.Field{&types.FieldMemberLongValue{Value: 12}}},
			},
		},
	}
	client := newTestClient(fake)

	out, err := client.BatchExecuteStatement(context.Background(), BatchExecuteStatementInput{
		SQL: "INSERT INTO items (name) VALUES (:name)",
		ParameterSets: []map[string]any{
			{"name": "item-a"},
			{"name": "item-b"},
			{"name": "item-c"},
		},
	})
	if err != nil {
		t.Fatalf("BatchExecuteStatement error: %v", err)
	}

	// Single SDK call with all three parameter sets.
	if fake.batchIn == nil {
		t.Fatal("batch execute was not called")
	}
	if len(fake.batchIn.ParameterSets) != 3 {
		t.Errorf("expected 3 parameter sets in SDK call, got %d", len(fake.batchIn.ParameterSets))
	}

	// Decoded results.
	if len(out.UpdateResults) != 3 {
		t.Fatalf("expected 3 UpdateResults, got %d", len(out.UpdateResults))
	}
	for i, r := range out.UpdateResults {
		if len(r.GeneratedFields) != 1 {
			t.Errorf("result %d: expected 1 generated field, got %d", i, len(r.GeneratedFields))
			continue
		}
		want := int64(10 + i)
		if r.GeneratedFields[0] != want {
			t.Errorf("result %d: GeneratedFields[0] = %v, want %v", i, r.GeneratedFields[0], want)
		}
	}
}

func TestExecuteStatement_SDKError(t *testing.T) {
	t.Parallel()

	sdkErr := errors.New("connection refused")
	fake := &fakeRDSData{executeErr: sdkErr}
	client := newTestClient(fake)

	_, err := client.ExecuteStatement(context.Background(), ExecuteStatementInput{SQL: "SELECT 1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sdkErr) {
		t.Errorf("expected wrapped sdkErr, got: %v", err)
	}
}
