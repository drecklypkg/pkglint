package pkglint

import (
	"net"
	"net/http"
	"syscall"
	"time"
)

// HomepageChecker runs the checks for a HOMEPAGE definition.
//
// When pkglint is in network mode (which has to be enabled explicitly using
// --network), it checks whether the homepage is actually reachable.
//
// The homepage URLs should use https as far as possible.
// To achieve this goal, the HomepageChecker can migrate homepages
// from less preferred URLs to preferred URLs.
//
// For most sites, the list of possible URLs is:
//  - https://$rest (preferred)
//  - http://$rest (less preferred)
//
// For SourceForge, it's a little more complicated:
//  - https://$project.sourceforge.io/$path
//  - http://$project.sourceforge.net/$path
//  - http://$project.sourceforge.io/$path (not officially supported)
//  - https://$project.sourceforge.net/$path (not officially supported)
//  - https://sourceforge.net/projects/$project/
//  - http://sourceforge.net/projects/$project/
//  - https://sf.net/projects/$project/
//  - http://sf.net/projects/$project/
//  - https://sf.net/p/$project/
//  - http://sf.net/p/$project/
//
// TODO: implement complete homepage migration for SourceForge.
// TODO: allow to suppress the automatic migration for SourceForge,
//  even if it is not about https vs. http.
type HomepageChecker struct {
	Value      string
	ValueNoVar string
	MkLine     *MkLine
	MkLines    *MkLines

	// Can be mocked for the tests.
	isReachable func(url string) YesNoUnknown
	Timeout     time.Duration
}

func NewHomepageChecker(value string, valueNoVar string, mkline *MkLine, mklines *MkLines) *HomepageChecker {
	ck := HomepageChecker{value, valueNoVar, mkline, mklines, nil, 3 * time.Second}
	ck.isReachable = ck.isReachableOnline
	return &ck
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

	mkline.Warnf("An FTP URL is not a user-friendly homepage.")
	mkline.Explain(
		"This homepage URL has probably been generated by url2pkg",
		"and not been reviewed by the package author.",
		"",
		"In most cases there exists a more welcoming URL,",
		"which is usually served via HTTP.")
}

func (ck *HomepageChecker) checkHttp() {
	if ck.MkLine.HasRationale("http", "https") {
		return
	}

	migrate, from, to := ck.migrate(ck.Value)
	if !migrate {
		return
	}

	fix := ck.MkLine.Autofix()
	fix.Warnf("HOMEPAGE should migrate from %s to %s.", from, to)
	fix.Replace(from, to)
	if from == "http" {
		fix.Explain(
			"To provide secure communication by default,",
			"the HOMEPAGE URL should use the https protocol if available.",
			"",
			"If the HOMEPAGE really does not support https,",
			"add a comment near the HOMEPAGE variable stating this clearly.")
	}
	fix.Apply()
}

