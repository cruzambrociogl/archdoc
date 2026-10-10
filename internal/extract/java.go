package extract

import (
	"encoding/xml"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Java applications, by their build files: Maven's pom.xml, Gradle's build.gradle or
// build.gradle.kts. What a module is, is judged from what it depends on, as a package.json's is: a
// Spring Boot starter makes a service, a pom that only lists modules gathers the others. Gradle's
// build file is a program, not data; its dependency lines are read by pattern, which is what nearly
// every one of them looks like.

// javaFrameworks decide a Java module's role by its dependencies' artifact IDs, in order: the first
// match wins, so a gateway that is also a Spring Boot application is named a gateway.
var javaFrameworks = []struct {
	artifact, name string
}{
	{"spring-cloud-config-server", "Spring Cloud Config"},
	{"spring-cloud-starter-netflix-eureka-server", "Eureka"},
	{"spring-boot-admin-starter-server", "Spring Boot Admin"},
	// A servlet application that also pulls in the gateway's starter is not a gateway: Spring Cloud
	// Gateway runs on WebFlux (petclinic's genai service has both).
	{"spring-boot-starter-web", "Spring Boot"},
	{"spring-boot-starter-webmvc", "Spring Boot"},
	{"spring-cloud-starter-gateway", "Spring Cloud Gateway"},
	{"spring-cloud-starter-gateway-", "Spring Cloud Gateway"},
	{"spring-boot-starter-webflux", "Spring Boot"},
	{"spring-boot-starter", "Spring Boot"},
	{"spring-boot-starter-", "Spring Boot"},
	{"quarkus-", "Quarkus"},
	{"micronaut-http-server", "Micronaut"},
	{"javalin", "Javalin"},
	{"dropwizard-core", "Dropwizard"},
	{"vertx-web", "Vert.x"},
	{"helidon-webserver", "Helidon"},
}

// javaTestScopes are the scopes a dependency of the tests alone is declared in.
var javaTestScopes = map[string]bool{"test": true, "testImplementation": true, "testRuntimeOnly": true,
	"testCompileOnly": true, "annotationProcessor": true, "developmentOnly": true}

func pomXML(root, rel string) *archdoc.App {
	content, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return nil
	}
	var pom struct {
		ArtifactID string   `xml:"artifactId"`
		Packaging  string   `xml:"packaging"`
		Modules    []string `xml:"modules>module"`
		Deps       []struct {
			GroupID    string `xml:"groupId"`
			ArtifactID string `xml:"artifactId"`
			Version    string `xml:"version"`
			Scope      string `xml:"scope"`
		} `xml:"dependencies>dependency"`
	}
	if xml.Unmarshal(content, &pom) != nil {
		return nil
	}
	dir := dirOf(rel)
	app := &archdoc.App{Name: pom.ArtifactID, Dir: dir, Manifest: rel, Language: javaLanguage(root, dir),
		Prov: archdoc.Provenance{File: rel, Line: lineOf(content, "<artifactId>"+pom.ArtifactID+"</artifactId>")}}
	if app.Name == "" {
		app.Name = path.Base(dir)
	}
	for _, d := range pom.Deps {
		app.Requires = append(app.Requires, archdoc.Requirement{Name: d.GroupID + ":" + d.ArtifactID, Version: d.Version,
			Dev: javaTestScopes[d.Scope], Prov: archdoc.Provenance{File: rel, Line: lineOf(content, "<artifactId>"+d.ArtifactID+"</artifactId>")}})
	}
	if pom.Packaging == "pom" {
		app.Role, app.Why = archdoc.RoleWorkspace, "a Maven parent: it gathers the other modules"
		if len(pom.Modules) > 0 {
			app.Prov = archdoc.Provenance{File: rel, Line: lineOf(content, "<modules>")}
		}
		return app
	}
	return javaRole(app)
}

// gradleDep is a dependency line: implementation("org.springframework.boot:spring-boot-starter-web"),
// or the Groovy form, testImplementation 'org.junit:junit:4.13'.
var gradleDep = regexp.MustCompile(`(?m)^\s*(\w+)\s*\(?\s*['"]([^:'"\s]+):([^:'"\s]+)(?::([^'"\s]+))?['"]`)

// gradlePlugin names a plugin applied by ID: id("org.springframework.boot"), id 'org.springframework.boot'.
var gradlePlugin = regexp.MustCompile(`(?m)^\s*id\s*\(?\s*['"]([^'"]+)['"]`)

