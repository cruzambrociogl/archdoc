package model

import (
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

func dec(name, arg string) core.Decorator {
	return core.Decorator{Name: name, Arg: arg, HasArg: arg != "", Prov: core.Provenance{File: "x", Line: 1}}
}

// Spring and ASP.NET controllers: the class's mapping is the prefix, each method's the route; a
// controller that renders views answers a GET with a page; [controller] and [action] are the names.
func TestControllerRoutes(t *testing.T) {
	requestMapping := dec("RequestMapping", "/fallback")
	requestMapping.Exprs = map[string]string{"method": "{RequestMethod.GET, RequestMethod.POST}"}
	src := core.Source{App: "app", Files: []core.SourceFile{
		{Path: "Owners.java", Language: "Java", Classes: []core.Class{
			{Name: "OwnerResource", Decorators: []core.Decorator{dec("RestController", ""), dec("RequestMapping", "/owners")},
				Methods: []core.Method{{Name: "find", Decorators: []core.Decorator{dec("GetMapping", "/{id}")}},
					{Name: "fallback", Decorators: []core.Decorator{requestMapping}}}},
			{Name: "WelcomeController", Decorators: []core.Decorator{dec("Controller", "")},
				Methods: []core.Method{{Name: "welcome", Decorators: []core.Decorator{dec("GetMapping", "/")}},
					{Name: "save", Decorators: []core.Decorator{dec("PostMapping", "/new")}}}},
		}},
		{Path: "Orders.cs", Language: "C#", Classes: []core.Class{
			{Name: "OrdersController", Extends: []string{"ControllerBase"}, Decorators: []core.Decorator{dec("ApiController", ""), dec("Route", "api/[controller]")},
				Methods: []core.Method{{Name: "Get", Decorators: []core.Decorator{dec("HttpGet", "{id}")}},
					{Name: "Health", Decorators: []core.Decorator{dec("HttpGet", "/health")}}}},
			{Name: "OrderController", Extends: []string{"Controller"}, Decorators: []core.Decorator{dec("Route", "[controller]/[action]")},
				Methods: []core.Method{{Name: "MyOrdersAsync", Decorators: []core.Decorator{dec("HttpGet", "")}}}},
		}},
	}}
	got := map[string]string{}
	for _, e := range controllerRoutes(src, "app:app", map[string]string{}) {
		got[e.ID] = e.Handler
	}
	for id, handler := range map[string]string{
		"route:app GET /owners/{id}":      "OwnerResource.find",
		"route:app GET /owners/fallback":  "OwnerResource.fallback",
		"route:app POST /owners/fallback": "OwnerResource.fallback",
		"page:app /":                      "WelcomeController.welcome",
		"route:app POST /new":             "WelcomeController.save",
		"route:app GET /api/Orders/{id}":  "OrdersController.Get",
		"route:app GET /health":           "OrdersController.Health",
		"page:app /Order/MyOrders":        "OrderController.MyOrdersAsync",
	} {
		if got[id] != handler {
			t.Errorf("%s: %q, want %q", id, got[id], handler)
		}
	}
	if len(got) != 8 {
		t.Errorf("%d routes: %v", len(got), got)
	}
}

// JPA and EF Core tables: a JPA entity is named by its @Table, its superclass's id is a column, an
// owned relation is a foreign key named by its @JoinColumn, and the inverse side is no column. An
// EF Core table is named after its DbSet, keyed by Id, and <Entity>Id refers to that entity — the
// class beside the context, not a same-named class elsewhere.
func TestJPAAndEFCoreTables(t *testing.T) {
	m := core.Model{Nodes: []core.Node{
		{ID: "app:java", Kind: core.Application, Dir: "java"},
		{ID: "app:web", Kind: core.Application, Dir: "web"},
	}}
	field := func(name, typ string, decs ...core.Decorator) core.Field {
		return core.Field{Name: name, Type: typ, Decorators: decs, Prov: core.Provenance{File: "lib/Data.cs", Line: 1}}
	}
	join := dec("JoinColumn", "")
	join.Options = map[string]string{"name": "type_id"}
	table := dec("Table", "")
	table.Options = map[string]string{"name": "pets"}
	many := dec("OneToMany", "")
	many.Target = "Visit"
	manyToOne := dec("ManyToOne", "")
	manyToOne.Target = "PetType"
	java := core.Source{App: "java", Files: []core.SourceFile{{Path: "java/Pet.java", Language: "Java", Classes: []core.Class{
		{Name: "BaseEntity", Decorators: []core.Decorator{dec("MappedSuperclass", "")}, Fields: []core.Field{field("id", "Integer", dec("Id", ""))}},
		{Name: "Pet", Extends: []string{"BaseEntity"}, Decorators: []core.Decorator{dec("Entity", ""), table},
			Fields: []core.Field{field("name", "String"), field("type", "PetType", manyToOne, join), field("visits", "Set<Visit>", many), field("age", "int")}},
		{Name: "PetType", Decorators: []core.Decorator{dec("Entity", "")}, Fields: []core.Field{field("id", "Integer", dec("Id", ""))}},
		{Name: "Visit", Decorators: []core.Decorator{dec("Entity", "")}, Fields: []core.Field{field("id", "Integer", dec("Id", ""))}},
	}}}}
	web := core.Source{App: "web",
		Files: []core.SourceFile{{Path: "web/Basket.cs", Language: "C#", Classes: []core.Class{{Name: "Basket", Extends: []string{"ViewComponent"}}}}},
		Schemas: []core.SourceFile{{Path: "lib/Data.cs", Language: "C#", Classes: []core.Class{
			{Name: "ShopContext", Extends: []string{"DbContext"}, Fields: []core.Field{field("Baskets", "DbSet<Basket>"), field("Items", "DbSet<BasketItem>")}},
			{Name: "BaseEntity", Fields: []core.Field{field("Id", "int")}},
			{Name: "Basket", Extends: []string{"BaseEntity"}, Fields: []core.Field{field("BuyerId", "string?"), field("Items", "IReadOnlyCollection<BasketItem>", core.Decorator{Name: "Collection", Target: "BasketItem"})}},
			{Name: "BasketItem", Extends: []string{"BaseEntity"}, Fields: []core.Field{field("BasketId", "int"), field("Basket", "Basket"), field("_secret", "int")}},
		}}}}
	tables(&m, []core.Source{java, web})
	cols := map[string][]core.Column{}
	for _, n := range m.Nodes {
		if n.Kind == core.Table {
			cols[n.ID] = n.Columns
		}
	}
	pets := cols["tbl:java/pets"]
	if len(pets) != 4 || pets[0].Name != "id" || !pets[0].Primary || pets[2].Name != "type_id" || pets[2].References != "tbl:java/PetType" ||
		!pets[1].Nullable || pets[3].Nullable {
		t.Errorf("pets %+v", pets)
	}
	baskets := cols["tbl:web/Baskets"]
	if len(baskets) != 2 || baskets[0].Name != "Id" || !baskets[0].Primary || !baskets[1].Nullable {
		t.Errorf("Baskets %+v — the entity beside the context, not the view component", baskets)
	}
	items := cols["tbl:web/Items"]
	if len(items) != 2 || items[1].Name != "BasketId" || items[1].References != "tbl:web/Baskets" {
		t.Errorf("Items %+v", items)
	}
	edges := map[string]string{}
	for _, e := range m.Edges {
		edges[e.From+">"+e.To] = e.Label
	}
	if edges["tbl:java/pets>tbl:java/Visit"] != "relates to" || edges["tbl:web/Baskets>tbl:web/Items"] != "relates to" {
		t.Errorf("relations %v", edges)
	}
}

// A call through a field typed as a Spring Data repository queries its entity's table, and how is
// in the method's name; a C# class given an interface is followed into the one class implementing it.
func TestSpringDataAndImplementations(t *testing.T) {
	entity := dec("Entity", "")
	src := core.Source{App: "a", Files: []core.SourceFile{{Path: "a/Owners.java", Language: "Java", Classes: []core.Class{
		{Name: "Owner", Decorators: []core.Decorator{entity, {Name: "Table", Options: map[string]string{"name": "owners"}}}},
		{Name: "OwnerRepository", Interface: true, Options: map[string]string{"entity": "Owner"}},
		{Name: "OwnerService", Params: []core.Param{{Name: "owners", Type: "OwnerRepository"}},
			Methods: []core.Method{{Name: "rename", Invokes: []core.Invocation{{Object: "owners", Method: "findById"}, {Object: "owners", Method: "save"}, {Object: "owners", Method: "flush"}}}}},
	}}}}
	springData([]core.Source{src})
	q := src.Files[0].Classes[2].Methods[0].Queries
	if len(q) != 2 || q[0].Table != "owners" || q[0].Op != "reads" || q[1].Op != "writes" {
		t.Errorf("queries %+v", q)
	}

	classes := map[string]classAt{
		"IOrders":  {core.Class{Name: "IOrders", Interface: true}, "a.cs"},
		"Orders":   {core.Class{Name: "Orders", Extends: []string{"IOrders"}}, "a.cs"},
		"IPay":     {core.Class{Name: "IPay", Interface: true}, "a.cs"},
		"Card":     {core.Class{Name: "Card", Extends: []string{"IPay"}}, "a.cs"},
		"Transfer": {core.Class{Name: "Transfer", Extends: []string{"IPay"}}, "a.cs"},
	}
	if impl := implementations(classes); impl["IOrders"] != "Orders" || impl["IPay"] != "" {
		t.Errorf("implementations %v — one implementation is followed, two are a choice the code makes at run time", impl)
	}
}
