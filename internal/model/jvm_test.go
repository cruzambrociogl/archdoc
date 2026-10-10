package model

import (
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

func dec(name, arg string) archdoc.Decorator {
	return archdoc.Decorator{Name: name, Arg: arg, HasArg: arg != "", Prov: archdoc.Provenance{File: "x", Line: 1}}
}

// Spring and ASP.NET controllers: the class's mapping is the prefix, each method's the route; a
// controller that renders views answers a GET with a page; [controller] and [action] are the names.
func TestControllerRoutes(t *testing.T) {
	requestMapping := dec("RequestMapping", "/fallback")
	requestMapping.Exprs = map[string]string{"method": "{RequestMethod.GET, RequestMethod.POST}"}
	src := archdoc.Source{App: "app", Files: []archdoc.SourceFile{
		{Path: "Owners.java", Language: "Java", Classes: []archdoc.Class{
			{Name: "OwnerResource", Decorators: []archdoc.Decorator{dec("RestController", ""), dec("RequestMapping", "/owners")},
				Methods: []archdoc.Method{{Name: "find", Decorators: []archdoc.Decorator{dec("GetMapping", "/{id}")}},
					{Name: "fallback", Decorators: []archdoc.Decorator{requestMapping}}}},
			{Name: "WelcomeController", Decorators: []archdoc.Decorator{dec("Controller", "")},
				Methods: []archdoc.Method{{Name: "welcome", Decorators: []archdoc.Decorator{dec("GetMapping", "/")}},
					{Name: "save", Decorators: []archdoc.Decorator{dec("PostMapping", "/new")}}}},
		}},
		{Path: "Orders.cs", Language: "C#", Classes: []archdoc.Class{
			{Name: "OrdersController", Extends: []string{"ControllerBase"}, Decorators: []archdoc.Decorator{dec("ApiController", ""), dec("Route", "api/[controller]")},
				Methods: []archdoc.Method{{Name: "Get", Decorators: []archdoc.Decorator{dec("HttpGet", "{id}")}},
					{Name: "Health", Decorators: []archdoc.Decorator{dec("HttpGet", "/health")}}}},
			{Name: "OrderController", Extends: []string{"Controller"}, Decorators: []archdoc.Decorator{dec("Route", "[controller]/[action]")},
				Methods: []archdoc.Method{{Name: "MyOrdersAsync", Decorators: []archdoc.Decorator{dec("HttpGet", "")}}}},
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
	m := archdoc.Model{Nodes: []archdoc.Node{
		{ID: "app:java", Kind: archdoc.Application, Dir: "java"},
		{ID: "app:web", Kind: archdoc.Application, Dir: "web"},
	}}
	field := func(name, typ string, decs ...archdoc.Decorator) archdoc.Field {
		return archdoc.Field{Name: name, Type: typ, Decorators: decs, Prov: archdoc.Provenance{File: "lib/Data.cs", Line: 1}}
	}
	join := dec("JoinColumn", "")
	join.Options = map[string]string{"name": "type_id"}
	table := dec("Table", "")
	table.Options = map[string]string{"name": "pets"}
	many := dec("OneToMany", "")
	many.Target = "Visit"
	manyToOne := dec("ManyToOne", "")
	manyToOne.Target = "PetType"
	java := archdoc.Source{App: "java", Files: []archdoc.SourceFile{{Path: "java/Pet.java", Language: "Java", Classes: []archdoc.Class{
		{Name: "BaseEntity", Decorators: []archdoc.Decorator{dec("MappedSuperclass", "")}, Fields: []archdoc.Field{field("id", "Integer", dec("Id", ""))}},
		{Name: "Pet", Extends: []string{"BaseEntity"}, Decorators: []archdoc.Decorator{dec("Entity", ""), table},
			Fields: []archdoc.Field{field("name", "String"), field("type", "PetType", manyToOne, join), field("visits", "Set<Visit>", many), field("age", "int")}},
		{Name: "PetType", Decorators: []archdoc.Decorator{dec("Entity", "")}, Fields: []archdoc.Field{field("id", "Integer", dec("Id", ""))}},
		{Name: "Visit", Decorators: []archdoc.Decorator{dec("Entity", "")}, Fields: []archdoc.Field{field("id", "Integer", dec("Id", ""))}},
	}}}}
	web := archdoc.Source{App: "web",
		Files: []archdoc.SourceFile{{Path: "web/Basket.cs", Language: "C#", Classes: []archdoc.Class{{Name: "Basket", Extends: []string{"ViewComponent"}}}}},
		Schemas: []archdoc.SourceFile{{Path: "lib/Data.cs", Language: "C#", Classes: []archdoc.Class{
			{Name: "ShopContext", Extends: []string{"DbContext"}, Fields: []archdoc.Field{field("Baskets", "DbSet<Basket>"), field("Items", "DbSet<BasketItem>")}},
			{Name: "BaseEntity", Fields: []archdoc.Field{field("Id", "int")}},
			{Name: "Basket", Extends: []string{"BaseEntity"}, Fields: []archdoc.Field{field("BuyerId", "string?"), field("Items", "IReadOnlyCollection<BasketItem>", archdoc.Decorator{Name: "Collection", Target: "BasketItem"})}},
			{Name: "BasketItem", Extends: []string{"BaseEntity"}, Fields: []archdoc.Field{field("BasketId", "int"), field("Basket", "Basket"), field("_secret", "int")}},
		}}}}
	tables(&m, []archdoc.Source{java, web})
	cols := map[string][]archdoc.Column{}
	for _, n := range m.Nodes {
		if n.Kind == archdoc.Table {
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
	src := archdoc.Source{App: "a", Files: []archdoc.SourceFile{{Path: "a/Owners.java", Language: "Java", Classes: []archdoc.Class{
		{Name: "Owner", Decorators: []archdoc.Decorator{entity, {Name: "Table", Options: map[string]string{"name": "owners"}}}},
		{Name: "OwnerRepository", Interface: true, Options: map[string]string{"entity": "Owner"}},
		{Name: "OwnerService", Params: []archdoc.Param{{Name: "owners", Type: "OwnerRepository"}},
			Methods: []archdoc.Method{{Name: "rename", Invokes: []archdoc.Invocation{{Object: "owners", Method: "findById"}, {Object: "owners", Method: "save"}, {Object: "owners", Method: "flush"}}}}},
	}}}}
	springData([]archdoc.Source{src})
	q := src.Files[0].Classes[2].Methods[0].Queries
	if len(q) != 2 || q[0].Table != "owners" || q[0].Op != "reads" || q[1].Op != "writes" {
		t.Errorf("queries %+v", q)
	}

	classes := map[string]classAt{
		"IOrders":  {archdoc.Class{Name: "IOrders", Interface: true}, "a.cs"},
		"Orders":   {archdoc.Class{Name: "Orders", Extends: []string{"IOrders"}}, "a.cs"},
		"IPay":     {archdoc.Class{Name: "IPay", Interface: true}, "a.cs"},
		"Card":     {archdoc.Class{Name: "Card", Extends: []string{"IPay"}}, "a.cs"},
		"Transfer": {archdoc.Class{Name: "Transfer", Extends: []string{"IPay"}}, "a.cs"},
	}
	if impl := implementations(classes); impl["IOrders"] != "Orders" || impl["IPay"] != "" {
		t.Errorf("implementations %v — one implementation is followed, two are a choice the code makes at run time", impl)
	}
}
