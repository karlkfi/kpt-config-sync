// Tool for linting vendor/ packages LICENSES and optionally generating METADATA files.
//
// The METADATA files and unified LICENSE files that this generates are required for all Google
// code with third-party dependencies. See
// https://g3doc.corp.google.com/company/thirdparty/non-google3.md?cl=head.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/golang/dep"
	"github.com/pkg/errors"
)

var dir = flag.String("dir", "", "Directory containing dep lock file")
var noRestrictedLicense = flag.Bool("no-restricted-license", true, "Disallow restricted licenses")
var printDeps = flag.Bool("print-deps", false, "Print vendored deps")
var printAggregate = flag.Bool("print-aggregate", false, "Prints an aggregate LICENSE file for embedding in an image")

const (
	sep = "\n\n-----------------------------------------------------------------------\n\n"

	// MIT license.
	MIT = "Permission is hereby granted, free of charge, to any person"
	// BSD3clause matches BSD 3-clause licence.
	BSD3clause = "Redistribution and use in source and binary forms, with or without"
	// CCBySA40 matches CreativeCommons Attribution-ShareAlike 4.0 license.
	CCBySA40 = "Attribution-ShareAlike 4.0 International"
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
		CCBySA40:                 restricted,
	}
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

func (l licenseType) disallowed() bool {
	return l == restricted
}

type metadata struct {
	Name, Root, AbsPath, URL, Version, Description string
	LicenseType                                    licenseType
	// Text of all LICENSE files
	LicenseText [][]byte
}

func (m *metadata) String() string {
	return fmt.Sprintf("%s%s%sVersion:%s\nLicense:%s\nDescription:%q", sep, m.Root, sep, m.Version, m.LicenseType, m.Description)
}

type linter struct {
	dir   string
	metas []*metadata
}

func (l *linter) readDepLock() error {
	ctx := &dep.Ctx{
		Out:            log.New(os.Stdout, "", 0),
		Err:            log.New(os.Stderr, "", 0),
		DisableLocking: true,
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
		matches, err := codeLicenses(m.AbsPath)
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			return errors.Errorf("No license found in %s", m.AbsPath)
		}
		var resultType licenseType
		for i, f := range matches {
			t, content, err := classifyLicense(f)
			if err != nil {
				return err
			}
			if i == 0 {
				resultType = t
			} else if t < resultType {
				resultType = t
			}
			m.LicenseText = append(m.LicenseText, content)
		}
		if resultType.disallowed() && *noRestrictedLicense {
			return errors.Errorf("Licence type %q not allowed in %s", resultType, m.AbsPath)
		}
		m.LicenseType = resultType
	}

	for _, m := range l.metas {
		docMatches, err := filepath.Glob(path.Join(m.AbsPath, "LICENSE.docs"))
		if err != nil {
			return err
		}
		for _, f := range docMatches {
			if _, _, err := classifyLicense(f); err != nil {
				return err
			}
		}
	}
	return nil
}

// codeLicenses gets a list of files in absPath whose names are LICENSE*, but not LICENSE.doc. The
// idea is to get licenses referring to code, not to docs. Docs licenses are uncommon, but they do
// exist.
func codeLicenses(absPath string) ([]string, error) {
	matches, err := filepath.Glob(path.Join(absPath, "LICENSE*"))
	if err != nil {
		return nil, err
	}
	var codeMatches []string
	for _, m := range matches {
		if !strings.HasSuffix(m, "LICENSE.docs") {
			codeMatches = append(codeMatches, m)
		}
	}
	return codeMatches, nil
}

func classifyLicense(f string) (licenseType, []byte, error) {
	content, err := ioutil.ReadFile(f)
	if err != nil {
		return 0, nil, err
	}

	for k, v := range licenseCategories {
		if bytes.Contains(content, []byte(k)) {
			return v, content, nil
		}
	}
	return 0, nil, errors.Errorf("unrecognized licence: %s", f)
}

func aggregateLicenses(metas []*metadata) string {
	var out strings.Builder
	out.WriteString("THE FOLLOWING SETS FORTH ATTRIBUTION NOTICES FOR THIRD PARTY SOFTWARE THAT MAY BE CONTAINED IN PORTIONS OF THE ANTHOS CONFIG MANAGEMENT PRODUCT.\n")
	for _, m := range metas {
		for _, t := range m.LicenseText {
			out.WriteString("\n-----\n\n")
			out.WriteString(fmt.Sprintf("The following software may be included in this product: %s. This software contains the following license and notice below:\n\n", m.Root))
			out.Write(t)
		}
	}
	return out.String()
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

	if *printDeps {
		for _, m := range l.metas {
			fmt.Println(m)
		}
	}
	if *printAggregate {
		fmt.Println(aggregateLicenses(l.metas))
	}
}
