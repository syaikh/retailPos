package config

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file checks that every deployment manifest in deploy/ actually supplies
// the environment the server reads.
//
// Every gap recorded in docs/audits/production-deploy-config-audit-2026-09-26.md
// was a variable that some Go file read and no manifest declared. Nothing in the
// normal test suite can catch that: unit tests set whatever variables they
// need, so the application is only ever exercised in an environment nobody
// would build by hand.
//
// Design note, because it constrains how this file may be extended: the check
// looks for an actual variable DECLARATION, not merely the name appearing in
// the file. A comment mentioning JWT_SECRET does not count. A first version of
// this file used plain substring matching and reported the podman script as
// correct purely because its comments name the variables, which is exactly the
// failure mode it exists to catch.
//
// This is a lint, not a security control. It confirms a declaration is present;
// it cannot confirm the value is correct, or that a secret is strong.

// requiredInProduction must be declared by every production manifest. Omitting
// any of these is a defect that either prevents startup or weakens security:
//
//   - ENV        selects production behaviour: JSON logs and sslmode=require.
//     Unset, the server runs as development with debug logging.
//   - CORS_ORIGIN falls back to http://localhost:5173, so a real domain is
//     rejected by both the CORS middleware and the WebSocket origin
//     check.
//   - COOKIE_SECURE unset, the 7-day refresh_token cookie is issued without the
//     Secure flag and is sent in cleartext on any http request.
var requiredInProduction = []string{
	"ENV",
	"CORS_ORIGIN",
	"COOKIE_SECURE",
}

// databaseConnection is required as a set, satisfied instead by DATABASE_URL.
// The server composes its DSN from DB_* only when DATABASE_URL is empty
// (cmd/server/main.go), so a manifest may use one form or the other but must
// supply one of them completely. DB_PASSWORD is excluded here because it is a
// secret; see requiredSecrets.
var databaseConnection = []string{
	"DB_HOST",
	"DB_PORT",
	"DB_USER",
	"DB_NAME",
}

// connectionViaURL is the alternative to databaseConnection. cmd/server/main.go
// composes the DSN from DB_* only when DATABASE_URL is empty, so a manifest may
// declare one form or the other, never a mix. DATABASE_URL is therefore classified
// for exhaustiveness but never itself asserted as required.
var connectionViaURL = []string{
	"DATABASE_URL",
}

// recommended have a working default, so their absence is not a defect. They
// are asserted anyway because the defaults hide problems: an unset LOG_LEVEL
// means debug logging, and an unset DB_SSLMODE in production means a connection
// the database may refuse.
var recommended = []string{
	"LOG_LEVEL",
	"DB_SSLMODE",
	"PORT",
}

// requiredSecrets must reach the backend through the secret file rather than a
// literal. JWT_SECRET is absent here because the server panics without it, which
// makes the omission a startup failure rather than a silent weakness.
//
// JWT_SECRET_REFRESH is listed for a different reason and would be misleading
// alongside the others: internal/config falls back to JWT_SECRET when it is
// unset, so the process starts fine without it. It is here because the *name*
// must be settable in the secret file, so an operator who wants independent
// rotation has somewhere to put it and no manifest tempts them toward a literal.
// The template documents it as optional; do not "fix" that mismatch by deleting
// the key.
var requiredSecrets = []string{
	"JWT_SECRET",
	"JWT_SECRET_REFRESH",
	"DB_PASSWORD",
}

// optionalWithDefaults are the rate-limit knobs. Every one has a default in
// internal/middleware/rate_limit.go, so no manifest needs to set them.
var optionalWithDefaults = []string{
	"RATE_LIMIT_RPS",
	"RATE_LIMIT_BURST",
	"LOGIN_RATE_LIMIT_RPM",
	"LOGIN_RATE_LIMIT_BURST",
	"REFRESH_RATE_LIMIT_RPM",
	"REFRESH_RATE_LIMIT_BURST",
	"WS_RATE_LIMIT_RPM",
	"WS_RATE_LIMIT_BURST",
}

