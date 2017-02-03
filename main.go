package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-oci8"
	"github.com/nu7hatch/gouuid"
	"log"
	"net/http"
)

type FrontEnd struct {
	DataHandler
}

type DataHandler interface {
	NewDB(string, string) (string, error)
	NewQuery(string, string) (string, error)
	QueryKey(string) (interface{}, error)
	QueryDBC(string, string) (interface{}, error)
}

type DBS struct {
	// key is uuid of dbc
	dbcs map[string]*sql.DB
	// key is uuid of a saved query
	queries map[string]Querier
}

type Querier struct {
	dbckey string
	query  string
}

type CredsJSON struct {
	Cnt string `json:"connection"`
}

type QueryJSON struct {
	Key   string `json:"key"`
	Query string `json:"query"`
}

func main() {
	dbcs := make(map[string]*sql.DB)
	defer func() {
		for _, dbc := range dbcs {
			dbc.Close()
		}
	}()

	queries := make(map[string]Querier)
	dbs := DBS{dbcs, queries}
	router := NewRouter(dbs)

	log.Printf("Listen on port: %d", 8800)
	log.Fatal(http.ListenAndServe(":8800", router))
}

func NewRouter(dbs DataHandler) *mux.Router {
	fe := FrontEnd{DataHandler: dbs}
	router := mux.NewRouter()
	router.Methods("POST").Path("/creds").Name("PostCREDS").Handler(http.HandlerFunc(fe.PostCREDS))
	router.Methods("POST").Path("/query").Name("PostQUERY").Handler(http.HandlerFunc(fe.PostQUERY))
	router.Methods("GET").Path("/query/{qkey}").Name("GetQUERY").Handler(http.HandlerFunc(fe.GetQUERY))
	return router
}

// Takes an interface and marshals to json
// Returns byte slice of json
func encode(mjson interface{}) []byte {
	encoded, err := json.Marshal(mjson)
	if err == nil {
		return encoded
	}
	log.Println(err)
	return []byte(`{"err": "internal server error while marshalling error message"}`)
}

// Post credtials connection url
// 200 returns url to query
// 406 if fails
func (fe FrontEnd) PostCREDS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	mjson := make(map[string]interface{})

	decoder := json.NewDecoder(r.Body)
	var t CredsJSON
	err := decoder.Decode(&t)
	if err != nil {
		log.Println(err)
		mjson["err"] = err.Error()
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write(encode(mjson))
		return
	}
	defer r.Body.Close()

	key, err := fe.NewDB("oci8", t.Cnt)
	if err != nil {
		log.Println(err)
		mjson["err"] = err.Error()
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write(encode(mjson))
		return
	}

	mjson["key"] = key

	w.WriteHeader(http.StatusOK)
	w.Write(encode(mjson))
}

// Post query for given key
// 200 returns json map of query response
// 406 if fails
func (fe FrontEnd) PostQUERY(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	mjson := make(map[string]interface{})

	decoder := json.NewDecoder(r.Body)
	var t QueryJSON
	err := decoder.Decode(&t)
	if err != nil {
		log.Println(err)
		mjson["err"] = err.Error()
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write(encode(mjson))
		return
	}
	defer r.Body.Close()

	data, err := fe.QueryDBC(t.Key, t.Query)
	if err != nil {
		log.Println(err)
		mjson["err"] = err.Error()
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write(encode(mjson))
		return
	}

	qkey, err := fe.NewQuery(t.Key, t.Query)
	if err != nil {
		log.Println(err)
		mjson["err"] = err.Error()
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write(encode(mjson))
		return
	}

	mjson["data"] = data
	mjson["qkey"] = qkey

	w.WriteHeader(http.StatusOK)
	w.Write(encode(mjson))
}

// Get a saved query for given querykey
// 200 returns json map of query response
// 406 if fails
func (fe FrontEnd) GetQUERY(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	qkey := vars["qkey"]

	mjson, err := fe.QueryKey(qkey)
	if err != nil {
		log.Println(err)
		errjson := make(map[string]interface{})
		errjson["err"] = err.Error()
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write(encode(errjson))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(encode(mjson))
}

// Takes SQLDatatype and connection string
// Returns dbcKey or err
func (dbs DBS) NewDB(dt string, c string) (string, error) {
	db, err := sql.Open(dt, c)
	if err != nil {
		return "", err
	}

	key, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	dbs.dbcs[key.String()] = db

	return key.String(), nil
}

// Takes dbcKey and query string
// Returns a queryKey
func (dbs DBS) NewQuery(key string, query string) (string, error) {
	qkey, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	qr := Querier{key, query}
	dbs.queries[qkey.String()] = qr

	return qkey.String(), nil
}

// Takes querykey
// Returns Query
func (dbs DBS) QueryKey(qkey string) (interface{}, error) {
	if q, ok := dbs.queries[qkey]; ok {
		return dbs.QueryDBC(q.dbckey, q.query)
	} else {
		return nil, errors.New("key does not exist")
	}
}

// Takes dbcKey and query string
// Returns DB rows or err
func (dbs DBS) QueryDBC(key string, query string) (interface{}, error) {
	if _, ok := dbs.dbcs[key]; !ok {
		return nil, errors.New("key does not exist")
	}

	db := dbs.dbcs[key]
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	final_result := make(map[string][]map[string]interface{})

	for rows.Next() {
		for i, _ := range columns {
			valuePtrs[i] = &values[i]
		}

		err = rows.Scan(valuePtrs...)
		if err != nil {
			return nil, err
		}

		var tmp_struct = make(map[string]interface{})
		for i, col := range columns {
			tmp_struct[col] = values[i]
		}

		final_result["data"] = append(final_result["data"], tmp_struct)
	}

	return final_result["data"], nil
}
