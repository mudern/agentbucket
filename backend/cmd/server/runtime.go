package main

func supportedRuntimes() []string {
	return []string{"codex", "claudecode", "opencode"}
}

func isSupportedRuntime(runtime string) bool {
	for _, item := range supportedRuntimes() {
		if item == runtime {
			return true
		}
	}
	return false
}
