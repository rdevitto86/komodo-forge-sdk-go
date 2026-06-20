package dynamodb

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rdevitto86/komodo-forge-sdk-go/testing/testutil"
)

type crudFake struct {
	dynamoDBAPI
	getItem    func(ctx context.Context, in *dynamodb.GetItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	putItem    func(ctx context.Context, in *dynamodb.PutItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	deleteItem func(ctx context.Context, in *dynamodb.DeleteItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	updateItem func(ctx context.Context, in *dynamodb.UpdateItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	query      func(ctx context.Context, in *dynamodb.QueryInput, opts ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	scan       func(ctx context.Context, in *dynamodb.ScanInput, opts ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

func (f crudFake) GetItem(ctx context.Context, in *dynamodb.GetItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return f.getItem(ctx, in, opts...)
}

func (f crudFake) PutItem(ctx context.Context, in *dynamodb.PutItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return f.putItem(ctx, in, opts...)
}

func (f crudFake) DeleteItem(ctx context.Context, in *dynamodb.DeleteItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	return f.deleteItem(ctx, in, opts...)
}

func (f crudFake) UpdateItem(ctx context.Context, in *dynamodb.UpdateItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	return f.updateItem(ctx, in, opts...)
}

func (f crudFake) Query(ctx context.Context, in *dynamodb.QueryInput, opts ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return f.query(ctx, in, opts...)
}

func (f crudFake) Scan(ctx context.Context, in *dynamodb.ScanInput, opts ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	return f.scan(ctx, in, opts...)
}

func sk(v string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: v}}
}

func TestBuildKey(t *testing.T) {
	testutil.Component(t)
	c := newWithAPI(crudFake{}, 1)

	key, err := c.BuildKey("pk", "v", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != 1 {
		t.Fatalf("expected 1 key attribute, got %d", len(key))
	}

	key2, err := c.BuildKey("pk", "v", "sk", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key2) != 2 {
		t.Fatalf("expected 2 key attributes, got %d", len(key2))
	}
}

func TestGetItem_SuccessAndNotFound(t *testing.T) {
	testutil.Component(t)

	found := crudFake{getItem: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
		return &dynamodb.GetItemOutput{Item: sk("x")}, nil
	}}
	got, err := newWithAPI(found, 1).GetItem(context.Background(), "t", sk("x"), false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected an item")
	}

	missing := crudFake{getItem: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
		return &dynamodb.GetItemOutput{}, nil
	}}
	_, err = newWithAPI(missing, 1).GetItem(context.Background(), "t", sk("x"), false, nil)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetItemAs_Unmarshals(t *testing.T) {
	testutil.Component(t)

	f := crudFake{getItem: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
		return &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{
			"name": &types.AttributeValueMemberS{Value: "bob"},
		}}, nil
	}}
	var out struct {
		Name string `dynamodbav:"name"`
	}
	if err := newWithAPI(f, 1).GetItemAs(context.Background(), "t", sk("x"), false, nil, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "bob" {
		t.Fatalf("expected name bob, got %q", out.Name)
	}
}

func TestWriteItem_PutWithCondition(t *testing.T) {
	testutil.Component(t)

	var captured *dynamodb.PutItemInput
	f := crudFake{putItem: func(_ context.Context, in *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
		captured = in
		return &dynamodb.PutItemOutput{}, nil
	}}
	cond := "attribute_not_exists(id)"
	if err := newWithAPI(f, 1).WriteItem(context.Background(), "t", sk("x"), false, nil, &cond); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured == nil || captured.ConditionExpression == nil {
		t.Fatal("expected condition expression to be applied")
	}
}

func TestDeleteItem(t *testing.T) {
	testutil.Component(t)

	f := crudFake{deleteItem: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
		return &dynamodb.DeleteItemOutput{}, nil
	}}
	if err := newWithAPI(f, 1).DeleteItem(context.Background(), "t", sk("x"), false, nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateItem(t *testing.T) {
	testutil.Component(t)

	f := crudFake{updateItem: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
		return &dynamodb.UpdateItemOutput{Attributes: map[string]types.AttributeValue{"v": &types.AttributeValueMemberN{Value: "2"}}}, nil
	}}
	attrs, err := newWithAPI(f, 1).UpdateItem(
		context.Background(), "t", sk("x"),
		"SET v = :v",
		map[string]types.AttributeValue{":v": &types.AttributeValueMemberN{Value: "2"}},
		nil, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attrs) != 1 {
		t.Fatalf("expected 1 returned attribute, got %d", len(attrs))
	}
}

func TestQueryAll_Paginates(t *testing.T) {
	testutil.Component(t)

	var calls int
	f := crudFake{query: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
		calls++
		if calls == 1 {
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{sk("a")}, LastEvaluatedKey: sk("a")}, nil
		}
		return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{sk("b")}}, nil
	}}
	items, err := newWithAPI(f, 1).QueryAll(context.Background(), QueryInput{TableName: "t", KeyConditionExpression: "id = :id"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items across pages, got %d", len(items))
	}
}

func TestScan(t *testing.T) {
	testutil.Component(t)

	f := crudFake{scan: func(_ context.Context, _ *dynamodb.ScanInput, _ ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
		return &dynamodb.ScanOutput{Items: []map[string]types.AttributeValue{sk("a")}, Count: 1}, nil
	}}
	out, err := newWithAPI(f, 1).Scan(context.Background(), ScanInput{TableName: "t"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(out.Items))
	}
}
