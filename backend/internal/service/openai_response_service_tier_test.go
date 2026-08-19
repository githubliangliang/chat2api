package service

import "testing"

func TestExtractOpenAIResponseServiceTierPrefersProviderDeclaration(t *testing.T) {
	got := extractOpenAIResponseServiceTierFromJSONBytes([]byte(`{"service_tier":"default","usage":{"input_tokens":1}}`))
	if got == nil || *got != "default" {
		t.Fatalf("service tier = %v, want default", got)
	}
}

func TestExtractOpenAIResponseServiceTierReadsResponsesEnvelope(t *testing.T) {
	got := extractOpenAIResponseServiceTierFromJSONBytes([]byte(`{"type":"response.completed","response":{"service_tier":"flex","usage":{"input_tokens":1}}}`))
	if got == nil || *got != "flex" {
		t.Fatalf("service tier = %v, want flex", got)
	}
}

func TestExtractOpenAIResponseServiceTierReturnsNilWhenAbsent(t *testing.T) {
	if got := extractOpenAIResponseServiceTierFromJSONBytes([]byte(`{"usage":{"input_tokens":1}}`)); got != nil {
		t.Fatalf("service tier = %v, want nil", got)
	}
}
