package connect

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

type fakeConnectAPI struct {
	outboundIn  *connect.StartOutboundVoiceContactInput
	outboundOut *connect.StartOutboundVoiceContactOutput
	outboundErr error

	getAttrIn  *connect.GetContactAttributesInput
	getAttrOut *connect.GetContactAttributesOutput
	getAttrErr error

	updateAttrIn  *connect.UpdateContactAttributesInput
	updateAttrOut *connect.UpdateContactAttributesOutput
	updateAttrErr error

	listFlowsIn  *connect.ListContactFlowsInput
	listFlowsOut *connect.ListContactFlowsOutput
	listFlowsErr error
}

func (f *fakeConnectAPI) StartOutboundVoiceContact(_ context.Context, in *connect.StartOutboundVoiceContactInput, _ ...func(*connect.Options)) (*connect.StartOutboundVoiceContactOutput, error) {
	f.outboundIn = in
	return f.outboundOut, f.outboundErr
}

func (f *fakeConnectAPI) GetContactAttributes(_ context.Context, in *connect.GetContactAttributesInput, _ ...func(*connect.Options)) (*connect.GetContactAttributesOutput, error) {
	f.getAttrIn = in
	return f.getAttrOut, f.getAttrErr
}

func (f *fakeConnectAPI) UpdateContactAttributes(_ context.Context, in *connect.UpdateContactAttributesInput, _ ...func(*connect.Options)) (*connect.UpdateContactAttributesOutput, error) {
	f.updateAttrIn = in
	return f.updateAttrOut, f.updateAttrErr
}

func (f *fakeConnectAPI) ListContactFlows(_ context.Context, in *connect.ListContactFlowsInput, _ ...func(*connect.Options)) (*connect.ListContactFlowsOutput, error) {
	f.listFlowsIn = in
	return f.listFlowsOut, f.listFlowsErr
}

// ── Unit Tests ───────────────────────────────────────────────────────────────

func TestNew_MissingRegion(t *testing.T) {
	_, err := New(context.Background(), Config{})
	if err == nil {
		t.Fatal("expected error for missing region, got nil")
	}
	if err == nil || err.Error() != "missing region" {
		t.Errorf("got %v, want \"missing region\"", err)
	}
}

