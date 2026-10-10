package extract

import (
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Java modules and .NET projects, judged by their build files: a starter or an SDK makes a service,
// a parent pom gathers, a test framework makes a test suite, a project another references is a
// library — and a pom beside a build.gradle is one module, not two.
func TestJavaAndDotnetManifests(t *testing.T) {
	root := tree(t, map[string]string{
		"pom.xml": "<project><artifactId>shop</artifactId><packaging>pom</packaging><modules><module>orders</module></modules></project>",
		"orders/pom.xml": "<project>\n<artifactId>orders</artifactId>\n<dependencies>\n<dependency><groupId>org.springframework.boot</groupId>" +
			"<artifactId>spring-boot-starter-web</artifactId></dependency>\n<dependency><groupId>org.junit</groupId><artifactId>junit</artifactId>" +
			"<scope>test</scope></dependency>\n</dependencies>\n</project>",
		"orders/build.gradle":        "plugins { id 'org.springframework.boot' }\n",
		"orders/src/main/java/.keep": "",
		"gateway/build.gradle.kts": "plugins {\n  id(\"org.springframework.boot\") version \"3.3.0\"\n}\ndependencies {\n" +
			"  implementation(\"org.springframework.cloud:spring-cloud-starter-gateway\")\n}\n",
		"src/Web/Web.csproj":     "\xef\xbb\xbf<Project Sdk=\"Microsoft.NET.Sdk.Web\">\n<ItemGroup><ProjectReference Include=\"..\\Core\\Core.csproj\" /></ItemGroup>\n</Project>",
		"src/Core/Core.csproj":   `<Project Sdk="Microsoft.NET.Sdk"></Project>`,
		"src/Admin/Admin.csproj": `<Project Sdk="Microsoft.NET.Sdk.BlazorWebAssembly"></Project>`,
		"src/Tool/Tool.csproj":   `<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><OutputType>Exe</OutputType></PropertyGroup></Project>`,
		"src/Host/Host.csproj":   `<Project Sdk="Microsoft.NET.Sdk"><Sdk Name="Aspire.AppHost.Sdk" /></Project>`,
		"tests/Unit/Unit.csproj": `<Project Sdk="Microsoft.NET.Sdk"><ItemGroup><PackageReference Include="xunit" /></ItemGroup></Project>`,
	})
	apps := byDir(Apps(root))
	want := map[string]archdoc.AppRole{
		".": archdoc.RoleWorkspace, "orders": archdoc.RoleService, "gateway": archdoc.RoleService,
		"src/Web": archdoc.RoleService, "src/Core": archdoc.RoleLibrary, "src/Admin": archdoc.RoleWeb,
		"src/Tool": archdoc.RoleCLI, "src/Host": archdoc.RoleTooling, "tests/Unit": archdoc.RoleTest,
	}
	for dir, role := range want {
		if apps[dir].Role != role {
			t.Errorf("%s: role %q, want %q (%s)", dir, apps[dir].Role, role, apps[dir].Why)
		}
	}
	if o := apps["orders"]; o.Manifest != "orders/pom.xml" || o.Framework != "Spring Boot" || o.Language != "Java" || o.FrameworkProv.Line != 4 {
		t.Errorf("orders: %+v — the pom, not the build.gradle beside it", o)
	}
	if g := apps["gateway"]; g.Framework != "Spring Cloud Gateway" || g.Requires[0].Prov.Line != 5 {
		t.Errorf("gateway: %s, its dependency at line %d", g.Framework, g.Requires[0].Prov.Line)
	}
	if w := apps["src/Web"]; w.Framework != "ASP.NET Core" || w.Language != "C#" || len(w.Requires) != 1 || w.Requires[0].Name != "Core" {
		t.Errorf("web: %+v", w)
	}
	if libs := libraries(apps["src/Web"], Apps(root)); len(libs) != 1 || libs[0].Dir != "src/Core" {
		t.Errorf("web is built with %+v, want Core", libs)
	}
}

// A Maven module's image carries the module's name; a service running it is tied to it by that,
// and says so. Only a Java module's image is read so.
func TestJavaModuleTiedByItsImage(t *testing.T) {
	root := tree(t, map[string]string{
		"docker-compose.yml": "services:\n  vets:\n    image: acme/shop-vets-service:1.0\n  web:\n    image: acme/site\n",
		"shop-vets-service/pom.xml": "<project><artifactId>shop-vets-service</artifactId><dependencies><dependency><groupId>org.springframework.boot</groupId>" +
			"<artifactId>spring-boot-starter-web</artifactId></dependency></dependencies></project>",
		"site/package.json": `{"name": "site", "dependencies": {"react": "^19"}}`,
	})
	fs, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	apps := byDir(fs.Apps)
	if d := apps["shop-vets-service"].Deployed; d == nil || d.Service != "vets" || !d.ByName || d.Prov.Note == "" {
		t.Errorf("the module is not tied to the service running its image: %+v", d)
	}
	if d := apps["site"].Deployed; d != nil {
		t.Errorf("a JavaScript package was tied by its image's name: %+v", d)
	}
}
