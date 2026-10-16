package main

const version = "2.0.4"

func versionRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--version" {
			return true
		}
	}
	return false
}
