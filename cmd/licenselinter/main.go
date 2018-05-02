/*
Copyright 2017 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Tool for linting vendor/ packages LICENSES and optionally generating METADATA files.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/golang/dep"
	"github.com/pkg/errors"
)

var dir = flag.String("dir", "", "Directory containing dep lock file")
var printDeps = flag.Bool("print-deps", false, "Print vendored deps")
var renameFiles = flag.Bool("rename-files", false, "Rename/merge LICENSE files")
var noRestrictedLicense = flag.Bool("no-restricted-license", true, "Disallow restricted licenses")
var generateMetaFile = flag.Bool("generate-meta-file", false, "Generate METADATA file")

const (
	metadataTxt = `name: "{{.Name}}"
description: "{{.Description}}"

third_party {
  url {
    type: GIT
    value: "{{.URL}}"
  }
  version: "{{.Version}}"
  last_upgrade_date { year: 2018 month: 4 day: 25 }
  license_type: {{.License}}
}
`
	sep = "\n\n-----------------------------------------------------------------------\n\n"

	// MIT license.
	MIT = "Permission is hereby granted, free of charge, to any person"
	// BSD3clause matches BSD 3-clause licence.
	BSD3clause = "Redistribution and use in source and binary forms, with or without"
)

const (
	notice licenseType = iota
	reciprocal
	restricted
)

var (
	licenseCategories = map[string]licenseType{
		MIT:                      notice,
		BSD3clause:               notice,
		"Apache License":         notice,
		"ISC License":            notice,
		"Mozilla Public License": reciprocal,
		"LGPLv3":                 restricted,
	}

	descRe = regexp.MustCompile(`(?m)^([A-Z](.+\n?)+)[.]?\s`)

	metadataTemplate = template.Must(template.New("metadata").Parse(metadataTxt))
)

type licenseType int

func (l licenseType) String() string {
	switch l {
	case notice:
		return "NOTICE"
	case restricted:
		return "RESTRICTED"
	case reciprocal:
		return "RECIPROCAL"
	default:
		panic("unknown license type")
	}
}

func (l licenseType) Disallowed() bool {
	return l == restricted
}

type metadata struct {
	Name, Root, AbsPath, URL, Version, Description, License string
}

func (m *metadata) String() string {
	return fmt.Sprintf("%s%s%sVersion:%s\nLicense:%s\nDescription:%q", sep, m.Root, sep, m.Version, m.License, m.Description)
}

func (m *metadata) populateDescription() error {
	content, p, err := getREADME(m)
	if err != nil {
		fmt.Println(errors.Wrapf(err, "Error getting README for %s\n", m.Root))
		return nil
	}
	d := descRe.Find(content)
	if d == nil {
		return errors.Errorf("No snippet found in %s %s\n", p, m.URL)
	}
	m.Description = strings.Replace(strings.TrimSpace(string(d[:])), "\n", " ", -1)
	return nil
}

func getREADME(m *metadata) ([]byte, string, error) {
	// Search local filesystem
	matches, err := filepath.Glob(path.Join(m.AbsPath, "/README*"))
	if err != nil {
		return nil, "", err
	}
	if len(matches) > 0 {
		content, err2 := ioutil.ReadFile(matches[0])
		if err != nil {
			return nil, "", err2
		}
		return content, matches[0], nil
	}

	// Sometimes dep doesn't pull README files, so try to download it
	url := strings.Replace(m.URL, "code.googlesource.com/gocloud", "github.com/GoogleCloudPlatform/google-cloud-go", 1)
	url = strings.Replace(url, "go.googlesource.com", "github.com/golang", 1)
	if !strings.Contains(url, "github.com") {
		return nil, "", errors.Errorf("No README found in %s", url)
	}
	url = fmt.Sprintf("https://raw.githubusercontent.com/%s/master/README.md", strings.SplitAfterN(url, "/", 4)[3])
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		url = strings.Replace(url, "README.md", "README", 1)
		resp, err = http.Get(url)
		if err != nil || resp.StatusCode != http.StatusOK {
			return nil, "", errors.Errorf("fail to GET README[.md] at %s", url)
		}
	}
	// nolint:errcheck
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return content, url, nil
}

type linter struct {
	dir   string
	metas []*metadata
}

func (l *linter) readDepLock() error {
	ctx := &dep.Ctx{
		Out: log.New(os.Stdout, "", 0),
		Err: log.New(os.Stderr, "", 0),
	}
	gopath, ok := os.LookupEnv("GOPATH")
	if !ok {
		return errors.New("must set GOPATH")
	}
	if err := ctx.SetPaths(l.dir, gopath); err != nil {
		return err
	}
	p, err := ctx.LoadProject()
	if err != nil {
		return err
	}
	sm, err := ctx.SourceManager()
	if err != nil {
		return err
	}
	for _, lp := range p.Lock.Projects() {
		var m metadata
		m.Version = lp.Version().String()
		m.Root = string(lp.Ident().ProjectRoot)
		m.Name = path.Base(m.Root)
		m.AbsPath = path.Join(l.dir, "vendor", m.Root)
		urls, err := sm.SourceURLsForPath(m.Root)
		if err != nil {
			return err
		}
		m.URL = urls[0].String()
		l.metas = append(l.metas, &m)
	}
	return nil
}

func (l *linter) detectLicenseType() error {
	for _, m := range l.metas {
		matches, err := filepath.Glob(path.Join(m.AbsPath, "/LICENSE*"))
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			return errors.Errorf("No license found in %s", m.AbsPath)
		}
		var resultType licenseType
		for i, f := range matches {
			t, err := classifyLicense(f)
			if err != nil {
				return err
			}
			if i == 0 {
				resultType = t
			} else if t < resultType {
				resultType = t
			}
		}
		if resultType.Disallowed() && *noRestrictedLicense {
			return errors.Errorf("Licence type %q not allowed in %s", resultType, m.AbsPath)
		}
		m.License = resultType.String()
		if err := fixLicenseFiles(matches); err != nil {
			return err
		}
	}
	return nil
}

func (l *linter) generateMetaFile() error {
	for _, m := range l.metas {
		p := path.Join(m.AbsPath, "METADATA")
		_, err := os.Stat(p)
		if err == nil {
			continue
		}
		if !os.IsNotExist(err) {
			return err
		}
		if !*generateMetaFile {
			return errors.Errorf("missing METADATA file, rerun with -generate-meta-file: %s", p)
		}
		if err = m.populateDescription(); err != nil {
			return err
		}
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		// nolint:errcheck
		defer f.Close()
		if err := metadataTemplate.Execute(f, m); err != nil {
			return err
		}
	}
	return nil
}

func fixLicenseFiles(licenses []string) error {
	correctPath := path.Join(path.Dir(licenses[0]), "LICENSE")
	if len(licenses) == 1 && licenses[0] == correctPath {
		return nil
	}

	if !*renameFiles {
		return errors.New("need to fix license files, rerun with -rename-files")
	}
	var content []string
	for _, l := range licenses {
		c, err := ioutil.ReadFile(l)
		if err != nil {
			return err
		}
		content = append(content, string(c[:]))
		if err := os.Remove(l); err != nil {
			return err
		}
	}
	return ioutil.WriteFile(correctPath, []byte(strings.Join(content, sep)), 0644)
}

func classifyLicense(f string) (licenseType, error) {
	content, err := ioutil.ReadFile(f)
	if err != nil {
		return 0, err
	}

	for k, v := range licenseCategories {
		if bytes.Contains(content, []byte(k)) {
			return v, nil
		}
	}
	return 0, errors.Errorf("unrecognized licence: %s", f)
}

func main() {
	flag.Parse()

	l := linter{dir: *dir}
	if err := l.readDepLock(); err != nil {
		log.Fatal(err)
	}
	if err := l.detectLicenseType(); err != nil {
		log.Fatal(err)
	}
	if err := l.generateMetaFile(); err != nil {
		log.Fatal(err)
	}

	if *printDeps {
		for _, m := range l.metas {
			fmt.Println(m)
		}
	}
}
