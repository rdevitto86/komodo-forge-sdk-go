// Package dynamodb re-exports aws/dynamo at the legacy aws/dynamodb import path.
// Services should migrate to github.com/rdevitto86/komodo-forge-sdk-go/aws/dynamo.
package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	d "github.com/rdevitto86/komodo-forge-sdk-go/aws/dynamo"
)

type Config = d.Config
type QueryInput = d.QueryInput
type QueryOutput = d.QueryOutput
type ScanInput = d.ScanInput
type ScanOutput = d.ScanOutput

var ErrClientNotInitialized = d.ErrClientNotInitialized

func Init(cfg Config) error        { return d.Init(cfg) }
func IsInitialized() bool          { return d.IsInitialized() }
func WrapError(err error, op string) error { return d.WrapError(err, op) }

func BuildKey(pk string, pv interface{}, sk string, sv interface{}) (map[string]types.AttributeValue, error) {
	return d.BuildKey(pk, pv, sk, sv)
}

func GetItem(ctx context.Context, table string, key map[string]types.AttributeValue, batch bool, keys []map[string]types.AttributeValue) (interface{}, error) {
	return d.GetItem(ctx, table, key, batch, keys)
}
func GetItemAs(ctx context.Context, table string, key map[string]types.AttributeValue, batch bool, keys []map[string]types.AttributeValue, out interface{}) error {
	return d.GetItemAs(ctx, table, key, batch, keys, out)
}
func WriteItem(ctx context.Context, table string, item map[string]types.AttributeValue, batch bool, items []map[string]types.AttributeValue, cond *string) error {
	return d.WriteItem(ctx, table, item, batch, items, cond)
}
func WriteItemFrom(ctx context.Context, table string, item interface{}, batch bool, items interface{}, cond *string) error {
	return d.WriteItemFrom(ctx, table, item, batch, items, cond)
}
func UpdateItem(ctx context.Context, table string, key map[string]types.AttributeValue, updateExpr string, exprValues map[string]types.AttributeValue, exprNames map[string]string, cond *string) (map[string]types.AttributeValue, error) {
	return d.UpdateItem(ctx, table, key, updateExpr, exprValues, exprNames, cond)
}
func UpdateItemAs(ctx context.Context, table string, key map[string]types.AttributeValue, updateExpr string, exprValues map[string]types.AttributeValue, exprNames map[string]string, cond *string, out interface{}) error {
	return d.UpdateItemAs(ctx, table, key, updateExpr, exprValues, exprNames, cond, out)
}
func DeleteItem(ctx context.Context, table string, key map[string]types.AttributeValue, batch bool, keys []map[string]types.AttributeValue, cond *string) error {
	return d.DeleteItem(ctx, table, key, batch, keys, cond)
}
func Query(ctx context.Context, input QueryInput) (*QueryOutput, error) { return d.Query(ctx, input) }
func QueryAs(ctx context.Context, input QueryInput, out interface{}) (*QueryOutput, error) {
	return d.QueryAs(ctx, input, out)
}
func QueryAll(ctx context.Context, input QueryInput) ([]map[string]types.AttributeValue, error) {
	return d.QueryAll(ctx, input)
}
func QueryAllAs(ctx context.Context, input QueryInput, out interface{}) error {
	return d.QueryAllAs(ctx, input, out)
}
func Scan(ctx context.Context, input ScanInput) (*ScanOutput, error) { return d.Scan(ctx, input) }
func ScanAll(ctx context.Context, input ScanInput) ([]map[string]types.AttributeValue, error) {
	return d.ScanAll(ctx, input)
}