// deliberatelyUnset lists variables that must stay unassigned unless someone
// changes their mind on purpose, with the reason recorded here.
//
// COOKIE_DOMAIN is left unset on purpose: an empty value produces a host-only
// cookie, which is correct. Setting it to a parent domain would share the
// refresh token with every subdomain.
var deliberatelyUnset = map[string]string{
	"COOKIE_DOMAIN": "a parent-domain value would share the refresh token across all subdomains",
}

// secretFileTemplate documents the contents of the secret file. It is the
// contract for what the operator puts in /etc/retail-pos/backend.env, and it is
// what a manifest's secret-file mechanism actually delivers.
const secretFileTemplate = "deploy/.env.example"

// secretFileMarkers are the mechanisms a manifest may use to receive secrets
// without embedding them.
var secretFileMarkers = []string{
	"--env-file",
	"env_file:",
	"EnvironmentFile=",
}

var (
	getenvLiteral = regexp.MustCompile(`os\.Getenv\("([A-Z][A-Z0-9_]*)"\)`)

	// shellEnvFlag matches `podman run -e NAME=value` and bare `-e NAME`, which
	// reads the value from the caller's exported environment.
	shellEnvFlag = regexp.MustCompile(`(?m)^\s*-e\s+([A-Z][A-Z0-9_]*)\s*(?:=|$)`)
	// composeEnvKey matches an environment mapping key at the indentation
	// compose uses inside a service.
	composeEnvKey = regexp.MustCompile(`(?m)^\s{4,}([A-Z][A-Z0-9_]*):`)
	// longHex matches a value that looks like a generated secret. Scoped to
	// lines that also name a secret so image digests do not match.
	longHex = regexp.MustCompile(`[0-9a-fA-F]{32,}`)

	// envKeyLine matches a `NAME=` assignment in the secret file template.
	envKeyLine = regexp.MustCompile(`(?m)^([A-Z][A-Z0-9_]*)=`)

	// quadletEnv matches `Environment=NAME=value` in a .container/.pod unit.
	quadletEnv = regexp.MustCompile(`(?m)^Environment=([A-Z][A-Z0-9_]*)=`)
	// quadletPublish matches `PublishPort=HOST:CONTAINER` in a .pod unit.
	quadletPublish = regexp.MustCompile(`(?m)^PublishPort=(\d+):(\d+)`)
	// composePort matches a `ports:` entry, `"HOST:CONTAINER"` or bare `CONTAINER`.
	composePort = regexp.MustCompile(`"?(\d+)(?::(\d+))?"?`)
	// healthPort matches a healthcheck URL aimed at the container itself. The
	// port group is optional because a URL with no explicit port is implicitly
	// 80 for http and 443 for https; see addHealthPorts.
	healthPort = regexp.MustCompile(`https?://(?:localhost|127\.0\.0\.1)(?::(\d+))?`)
	// exposePorts matches `EXPOSE 8080` in a Dockerfile.
	exposePorts = regexp.MustCompile(`(?mi)^EXPOSE\s+([0-9\s]+)$`)
)

// addHealthPorts records the port each healthcheck URL in text refers to. A URL
// with no explicit port means 80 for http and 443 for https, which is how a
// healthcheck aimed at a container that only listens on 8081 goes wrong.
func addHealthPorts(text string, into map[string]bool) {
	for _, m := range healthPort.FindAllStringSubmatch(text, -1) {
		switch {
		case m[1] != "":
			into[m[1]] = true
		case strings.HasPrefix(m[0], "https"):
			into["443"] = true
		default:
			into["80"] = true
		}
	}
}

// manifestEnv is what a deploy path actually declares for the backend.
type manifestEnv struct {
	declared   map[string]bool
	secretFile bool
	raw        string
	// listenPorts are the container-side ports this manifest expects something
	// to be listening on, gathered from published ports and healthcheck URLs.
	// Checked against the image's EXPOSE list; see TestPortsAreExposedByImage.
	listenPorts map[string]bool
}

func repoFile(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", rel))
	require.NoErrorf(t, err, "cannot read %s", rel)
	return string(data)
}

