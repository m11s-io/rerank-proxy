package qwen

const (
	prefix = "<|im_start|>system\n" +
		"Judge whether the Document meets the requirements based on the Query and the Instruct provided. Note that the answer can only be \"yes\" or \"no\"." +
		"<|im_end|>\n<|im_start|>user\n"
	suffix = "<|im_end|>\n<|im_start|>assistant\n<think>\n\n</think>\n\n"
)

// FormatQuery applies the query half of Qwen3's reranker prompt template.
func FormatQuery(instruction, query string) string {
	return prefix + "<Instruct>: " + instruction + "\n<Query>: " + query + "\n"
}

// FormatDocument applies the document half of Qwen3's reranker prompt template.
func FormatDocument(document string) string {
	return "<Document>: " + document + suffix
}
