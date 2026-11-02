package llm

// Backend mirrors Java LlmBackendSelector.Backend.
type Backend string

// Backends parity.
const (
	BackendLocal   Backend = "local"
	BackendPiSSH   Backend = "pi_ssh"
	BackendOpenAI  Backend = "openai"
	BackendOllama  Backend = "ollama"
	BackendLaya    Backend = "laya"
	BackendPiAgent Backend = "pi_agent"
)

// BackendFromKey parity.
func BackendFromKey(key string) Backend {
	switch key {
	case "local":
		return BackendLocal
	case "pi_ssh":
		return BackendPiSSH
	case "openai", "gpuhub":
		return BackendOpenAI
	case "ollama":
		return BackendOllama
	case "laya":
		return BackendLaya
	case "pi_agent":
		return BackendPiAgent
	default:
		return BackendLocal
	}
}