// stripComments drops whole-line comments so documentation cannot satisfy a
// declaration check. Inline comments are left alone; a variable named in one is
// still a mention worth reviewing by eye.
func stripComments(content, prefix string) string {
	var kept []string
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// extractShellBackendEnv reads the environment the podman script declares for
// the backend container specifically, not the whole file. The postgres block
// legitimately declares POSTGRES_* and must not be mistaken for backend config.
func extractShellBackendEnv(content string) manifestEnv {
	env := manifestEnv{declared: map[string]bool{}, raw: content}

	start := strings.Index(content, "start_backend()")
	if start < 0 {
		return env
	}
	rest := content[start:]
	if end := strings.Index(rest, "\n}"); end > 0 {
		rest = rest[:end]
	}
	body := stripComments(rest, "#")

	for _, m := range shellEnvFlag.FindAllStringSubmatch(body, -1) {
		env.declared[m[1]] = true
	}
	// --env-file delivers everything in the secret file, so its presence is
	// itself a declaration mechanism.
	if strings.Contains(body, "--env-file") {
		env.secretFile = true
	}
	return env
}

// extractComposeBackendEnv reads the `backend:` service block. compose is parsed
// with a regex rather than a YAML library to avoid adding a dependency for a
// lint; the indentation assumption is compose's two-space service / four-space
// mapping convention, and this function is the single place to fix if a manifest
// ever uses tabs.
func extractComposeBackendEnv(content string) manifestEnv {
	env := manifestEnv{
		declared:    map[string]bool{},
		raw:         content,
		listenPorts: map[string]bool{},
	}

	lines := strings.Split(content, "\n")
	inBackend := false
	inEnvironment := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		// A new top-level service or a top-level key ends the block.
		if !inBackend {
			if trimmed == "backend:" {
				inBackend = true
			}
			continue
		}
		if !strings.HasPrefix(line, " ") && trimmed != "" {
			break
		}
		if trimmed == "environment:" {
			inEnvironment = true
			continue
		}
		if inEnvironment {
			// Dedent out of the environment block.
			if strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, ": ") {
				inEnvironment = false
				continue
			}
			if m := composeEnvKey.FindStringSubmatch(line); m != nil {
				env.declared[m[1]] = true
			}
		}
		if strings.HasPrefix(trimmed, "env_file:") {
			env.secretFile = true
		}
	}
	collectPorts(content, env.listenPorts)
	return env
}

// extractQuadletEnv reads a Quadlet unit. A .container file is one unit per
// file with no nested services, so unlike the compose and shell readers this
// does not need to locate a block: the unit is the file.
func extractQuadletEnv(content string) manifestEnv {
	env := manifestEnv{
		declared:    map[string]bool{},
		raw:         content,
		listenPorts: map[string]bool{},
	}
	body := stripComments(content, "#")
	for _, m := range quadletEnv.FindAllStringSubmatch(body, -1) {
		env.declared[m[1]] = true
	}
	if strings.Contains(body, "EnvironmentFile=") {
		env.secretFile = true
	}
	for _, m := range quadletPublish.FindAllStringSubmatch(body, -1) {
		env.listenPorts[m[2]] = true
	}
	collectPorts(content, env.listenPorts)
	return env
}

// collectPorts records every container-side port a manifest names: published
// ports and healthcheck URLs. Only used to check the manifest against what the
// image actually EXPOSEs, so a false positive costs a test failure, not a
// false pass.
func collectPorts(content string, into map[string]bool) {
	addHealthPorts(content, into)
}

