package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func syncSection(uri, arch, sectionName, rootDir string) error {
	sectionDir := filepath.Join(rootDir, arch, sectionName)
	if mkdirErr := os.MkdirAll(sectionDir, 0755); mkdirErr != nil {
		return mkdirErr
	}

	dbArc := fmt.Sprintf("%s.db.tar.gz", sectionName)
	dbFiles := []string{
		dbArc,
		fmt.Sprintf("%s.files.tar.gz", sectionName),
	}
	baseUrl := formatUrl(uri, arch, sectionName)

	var downErr error
	if downErr = downloadFiles(baseUrl, sectionDir, dbFiles); downErr != nil {
		return downErr
	}

	allPkgs, loadErr := loadDescFromDB(filepath.Join(sectionDir, dbArc))
	if loadErr != nil {
		return loadErr
	}

	updatedOk := false
	for attempt := 1; attempt <= 2; attempt++ {
		needUpdPkgs, checkErr := getPkgsToUpdate(sectionDir, allPkgs)
		if checkErr != nil {
			return checkErr
		}
		if len(needUpdPkgs) == 0 {
			updatedOk = true
			break
		}
		defPrinter.info("Updating packages...")
		names := namesFromDescs(needUpdPkgs)
		if downErr = downloadFiles(baseUrl, sectionDir, names); downErr != nil {
			return nil
		}
	}
	if !updatedOk {
		return fmt.Errorf("unable to update packages, all attempts failed")
	}

	if rmErr := removeRedundantFiles(sectionDir, sectionName, allPkgs); rmErr != nil {
		return rmErr
	}
	return fixupSymlinks(sectionDir, sectionName)
}

func syncLocalMirror() error {
	var beQuiet bool
	var cfgPath string
	var rootDir string
	var mirrorNames string
	var listMirrors bool

	flag.BoolVar(&beQuiet, "quiet", false, "quiet mode")
	flag.StringVar(&cfgPath, "config", "~/.config/amt.toml", "config file path")
	flag.StringVar(&rootDir, "rootdir", "", "root directory (read from config, use current if not set)")
	flag.StringVar(&mirrorNames, "mirrors", "", "mirrors (read from config, use enabled if set)")
	flag.BoolVar(&listMirrors, "list", false, "list configured mirrors and quit")
	flag.Parse()

	if beQuiet {
		defPrinter.setQuiet()
	}

	cfg, cfgErr := readConfig(cfgPath)
	if cfgErr != nil {
		return cfgErr
	}

	if listMirrors {
		if len(cfg.Mirrors) > 0 {
			fmt.Println("Hint: Enabled mirrors are marked with '*'.")
		}
		for name, mirror := range cfg.Mirrors {
			mark := ' '
			if mirror.Enabled {
				mark = '*'
			}
			fmt.Printf(
				"%c%s {\n\turi: %s\n\tarch: %s\n\tsections: [%s]\n}\n",
				mark, name, mirror.Uri, mirror.Arch, strings.Join(mirror.Sections, ","),
			)
		}
		return nil
	}

	if rootDir == "" {
		rootDir = cfg.RootDir
		if rootDir == "" {
			var dirErr error
			rootDir, dirErr = os.Getwd()
			if dirErr != nil {
				return dirErr
			}
		}
	}

	enabledNames := make([]string, 0)
	if mirrorNames == "" {
		for name, mirror := range cfg.Mirrors {
			if mirror.Enabled {
				enabledNames = append(enabledNames, name)
			}
		}
	} else {
		for _, name := range strings.Split(mirrorNames, ",") {
			name = strings.TrimSpace(strings.ToLower(name))
			_, found := cfg.Mirrors[name]
			if !found {
				return fmt.Errorf("mirror '%s' not found in config: %s", name, cfgPath)
			}
			enabledNames = append(enabledNames, name)
		}
	}
	enabledCount := len(enabledNames)
	if enabledCount == 0 {
		return fmt.Errorf("no enabled mirrors found in config '%s'", cfgPath)
	}

	defPrinter.info("Using '%s' as a root directory.", rootDir)
	for midx, name := range enabledNames {
		mirror := cfg.Mirrors[name]
		for sidx, section := range mirror.Sections {
			defPrinter.info(
				"Syncing section '%s' (%d/%d), mirror '%s'@%s (%d/%d)...",
				section, sidx+1, len(mirror.Sections),
				name, mirror.Arch, midx+1, enabledCount,
			)
			if syncErr := syncSection(mirror.Uri, mirror.Arch, section, rootDir); syncErr != nil {
				return syncErr
			}
			defPrinter.info("Syncing section '%s', mirror '%s': done.", section, name)
		}
	}

	if tsErr := mkLastUpdateStamp(rootDir); tsErr != nil {
		return tsErr
	}
	defPrinter.info("Local packages synced successfully.")
	return nil
}

func main() {
	if err := syncLocalMirror(); err != nil {
		defPrinter.error("Unable to sync local packages: %s.", err)
		os.Exit(1)
	}
}
