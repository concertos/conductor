package resources

import (
	"github.com/emicklei/go-restful"
	"net/http"
	"github.com/emicklei/go-restful-openapi"
	"github.com/concertos/conductor/pkg/conductor"
	"context"
	"log"
	"encoding/json"
	"github.com/ventu-io/go-shortid"
	"strings"
)

type User struct {
	Id   string `json:"id" description:"identifier of the user"`
	Password string `json:"password" description:"password of user"`
	Name string `json:"name" description:"name of the user"`
	Created  int    `json:"created" description:"created time`
}

type UserResource struct {
	// normally one would use DAO (data access object)
	users map[string]User
}

// WebService creates a new service that can handle REST requests for User resources.
func (u UserResource) WebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
	Path("/users").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON) // you can specify this per route as well

	tags := []string{"users"}

	ws.Route(ws.GET("/").To(u.findAllUsers).
	// docs
		Doc("get all users").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]User{}).
		Returns(200, "OK", []User{}))

	ws.Route(ws.GET("/{user-id}").To(u.findUser).
	// docs
		Doc("get a user").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("integer").DefaultValue("1")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(User{}). // on the response
		Returns(200, "OK", User{}).
		Returns(404, "Not Found", nil))

	ws.Route(ws.PUT("/{user-id}").To(u.updateUser).
	// docs
		Doc("update a user").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(User{})) // from the request

	ws.Route(ws.POST("").To(u.createUser).
	// docs
		Doc("create a user").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(User{})) // from the request

	ws.Route(ws.DELETE("/{user-id}").To(u.removeUser).
	// docs
		Doc("delete a user").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")))

	return ws
}

func (u UserResource) findAllUsers(request *restful.Request, response *restful.Response) {
	c := conductor.GetConductor()
	resp, err := c.KeysAPI.Get(context.Background(), "/users", nil)
	if err != nil {
		log.Println("err read users")
	}
	var users []User
	for _, v := range resp.Node.Nodes {
		var user User
		json.Unmarshal([]byte(v.Value), &user)

		// userId is last string after /
		arr := strings.Split(v.Key, "/")
		user.Id = arr[len(arr) - 1]

		users = append(users, user)
	}

	response.WriteEntity(users)
}

func (u UserResource) findUser(request *restful.Request, response *restful.Response) {
	c := conductor.GetConductor()
	id := request.PathParameter("user-id")
	resp, err := c.KeysAPI.Get(context.Background(), "/users/" + id, nil)
	if err != nil {
		log.Println("catnot get user from etcd")
		response.WriteErrorString(http.StatusNotFound, "User could not be found.")
		return
	}
	var user User
	json.Unmarshal([]byte(resp.Node.Value), &user)
	response.WriteEntity(user)

}

func (u *UserResource) updateUser(request *restful.Request, response *restful.Response) {
	c := conductor.GetConductor()
	usr := new(User)
	err := request.ReadEntity(&usr)
	if err == nil {
		str, _ := json.Marshal(usr)
		c.KeysAPI.Set(context.Background(), "/users/" + usr.Id, string(str), nil)
		response.WriteHeaderAndEntity(http.StatusCreated, usr)
		response.WriteEntity(usr)
	} else {
		response.WriteError(http.StatusInternalServerError, err)
	}
}

func (u *UserResource) createUser(request *restful.Request, response *restful.Response) {
	c := conductor.GetConductor()
	sid, _ := shortid.New(1, shortid.DefaultABC, 2342)
	uid, _ := sid.Generate()
	usr := User{Id: uid}
	err := request.ReadEntity(&usr)
	if err == nil {
		str, _ := json.Marshal(usr)
		c.KeysAPI.Set(context.Background(), "/users/" + usr.Id, string(str), nil)
		response.WriteHeaderAndEntity(http.StatusCreated, usr)
	} else {
		response.WriteError(http.StatusInternalServerError, err)
	}
}

func (u *UserResource) removeUser(request *restful.Request, response *restful.Response) {
	c := conductor.GetConductor()
	id := request.PathParameter("user-id")

	c.KeysAPI.Delete(context.Background(), "/users/" + id, nil)
}