func gradleBuild(root, rel string) *archdoc.App {
	content, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return nil
	}
	dir := dirOf(rel)
	app := &archdoc.App{Name: path.Base(dir), Dir: dir, Manifest: rel, Language: javaLanguage(root, dir),
		Prov: archdoc.Provenance{File: rel, Line: 1}}
	if dir == "." {
		abs, _ := filepath.Abs(root)
		app.Name = filepath.Base(abs)
		if s, err := os.ReadFile(filepath.Join(root, "settings.gradle")); err == nil {
			if m := regexp.MustCompile(`rootProject\.name\s*=\s*['"]([^'"]+)['"]`).FindSubmatch(s); m != nil {
				app.Name = string(m[1])
			}
		}
	}
	for _, m := range gradleDep.FindAllSubmatchIndex(content, -1) {
		conf, group, artifact := string(content[m[2]:m[3]]), string(content[m[4]:m[5]]), string(content[m[6]:m[7]])
		version := ""
		if m[8] >= 0 {
			version = string(content[m[8]:m[9]])
		}
		app.Requires = append(app.Requires, archdoc.Requirement{Name: group + ":" + artifact, Version: version,
			Dev: javaTestScopes[conf], Prov: archdoc.Provenance{File: rel, Line: 1 + strings.Count(string(content[:m[0]]), "\n") + leadingNewlines(content[m[0]:m[1]])}})
	}
	if gradleWorkspace(root, dir, app) {
		return app
	}
	// The dependencies decide first, as in a pom — a gateway is a gateway — and the Spring Boot
	// plugin, which makes the module an application whatever it depends on, after them.
	var boot *archdoc.Provenance
	for _, m := range gradlePlugin.FindAllSubmatchIndex(content, -1) {
		if string(content[m[2]:m[3]]) == "org.springframework.boot" {
			boot = &archdoc.Provenance{File: rel, Line: 1 + strings.Count(string(content[:m[2]]), "\n")}
		}
	}
	if javaRole(app); app.Role != archdoc.RoleLibrary || boot == nil {
		return app
	}
	app.Framework, app.FrameworkProv, app.Role = "Spring Boot", *boot, archdoc.RoleService
	app.Why = whyRole(archdoc.RoleService, "Spring Boot")
	if inTestDir(app.Dir) {
		app.Role, app.Why = archdoc.RoleTest, "lives in a test directory"
	}
	return app
}

func gradleWorkspace(root, dir string, app *archdoc.App) bool {
	if dir != "." {
		return false
	}
	for _, f := range []string{"settings.gradle", "settings.gradle.kts"} {
		if s, err := os.ReadFile(filepath.Join(root, f)); err == nil && regexp.MustCompile(`(?m)^\s*include\b`).Match(s) {
			app.Role, app.Why = archdoc.RoleWorkspace, "a Gradle build whose "+f+" includes the other projects"
			app.Prov = archdoc.Provenance{File: f, Line: 1}
			return true
		}
	}
	return false
}

func leadingNewlines(b []byte) int {
	return len(b) - len(strings.TrimLeft(string(b), "\n"))
}

// javaRole judges a Java module by its dependencies: a framework makes it a service; nothing makes
// it a library — Java code that is not an application runs inside one.
func javaRole(app *archdoc.App) *archdoc.App {
	for _, f := range javaFrameworks {
		for _, r := range app.Requires {
			_, artifact, _ := strings.Cut(r.Name, ":")
			if r.Dev || !(artifact == f.artifact || strings.HasSuffix(f.artifact, "-") && strings.HasPrefix(artifact, f.artifact)) {
				continue
			}
			app.Framework, app.FrameworkProv, app.Role = f.name, r.Prov, archdoc.RoleService
			app.Why = whyRole(archdoc.RoleService, f.name)
			return testOrNot(app)
		}
	}
	app.Role, app.Why = archdoc.RoleLibrary, "a library: it declares no application framework, and runs inside what uses it"
	return testOrNot(app)
}

func testOrNot(app *archdoc.App) *archdoc.App {
	if inTestDir(app.Dir) {
		app.Role, app.Why = archdoc.RoleTest, "lives in a test directory"
	}
	return app
}

// javaLanguage is Java, unless the module's code is Kotlin — which archdoc does not read yet.
func javaLanguage(root, dir string) string {
	_, java := os.Stat(filepath.Join(root, dir, "src", "main", "java"))
	_, kotlin := os.Stat(filepath.Join(root, dir, "src", "main", "kotlin"))
	if kotlin == nil && java != nil {
		return "Kotlin"
	}
	return "Java"
}

// oneBuildPerDir keeps one Java build file where a directory has both — Spring's own petclinic ships
// a pom.xml and a build.gradle for the same code. The pom is kept: it is data, not a program.
func oneBuildPerDir(apps []archdoc.App) []archdoc.App {
	pom := map[string]bool{}
	for _, a := range apps {
		if path.Base(a.Manifest) == "pom.xml" {
			pom[a.Dir] = true
		}
	}
	out := apps[:0]
	for _, a := range apps {
		if b := path.Base(a.Manifest); (b == "build.gradle" || b == "build.gradle.kts") && pom[a.Dir] {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Manifest < out[j].Manifest })
	return out
}