func TestStartOutboundVoiceContact_HappyPath(t *testing.T) {
	fake := &fakeConnectAPI{
		outboundOut: &connect.StartOutboundVoiceContactOutput{
			ContactId: aws.String("contact-123"),
		},
	}
	c := newWithAPI(fake)

	contactID, err := c.StartOutboundVoiceContact(context.Background(), OutboundVoiceContactInput{
		InstanceID:       "instance-abc",
		ContactFlowID:    "flow-def",
		DestinationPhone: "+15551234567",
		SourcePhone:      "+15557654321",
		Attributes:       map[string]string{"foo": "bar"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contactID != "contact-123" {
		t.Errorf("contactID = %q, want %q", contactID, "contact-123")
	}

	in := fake.outboundIn
	if aws.ToString(in.InstanceId) != "instance-abc" {
		t.Errorf("InstanceId = %q, want %q", aws.ToString(in.InstanceId), "instance-abc")
	}
	if aws.ToString(in.ContactFlowId) != "flow-def" {
		t.Errorf("ContactFlowId = %q, want %q", aws.ToString(in.ContactFlowId), "flow-def")
	}
	if aws.ToString(in.DestinationPhoneNumber) != "+15551234567" {
		t.Errorf("DestinationPhoneNumber = %q, want %q", aws.ToString(in.DestinationPhoneNumber), "+15551234567")
	}
	if aws.ToString(in.SourcePhoneNumber) != "+15557654321" {
		t.Errorf("SourcePhoneNumber = %q, want %q", aws.ToString(in.SourcePhoneNumber), "+15557654321")
	}
	if in.Attributes["foo"] != "bar" {
		t.Errorf("Attributes[foo] = %q, want %q", in.Attributes["foo"], "bar")
	}
}

func TestStartOutboundVoiceContact_SDKError(t *testing.T) {
	sdkErr := errors.New("connect unavailable")
	fake := &fakeConnectAPI{outboundErr: sdkErr}
	c := newWithAPI(fake)

	_, err := c.StartOutboundVoiceContact(context.Background(), OutboundVoiceContactInput{
		InstanceID:       "i",
		ContactFlowID:    "f",
		DestinationPhone: "+1",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sdkErr) {
		t.Errorf("error chain does not wrap SDK error: %v", err)
	}
}

func TestStartOutboundVoiceContact_ValidationErrors(t *testing.T) {
	c := newWithAPI(&fakeConnectAPI{})

	cases := []struct {
		name  string
		input OutboundVoiceContactInput
	}{
		{"missing InstanceID", OutboundVoiceContactInput{ContactFlowID: "f", DestinationPhone: "+1"}},
		{"missing ContactFlowID", OutboundVoiceContactInput{InstanceID: "i", DestinationPhone: "+1"}},
		{"missing DestinationPhone", OutboundVoiceContactInput{InstanceID: "i", ContactFlowID: "f"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := c.StartOutboundVoiceContact(context.Background(), tc.input)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestGetContactAttributes_HappyPath(t *testing.T) {
	fake := &fakeConnectAPI{
		getAttrOut: &connect.GetContactAttributesOutput{
			Attributes: map[string]string{"key1": "val1", "key2": "val2"},
		},
	}
	c := newWithAPI(fake)

	attrs, err := c.GetContactAttributes(context.Background(), "inst-1", "contact-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attrs["key1"] != "val1" {
		t.Errorf("attrs[key1] = %q, want %q", attrs["key1"], "val1")
	}
	if attrs["key2"] != "val2" {
		t.Errorf("attrs[key2] = %q, want %q", attrs["key2"], "val2")
	}

	if aws.ToString(fake.getAttrIn.InstanceId) != "inst-1" {
		t.Errorf("InstanceId = %q, want %q", aws.ToString(fake.getAttrIn.InstanceId), "inst-1")
	}
	if aws.ToString(fake.getAttrIn.InitialContactId) != "contact-1" {
		t.Errorf("InitialContactId = %q, want %q", aws.ToString(fake.getAttrIn.InitialContactId), "contact-1")
	}
}

func TestGetContactAttributes_SDKError(t *testing.T) {
	sdkErr := errors.New("not found")
	fake := &fakeConnectAPI{getAttrErr: sdkErr}
	c := newWithAPI(fake)

	_, err := c.GetContactAttributes(context.Background(), "i", "c")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sdkErr) {
		t.Errorf("error chain does not wrap SDK error: %v", err)
	}
}

func TestUpdateContactAttributes_HappyPath(t *testing.T) {
	fake := &fakeConnectAPI{
		updateAttrOut: &connect.UpdateContactAttributesOutput{},
	}
	c := newWithAPI(fake)

	err := c.UpdateContactAttributes(context.Background(), "inst-1", "contact-1", map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	in := fake.updateAttrIn
	if aws.ToString(in.InstanceId) != "inst-1" {
		t.Errorf("InstanceId = %q, want %q", aws.ToString(in.InstanceId), "inst-1")
	}
	if aws.ToString(in.InitialContactId) != "contact-1" {
		t.Errorf("InitialContactId = %q, want %q", aws.ToString(in.InitialContactId), "contact-1")
	}
	if in.Attributes["foo"] != "bar" {
		t.Errorf("Attributes[foo] = %q, want %q", in.Attributes["foo"], "bar")
	}
}

func TestUpdateContactAttributes_SDKError(t *testing.T) {
	sdkErr := errors.New("update failed")
	fake := &fakeConnectAPI{updateAttrErr: sdkErr}
	c := newWithAPI(fake)

	err := c.UpdateContactAttributes(context.Background(), "i", "c", map[string]string{"k": "v"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sdkErr) {
		t.Errorf("error chain does not wrap SDK error: %v", err)
	}
}

func TestListContactFlows_HappyPath(t *testing.T) {
	fake := &fakeConnectAPI{
		listFlowsOut: &connect.ListContactFlowsOutput{
			ContactFlowSummaryList: []types.ContactFlowSummary{
				{
					Id:              aws.String("flow-1"),
					Arn:             aws.String("arn:aws:connect:::flow-1"),
					Name:            aws.String("Inbound Flow"),
					ContactFlowType: types.ContactFlowTypeContactFlow,
				},
				{
					Id:              aws.String("flow-2"),
					Arn:             aws.String("arn:aws:connect:::flow-2"),
					Name:            aws.String("Transfer Flow"),
					ContactFlowType: types.ContactFlowTypeContactFlow,
				},
			},
		},
	}
	c := newWithAPI(fake)

	flows, err := c.ListContactFlows(context.Background(), "inst-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flows) != 2 {
		t.Fatalf("len(flows) = %d, want 2", len(flows))
	}
	if flows[0].ID != "flow-1" {
		t.Errorf("flows[0].ID = %q, want %q", flows[0].ID, "flow-1")
	}
	if flows[0].ARN != "arn:aws:connect:::flow-1" {
		t.Errorf("flows[0].ARN = %q, want %q", flows[0].ARN, "arn:aws:connect:::flow-1")
	}
	if flows[0].Name != "Inbound Flow" {
		t.Errorf("flows[0].Name = %q, want %q", flows[0].Name, "Inbound Flow")
	}
	if flows[0].Type != string(types.ContactFlowTypeContactFlow) {
		t.Errorf("flows[0].Type = %q, want %q", flows[0].Type, string(types.ContactFlowTypeContactFlow))
	}
	if aws.ToString(fake.listFlowsIn.InstanceId) != "inst-1" {
		t.Errorf("InstanceId = %q, want %q", aws.ToString(fake.listFlowsIn.InstanceId), "inst-1")
	}
}

func TestListContactFlows_SDKError(t *testing.T) {
	sdkErr := errors.New("list failed")
	fake := &fakeConnectAPI{listFlowsErr: sdkErr}
	c := newWithAPI(fake)

	_, err := c.ListContactFlows(context.Background(), "inst-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sdkErr) {
		t.Errorf("error chain does not wrap SDK error: %v", err)
	}
}
