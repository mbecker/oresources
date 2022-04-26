package main

import (
	// add this

	"database/sql"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
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

	stmt2, err := db.db.Preparex(`select r.resource_id , r."name", r."type", r2.scope_id, sc."name" as scope_name from resources r left join public.resourcesscopes r2 on r2.resource_id = r.resource_id left join public.scopes sc on sc.scope_id = r2.scope_id left join public.userpermissions u on u.resourcesscope_id = r2.resourcesscope_id where u.uuid = $1 and r."type" like '$2%';`)
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
	app.Get("/roles/:uuid", db.groupsHandler)
	app.Get("/rolestest/:uuid", db.rolesHandler)

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

func (db *DB) rolesHandler(c *fiber.Ctx) error {
	uuid := c.Params("uuid")
	if uuid == "" {
		c.Status(http.StatusInternalServerError).JSON("Error requesting resources")
		return nil
	}

	rows, err := db.db.Queryx(`select r.role_id, r.name from public.roles r left join public.userroles u ON u.role_id = r.role_id where u.uuid =$1;`, uuid)

	if err != nil {
		log.Println(err)
		c.Status(http.StatusInternalServerError).JSON("Error requesting resources")
		return nil
	}

	resources := []string{}
	roles := []string{}
	rolesResources := dto.ResourcesRolesResult{
		ResourcesRoles: dto.ResourcesRoles{},
	}
	for rows.Next() {
		var dbRole dto.DBRole
		rows.StructScan(&dbRole)
		log.Printf("%#v", dbRole)
		resroles := strings.Split(dbRole.Name, ":")
		if len(resroles) == 0 {
			continue
		}
		lenresroles := len(resroles)
		role := resroles[lenresroles-1]
		resource := strings.Join(resroles[0:lenresroles-1], ":")
		resources = append(resources, resource)
		roles = append(roles, role)
		_, exists := rolesResources.ResourcesRoles[resource]
		if !exists {
			rolesResources.ResourcesRoles[resource] = []string{}
		}
		rolesResources.ResourcesRoles[resource] = append(rolesResources.ResourcesRoles[resource], role)

	}

	if len(roles) == 0 {
		c.Status(http.StatusInternalServerError).JSON("No roles")
		return nil
	}

	q := fmt.Sprintf("select r.resource_id , r.name, r.type, r2.scope_id, sc.name as scope_name from public.resources r left join public.resourcesscopes r2 on r2.resource_id = r.resource_id left join public.scopes sc on sc.scope_id = r2.scope_id left join public.userpermissions u on u.resourcesscope_id = r2.resourcesscope_id where u.uuid = '%s' or r.type ~* '%s';", uuid, strings.Join(resources, "|"))
	log.Println(q)
	resourceRows, err := db.db.Queryx(q)
	if err != nil {
		log.Println(err)
		c.Status(http.StatusInternalServerError).JSON("Error requesting resources")
		return nil
	}
	rTree := getResourceTree(resourceRows, "", 999999999, rolesResources.ResourcesRoles)
	rolesResources.ResourceTee = *rTree
	return c.JSON(rolesResources)

}

func (db *DB) groupsHandler(c *fiber.Ctx) error {
	uuid := c.Params("uuid")
	if uuid == "" {
		c.Status(http.StatusInternalServerError).JSON("Error requesting resources")
		return nil
	}

	rows, err := db.db.Queryx(`select r.name from public.roles r left join public.userroles u ON u.role_id = r.role_id where u.uuid =$1;`, uuid)

	if err != nil {
		log.Println(err)
		c.Status(http.StatusInternalServerError).JSON("Error requesting resources")
		return nil
	}

	roles := []dto.Role{}
	for rows.Next() {
		var dbRole dto.DBRole
		rows.StructScan(&dbRole)
		log.Printf("%#v", dbRole)
		if dbRole.Name == "" {
			continue
		}
		role := dto.Role{
			Name:       dbRole.Name,
			Permission: []dto.ResourcesPermission{},
		}
		// Has "type" in name
		types := strings.Split(dbRole.Name, ":")
		if len(types) > 0 {
			role.Role = types[len(types)-1]
			role.Type = strings.Join(types[0:len(types)-1], ":")
		}
		q := fmt.Sprintf("select r.resource_id , r.name, r.type, r2.scope_id, sc.name as scope_name from public.resources r left join public.resourcesscopes r2 on r2.resource_id = r.resource_id left join public.scopes sc on sc.scope_id = r2.scope_id where r.type like '%s%%';", role.Type)
		resourceRows, err := db.db.Queryx(q)
		if err != nil {
			log.Println(err)
			continue
		}
		rTree := getResourceTree(resourceRows, "", 1000000, dto.NewResourcesRoles())
		role.ResourceTee = *rTree
		for resourceRows.Next() {
			var resourcesPermission dto.ResourcesPermission
			resourceRows.StructScan(&resourcesPermission)
			role.Permission = append(role.Permission, resourcesPermission)
		}

		roles = append(roles, role)

	}
	return c.JSON(roles)
}

// http://localhost:3000/resources/e3cb82c9-6b37-4d13-8583-344e83ad74af
// http://localhost:3000/resources/e3cb82c9-6b37-4d13-8583-344e83ad74af?type=org:team
// http://localhost:3000/resources/e3cb82c9-6b37-4d13-8583-344e83ad74af?type=org
// http://localhost:3000/resources/e3cb82c9-6b37-4d13-8583-344e83ad74af?depth=2&type=org:team
func (db *DB) resourcesHandler(c *fiber.Ctx) error {

	uuid := c.Params("uuid")
	if uuid == "" {
		c.Status(http.StatusInternalServerError).JSON("Error requesting resources")
		return nil
	}

	// Query Parameters
	qType := c.Query("type")
	qDepth, errAtoi := strconv.Atoi(c.Query("depth"))
	if errAtoi != nil {
		qDepth = math.MaxInt
	}

	var rows *sqlx.Rows
	var err error

	if len(qType) > 0 {
		log.Printf("Query resources with type: %s", qType)
		q := fmt.Sprintf("select r.resource_id , r.name, r.type, r2.scope_id, sc.name as scope_name from resources r left join public.resourcesscopes r2 on r2.resource_id = r.resource_id left join public.scopes sc on sc.scope_id = r2.scope_id left join public.userpermissions u on u.resourcesscope_id = r2.resourcesscope_id where u.uuid = '%s' and r.type like '%s%%';", uuid, qType)
		log.Println(q)
		rows, err = db.db.Queryx(q)
	} else {
		rows, err = db.resourcesUUIDStmt.Queryx(uuid)
	}

	// Error no rows: Returny empty object
	if err == sql.ErrNoRows {
		log.Println(err)
		c.JSON(map[string]dto.ResourcesTree{})
		return nil
	}
	// Generic error
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
	rTree := getResourceTree(rows, qType, qDepth, dto.NewResourcesRoles())

	return c.JSON(rTree)
}
func (db *DB) putHandler(c *fiber.Ctx) error {
	return c.SendString("Hello")
}
func (db *DB) deleteHandler(c *fiber.Ctx) error {
	return c.SendString("Hello")
}

func getResourceTree(rows *sqlx.Rows, qType string, qDepth int, resourcesRoles dto.ResourcesRoles) *map[string]dto.ResourcesTree {
	// Create the original 'resource tree'
	rTree := map[string]dto.ResourcesTree{}
	log.Println("RESULTS:")
	for rows.Next() {
		var resourcesPermission dto.ResourcesPermission
		rows.StructScan(&resourcesPermission)
		fmt.Printf("%#v\n", resourcesPermission)
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
		x := 0
		// Range the 'names' ["ruv"", "kompass"]
		roles := []string{}
		for i, n := range names {

			// Create the 'type path': We are in loop 0; get all "types" from "0" to "0+1" and join them with ":"
			typePath = strings.Join(types[0:i+1], ":")
			if len(qType) > 0 && !strings.HasPrefix(typePath, qType) {
				continue
			}

			if x > qDepth {
				continue
			}
			x++

			// Get Resources Roles for the current "typePath" ["org", "org:team", "org:team:service" ...] and append it to the current roles array; the roles are added to each each resource in the tree
			rr, existsrr := resourcesRoles[typePath]
			if existsrr {
				roles = addIfNotExists(roles, rr)
			}

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
						Name:         n,
						Roles:        roles,
						OriginalName: resourcesPermission.Name,
						Type:         typePath,
						Scopes:       []string{resourcesPermission.ScopeName},
						Resources:    resTree,
					}
				} else {
					// All elements before 'last name' in 'names'
					// The 'scopes' are in the returned SQL row are only valid for the last name; so just create an empty array of strings
					// Assign the the new resource tree 'resTree' that in the next loop we can add a new element to it
					rt[n] = dto.ResourcesTree{
						Name:      n,
						Roles:     roles,
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
					rNode.Roles = addIfNotExists(rNode.Roles, roles)
					rt[n] = rNode
				}

			}
			rTreeP = &resTree
		}
	}
	return &rTree
}

func removeDuplicate(array []string) []string {
	m := make(map[string]string)
	for _, x := range array {
		m[x] = x
	}
	var ClearedArr []string
	for x, _ := range m {
		ClearedArr = append(ClearedArr, x)
	}
	return ClearedArr
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func addIfNotExists(primary []string, secondary []string) []string {
	for _, s := range secondary {
		if !contains(primary, s) {
			primary = append(primary, s)
		}
	}
	return primary
}
