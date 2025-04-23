package authz

import (
	"fmt"

	casbin "github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	"github.com/casbin/casbin/v3/persist"
	"go.etcd.io/bbolt"
)

const Model = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && r.act == p.act
`

var roles = [][2]string{
	{"super_admin", "admin"},
}

var permissions = [][3]string{
	{"super_admin", "/users", "POST"},
	{"super_admin", "/users/*/new-device-token", "POST"},

	{"admin", "/users", "GET"},
}

func New(a persist.Adapter) *casbin.Enforcer {
	m, err := model.NewModelFromString(Model)
	if err != nil {
		panic(fmt.Errorf("setting up authz model : %w", err))
	}

	e, err := casbin.NewEnforcer(m, a)
	if err != nil {
		panic(fmt.Errorf("setting up authz enforcer : %w", err))
	}

	for _, role := range roles {
		_, err := e.AddRoleForUser(role[0], role[1])
		if err != nil {
			panic(fmt.Errorf("adding role : %w", err))
		}
	}

	for _, permission := range permissions {
		_, err := e.AddPolicy(permission[0], permission[1], permission[2])
		if err != nil {
			panic(fmt.Errorf("adding permission : %w", err))
		}
	}

	return e

}

func NewBbolt(db *bbolt.DB) *casbin.Enforcer {
	adapter, err := NewAdapter(db, "Authz")
	if err != nil {
		panic(fmt.Errorf("setting up authz adapter : %w", err))
	}

	return New(adapter)
}
