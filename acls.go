package main

// This file defines the specific type of some variables, according to
// their current use in pkgsrc.
//
// The type YesNo is used for variables that are checked using
//     .if defined(VAR) && !empty(VAR:M[Yy][Ee][Ss])
//
// The type Yes is used for variables that are checked using
//     .if defined(VAR)
//
// The type List is used for lists of things. There are two types of lists,
// InternalList and List, which are described in the pkgsrc guide, chapter
// "Makefiles".
//
// The other types are described in pkglint.pl, checkline_mk_vartype_basic.
//
// Last synced with mk/defaults/mk.conf revision 1.118

var aclVartypes = make(map[string]*Vartype)

func getBasicType(typename string) *Vartype {
	notImplemented("getBasicType")
	return nil
}

func acl(varname string, vartype string, aclentries ...string) {
	m := mustMatch(`^([A-Z_.][A-Z0-9_]*)(|\*|\.\*)$`, varname)
	varbase, varparam := m[1], m[2]
	m = mustMatch(`^(InternalList of |List of |)(?:([\w\d_]+)|\{\s*([\w\d_+,\-.\s]+?)\s*\})$`, vartype)
	list, basicType, enumValues := m[1], m[2], m[3]

	var kindOfList KindOfList
	switch list {
	case "InternalList of ":
		kindOfList = LK_INTERNAL
	case "List of ":
		kindOfList = LK_EXTERNAL
	case "":
		kindOfList = LK_NONE
	default:
		panic(list)
	}

	var vtype *Vartype
	if basicType != "" {
		vtype = newBasicVartype(kindOfList, basicType, parseAclEntries(aclentries), NOT_GUESSED)
	} else {
		vtype = newEnumVartype(kindOfList, enumValues, parseAclEntries(aclentries), NOT_GUESSED)
	}

	if varparam == "" || varparam == "*" {
		aclVartypes[varbase] = vtype
	}
	if varparam == "*" || varparam == ".*" {
		aclVartypes[varbase+".*"] = vtype
	}
}

// A package-defined variable may be set in all Makefiles except buildlink3.mk and builtin.mk.
func pkg(varname, vartype string) {
	acl(varname, vartype, "Makefile:su", "Makefile.common:dsu", "buildlink3.mk:", "builtin.mk:", "*.mk:dsu")
}

// A package-defined list may be appended to in all Makefiles except buildlink3.mk and builtin.mk.
// Simple assignment (instead of appending) is only allowed in Makefile and Makefile.common.
func pkglist(varname, vartype string) {
	acl(varname, vartype, "Makefile:asu", "Makefile.common:asu", "buildlink3.mk:", "builtin.mk:", "*.mk:au")
}

// A user-defined or system-defined variable must not be set by any
// package file. It also must not be used in buildlink3.mk and
// builtin.mk files or at load-time, since the system/user preferences
// may not have been loaded when these files are included.
func sys(varname, vartype string) {
	acl(varname, vartype, "buildlink3.mk:", "builtin.mk:", "*:u")
}
func usr(varname, vartype string) {
	acl(varname, vartype, "buildlink3.mk:", "builtin.mk:", "*:u")
}
func bl3list(varname, vartype string) {
	acl(varname, vartype, "buildlink3.mk:a", "builtin.mk:a")
}
func cmdline(varname, vartype string) {
	acl(varname, vartype, "buildlink3.mk:", "builtin.mk:", "*:pu")
}

