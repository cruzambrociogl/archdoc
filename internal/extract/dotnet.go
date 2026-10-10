package extract

import (
	"bytes"
	"encoding/xml"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// .NET projects, by their .csproj. The project's SDK says most of what it is — Microsoft.NET.Sdk.Web
// is a web application, .BlazorWebAssembly a front end in the browser, .Worker a background
// service — and its package references say the rest: a test framework makes a test suite, Aspire's
// hosting package a project that only starts the others in development.

// dotnetTests are the packages a test project references.
var dotnetTests = []string{"Microsoft.NET.Test.Sdk", "xunit", "xunit.v3", "NUnit", "MSTest.TestFramework", "MSTest"}

func csproj(root, rel string) *archdoc.App {
	content, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return nil
	}
	var proj struct {
		Sdk  string `xml:"Sdk,attr"`
		Sdks []struct {
			Name string `xml:"Name,attr"`
		} `xml:"Sdk"`
		Props []struct {
			OutputType    string `xml:"OutputType"`
			IsTestProject string `xml:"IsTestProject"`
			IsAspireHost  string `xml:"IsAspireHost"`
			UseMaui       string `xml:"UseMaui"`
			AssemblyName  string `xml:"AssemblyName"`
		} `xml:"PropertyGroup"`
		Items []struct {
			Packages []struct {
				Include string `xml:"Include,attr"`
				Version string `xml:"Version,attr"`
			} `xml:"PackageReference"`
			Projects []struct {
				Include string `xml:"Include,attr"`
			} `xml:"ProjectReference"`
		} `xml:"ItemGroup"`
	}
	if xml.Unmarshal(bytes.TrimPrefix(content, []byte("\xef\xbb\xbf")), &proj) != nil {
		return nil
	}
	dir := dirOf(rel)
	name := strings.TrimSuffix(path.Base(rel), ".csproj")
	app := &archdoc.App{Name: name, Dir: dir, Manifest: rel, Language: "C#",
		Prov: archdoc.Provenance{File: rel, Line: lineOf(content, "<Project")}}
	prop := func(get func(i int) string) string {
		for i := range proj.Props {
			if v := strings.TrimSpace(get(i)); v != "" {
				return v
			}
		}
		return ""
	}
	if a := prop(func(i int) string { return proj.Props[i].AssemblyName }); a != "" {
		app.Name = a
	}
	has := map[string]bool{}
	for _, it := range proj.Items {
		for _, p := range it.Packages {
			has[p.Include] = true
			app.Requires = append(app.Requires, archdoc.Requirement{Name: p.Include, Version: p.Version,
				Prov: archdoc.Provenance{File: rel, Line: lineOf(content, `"`+p.Include+`"`)}})
		}
		for _, p := range it.Projects {
			// Another project of this repository, by the name its file gives it: what makes a class
			// library a library is that a project here references it.
			other := strings.TrimSuffix(path.Base(strings.ReplaceAll(p.Include, `\`, "/")), ".csproj")
			app.Requires = append(app.Requires, archdoc.Requirement{Name: other,
				Prov: archdoc.Provenance{File: rel, Line: lineOf(content, p.Include)}})
		}
	}
	sdk := proj.Sdk
	for _, s := range proj.Sdks {
		sdk += ";" + s.Name
	}
	sdkProv := archdoc.Provenance{File: rel, Line: lineOf(content, "Sdk")}

	for _, t := range dotnetTests {
		if has[t] {
			app.Role, app.Why, app.FrameworkProv = archdoc.RoleTest, "a test project ("+t+")", archdoc.Provenance{File: rel, Line: lineOf(content, `"`+t+`"`)}
			return app
		}
	}
	switch {
	case inTestDir(dir) || strings.EqualFold(prop(func(i int) string { return proj.Props[i].IsTestProject }), "true"):
		app.Role, app.Why = archdoc.RoleTest, "a test project"
	case strings.Contains(sdk, "Aspire.AppHost") || has["Aspire.Hosting.AppHost"] ||
		strings.EqualFold(prop(func(i int) string { return proj.Props[i].IsAspireHost }), "true"):
		app.Role, app.Why = archdoc.RoleTooling, "an Aspire host: it starts the other projects in development, and is not deployed"
	case strings.Contains(sdk, "Microsoft.NET.Sdk.BlazorWebAssembly"):
		app.Framework, app.FrameworkProv, app.Role = "Blazor WebAssembly", sdkProv, archdoc.RoleWeb
		app.Why = whyRole(archdoc.RoleWeb, "Blazor WebAssembly")
	case strings.Contains(sdk, "Microsoft.NET.Sdk.Web"):
		app.Framework, app.FrameworkProv, app.Role = "ASP.NET Core", sdkProv, archdoc.RoleService
		app.Why = "a service: its SDK is Microsoft.NET.Sdk.Web"
	case strings.Contains(sdk, "Microsoft.NET.Sdk.Worker"):
		app.Framework, app.FrameworkProv, app.Role = ".NET Worker", sdkProv, archdoc.RoleService
		app.Why = "a background service: its SDK is Microsoft.NET.Sdk.Worker"
	case strings.EqualFold(prop(func(i int) string { return proj.Props[i].UseMaui }), "true"):
		app.Framework, app.FrameworkProv, app.Role = ".NET MAUI", archdoc.Provenance{File: rel, Line: lineOf(content, "<UseMaui>")}, archdoc.RoleMobile
		app.Why = "an app: it is built with .NET MAUI"
	case strings.EqualFold(prop(func(i int) string { return proj.Props[i].OutputType }), "Exe"):
		app.Role, app.FrameworkProv = archdoc.RoleCLI, archdoc.Provenance{File: rel, Line: lineOf(content, "<OutputType>")}
		app.Why = "a program a person runs: its OutputType is Exe"
	default:
		app.Role, app.Why = archdoc.RoleLibrary, "a class library: it builds code for other projects and runs inside them"
	}
	return app
}