// composeServicePorts returns the container-side ports the compose service using
// the given image declares: the published ones and the one its healthcheck URL
// names. Both are attributed to the same service so a health check cannot be
// credited to the wrong image.
//
// Indentation follows compose's own convention: two spaces per level, so a
// service key sits at 2, its keys at 4, and their list entries or values at 6.
// That assumption is documented above extractComposeBackendEnv; this is the
// second place to fix if a manifest ever uses tabs.
func composeServicePorts(content, image string) map[string]bool {
	ports := map[string]bool{}
	isTarget := false
	block := ""
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))

		if indent == 2 {
			// A new service: `  name:`
			isTarget, block = false, ""
			continue
		}
		if indent < 4 {
			continue
		}
		if indent == 4 {
			key, value, _ := strings.Cut(trimmed, ":")
			key, value = strings.TrimSpace(key), strings.TrimSpace(value)
			if key == "image" {
				isTarget = value == image
				block = ""
				continue
			}
			if value != "" {
				block = "" // a mapping, not a nested block
				continue
			}
			block = key
			continue
		}
		// indent >= 6: a value or list entry belonging to the current block.
		if !isTarget {
			continue
		}
		if indent >= 8 {
			continue
		}
		switch block {
		case "ports":
			entry := strings.TrimPrefix(trimmed, "- ")
			if m := composePort.FindStringSubmatch(entry); m != nil {
				// "HOST:CONTAINER" or a bare "CONTAINER".
				containerPort := m[2]
				if containerPort == "" {
					containerPort = m[1]
				}
				ports[containerPort] = true
			}
		case "healthcheck":
			if strings.HasPrefix(trimmed, "test:") {
				addHealthPorts(trimmed, ports)
			}
		}
	}
	return ports
}

var extractors = map[string]func(string) manifestEnv{
	"podman-deploy.sh":                     extractShellBackendEnv,
	"docker-compose.yml":                   extractComposeBackendEnv,
	"quadlet/retail-pos-backend.container": extractQuadletEnv,
}

var manifestNames = []string{
	"podman-deploy.sh",
	"docker-compose.yml",
	"quadlet/retail-pos-backend.container",
}

func envFor(t *testing.T, name string) manifestEnv {
	t.Helper()
	extract, ok := extractors[name]
	require.Truef(t, ok, "no extractor for deploy/%s; register one in extractors", name)
	return extract(repoFile(t, filepath.Join("deploy", name)))
}

// manifestGlobs matches the files under deploy/ that describe how the
// application is started. A new one must be registered here or it escapes every
// check below. Matching is on the path relative to deploy/, so deploy/quadlet/
// is covered too.
var manifestGlobs = []string{
	"docker-compose*.yml",
	"docker-compose*.yaml",
	"compose*.yml",
	"*.service",
	"*.sh",
	"*.pod",
	"*.container",
}

// brokenAwaitingFix lists deploy manifests known to be non-functional, each with
// the reason. It is empty on purpose: every earlier P1-P8 finding lived in a
// file here, and a new broken path has to be named in this map rather than
// quietly filed under notABackendManifest. See Recommendation 7 in
// docs/audits/production-deploy-config-audit-2026-09-26.md.
var brokenAwaitingFix = map[string]string{}

// notABackendManifest lists deploy files that legitimately do not configure the
// backend, so the backend check can be pointed at the one that does.
var notABackendManifest = map[string]string{
	"quadlet/retail-pos.pod":                "only creates the network namespace and the port forwards",
	"quadlet/retail-pos-postgres.container": "configures the database container, checked via its own healthcheck test",
	"quadlet/retail-pos-frontend.container": "serves static files; VITE_API_URL is compiled in at image build time",
}