func initacls() {
	usr("ALLOW_VULNERABLE_PACKAGES", "Yes")
	usr("MANINSTALL", "List of { maninstall catinstall }")
	usr("MANZ", "Yes")
	usr("GZIP", "List of ShellWord")
	usr("MKCRYPTO", "YesNo")
	usr("OBJHOSTNAME", "Yes")
	usr("OBJMACHINE", "Yes")
	usr("PKG_SUFX", "Filename")
	usr("PKGSRC_LOCKTYPE", "{ none sleep once }")
	usr("PKGSRC_SLEEPSECS", "Integer")
	usr("USETBL", "Yes")
	usr("ABI", "{ 32 64 }")
	usr("PKG_DEVELOPER", "Yes")
	usr("USE_ABI_DEPENDS", "YesNo")
	usr("PKG_REGISTER_SHELLS", "{ YES NO }")
	usr("PKGSRC_COMPILER", "List of { ccache ccc clang distcc f2c gcc hp icc ido gcc mipspro mipspro-ucode pcc sunpro xlc }")
	usr("PKGSRC_MESSAGE_RECIPIENTS", "List of MailAddress")
	usr("PKGSRC_SHOW_BUILD_DEFS", "YesNo")
	usr("PKGSRC_SHOW_PATCH_ERRORMSG", "YesNo")
	usr("PKGSRC_RUN_TEST", "YesNo")
	usr("PREFER_PKGSRC", "List of Identifier")
	usr("PREFER_NATIVE", "List of Identifier")
	usr("PREFER_NATIVE_PTHREADS", "YesNo")
	usr("LOCALBASE", "Pathname")
	usr("CROSSBASE", "Pathname")
	usr("VARBASE", "Pathname")
	usr("X11_TYPE", "{ modular native }")
	usr("X11BASE", "Pathname")
	usr("MOTIFBASE", "Pathname")
	usr("PKGINFODIR", "Pathname")
	usr("PKGMANDIR", "Pathname")
	usr("USE_XPKGWEDGE", "YesNo")
	usr("BSDSRCDIR", "Pathname")
	usr("BSDXSRCDIR", "Pathname")
	usr("DISTDIR", "Pathname")
	usr("DIST_PATH", "Pathlist")
	usr("DEFAULT_VIEW", "Unchecked")
	usr("FETCH_CMD", "ShellCommand")
	usr("FETCH_USING", "{ curl custom fetch ftp manual wget }")
	usr("FETCH_RESUME_ARGS", "List of ShellWord")
	usr("FETCH_OUTPUT_ARGS", "List of ShellWord")
	usr("LIBTOOLIZE_PLIST", "YesNo")
	usr("PKG_RESUME_TRANSFERS", "YesNo")
	usr("PKG_SYSCONFBASE", "Pathname")
	usr("RCD_SCRIPTS_DIR", "Pathname")
	usr("PACKAGES", "Pathname")
	usr("PKGVULNDIR", "Pathname")
	usr("PASSIVE_FETCH", "Yes")
	usr("PATCH_FUZZ_FACTOR", "{ -F0 -F1 -F2 -F3 }")
	usr("ACCEPTABLE_LICENSES", "List of Identifier")
	usr("SPECIFIC_PKGS", "Yes")
	usr("SITE_SPECIFIC_PKGS", "List of PkgPath")
	usr("HOST_SPECIFIC_PKGS", "List of PkgPath")
	usr("GROUP_SPECIFIC_PKGS", "List of PkgPath")
	usr("USER_SPECIFIC_PKGS", "List of PkgPath")
	usr("FAILOVER_FETCH", "Yes")
	usr("MASTER_SORT", "List of Unchecked")
	usr("MASTER_SORT_REGEX", "List of Unchecked")
	usr("PATCH_DEBUG", "Yes")
	usr("PKG_FC", "ShellCommand")
	usr("IMAKE", "ShellCommand")
	usr("IMAKEOPTS", "List of ShellWord")
	usr("PRE_ROOT_CMD", "ShellCommand")
	pkg("USE_GAMESGROUP", "YesNo")
	usr("SU_CMD", "ShellCommand")
	usr("SU_CMD_PATH_APPEND", "Pathlist")
	usr("FATAL_OBJECT_FMT_SKEW", "YesNo")
	usr("WARN_NO_OBJECT_FMT", "YesNo")
	usr("SMART_MESSAGES", "Yes")
	usr("BINPKG_SITES", "List of URL")
	usr("BIN_INSTALL_FLAG", "List of ShellWord")
	usr("LOCALPATCHES", "Pathname")

	// some other variables, sorted alphabetically

	sys(".CURDIR", "Pathname")
	sys(".TARGET", "Pathname")
	acl("ALL_ENV", "List of ShellWord")
	acl("ALTERNATIVES_FILE", "Filename")
	acl("ALTERNATIVES_SRC", "List of Pathname")
	pkg("APACHE_MODULE", "Yes")
	sys("AR", "ShellCommand")
	sys("AS", "ShellCommand")
	pkglist("AUTOCONF_REQD", "List of Version")
	acl("AUTOMAKE_OVERRIDE", "List of Pathmask")
	pkglist("AUTOMAKE_REQD", "List of Version")
	pkg("AUTO_MKDIRS", "YesNo")
	usr("BATCH", "Yes")
	acl("BDB185_DEFAULT", "Unchecked")
	sys("BDBBASE", "Pathname")
	pkg("BDB_ACCEPTED", "List of { db1 db2 db3 db4 db5 }")
	acl("BDB_DEFAULT", "{ db1 db2 db3 db4 db5 }")
	sys("BDB_LIBS", "List of LdFlag")
	sys("BDB_TYPE", "{ db1 db2 db3 db4 db5 }")
	sys("BINGRP", "UserGroupName")
	sys("BINMODE", "FileMode")
	sys("BINOWN", "UserGroupName")
	acl("BOOTSTRAP_DEPENDS", "InternalList of DependencyWithPath", "Makefile.common:a", "Makefile:a", "options.mk:a", "*.mk:a")
	pkg("BOOTSTRAP_PKG", "YesNo")
	acl("BROKEN", "Message")
	pkg("BROKEN_GETTEXT_DETECTION", "YesNo")
	pkglist("BROKEN_EXCEPT_ON_PLATFORM", "List of PlatformTriple")
	pkglist("BROKEN_ON_PLATFORM", "InternalList of PlatformTriple")
	sys("BSD_MAKE_ENV", "List of ShellWord")
	acl("BUILDLINK_ABI_DEPENDS.*", "InternalList of Dependency", "*:a")
	acl("BUILDLINK_API_DEPENDS.*", "InternalList of Dependency", "*:a")
	acl("BUILDLINK_CONTENTS_FILTER", "List of ShellWord")
	// ^^ should rather be ShellCommand
	sys("BUILDLINK_CFLAGS", "List of CFlag")
	bl3list("BUILDLINK_CFLAGS.*", "List of CFlag")
	sys("BUILDLINK_CPPFLAGS", "List of CFlag")
	bl3list("BUILDLINK_CPPFLAGS.*", "List of CFlag")
	acl("BUILDLINK_CONTENTS_FILTER.*", "ShellCommand", "buildlink3.mk:s")
	acl("BUILDLINK_DEPENDS", "InternalList of Identifier", "buildlink3.mk:a")
	acl("BUILDLINK_DEPMETHOD.*", "List of BuildlinkDepmethod", "buildlink3.mk:ad", "Makefile:as", "Makefile.common:a", "*.mk:a")
	// ^^ FIXME: b:d may lead to unexpected behavior.
	acl("BUILDLINK_DEPTH", "BuildlinkDepth", "buildlink3.mk:ps", "builtin.mk:ps")
	sys("BUILDLINK_DIR", "Pathname")
	bl3list("BUILDLINK_FILES.*", "List of Pathmask")
	acl("BUILDLINK_FILES_CMD.*", "List of ShellWord")
	// ^^ ShellCommand
	acl("BUILDLINK_INCDIRS.*", "List of Pathname", "buildlink3.mk:ad") // b:d?
	acl("BUILDLINK_JAVA_PREFIX.*", "Pathname", "buildlink3.mk:s")
	acl("BUILDLINK_LDADD.*", "List of LdFlag", "builtin.mk:adsu", "buildlink3.mk:", "Makefile:u", "Makefile.common:u", "*.mk:u")
	sys("BUILDLINK_LDFLAGS", "List of LdFlag")
	bl3list("BUILDLINK_LDFLAGS.*", "List of LdFlag")
	bl3list("BUILDLINK_LIBDIRS.*", "List of Pathname")
	acl("BUILDLINK_LIBS.*", "List of LdFlag", "buildlink3.mk:a")
	acl("BUILDLINK_PACKAGES", "BuildlinkPackages", "buildlink3.mk:aps")
	acl("BUILDLINK_PASSTHRU_DIRS", "List of Pathname", "Makefile:a", "Makefile.common:a", "buildlink3.mk:a", "hacks.mk:a")
	acl("BUILDLINK_PASSTHRU_RPATHDIRS", "List of Pathname", "Makefile:a", "Makefile.common:a", "buildlink3.mk:a", "hacks.mk:a")
	acl("BUILDLINK_PKGSRCDIR.*", "RelativePkgDir", "buildlink3.mk:dp")
	acl("BUILDLINK_PREFIX.*", "Pathname", "builtin.mk:su", "buildlink3.mk:", "Makefile:u", "Makefile.common:u", "*.mk:u")
	acl("BUILDLINK_RPATHDIRS.*", "List of Pathname", "buildlink3.mk:a")
	acl("BUILDLINK_TARGETS", "List of Identifier")
	acl("BUILDLINK_FNAME_TRANSFORM.*", "SedCommands", "Makefile:a", "builtin.mk:a", "hacks.mk:a", "buildlink3.mk:a")
	acl("BUILDLINK_TRANSFORM", "List of WrapperTransform", "*:a")
	acl("BUILDLINK_TREE", "List of Identifier", "buildlink3.mk:a")
	acl("BUILD_DEFS", "List of Varname", "Makefile:a", "Makefile.common:a", "options.mk:a")
	acl("BUILD_DEPENDS", "InternalList of DependencyWithPath", "Makefile.common:a", "Makefile:a", "options.mk:a", "*.mk:a")
	pkglist("BUILD_DIRS", "List of WrksrcSubdirectory")
	pkglist("BUILD_ENV", "List of ShellWord")
	sys("BUILD_MAKE_CMD", "ShellCommand")
	pkglist("BUILD_MAKE_FLAGS", "List of ShellWord")
	pkg("BUILD_TARGET", "List of Identifier")
	pkg("BUILD_USES_MSGFMT", "Yes")
	acl("BUILTIN_PKG", "Identifier", "builtin.mk:psu")
	acl("BUILTIN_PKG.*", "PkgName", "builtin.mk:psu")
	acl("BUILTIN_FIND_FILES_VAR", "List of Varname", "builtin.mk:s")
	acl("BUILTIN_FIND_FILES.*", "List of Pathname", "builtin.mk:s")
	acl("BUILTIN_FIND_GREP.*", "String", "builtin.mk:s")
	acl("BUILTIN_FIND_LIBS", "List of Pathname", "builtin.mk:s")
	acl("BUILTIN_IMAKE_CHECK", "List of Unchecked", "builtin.mk:s")
	acl("BUILTIN_IMAKE_CHECK.*", "YesNo")
	sys("BUILTIN_X11_TYPE", "Unchecked")
	sys("BUILTIN_X11_VERSION", "Unchecked")
	acl("CATEGORIES", "List of Category", "Makefile:as", "Makefile.common:ads")
	sys("CC_VERSION", "Message")
	sys("CC", "ShellCommand")
	pkglist("CFLAGS*", "List of CFlag")
	// ^^ may also be changed by the user
	acl("CHECK_BUILTIN", "YesNo", "builtin.mk:d", "Makefile:s")
	acl("CHECK_BUILTIN.*", "YesNo", "*:p")
	acl("CHECK_FILES_SKIP", "List of Pathmask", "Makefile:a", "Makefile.common:a")
	pkg("CHECK_FILES_SUPPORTED", "YesNo")
	usr("CHECK_HEADERS", "YesNo")
	pkglist("CHECK_HEADERS_SKIP", "List of Pathmask")
	usr("CHECK_INTERPRETER", "YesNo")
	pkglist("CHECK_INTERPRETER_SKIP", "List of Pathmask")
	usr("CHECK_PERMS", "YesNo")
	pkglist("CHECK_PERMS_SKIP", "List of Pathmask")
	//CHECK_PERMS_AUTOFIX", "YesNo", pkg...)
	// ^^ experimental
	usr("CHECK_PORTABILITY", "YesNo")
	pkglist("CHECK_PORTABILITY_SKIP", "List of Pathmask")
	acl("CHECK_SHLIBS", "YesNo", "Makefile:s")
	pkglist("CHECK_SHLIBS_SKIP", "List of Pathmask")
	acl("CHECK_SHLIBS_SUPPORTED", "YesNo", "Makefile:s")
	pkglist("CHECK_WRKREF_SKIP", "List of Pathmask")
	pkg("CMAKE_ARG_PATH", "Pathname")
	pkglist("CMAKE_ARGS", "List of ShellWord")
	acl("COMMENT", "Comment", "Makefile:as", "Makefile.common:as")
	sys("COMPILER_RPATH_FLAG", "{ -Wl,-rpath }")
	pkglist("CONFIGURE_ARGS", "List of ShellWord")
	pkglist("CONFIGURE_DIRS", "List of WrksrcSubdirectory")
	pkglist("CONFIGURE_ENV", "List of ShellWord")
	pkg("CONFIGURE_HAS_INFODIR", "YesNo")
	pkg("CONFIGURE_HAS_LIBDIR", "YesNo")
	pkg("CONFIGURE_HAS_MANDIR", "YesNo")
	pkg("CONFIGURE_SCRIPT", "Pathname")
	acl("CONFIG_GUESS_OVERRIDE", "List of Pathmask", "Makefile:as", "Makefile.common:as")
	acl("CONFIG_STATUS_OVERRIDE", "List of Pathmask", "Makefile:as", "Makefile.common:as")
	acl("CONFIG_SHELL", "Pathname", "Makefile:s", "Makefile.common:s")
	acl("CONFIG_SUB_OVERRIDE", "List of Pathmask", "Makefile:as", "Makefile.common:as")
	pkglist("CONFLICTS", "InternalList of Dependency")
	pkglist("CONF_FILES", "List of ShellWord")
	pkg("CONF_FILES_MODE", "{ 0644 0640 0600 0400 }")
	pkglist("CONF_FILES_PERMS", "List of ShellWord")
	sys("COPY", "{ -c }")
	// ^^ the flag that tells ${INSTALL} to copy a file
	sys("CPP", "ShellCommand")
	pkglist("CPPFLAGS*", "List of CFlag")
	acl("CRYPTO", "Yes", "Makefile:s")
	sys("CXX", "ShellCommand")
	pkglist("CXXFLAGS*", "List of CFlag")
	acl("DEINSTALL_FILE", "Pathname", "Makefile:s")
	acl("DEINSTALL_SRC", "List of Pathname", "Makefile:s", "Makefile.common:ds")
	acl("DEINSTALL_TEMPLATES", "List of Pathname", "Makefile:as", "Makefile.common:ads")
	sys("DELAYED_ERROR_MSG", "ShellCommand")
	sys("DELAYED_WARNING_MSG", "ShellCommand")
	pkglist("DEPENDS", "InternalList of DependencyWithPath")
	usr("DEPENDS_TARGET", "List of Identifier")
	acl("DESCR_SRC", "List of Pathname", "Makefile:s", "Makefile.common:ds")
	sys("DESTDIR", "Pathname")
	acl("DESTDIR_VARNAME", "Varname", "Makefile:s", "Makefile.common:s")
	sys("DEVOSSAUDIO", "Pathname")
	sys("DEVOSSSOUND", "Pathname")
	pkglist("DISTFILES", "List of Filename")
	pkg("DISTINFO_FILE", "RelativePkgPath")
	pkg("DISTNAME", "Filename")
	pkg("DIST_SUBDIR", "Pathname")
	acl("DJB_BUILD_ARGS", "List of ShellWord")
	acl("DJB_BUILD_TARGETS", "List of Identifier")
	acl("DJB_CONFIG_CMDS", "List of ShellWord", "options.mk:s")
	// ^^ ShellCommand, terminated by a semicolon
	acl("DJB_CONFIG_DIRS", "List of WrksrcSubdirectory")
	acl("DJB_CONFIG_HOME", "Filename")
	acl("DJB_CONFIG_PREFIX", "Pathname")
	acl("DJB_INSTALL_TARGETS", "List of Identifier")
	acl("DJB_MAKE_TARGETS", "YesNo")
	acl("DJB_RESTRICTED", "YesNo", "Makefile:s")
	acl("DJB_SLASHPACKAGE", "YesNo")
	acl("DLOPEN_REQUIRE_PTHREADS", "YesNo")
	acl("DL_AUTO_VARS", "Yes", "Makefile:s", "Makefile.common:s", "options.mk:s")
	acl("DL_LIBS", "List of LdFlag")
	sys("DOCOWN", "UserGroupName")
	sys("DOCGRP", "UserGroupName")
	sys("DOCMODE", "FileMode")
	sys("DOWNLOADED_DISTFILE", "Pathname")
	sys("DO_NADA", "ShellCommand")
	pkg("DYNAMIC_SITES_CMD", "ShellCommand")
	pkg("DYNAMIC_SITES_SCRIPT", "Pathname")
	sys("ECHO", "ShellCommand")
	sys("ECHO_MSG", "ShellCommand")
	sys("ECHO_N", "ShellCommand")
	pkg("EGDIR", "Pathname")
	// ^^ This variable is not defined by the system, but has been established
	// as a convention.
	sys("EMACS_BIN", "Pathname")
	sys("EMACS_ETCPREFIX", "Pathname")
	sys("EMACS_FLAVOR", "{ emacs xemacs }")
	sys("EMACS_INFOPREFIX", "Pathname")
	sys("EMACS_LISPPREFIX", "Pathname")
	acl("EMACS_MODULES", "List of Identifier", "Makefile:as", "Makefile.common:as")
	sys("EMACS_PKGNAME_PREFIX", "Identifier")
	// ^^ or the empty string.
	sys("EMACS_TYPE", "{ emacs xemacs }")
	acl("EMACS_USE_LEIM", "Yes")
	acl("EMACS_VERSIONS_ACCEPTED", "List of { emacs25 emacs24 emacs24nox emacs23 emacs23nox emacs22 emacs22nox emacs21 emacs21nox emacs20 xemacs215 xemacs215nox xemacs214 xemacs214nox}", "Makefile:s")
	sys("EMACS_VERSION_MAJOR", "Integer")
	sys("EMACS_VERSION_MINOR", "Integer")
	acl("EMACS_VERSION_REQD", "List of { emacs24 emacs24nox emacs23 emacs23nox emacs22 emacs22nox emacs21 emacs21nox emacs20 xemacs215 xemacs214 }", "Makefile:as")
	sys("EMULDIR", "Pathname")
	sys("EMULSUBDIR", "Pathname")
	sys("OPSYS_EMULDIR", "Pathname")
	sys("EMULSUBDIRSLASH", "Pathname")
	sys("EMUL_ARCH", "{ i386 none }")
	sys("EMUL_DISTRO", "Identifier")
	sys("EMUL_IS_NATIVE", "Yes")
	pkg("EMUL_MODULES.*", "List of Identifier")
	sys("EMUL_OPSYS", "{ freebsd hpux irix linux osf1 solaris sunos none }")
	pkg("EMUL_PKG_FMT", "{ plain rpm }")
	usr("EMUL_PLATFORM", "EmulPlatform")
	pkg("EMUL_PLATFORMS", "List of EmulPlatform")
	usr("EMUL_PREFER", "List of EmulPlatform")
	pkg("EMUL_REQD", "InternalList of Dependency")
	usr("EMUL_TYPE.*", "{ native builtin suse suse-9.1 suse-9.x suse-10.0 suse-10.x }")
	sys("ERROR_CAT", "ShellCommand")
	sys("ERROR_MSG", "ShellCommand")
	acl("EVAL_PREFIX", "InternalList of ShellWord", "Makefile:a", "Makefile.common:a")
	// ^^ FIXME: Looks like a type mismatch.
	sys("EXPORT_SYMBOLS_LDFLAGS", "List of LdFlag")
	sys("EXTRACT_CMD", "ShellCommand")
	pkg("EXTRACT_DIR", "Pathname")
	pkglist("EXTRACT_ELEMENTS", "List of Pathmask")
	pkglist("EXTRACT_ENV", "List of ShellWord")
	pkglist("EXTRACT_ONLY", "List of Pathname")
	acl("EXTRACT_OPTS", "List of ShellWord", "Makefile:as", "Makefile.common:as")
	acl("EXTRACT_OPTS_BIN", "List of ShellWord", "Makefile:as", "Makefile.common:as")
	acl("EXTRACT_OPTS_LHA", "List of ShellWord", "Makefile:as", "Makefile.common:as")
	acl("EXTRACT_OPTS_PAX", "List of ShellWord", "Makefile:as", "Makefile.common:as")
	acl("EXTRACT_OPTS_RAR", "List of ShellWord", "Makefile:as", "Makefile.common:as")
	acl("EXTRACT_OPTS_TAR", "List of ShellWord", "Makefile:as", "Makefile.common:as")
	acl("EXTRACT_OPTS_ZIP", "List of ShellWord", "Makefile:as", "Makefile.common:as")
	acl("EXTRACT_OPTS_ZOO", "List of ShellWord", "Makefile:as", "Makefile.common:as")
	pkg("EXTRACT_SUFX", "DistSuffix")
	pkg("EXTRACT_USING", "{ bsdtar gtar nbtar pax }")
	sys("FAIL_MSG", "ShellCommand")
	sys("FAMBASE", "Pathname")
	pkg("FAM_ACCEPTED", "List of { fam gamin }")
	usr("FAM_DEFAULT", "{ fam gamin }")
	sys("FAM_TYPE", "{ fam gamin }")
	acl("FETCH_BEFORE_ARGS", "List of ShellWord", "Makefile:as")
	pkglist("FETCH_MESSAGE", "List of ShellWord")
	pkg("FILESDIR", "RelativePkgPath")
	pkglist("FILES_SUBST", "List of ShellWord")
	acl("FILES_SUBST_SED", "List of ShellWord")
	pkglist("FIX_RPATH", "List of Varname")
	pkglist("FLEX_REQD", "List of Version")
	acl("FONTS_DIRS.*", "List of Pathname", "Makefile:as", "Makefile.common:a")
	sys("GAMEDATAMODE", "FileMode")
	sys("GAMES_GROUP", "UserGroupName")
	sys("GAMEMODE", "FileMode")
	sys("GAMES_USER", "UserGroupName")
	pkglist("GCC_REQD", "List of Version")
	pkglist("GENERATE_PLIST", "List of ShellWord")
	// ^^ List of Shellcommand, terminated with a semicolon
	pkg("GITHUB_PROJECT", "Identifier")
	pkg("GITHUB_TAG", "Identifier")
	pkg("GITHUB_RELEASE", "Filename")
	pkg("GITHUB_TYPE", "{ tag release }")
	acl("GNU_ARCH", "{ mips }")
	acl("GNU_CONFIGURE", "Yes", "Makefile.common:s", "Makefile:s")
	acl("GNU_CONFIGURE_INFODIR", "Pathname", "Makefile:s", "Makefile.common:s")
	acl("GNU_CONFIGURE_LIBDIR", "Pathname", "Makefile:s", "Makefile.common:s")
	pkg("GNU_CONFIGURE_LIBSUBDIR", "Pathname")
	acl("GNU_CONFIGURE_MANDIR", "Pathname", "Makefile:s", "Makefile.common:s")
	acl("GNU_CONFIGURE_PREFIX", "Pathname", "Makefile:s")
	acl("HAS_CONFIGURE", "Yes", "Makefile:s", "Makefile.common:s")
	pkglist("HEADER_TEMPLATES", "List of Pathname")
	pkg("HOMEPAGE", "URL")
	acl("IGNORE_PKG.*", "Yes", "*:sp")
	acl("INCOMPAT_CURSES", "InternalList of PlatformTriple", "Makefile:as")
	acl("INCOMPAT_ICONV", "InternalList of PlatformTriple")
	acl("INFO_DIR", "Pathname")
	// ^^ relative to PREFIX")
	pkg("INFO_FILES", "Yes")
	sys("INSTALL", "ShellCommand")
	pkglist("INSTALLATION_DIRS", "List of PrefixPathname")
	pkg("INSTALLATION_DIRS_FROM_PLIST", "Yes")
	sys("INSTALL_DATA", "ShellCommand")
	sys("INSTALL_DATA_DIR", "ShellCommand")
	pkglist("INSTALL_DIRS", "List of WrksrcSubdirectory")
	pkglist("INSTALL_ENV", "List of ShellWord")
	acl("INSTALL_FILE", "Pathname", "Makefile:s")
	sys("INSTALL_GAME", "ShellCommand")
	sys("INSTALL_GAME_DATA", "ShellCommand")
	sys("INSTALL_LIB", "ShellCommand")
	sys("INSTALL_LIB_DIR", "ShellCommand")
	pkglist("INSTALL_MAKE_FLAGS", "List of ShellWord")
	sys("INSTALL_MAN", "ShellCommand")
	sys("INSTALL_MAN_DIR", "ShellCommand")
	sys("INSTALL_PROGRAM", "ShellCommand")
	sys("INSTALL_PROGRAM_DIR", "ShellCommand")
	sys("INSTALL_SCRIPT", "ShellCommand")
	acl("INSTALL_SCRIPTS_ENV", "List of ShellWord")
	sys("INSTALL_SCRIPT_DIR", "ShellCommand")
	acl("INSTALL_SRC", "List of Pathname", "Makefile:s", "Makefile.common:ds")
	pkg("INSTALL_TARGET", "List of Identifier")
	acl("INSTALL_TEMPLATES", "List of Pathname", "Makefile:as", "Makefile.common:ads")
	acl("INSTALL_UNSTRIPPED", "YesNo", "Makefile:s", "Makefile.common:s")
	pkg("INTERACTIVE_STAGE", "List of { fetch extract configure build install }")
	acl("IS_BUILTIN.*", "YesNo_Indirectly", "builtin.mk:psu")
	sys("JAVA_BINPREFIX", "Pathname")
	pkg("JAVA_CLASSPATH", "ShellWord")
	pkg("JAVA_HOME", "Pathname")
	pkg("JAVA_NAME", "Filename")
	pkglist("JAVA_UNLIMIT", "List of { cmdsize datasize stacksize }")
	pkglist("JAVA_WRAPPERS", "InternalList of Filename")
	pkg("JAVA_WRAPPER_BIN.*", "Pathname")
	sys("KRB5BASE", "Pathname")
	acl("KRB5_ACCEPTED", "List of { heimdal mit-krb5 }")
	usr("KRB5_DEFAULT", "{ heimdal mit-krb5 }")
	sys("KRB5_TYPE", "Unchecked")
	sys("LD", "ShellCommand")
	pkglist("LDFLAGS*", "List of LdFlag")
	sys("LIBGRP", "UserGroupName")
	sys("LIBMODE", "FileMode")
	sys("LIBOWN", "UserGroupName")
	sys("LIBOSSAUDIO", "Pathname")
	pkglist("LIBS*", "List of LdFlag")
	sys("LIBTOOL", "ShellCommand")
	acl("LIBTOOL_OVERRIDE", "List of Pathmask", "Makefile:as")
	pkglist("LIBTOOL_REQD", "List of Version")
	acl("LICENCE", "License", "Makefile:s", "Makefile.common:s", "options.mk:s")
	acl("LICENSE", "License", "Makefile:s", "Makefile.common:s", "options.mk:s")
	pkg("LICENSE_FILE", "Pathname")
	sys("LINKER_RPATH_FLAG", "ShellWord")
	sys("LOWER_OPSYS", "Identifier")
	acl("LTCONFIG_OVERRIDE", "List of Pathmask", "Makefile:as", "Makefile.common:a")
	sys("MACHINE_ARCH", "Identifier")
	sys("MACHINE_GNU_PLATFORM", "PlatformTriple")
	acl("MAINTAINER", "MailAddress", "Makefile:s", "Makefile.common:d")
	sys("MAKE", "ShellCommand")
	pkglist("MAKEFLAGS", "List of ShellWord")
	acl("MAKEVARS", "List of Varname", "builtin.mk:a", "buildlink3.mk:a", "hacks.mk:a")
	pkglist("MAKE_DIRS", "List of Pathname")
	pkglist("MAKE_DIRS_PERMS", "List of ShellWord")
	pkglist("MAKE_ENV", "List of ShellWord")
	pkg("MAKE_FILE", "Pathname")
	pkglist("MAKE_FLAGS", "List of ShellWord")
	usr("MAKE_JOBS", "Integer")
	pkg("MAKE_JOBS_SAFE", "YesNo")
	pkg("MAKE_PROGRAM", "ShellCommand")
	acl("MANCOMPRESSED", "YesNo", "Makefile:s", "Makefile.common:ds")
	acl("MANCOMPRESSED_IF_MANZ", "Yes", "Makefile:s", "Makefile.common:ds")
	sys("MANGRP", "UserGroupName")
	sys("MANMODE", "FileMode")
	sys("MANOWN", "UserGroupName")
	pkglist("MASTER_SITES", "List of FetchURL")
	sys("MASTER_SITE_APACHE", "List of FetchURL")
	sys("MASTER_SITE_BACKUP", "List of FetchURL")
	sys("MASTER_SITE_CYGWIN", "List of FetchURL")
	sys("MASTER_SITE_DEBIAN", "List of FetchURL")
	sys("MASTER_SITE_FREEBSD", "List of FetchURL")
	sys("MASTER_SITE_FREEBSD_LOCAL", "List of FetchURL")
	sys("MASTER_SITE_GENTOO", "List of FetchURL")
	sys("MASTER_SITE_GITHUB", "List of FetchURL")
	sys("MASTER_SITE_GNOME", "List of FetchURL")
	sys("MASTER_SITE_GNU", "List of FetchURL")
	sys("MASTER_SITE_GNUSTEP", "List of FetchURL")
	sys("MASTER_SITE_IFARCHIVE", "List of FetchURL")
	sys("MASTER_SITE_HASKELL_HACKAGE", "List of FetchURL")
	sys("MASTER_SITE_KDE", "List of FetchURL")
	sys("MASTER_SITE_LOCAL", "List of FetchURL")
	sys("MASTER_SITE_MOZILLA", "List of FetchURL")
	sys("MASTER_SITE_MOZILLA_ALL", "List of FetchURL")
	sys("MASTER_SITE_MOZILLA_ESR", "List of FetchURL")
	sys("MASTER_SITE_MYSQL", "List of FetchURL")
	sys("MASTER_SITE_NETLIB", "List of FetchURL")
	sys("MASTER_SITE_OPENOFFICE", "List of FetchURL")
	sys("MASTER_SITE_OSDN", "List of FetchURL")
	sys("MASTER_SITE_PERL_CPAN", "List of FetchURL")
	sys("MASTER_SITE_R_CRAN", "List of FetchURL")
	sys("MASTER_SITE_RUBYGEMS", "List of FetchURL")
	sys("MASTER_SITE_SOURCEFORGE", "List of FetchURL")
	sys("MASTER_SITE_SUNSITE", "List of FetchURL")
	sys("MASTER_SITE_SUSE", "List of FetchURL")
	sys("MASTER_SITE_TEX_CTAN", "List of FetchURL")
	sys("MASTER_SITE_XCONTRIB", "List of FetchURL")
	sys("MASTER_SITE_XEMACS", "List of FetchURL")
	pkglist("MESSAGE_SRC", "List of Pathname")
	acl("MESSAGE_SUBST", "List of ShellWord", "Makefile.common:a", "Makefile:a", "options.mk:a")
	pkg("META_PACKAGE", "Yes")
	sys("MISSING_FEATURES", "List of Identifier")
	acl("MYSQL_VERSIONS_ACCEPTED", "List of { 51 55 56 }", "Makefile:s")
	usr("MYSQL_VERSION_DEFAULT", "Version")
	sys("NM", "ShellCommand")
	sys("NONBINMODE", "FileMode")
	pkg("NOT_FOR_COMPILER", "List of { ccache ccc clang distcc f2c gcc hp icc ido mipspro mipspro-ucode pcc sunpro xlc }")
	pkglist("NOT_FOR_PLATFORM", "InternalList of PlatformTriple")
	pkg("NOT_FOR_UNPRIVILEGED", "YesNo")
	acl("NO_BIN_ON_CDROM", "Restricted", "Makefile:s", "Makefile.common:s")
	acl("NO_BIN_ON_FTP", "Restricted", "Makefile:s", "Makefile.common:s")
	acl("NO_BUILD", "Yes", "Makefile:s", "Makefile.common:s", "Makefile.*:ds")
	pkg("NO_CHECKSUM", "Yes")
	pkg("NO_CONFIGURE", "Yes")
	acl("NO_EXPORT_CPP", "Yes", "Makefile:s")
	pkg("NO_EXTRACT", "Yes")
	pkg("NO_INSTALL_MANPAGES", "Yes")
	// ^^ only has an effect for Imake packages.
	acl("NO_PKGTOOLS_REQD_CHECK", "Yes", "Makefile:s")
	acl("NO_SRC_ON_CDROM", "Restricted", "Makefile:s", "Makefile.common:s")
	acl("NO_SRC_ON_FTP", "Restricted", "Makefile:s", "Makefile.common:s")
	pkglist("ONLY_FOR_COMPILER", "List of { ccc clang gcc hp icc ido mipspro mipspro-ucode pcc sunpro xlc }")
	pkglist("ONLY_FOR_PLATFORM", "InternalList of PlatformTriple")
	pkg("ONLY_FOR_UNPRIVILEGED", "YesNo")
	sys("OPSYS", "Identifier")
	acl("OPSYSVARS", "List of Varname", "Makefile:a", "Makefile.common:a")
	acl("OSVERSION_SPECIFIC", "Yes", "Makefile:s", "Makefile.common:s")
	sys("OS_VERSION", "Version")
	pkg("OVERRIDE_DIRDEPTH*", "Integer")
	pkg("OVERRIDE_GNU_CONFIG_SCRIPTS", "Yes")
	acl("OWNER", "MailAddress", "Makefile:s", "Makefile.common:d")
	pkglist("OWN_DIRS", "List of Pathname")
	pkglist("OWN_DIRS_PERMS", "List of ShellWord")
	sys("PAMBASE", "Pathname")
	usr("PAM_DEFAULT", "{ linux-pam openpam solaris-pam }")
	acl("PATCHDIR", "RelativePkgPath", "Makefile:s", "Makefile.common:ds")
	pkglist("PATCHFILES", "List of Filename")
	acl("PATCH_ARGS", "List of ShellWord")
	acl("PATCH_DIST_ARGS", "List of ShellWord", "Makefile:as")
	acl("PATCH_DIST_CAT", "ShellCommand")
	acl("PATCH_DIST_STRIP*", "ShellWord", "Makefile:s", "Makefile.common:s", "buildlink3.mk:", "builtin.mk:", "*.mk:s")
	acl("PATCH_SITES", "List of URL", "Makefile:s", "options.mk:s", "Makefile.common:s")
	acl("PATCH_STRIP", "ShellWord")
	pkg("PERL5_USE_PACKLIST", "YesNo")
	acl("PERL5_PACKLIST", "List of Perl5Packlist", "Makefile:s", "options.mk:sa")
	acl("PERL5_PACKLIST_DIR", "Pathname")
	sys("PGSQL_PREFIX", "Pathname")
	acl("PGSQL_VERSIONS_ACCEPTED", "List of { 82 83 84 90 91 }")
	usr("PGSQL_VERSION_DEFAULT", "Version")
	sys("PG_LIB_EXT", "{ dylib so }")
	sys("PGSQL_TYPE", "{ postgresql81-client postgresql80-client }")
	sys("PGPKGSRCDIR", "Pathname")
	sys("PHASE_MSG", "ShellCommand")
	usr("PHP_VERSION_REQD", "Version")
	sys("PKGBASE", "Identifier")
	acl("PKGCONFIG_OVERRIDE", "List of Pathmask", "Makefile:as", "Makefile.common:a")
	pkg("PKGCONFIG_OVERRIDE_STAGE", "Stage")
	pkg("PKGDIR", "RelativePkgDir")
	sys("PKGDIRMODE", "FileMode")
	sys("PKGLOCALEDIR", "Pathname")
	pkg("PKGNAME", "PkgName")
	sys("PKGNAME_NOREV", "PkgName")
	sys("PKGPATH", "Pathname")
	acl("PKGREPOSITORY", "Unchecked")
	acl("PKGREVISION", "PkgRevision", "Makefile:s")
	sys("PKGSRCDIR", "Pathname")
	acl("PKGSRCTOP", "Yes", "Makefile:s")
	acl("PKGTOOLS_ENV", "List of ShellWord")
	sys("PKGVERSION", "Version")
	sys("PKGWILDCARD", "Filemask")
	sys("PKG_ADMIN", "ShellCommand")
	sys("PKG_APACHE", "{ apache22 apache24 }")
	pkg("PKG_APACHE_ACCEPTED", "List of { apache22 apache24 }")
	usr("PKG_APACHE_DEFAULT", "{ apache22 apache24 }")
	usr("PKG_CONFIG", "Yes")
	// ^^ No, this is not the popular command from GNOME, but the setting
	// whether the pkgsrc user wants configuration files automatically
	// installed or not.
	sys("PKG_CREATE", "ShellCommand")
	sys("PKG_DBDIR", "Pathname")
	cmdline("PKG_DEBUG_LEVEL", "Integer")
	usr("PKG_DEFAULT_OPTIONS", "List of Option")
	sys("PKG_DELETE", "ShellCommand")
	acl("PKG_DESTDIR_SUPPORT", "List of { destdir user-destdir }", "Makefile:s", "Makefile.common:s")
	pkglist("PKG_FAIL_REASON", "List of ShellWord")
	acl("PKG_GECOS.*", "Message", "Makefile:s")
	acl("PKG_GID.*", "Integer", "Makefile:s")
	acl("PKG_GROUPS", "List of ShellWord", "Makefile:as")
	pkglist("PKG_GROUPS_VARS", "List of Varname")
	acl("PKG_HOME.*", "Pathname", "Makefile:s")
	acl("PKG_HACKS", "List of Identifier", "hacks.mk:a")
	sys("PKG_INFO", "ShellCommand")
	sys("PKG_JAVA_HOME", "Pathname")
	jvms := "{ blackdown-jdk13 jdk jdk14 kaffe run-jdk13 sun-jdk14 sun-jdk15 sun-jdk6 openjdk7 openjdk7-bin sun-jdk7}"
	sys("PKG_JVM", jvms)
	acl("PKG_JVMS_ACCEPTED", "List of "+jvms, "Makefile:s", "Makefile.common:ds")
	usr("PKG_JVM_DEFAULT", jvms)
	acl("PKG_LEGACY_OPTIONS", "List of Option")
	acl("PKG_LIBTOOL", "Pathname", "Makefile:s")
	acl("PKG_OPTIONS", "InternalList of Option", "bsd.options.mk:s", "*:pu")
	usr("PKG_OPTIONS.*", "InternalList of Option")
	acl("PKG_OPTIONS_DEPRECATED_WARNINGS", "List of ShellWord")
	acl("PKG_OPTIONS_GROUP.*", "InternalList of Option", "options.mk:s", "Makefile:s")
	acl("PKG_OPTIONS_LEGACY_OPTS", "InternalList of Unchecked", "Makefile:a", "Makefile.common:a", "options.mk:a")
	acl("PKG_OPTIONS_LEGACY_VARS", "InternalList of Unchecked", "Makefile:a", "Makefile.common:a", "options.mk:a")
	acl("PKG_OPTIONS_NONEMPTY_SETS", "InternalList of Identifier")
	acl("PKG_OPTIONS_OPTIONAL_GROUPS", "InternalList of Identifier", "options.mk:as")
	acl("PKG_OPTIONS_REQUIRED_GROUPS", "InternalList of Identifier", "options.mk:s", "Makefile:s")
	acl("PKG_OPTIONS_SET.*", "InternalList of Option")
	acl("PKG_OPTIONS_VAR", "PkgOptionsVar", "options.mk:s", "Makefile:s", "Makefile.common:s", "bsd.options.mk:p")
	acl("PKG_PRESERVE", "Yes", "Makefile:s")
	acl("PKG_SHELL", "Pathname", "Makefile:s", "Makefile.common:s")
	acl("PKG_SHELL.*", "Pathname", "Makefile:s", "Makefile.common:s")
	acl("PKG_SHLIBTOOL", "Pathname")
	pkglist("PKG_SKIP_REASON", "List of ShellWord")
	acl("PKG_SUGGESTED_OPTIONS", "List of Option", "options.mk:as", "Makefile:as", "Makefile.common:s")
	acl("PKG_SUPPORTED_OPTIONS", "List of Option", "options.mk:as", "Makefile:as", "Makefile.common:s")
	pkg("PKG_SYSCONFDIR*", "Pathname")
	pkglist("PKG_SYSCONFDIR_PERMS", "List of ShellWord")
	sys("PKG_SYSCONFBASEDIR", "Pathname")
	pkg("PKG_SYSCONFSUBDIR", "Pathname")
	acl("PKG_SYSCONFVAR", "Identifier")
	// ^^ FIXME: name/type mismatch.")
	acl("PKG_UID", "Integer", "Makefile:s")
	acl("PKG_USERS", "List of ShellWord", "Makefile:as")
	pkg("PKG_USERS_VARS", "List of Varname")
	acl("PKG_USE_KERBEROS", "Yes", "Makefile:s", "Makefile.common:s")
	//PLIST.*", "# has special handling code")
	pkglist("PLIST_VARS", "List of Identifier")
	pkglist("PLIST_SRC", "List of RelativePkgPath")
	pkglist("PLIST_SUBST", "List of ShellWord")
	acl("PLIST_TYPE", "{ dynamic static }")
	acl("PREPEND_PATH", "List of Pathname")
	acl("PREFIX", "Pathname", "*:u")
	acl("PREV_PKGPATH", "Pathname", "*:u") // doesn't exist any longer
	acl("PRINT_PLIST_AWK", "AwkCommand", "*:a")
	acl("PRIVILEGED_STAGES", "List of { install package clean }")
	acl("PTHREAD_AUTO_VARS", "YesNo", "Makefile:s")
	sys("PTHREAD_CFLAGS", "List of CFlag")
	sys("PTHREAD_LDFLAGS", "List of LdFlag")
	sys("PTHREAD_LIBS", "List of LdFlag")
	acl("PTHREAD_OPTS", "List of { native optional require }", "Makefile:as", "Makefile.common:a", "buildlink3.mk:a")
	sys("PTHREAD_TYPE", "Identifier")
	// ^^ or "native" or "none".
	pkg("PY_PATCHPLIST", "Yes")
	acl("PYPKGPREFIX", "{ py27 py33 py34 }", "*:pu", "pyversion.mk:s", "*:")
	pkg("PYTHON_FOR_BUILD_ONLY", "Yes")
	pkglist("REPLACE_PYTHON", "List of Pathmask")
	pkg("PYTHON_VERSIONS_ACCEPTED", "List of Version")
	pkg("PYTHON_VERSIONS_INCOMPATIBLE", "List of Version")
	usr("PYTHON_VERSION_DEFAULT", "Version")
	usr("PYTHON_VERSION_REQD", "Version")
	pkglist("PYTHON_VERSIONED_DEPENDENCIES", "List of PythonDependency")
	sys("RANLIB", "ShellCommand")
	pkglist("RCD_SCRIPTS", "List of Filename")
	acl("RCD_SCRIPT_SRC.*", "List of Pathname", "Makefile:s")
	acl("REPLACE.*", "String", "Makefile:s")
	pkglist("REPLACE_AWK", "List of Pathmask")
	pkglist("REPLACE_BASH", "List of Pathmask")
	pkglist("REPLACE_CSH", "List of Pathmask")
	acl("REPLACE_EMACS", "List of Pathmask")
	acl("REPLACE_FILES.*", "List of Pathmask", "Makefile:as", "Makefile.common:as")
	acl("REPLACE_INTERPRETER", "List of Identifier", "Makefile:a", "Makefile.common:a")
	pkglist("REPLACE_KSH", "List of Pathmask")
	pkglist("REPLACE_LOCALEDIR_PATTERNS", "List of Filemask")
	pkglist("REPLACE_LUA", "List of Pathmask")
	pkglist("REPLACE_PERL", "List of Pathmask")
	pkglist("REPLACE_PYTHON", "List of Pathmask")
	pkglist("REPLACE_SH", "List of Pathmask")
	pkglist("REQD_DIRS", "List of Pathname")
	pkglist("REQD_DIRS_PERMS", "List of ShellWord")
	pkglist("REQD_FILES", "List of Pathname")
	pkg("REQD_FILES_MODE", "{ 0644 0640 0600 0400 }")
	pkglist("REQD_FILES_PERMS", "List of ShellWord")
	pkg("RESTRICTED", "Message")
	usr("ROOT_USER", "UserGroupName")
	usr("ROOT_GROUP", "UserGroupName")
	usr("RUBY_VERSION_REQD", "Version")
	sys("RUN", "ShellCommand")
	acl("SCRIPTS_ENV", "List of ShellWord", "Makefile:a", "Makefile.common:a")
	usr("SETUID_ROOT_PERMS", "List of ShellWord")
	sys("SHAREGRP", "UserGroupName")
	sys("SHAREMODE", "FileMode")
	sys("SHAREOWN", "UserGroupName")
	sys("SHCOMMENT", "ShellCommand")
	acl("SHLIB_HANDLING", "{ YES NO no }")
	acl("SHLIBTOOL", "ShellCommand")
	acl("SHLIBTOOL_OVERRIDE", "List of Pathmask", "Makefile:as", "Makefile.common:a")
	acl("SITES.*", "List of FetchURL", "Makefile:asu", "Makefile.common:asu", "options.mk:asu")
	pkglist("SPECIAL_PERMS", "List of ShellWord")
	sys("STEP_MSG", "ShellCommand")
	acl("SUBDIR", "List of Filename", "Makefile:a", "*:")
	acl("SUBST_CLASSES", "List of Identifier", "Makefile:a", "Makefile.common:a", "hacks.mk:a", "Makefile.*:a")
	acl("SUBST_FILES.*", "List of Pathmask", "Makefile:as", "Makefile.common:as", "hacks.mk:as", "options.mk:as", "Makefile.*:as")
	acl("SUBST_FILTER_CMD.*", "ShellCommand", "Makefile:s", "Makefile.common:s", "hacks.mk:s", "options.mk:s", "Makefile.*:s")
	acl("SUBST_MESSAGE.*", "Message", "Makefile:s", "Makefile.common:s", "hacks.mk:s", "options.mk:s", "Makefile.*:s")
	acl("SUBST_SED.*", "SedCommands", "Makefile:as", "Makefile.common:as", "hacks.mk:as", "options.mk:as", "Makefile.*:as")
	pkg("SUBST_STAGE.*", "Stage")
	pkglist("SUBST_VARS.*", "List of Varname")
	pkglist("SUPERSEDES", "InternalList of Dependency")
	pkglist("TEST_DIRS", "List of WrksrcSubdirectory")
	pkglist("TEST_ENV", "List of ShellWord")
	acl("TEST_TARGET", "List of Identifier", "Makefile:s", "Makefile.common:ds", "options.mk:as")
	acl("TEX_ACCEPTED", "List of { teTeX1 teTeX2 teTeX3 }", "Makefile:s", "Makefile.common:s")
	acl("TEX_DEPMETHOD", "{ build run }", "Makefile:s", "Makefile.common:s")
	pkglist("TEXINFO_REQD", "List of Version")
	acl("TOOL_DEPENDS", "InternalList of DependencyWithPath", "Makefile.common:a", "Makefile:a", "options.mk:a", "*.mk:a")
	sys("TOOLS_ALIASES", "List of Filename")
	sys("TOOLS_BROKEN", "List of Tool")
	sys("TOOLS_CREATE", "List of Tool")
	sys("TOOLS_DEPENDS.*", "InternalList of DependencyWithPath")
	sys("TOOLS_GNU_MISSING", "List of Tool")
	sys("TOOLS_NOOP", "List of Tool")
	sys("TOOLS_PATH.*", "Pathname")
	sys("TOOLS_PLATFORM.*", "ShellCommand")
	sys("TOUCH_FLAGS", "List of ShellWord")
	pkglist("UAC_REQD_EXECS", "List of PrefixPathname")
	acl("UNLIMIT_RESOURCES", "List of { datasize stacksize memorysize }", "Makefile:as", "Makefile.common:a")
	usr("UNPRIVILEGED_USER", "UserGroupName")
	usr("UNPRIVILEGED_GROUP", "UserGroupName")
	pkglist("UNWRAP_FILES", "List of Pathmask")
	usr("UPDATE_TARGET", "List of Identifier")
	pkg("USE_BSD_MAKEFILE", "Yes")
	acl("USE_BUILTIN.*", "YesNo_Indirectly", "builtin.mk:s")
	pkg("USE_CMAKE", "Yes")
	acl("USE_CROSSBASE", "Yes", "Makefile:s")
	pkg("USE_FEATURES", "List of Identifier")
	pkg("USE_GCC_RUNTIME", "YesNo")
	pkg("USE_GNU_CONFIGURE_HOST", "YesNo")
	acl("USE_GNU_ICONV", "Yes", "Makefile:s", "Makefile.common:s", "options.mk:s")
	acl("USE_IMAKE", "Yes", "Makefile:s")
	pkg("USE_JAVA", "{ run yes build }")
	pkg("USE_JAVA2", "{ YES yes no 1.4 1.5 6 7 8 }")
	acl("USE_LANGUAGES", "List of { ada c c99 c++ fortran fortran77 java objc }", "Makefile:s", "Makefile.common:s", "options.mk:s")
	pkg("USE_LIBTOOL", "Yes")
	pkg("USE_MAKEINFO", "Yes")
	pkg("USE_MSGFMT_PLURALS", "Yes")
	pkg("USE_NCURSES", "Yes")
	pkg("USE_OLD_DES_API", "YesNo")
	pkg("USE_PKGINSTALL", "Yes")
	pkg("USE_PKGLOCALEDIR", "YesNo")
	usr("USE_PKGSRC_GCC", "Yes")
	acl("USE_TOOLS", "List of Tool", "*:a")
	pkg("USE_X11", "Yes")
	sys("WARNING_MSG", "ShellCommand")
	sys("WARNING_CAT", "ShellCommand")
	acl("WRAPPER_REORDER_CMDS", "List of WrapperReorder", "buildlink3.mk:a", "Makefile.common:a", "Makefile:a")
	acl("WRAPPER_TRANSFORM_CMDS", "List of WrapperTransform", "buildlink3.mk:a", "Makefile.common:a", "Makefile:a")
	sys("WRKDIR", "Pathname")
	pkg("WRKSRC", "WrkdirSubdirectory")
	sys("X11_PKGSRCDIR.*", "Pathname")
	usr("XAW_TYPE", "{ 3d neXtaw standard xpm }")
	acl("XMKMF_FLAGS", "List of ShellWord")
}
