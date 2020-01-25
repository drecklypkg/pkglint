package pkglint

import (
	"net/http"
	"time"
)

type HomepageChecker struct {
	Value      string
	ValueNoVar string
	MkLine     *MkLine
	MkLines    *MkLines
}

func NewHomepageChecker(value string, valueNoVar string, mkline *MkLine, mklines *MkLines) *HomepageChecker {
	return &HomepageChecker{value, valueNoVar, mkline, mklines}
}

func (ck *HomepageChecker) Check() {
	ck.checkBasedOnMasterSites()
	ck.checkFtp()
	ck.checkHttp()
	ck.checkBadUrls()
	ck.checkReachable()
}

func (ck *HomepageChecker) checkBasedOnMasterSites() {
	m, wrong, sitename, subdir := match3(ck.Value, `^(\$\{(MASTER_SITE\w+)(?::=([\w\-/]+))?\})`)
	if !m {
		return
	}

	baseURL := G.Pkgsrc.MasterSiteVarToURL[sitename]
	if sitename == "MASTER_SITES" && ck.MkLines.pkg != nil {
		mkline := ck.MkLines.pkg.vars.FirstDefinition("MASTER_SITES")
		if mkline != nil {
			if !containsVarUse(mkline.Value()) {
				masterSites := ck.MkLine.ValueFields(mkline.Value())
				if len(masterSites) > 0 {
					baseURL = masterSites[0]
				}
			}
		}
	}

	fixedURL := baseURL + subdir

	fix := ck.MkLine.Autofix()
	if baseURL != "" {
		// TODO: Don't suggest any of checkBadUrls.
		fix.Warnf("HOMEPAGE should not be defined in terms of MASTER_SITEs. Use %s directly.", fixedURL)
	} else {
		fix.Warnf("HOMEPAGE should not be defined in terms of MASTER_SITEs.")
	}
	fix.Explain(
		"The HOMEPAGE is a single URL, while MASTER_SITES is a list of URLs.",
		"As long as this list has exactly one element, this works, but as",
		"soon as another site is added, the HOMEPAGE would not be a valid",
		"URL anymore.",
		"",
		"Defining MASTER_SITES=${HOMEPAGE} is ok, though.")
	if baseURL != "" {
		fix.Replace(wrong, fixedURL)
	}
	fix.Apply()
}

func (ck *HomepageChecker) checkFtp() {
	if !hasPrefix(ck.Value, "ftp://") {
		return
	}

	mkline := ck.MkLine
	if mkline.HasRationale("ftp", "FTP", "http", "https", "HTTP") {
		return
	}

	mkline.Warnf("An FTP URL does not represent a user-friendly homepage.")
	mkline.Explain(
		"This homepage URL has probably been generated by url2pkg",
		"and not been reviewed by the package author.",
		"",
		"In most cases there exists a more welcoming URL,",
		"which is usually served via HTTP.")
}

func (ck *HomepageChecker) checkHttp() {
	m, host := match1(ck.Value, `http://([A-Za-z0-9-.]+)`)
	if !m {
		return
	}

	if ck.MkLine.HasRationale("http", "https") {
		return
	}

	// Don't warn about sites that don't support https at all.
	if ck.hasAnySuffix(host,
		"www.gnustep.org", // 2020-01-18
		"aspell.net",      // 2020-01-18
	) {
		return
	}

	if ck.hasAnySuffix(host, ".sf.net", ".sourceforge.net") {
		// Exclude SourceForge subdomains since each of these projects
		// must migrate to https manually and individually.
		// As of January 2020, only around 50% of the projects have done that.
		return
	}

	supportsHttps := ck.hasAnySuffix(host,
		"apache.org",
		"archive.org",
		"ctan.org",
		"freedesktop.org",
		"github.com",
		"github.io",
		"gnome.org",
		"gnu.org",
		"kde.org",
		"kldp.net",
		"linuxfoundation.org",
		"NetBSD.org",
		"nongnu.org",
		"tryton.org",
		"tug.org")

	from := "http"
	to := "https"
	if host == "sf.net" {
		from = "http://sf.net"
		to = "https://sourceforge.net"
	}

	fix := ck.MkLine.Autofix()
	fix.Warnf("HOMEPAGE should migrate from %s to %s.", from, to)
	if supportsHttps {
		fix.Replace(from, to)
	}
	fix.Explain(
		"To provide secure communication by default,",
		"the HOMEPAGE URL should use the https protocol if available.",
		"",
		"If the HOMEPAGE really does not support https,",
		"add a comment near the HOMEPAGE variable stating this clearly.")
	fix.Apply()
}

func (ck *HomepageChecker) checkBadUrls() {
	// TODO: Warn about dl.sourceforge.net
	// TODO: Warn about downloads.sourceforge.net
}

func (ck *HomepageChecker) checkReachable() {
	mkline := ck.MkLine

	if !G.Opts.Network || ck.Value != ck.ValueNoVar {
		return
	}
	if !matches(ck.Value, `^https?://[A-Za-z0-9-.]+(?::[0-9]+)?/[!-~]*$`) {
		return
	}

	var client http.Client
	client.Timeout = 3 * time.Second
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	request, err := http.NewRequest("GET", ck.Value, nil)
	if err != nil {
		mkline.Errorf("Invalid URL %q.", ck.Value)
		return
	}
	response, err := client.Do(request)
	if err != nil {
		mkline.Warnf("Do error: %s", err)
		return
	}
	location, err := response.Location()
	if err == nil {
		mkline.Warnf("Status: %s, location: %s", response.Status, location.String())
		return
	}
	if response.StatusCode != 200 {
		mkline.Warnf("Status: %s", response.Status)
		return
	}
}

func (*HomepageChecker) hasAnySuffix(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if hasSuffix(s, suffix) {
			dotIndex := len(s) - len(suffix)
			if dotIndex == 0 || s[dotIndex-1] == '.' || suffix[0] == '.' {
				return true
			}
		}
	}
	return false
}
