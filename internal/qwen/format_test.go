package qwen

import "testing"

func TestFormatQuery(t *testing.T) {
	want := "<|im_start|>system\n" +
		"Judge whether the Document meets the requirements based on the Query and the Instruct provided. Note that the answer can only be \"yes\" or \"no\"." +
		"<|im_end|>\n<|im_start|>user\n" +
		"<Instruct>: find relevant passages\n<Query>: what is OpenVINO?\n"

	if got := FormatQuery("find relevant passages", "what is OpenVINO?"); got != want {
		t.Fatalf("FormatQuery() = %q, want %q", got, want)
	}
}

func TestFormatDocument(t *testing.T) {
	want := "<Document>: OpenVINO is an inference toolkit." +
		"<|im_end|>\n<|im_start|>assistant\n<think>\n\n</think>\n\n"

	if got := FormatDocument("OpenVINO is an inference toolkit."); got != want {
		t.Fatalf("FormatDocument() = %q, want %q", got, want)
	}
}
