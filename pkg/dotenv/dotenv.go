package dotenv

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Load reads a .env file in the current working directory when the file exists
// and merges variables into the process environment. Variables already set in
// the environment are not overwritten (same semantics as godotenv.Load).
func Load() {
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	path := filepath.Join(wd, ".env")
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return
	}
	if err := godotenv.Load(path); err != nil {
		log.Printf("dotenv: load %s: %v", path, err)
	}
}