func TestAllManifestsHaveAnExtractor(t *testing.T) {
	// A new deploy path that nobody registered would silently escape this file,
	// which is how P1-P8 accumulated in the first place: they were all in files
	// this check would have rejected.
	deployDir := filepath.Join("..", "..", "deploy")

	var found []string
	err := filepath.WalkDir(deployDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(deployDir, path)
		if err != nil {
			return err
		}
		for _, g := range manifestGlobs {
			// Matched against the base name: filepath.Match will not let *
			// cross a separator, and every glob here is extension-style.
			if matched, _ := filepath.Match(g, d.Name()); matched {
				found = append(found, rel)
				break
			}
		}
		return nil
	})
	require.NoError(t, err)
	sort.Strings(found)
	require.NotEmpty(t, found, "no deploy manifests found; the glob is wrong")

	var unaccounted, checked []string
	for _, name := range found {
		if _, registered := extractors[name]; registered {
			checked = append(checked, name)
			continue
		}
		if _, acknowledged := notABackendManifest[name]; acknowledged {
			continue
		}
		if _, known := brokenAwaitingFix[name]; known {
			continue
		}
		unaccounted = append(unaccounted, name)
	}
	assert.Emptyf(t, unaccounted,
		"these deploy manifests are neither checked by this file nor explained in "+
			"notABackendManifest or brokenAwaitingFix: %v.\n"+
			"Register them in extractors and manifestNames, or record why they are not a "+
			"path that starts the application.", unaccounted)

	assert.ElementsMatch(t, manifestNames, checked,
		"the set of deploy manifests on disk and the set checked here have diverged; "+
			"update manifestNames, extractors and the two explanation maps to match")

	// Keep the explanations honest: a stale entry that names a file which no
	// longer exists, or an empty reason, is worse than no entry at all.
	for _, m := range []map[string]string{notABackendManifest, brokenAwaitingFix} {
		for name, reason := range m {
			assert.NotEmptyf(t, reason, "%v lists deploy/%s with no reason", m, name)
			assert.Containsf(t, found, name,
				"an explanation map lists deploy/%s, which is no longer a manifest; remove the entry", name)
		}
	}
}

// TestServerEnvVarsAreClassified keeps the tables above exhaustive. Without it a
// newly added os.Getenv would be invisible here, the manifest check would stop
// covering it, and this file would drift into a lint that passes for the wrong
// reason.
func TestServerEnvVarsAreClassified(t *testing.T) {
	classified := map[string]bool{}
	for _, group := range [][]string{requiredInProduction, databaseConnection, connectionViaURL, recommended, requiredSecrets, optionalWithDefaults} {
		for _, name := range group {
			require.Falsef(t, classified[name], "%s appears in more than one group", name)
			classified[name] = true
		}
	}
	for name := range deliberatelyUnset {
		require.Falsef(t, classified[name], "%s is both asserted and deliberately unset", name)
		classified[name] = true
	}

	// Scan what the running server reads. cmd/dummy is a standalone seeder, not
	// part of the deployed service, so it is excluded.
	var found []string
	seen := map[string]bool{}
	for _, dir := range []string{"internal", "cmd/server"} {
		require.NoError(t, filepath.Walk(filepath.Join("..", "..", dir), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || strings.HasSuffix(path, "_test.go") || !strings.HasSuffix(path, ".go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, m := range getenvLiteral.FindAllStringSubmatch(string(data), -1) {
				if !seen[m[1]] {
					seen[m[1]] = true
					found = append(found, m[1])
				}
			}
			return nil
		}))
	}
	sort.Strings(found)

	var unclassified []string
	for _, name := range found {
		if !classified[name] {
			unclassified = append(unclassified, name)
		}
	}
	assert.Emptyf(t, unclassified,
		"environment variables read by the server but not classified above; add each to "+
			"requiredInProduction, databaseConnection, recommended, requiredSecrets, "+
			"optionalWithDefaults or deliberatelyUnset: %v", unclassified)

	// The reverse direction: a name asserted here that no code reads is a typo in
	// this file, and a typo here means a real gap passes silently.
	var unread []string
	for name := range classified {
		if !seen[name] {
			unread = append(unread, name)
		}
	}
	sort.Strings(unread)
	assert.Emptyf(t, unread, "asserted here but read by no code: %v", unread)
}

// templateKeys returns the variable names deploy/.env.example instructs the
// operator to put in the secret file. A manifest that reads the secret file can
// legitimately source any of these, so they count as supplied even when they are
// not repeated in the manifest body.
func templateKeys(t *testing.T) map[string]bool {
	t.Helper()
	body := stripComments(repoFile(t, secretFileTemplate), "#")
	keys := map[string]bool{}
	for _, m := range envKeyLine.FindAllStringSubmatch(body, -1) {
		keys[m[1]] = true
	}
	require.NotEmpty(t, keys, "no keys parsed from %s; the template format changed", secretFileTemplate)
	return keys
}