// migrate checks whether the homepage should be migrated from http to https
// and which part of the homepage URL needs to be modified for that.
//
// If for some reason the https URL should not be reachable but the
// corresponding http URL is, the homepage is changed back to http.
func (ck *HomepageChecker) migrate(url string) (bool, string, string) {
	m, scheme, host := match2(url, `(https?)://([A-Za-z0-9-.]+)(?:/|$)?`)
	if !m || containsVarRefLong(url) {
		return false, "", ""
	}

	if scheme == "http" && ck.hasAnySuffix(host,
		"www.gnustep.org",           // no https as of 2020-01-18
		"aspell.net",                // no https as of 2020-01-18
		"downloads.sourceforge.net", // gets another warning already
		"dl.sourceforge.net",        // gets another warning already
	) {
		return false, "", ""
	}

	if scheme == "http" && ck.hasAnySuffix(host,
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
		"tug.org") {
		return true, "http", "https"
	}

	if host == "sf.net" {
		// sf.net redirects to sourceforge.net
		return true, scheme + "://sf.net", "https://sourceforge.net"
	}

	from := scheme
	to := "https"

	// SourceForge projects use either http://project.sourceforge.net or
	// https://project.sourceforge.io (not net).
	if m, project, domain := match2(host, `^([\w-]+)\.((?:sf|sourceforge)\.net)$`); m {

		if scheme == "http" {
			// See https://sourceforge.net/p/forge/documentation/Custom%20VHOSTs
			from = "http://" + host
			to = "https://" + project + ".sourceforge.io"
		} else {
			from = domain
			to = "sourceforge.io"

			// Roll back wrong https SourceForge homepages generated by:
			// https://mail-index.netbsd.org/pkgsrc-changes/2020/01/18/msg205146.html
			_, migrated := replaceOnce(url, from, to)
			if ck.isReachable(migrated) == no {
				_, httpOnly := replaceOnce(url, "https://", "http://")
				if ck.isReachable(httpOnly) == yes && ck.isReachable(url) == no {
					return true, "https", "http"
				}
			}
		}
	}

	if from == to {
		return false, "", ""
	}

	_, migrated := replaceOnce(url, from, to)
	migrate := ck.isReachable(migrated)
	if migrate == yes {
		return true, from, to
	}

	return false, "", ""
}

func (ck *HomepageChecker) checkBadUrls() {
	m, host := match1(ck.Value, `https?://([A-Za-z0-9-.]+)`)
	if !m {
		return
	}

	if !ck.hasAnySuffix(host,
		".dl.sourceforge.net",
		"downloads.sourceforge.net",
		"cpan.metacpan.org") {
		return
	}

	mkline := ck.MkLine
	mkline.Warnf("A direct download URL is not a user-friendly homepage.")
	mkline.Explain(
		"This homepage URL has probably been generated by url2pkg",
		"and not been reviewed by the package author.",
		"",
		"In most cases there exists a more welcoming URL.")
}

func (ck *HomepageChecker) checkReachable() {
	mkline := ck.MkLine
	url := ck.Value

	if !G.Network || url != ck.ValueNoVar {
		return
	}
	if !matches(url, `^https?://[A-Za-z0-9-.]+(?::[0-9]+)?/[!-~]*$`) {
		return
	}

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		mkline.Errorf("Invalid URL %q.", url)
		return
	}

	client := http.Client{
		Timeout: ck.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(request)
	if err != nil {
		networkError := ck.classifyNetworkError(err)
		mkline.Warnf("Homepage %q cannot be checked: %s", url, networkError)
		return
	}
	defer func() { _ = response.Body.Close() }()

	location, err := response.Location()
	if err == nil {
		mkline.Warnf("Homepage %q redirects to %q.", url, location.String())
		return
	}

	if response.StatusCode != 200 {
		mkline.Warnf("Homepage %q returns HTTP status %q.", url, response.Status)
		return
	}
}

func (ck *HomepageChecker) isReachableOnline(url string) YesNoUnknown {
	switch {
	case !G.Network,
		containsVarRefLong(url),
		!matches(url, `^https?://[A-Za-z0-9-.]+(?::[0-9]+)?/[!-~]*$`):
		return unknown
	}

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return no
	}

	client := http.Client{
		Timeout: ck.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return no
	}

	_ = response.Body.Close()
	if response.StatusCode != 200 {
		return no
	}
	return yes
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

func (*HomepageChecker) classifyNetworkError(err error) string {
	cause := err
again:
	if wrapper, ok := cause.(interface{ Unwrap() error }); ok {
		cause = wrapper.Unwrap()
		goto again
	}

	if cause, ok := cause.(*net.DNSError); ok && cause.IsNotFound {
		return "name not found"
	}

	if cause, ok := cause.(syscall.Errno); ok {
		if cause == 10061 || cause == syscall.ECONNREFUSED {
			return "connection refused"
		}
	}

	if cause, ok := cause.(net.Error); ok && cause.Timeout() {
		return "timeout"
	}

	return sprintf("unknown network error: %s", err)
}
