package main

import (
	// add this

	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // add this
	"github.com/mbecker/oresources/dto"

	"github.com/gofiber/fiber/v2"
)

type DB struct {
	db                *sqlx.DB
	resourcesUUIDStmt *sqlx.Stmt
	resourcesTypeStmt *sqlx.Stmt
}

func main() {
	db := DB{}
	// POSTGRES
	connStr := "postgresql://keycloak:password@localhost:5432/api?sslmode=disable"
	// Connect to database
	dbb, err := sqlx.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	db.db = dbb

	// PREPARED STATEMENTS
	stmt, err := db.db.Preparex(`select r.resource_id , r."name", r."type", r2.scope_id, sc."name" as scope_name from resources r left join public.resourcesscopes r2 on r2.resource_id = r.resource_id left join public.scopes sc on sc.scope_id = r2.scope_id left join public.userpermissions u on u.resourcesscope_id = r2.resourcesscope_id where u.uuid = $1`)
	if err != nil {
		log.Fatal(err)
	}
	db.resourcesUUIDStmt = stmt

	stmt2, err := db.db.Preparex(`select r.resource_id , r."name", r."type", r2.scope_id, sc."name" as scope_name from resources r left join public.resourcesscopes r2 on r2.resource_id = r.resource_id left join public.scopes sc on sc.scope_id = r2.scope_id left join public.userpermissions u on u.resourcesscope_id = r2.resourcesscope_id where u.uuid = $1 and r."type" like '$2%'`)
	if err != nil {
		log.Fatal(err)
	}
	db.resourcesTypeStmt = stmt2

	// FIBER
	app := fiber.New()
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	app.Get("/", db.indexHandler) // Add this

	app.Get("/resources/:uuid", db.resourcesHandler) // Add this

	app.Put("/update", db.putHandler) // Add this

	app.Delete("/delete", db.deleteHandler) // Add this

	log.Fatalln(app.Listen(fmt.Sprintf(":%v", port)))
}

func (db *DB) indexHandler(c *fiber.Ctx) error {
	var resource dto.Resources
	var resources []dto.Resources
	rows, err := db.db.Queryx("SELECT * FROM public.resources")
	defer rows.Close()
	if err != nil {
		log.Println(err)
		c.Status(http.StatusInternalServerError).JSON("Error requesting resources")
		return nil
	}
	for rows.Next() {
		rows.StructScan(&resource)
		log.Printf("Resource: %#v", resource)
		resources = append(resources, resource)
	}
	return c.JSON(resources)
}
func (db *DB) resourcesHandler(c *fiber.Ctx) error {
	uuid := c.Params("uuid")

	var rows *sqlx.Rows
	var err error
	types := c.Query("type")
	if len(types) > 0 {
		log.Printf("Query resources with type: %s", types)
		rows, err = db.resourcesTypeStmt.Queryx(uuid, types)
	} else {
		rows, err = db.resourcesUUIDStmt.Queryx(uuid)
	}

	if err != nil {
		log.Println(err)
		c.Status(http.StatusInternalServerError).JSON("Error requesting resources")
		return nil
	}

	/*

		Following SQL rows / go structs are returned from the sql statement:

		dto.ResourcesPermission{ResourceID:2, Name:"ruv:kompass", Type:"org:team", ScopeID:"2", ScopeName:"org:update"}
		dto.ResourcesPermission{ResourceID:2, Name:"ruv:kompass", Type:"org:team", ScopeID:"3", ScopeName:"api:create"}
		dto.ResourcesPermission{ResourceID:2, Name:"ruv:kompass", Type:"org:team", ScopeID:"4", ScopeName:"api:delete"}
		dto.ResourcesPermission{ResourceID:2, Name:"ruv:kompass", Type:"org:team", ScopeID:"5", ScopeName:"api:update"}
		dto.ResourcesPermission{ResourceID:2, Name:"ruv:kompass", Type:"org:team", ScopeID:"6", ScopeName:"api:read"}
		dto.ResourcesPermission{ResourceID:3, Name:"ruv:racoon", Type:"org:team", ScopeID:"6", ScopeName:"api:read"}

	*/

	// Create the original 'resource tree'
	rTree := map[string]dto.ResourcesTree{}

	for rows.Next() {
		var resourcesPermission dto.ResourcesPermission
		rows.StructScan(&resourcesPermission)

		/*
			Split the "name" and the "types" to get the path of the name and the types as follows
			names := ["ruv", "kompass"]
			types := ["org", "team"]
		*/
		names := strings.Split(resourcesPermission.Name, ":")
		types := strings.Split(resourcesPermission.Type, ":")

		// For each SQL row reset the point to the 'original' rTree
		rTreeP := &rTree

		// For each SQL row create a new empty string for the type path like "org" ... "org:team" ... "org:team:service"
		var typePath string
		// Range the 'names' ["ruv"", "kompass"]
		for i, n := range names {

			// Create the 'type path': We are in loop 1; get all "types" from "0" to "1+1" and join them with ":"
			typePath = strings.Join(types[0:i+1], ":")

			// The loop internal 'resource tree' point to the 'pointer original tree'
			rt := *rTreeP
			// Check that the "name" (like "ruv" or "kompass" exists in the current internal pomted 'resource tree')
			rNode, exists := rt[n]
			// Create a new 'resource tree' to assign
			var resTree map[string]dto.ResourcesTree
			if !exists {
				resTree = map[string]dto.ResourcesTree{}
				// Last Element
				if i == len(types)-1 {
					rt[n] = dto.ResourcesTree{
						Name:      n,
						Type:      typePath,
						Scopes:    []string{resourcesPermission.ScopeName},
						Resources: resTree,
					}
				} else {
					// All elements before 'last name' in 'names'
					// The 'scopes' are in the returned SQL row are only valid for the last name; so just create an empty array of strings
					// Assign the the new resource tree 'resTree' that in the next loop we can add a new element to it
					rt[n] = dto.ResourcesTree{
						Name:      n,
						Type:      typePath,
						Scopes:    []string{},
						Resources: resTree,
					}
				}
			} else {
				// The 'name' like "ruv" already exists in the tree
				// Only for the last elemnt / 'name' the scope must be added to the already existing array of strings (scopes)
				// The internal 'resource Tree' "resTree" is now the 'resource tree' of the current node
				resTree = rNode.Resources
				if i == len(types)-1 {
					rNode.Scopes = append(rNode.Scopes, resourcesPermission.ScopeName)
					rt[n] = rNode
				}

			}
			rTreeP = &resTree
		}
	}
	return c.JSON(rTree)
}
func (db *DB) putHandler(c *fiber.Ctx) error {
	return c.SendString("Hello")
}
func (db *DB) deleteHandler(c *fiber.Ctx) error {
	return c.SendString("Hello")
}