// secretFileOnlyVars are per-deployment values that must arrive through the
// secret file. Listing one in a compose `environment:` block silently breaks
// that, because compose resolves `${VAR:-default}` against the host environment
// or project .env, never against a service's env_file — and `environment:`
// outranks env_file. So a line like
//
//	CORS_ORIGIN: ${CORS_ORIGIN:-http://localhost:5173}
//
// does not default the *secret file's* value; it replaces it with a
// host-derived one, and an operator who correctly wrote their real origin into
// /etc/retail-pos/backend.env would get the localhost fallback instead, with no
// warning. Both names below have the same defaults in internal/config, so
// omitting them from the manifest is behaviour-preserving when the file omits
// them too.
//
// This is deliberately narrower than "env_file must always win". DB_PORT is a
// counter-example: the template's 5433 is a host port, while compose needs 5432
// inside the network, so DB_PORT *must* be overridden in `environment:`.
var secretFileOnlyVars = []string{"DB_SSLMODE", "CORS_ORIGIN"}

func TestComposeDoesNotHostResolveSecretFileVars(t *testing.T) {
	env := envFor(t, "docker-compose.yml")

	for _, key := range secretFileOnlyVars {
		if env.declared[key] {
			t.Errorf("docker-compose.yml declares %s in `environment:`, which overrides the "+
				"secret file. If it is written as ${%s:-...} the value is resolved against the "+
				"host, not env_file, so the secret file's setting is silently discarded. "+
				"Omit the name and let it come from %s; internal/config already defaults it.",
				key, key, secretFileTemplate)
		}
	}
}

func TestTemplateDocumentsDBPortForPodmanOnly(t *testing.T) {
	// Guards the reasoning behind secretFileOnlyVars staying narrow. If the
	// template ever drops DB_PORT, the override in compose stops being
	// meaningful and this note goes stale.
	assert.True(t, templateKeys(t)["DB_PORT"],
		"%s no longer documents DB_PORT; re-check why compose overrides it in `environment:`",
		secretFileTemplate)
}

func TestManifestsDeclareRequiredVars(t *testing.T) {
	fromSecretFile := templateKeys(t)

	for _, name := range manifestNames {
		t.Run(name, func(t *testing.T) {
			env := envFor(t, name)

			want := append([]string{}, requiredInProduction...)
			want = append(want, recommended...)
			// DATABASE_URL replaces the whole DB_* group.
			if !env.declared["DATABASE_URL"] {
				want = append(want, databaseConnection...)
			}

			var missing []string
			for _, v := range want {
				if env.declared[v] {
					continue
				}
				// The secret file is a real channel, but only for what the template
				// tells the operator to write there.
				if env.secretFile && fromSecretFile[v] {
					continue
				}
				missing = append(missing, v)
			}
			assert.Emptyf(t, missing,
				"deploy/%s does not declare %v for the backend container.\n"+
					"Each was read by the server and declared by no deploy path; see "+
					"docs/audits/production-deploy-config-audit-2026-09-26.md. Declare the variable, "+
					"or record in deploy/%s why it does not apply to this path.",
				name, missing, name)
		})
	}
}

// imageExposedPorts reads the ports a Dockerfile declares with EXPOSE. This is
// the closest thing to ground truth the repo has about what an image listens on.
func imageExposedPorts(t *testing.T, dockerfile string) map[string]bool {
	t.Helper()
	ports := map[string]bool{}
	for _, m := range exposePorts.FindAllStringSubmatch(repoFile(t, dockerfile), -1) {
		for _, p := range strings.Fields(m[1]) {
			ports[p] = true
		}
	}
	require.NotEmptyf(t, ports, "%s has no EXPOSE line, so it cannot be checked", dockerfile)
	return ports
}

// externalImagePorts are container-side ports published by a pod unit on behalf
// of an image this repository does not build, so there is no Dockerfile to read
// an EXPOSE list from. The official postgres image documents 5432.
var externalImagePorts = map[string]bool{"5432": true}

