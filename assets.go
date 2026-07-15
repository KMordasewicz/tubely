package main

import (
	"fmt"
	"os"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func (cfg apiConfig) getVideoURL(fileName string) string {
	return fmt.Sprintf(
		"%s,%s",
		cfg.s3Bucket,
		fileName,
	)
}