// TestPortsAreExposedByImage catches a manifest that publishes or healthchecks a
// port the image never listens on. The compose frontend published 80:80 and
// probed port 80 while nginx.conf has a single `listen 8081;` — unreachable in
// both directions, inherited from the deleted systemd unit, which had the same
// defect. A variable-level guard cannot see this: every environment variable was
// correct in that file.
func TestPortsAreExposedByImage(t *testing.T) {
	cases := []struct {
		name       string
		dockerfile string
		image      string
		unit       string
	}{
		{
			name:       "backend",
			dockerfile: "deploy/backend/Dockerfile",
			image:      "retail-pos-backend:latest",
			unit:       "quadlet/retail-pos-backend.container",
		},
		{
			name:       "frontend",
			dockerfile: "deploy/frontend/Dockerfile",
			image:      "retail-pos-frontend:latest",
			unit:       "quadlet/retail-pos-frontend.container",
		},
	}

	compose := repoFile(t, "deploy/docker-compose.yml")
	podPorts := extractQuadletEnv(repoFile(t, "deploy/quadlet/retail-pos.pod")).listenPorts

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exposed := imageExposedPorts(t, tc.dockerfile)

			checked := map[string]bool{}
			for p := range composeServicePorts(compose, tc.image) {
				checked[p] = true
			}
			// The unit's own healthcheck URL, which is inside the container and
			// so is not routed through the pod.
			for p := range extractQuadletEnv(
				repoFile(t, filepath.Join("deploy", tc.unit))).listenPorts {
				if _, forwarded := podPorts[p]; forwarded {
					continue // attributed to the pod check below
				}
				checked[p] = true
			}

			for p := range checked {
				assert.Containsf(t, exposed, p,
					"the %s manifests publish/healthcheck port %s, but %s only EXPOSEs %v. "+
						"Publishing it reaches nothing.", tc.name, p, tc.dockerfile, keys(exposed))
			}
		})
	}

	// The pod forwards for all three containers at once, so it can only be
	// checked against the union of what every image listens on.
	t.Run("pod", func(t *testing.T) {
		forwardable := map[string]bool{}
		for k, v := range externalImagePorts {
			forwardable[k] = v
		}
		for _, df := range []string{"deploy/backend/Dockerfile", "deploy/frontend/Dockerfile"} {
			for k, v := range imageExposedPorts(t, df) {
				forwardable[k] = v
			}
		}
		for p := range podPorts {
			assert.Containsf(t, forwardable, p,
				"quadlet/retail-pos.pod forwards container port %s, which no image in the stack "+
					"listens on. Available: %v", p, keys(forwardable))
		}
	})
}

// TestQuadletUnitReferencesResolve checks that every unit name a Quadlet file
// points at corresponds to a file in deploy/quadlet. A typo in a Requires= or
// After= is not a startup error: systemd logs an unresolved dependency and
// starts the unit anyway, so the container would come up unordered and the
// breakage would only show as a race. This file already shipped one such typo
// (`Requires=retail-pos-pod.container`, which no unit can ever be named).
func TestQuadletUnitReferencesResolve(t *testing.T) {
	quadletDir := filepath.Join("..", "..", "deploy", "quadlet")
	entries, err := os.ReadDir(quadletDir)
	require.NoError(t, err)

	sourceFiles := map[string]bool{}
	for _, e := range entries {
		sourceFiles[e.Name()] = true
	}
	// Quadlet turns $name.container and $name.pod into $name.service, and
	// additionally $name.pod into $name-pod.service. These are what
	// Requires=/After= may name; Pod= may only name a source file.
	generatedUnits := map[string]bool{}
	for name := range sourceFiles {
		switch ext := filepath.Ext(name); ext {
		case ".container", ".pod":
			generatedUnits[strings.TrimSuffix(name, ext)+".service"] = true
		}
		if filepath.Ext(name) == ".pod" {
			generatedUnits[strings.TrimSuffix(name, ".pod")+"-pod.service"] = true
		}
	}

	unitRef := regexp.MustCompile(`(?m)^(Pod|Requires|After)=(.*)$`)
	for _, e := range entries {
		body := stripComments(repoFile(t, filepath.Join("deploy", "quadlet", e.Name())), "#")
		for _, m := range unitRef.FindAllStringSubmatch(body, -1) {
			key, valid := m[1], generatedUnits
			if key == "Pod" {
				valid = sourceFiles
			}
			for _, ref := range strings.Fields(m[2]) {
				if strings.HasSuffix(ref, ".target") {
					continue // a systemd target, nothing to resolve in this directory
				}
				assert.Truef(t, valid[ref],
					"deploy/quadlet/%s has %s=%s, which does not resolve. "+
						"Pod= must name a .pod file in this directory; Requires=/After= must name a "+
						"unit Quadlet generates ($name.container or $name.pod gives $name.service). "+
						"Pointing at a source file is not a unit, and systemd ignores such a "+
						"dependency without failing.", e.Name(), key, ref)
			}
		}
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestManifestsDeliverSecretsViaSecretFile(t *testing.T) {
	template := repoFile(t, secretFileTemplate)
	templateLines := stripComments(template, "#")

	for _, name := range manifestNames {
		t.Run(name, func(t *testing.T) {
			env := envFor(t, name)
			assert.Truef(t, env.secretFile,
				"deploy/%s has no secret-file mechanism for the backend (%s). "+
					"Passing secrets as `-e NAME=value` exposes them in the process table and in `inspect`.",
				name, strings.Join(secretFileMarkers, ", "))
		})
	}

	// The mechanism only delivers what the template tells the operator to put in
	// the file, so the template is half the contract.
	var undocumented []string
	for _, s := range requiredSecrets {
		if !strings.Contains(templateLines, s) {
			undocumented = append(undocumented, s)
		}
	}
	assert.Emptyf(t, undocumented,
		"%s does not document %v. A manifest's secret-file mechanism only supplies what the "+
			"template tells the operator to write, so an undocumented secret is a missing secret.",
		secretFileTemplate, undocumented)
}

func TestManifestsDoNotHardcodeSecrets(t *testing.T) {
	for _, name := range manifestNames {
		t.Run(name, func(t *testing.T) {
			for i, line := range strings.Split(repoFile(t, filepath.Join("deploy", name)), "\n") {
				mentionsSecret := false
				for _, s := range append(append([]string{}, requiredSecrets...), "POSTGRES_PASSWORD") {
					if strings.Contains(line, s) {
						mentionsSecret = true
						break
					}
				}
				if !mentionsSecret {
					continue
				}
				assert.NotRegexpf(t, longHex, line,
					"deploy/%s:%d looks like a hardcoded secret on a line naming a secret variable.\n"+
						"Generate values into the secret file instead: openssl rand -hex 32\n"+
						"line: %s", name, i+1, strings.TrimSpace(line))
			}
		})
	}
}

func TestDeliberatelyUnsetVarsAreNotAssigned(t *testing.T) {
	// Assignment-aware, so a comment saying "COOKIE_DOMAIN is deliberately left
	// unset" satisfies this rather than tripping it.
	assignPattern := regexp.MustCompile(`([A-Z][A-Z0-9_]*)\s*[:=]\s*\S`)
	for _, name := range manifestNames {
		t.Run(name, func(t *testing.T) {
			body := stripComments(repoFile(t, filepath.Join("deploy", name)), "#")
			for varName, reason := range deliberatelyUnset {
				for _, m := range assignPattern.FindAllStringSubmatch(body, -1) {
					if m[1] != varName {
						continue
					}
					// `COOKIE_DOMAIN:-default` and `${COOKIE_DOMAIN:-x}` are default
					// expressions, not assignments.
					if strings.Contains(m[0], ":-") {
						continue
					}
					assert.Failf(t, "unexpected assignment",
						"deploy/%s assigns %s, but %s. Prefer a host-only cookie; change this test only "+
							"alongside a deliberate decision to share the refresh token across subdomains.",
						name, varName, reason)
				}
			}
		})
	}
}
