/*   Copyright 2020 Mark Lakes
 *
 *   Licensed under the Apache License, Version 2.0 (the "License");
 *   you may not use this file except in compliance with the License.
 *   You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 *   Unless required by applicable law or agreed to in writing, software
 *   distributed under the License is distributed on an "AS IS" BASIS,
 *   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *   See the License for the specific language governing permissions and
 *   limitations under the License.
 */
package mongoclt

import (
	"context"
	"errors"
	"strings"
	"time"
	
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	keyField = "key" // default of expected key field for each entry called "key"

	// error keys in the error map : user may change their values with SetErrorMap()
	errNoDocsKey = "no documents in result"
	errNoDocsVal = "not found"
	errNoFindEntKey = "failed to find entity="
	errNoFindKeyKey = "failed to find key="
	errNoMatch = "no match found for entity="
	errNoDelKey = "failed to delete entity="
	errMissKeyKey = "missing key field"
)

var keyFieldName string = keyField


// key = partial mongo error string
// value = normalized string
var nerrorMap map[string]string = map[string]string{
	errNoDocsKey:errNoDocsVal,
	errNoFindEntKey:errNoFindEntKey,
	errNoFindKeyKey:errNoFindKeyKey,
	errMissKeyKey:errMissKeyKey,
	errNoMatch:errNoMatch,
	errNoDelKey:errNoDelKey,
}


// SetKeyFieldName allows user to over-ride the default name of the "key" field
func SetKeyFieldName(keyName string) {
	keyFieldName = keyName
}

// SetErrorMap allows user to create normalized errors for their apps to handle without exposing
// underlying mongodb specific errors
func SetErrorMap(mongoErrStr, normalizedErrStr string) {
	nerrorMap[mongoErrStr] = normalizedErrStr
}

type Client struct {
	client *mongo.Client
	opts *cltOptions
}

// ClientOption specifies an option for connecting to a mongodb server
type ClientOption struct {
	f func(*cltOptions)
}

type cltOptions struct {
	hostPorts string // "host:port" or "host:port,host2:port2..."
	dbUser string   // user to connect to database
	dbPasswd string // password for db user
	dbAuthDB string // name of auth database to auth the connection if needed
	dbName string   // name of database to connect to
	commTimeoutMS time.Duration  // millisecs
}

// ClientHostPorts specifies the host and port inwhich to access the database
// ex: "127.0.0.1:27017"
// Supports sharded db - can call multiple times host+ports
func ClientHostPort(hostPort string) ClientOption {
	return ClientOption{func(co *cltOptions) {
		if co.hostPorts == "" {
			co.hostPorts = hostPort
		} else {
			co.hostPorts = co.hostPorts + "," + hostPort
		}
	}}
}

// ClientDbUser specifies the user inwhich to access the database
func ClientDbUser(user string) ClientOption {
	return ClientOption{func(co *cltOptions) {
		co.dbUser = user
	}}
}

// ClientDbPasswd specifies the password for user to access the database
func ClientDbPasswd(passwd string) ClientOption {
	return ClientOption{func(co *cltOptions) {
		co.dbPasswd = passwd
	}}
}

// ClientDbAuthDbName specifies the name of auth database containing the user when connecting to the server
// Optional: This was required on Ubuntu but not on MAC
func ClientAuthDbName(name string) ClientOption {
	return ClientOption{func(co *cltOptions) {
		co.dbAuthDB = name
	}}
}

// ClientDbName specifies the name of database to write or read data from
func ClientDbName(name string) ClientOption {
	return ClientOption{func(co *cltOptions) {
		co.dbName = name
	}}
}

// ClientCommTimeout specifies the timeout for communicating to the mongo server
func ClientCommTimeout(timeOut int) ClientOption {
	return ClientOption{func(co *cltOptions) {
		co.commTimeoutMS = time.Duration(timeOut) * time.Millisecond
	}}
}


func NewClient(coptions ...ClientOption) (*Client, error) {
	opts := cltOptions{}
	for _, option := range coptions {
		option.f(&opts)
	}

	// using SCRAM auth
	loginCreds := opts.dbUser + ":" + opts.dbPasswd + "@"
	url := "mongodb://" + loginCreds + opts.hostPorts // works on mac without the auth db suffix

	if len(opts.dbAuthDB) > 0 {
		// use the database auth name when on ubuntu-18.04
		// ex: mongodb://foo:bar@localhost:27017/mydb
		url = url + "/" + opts.dbAuthDB
	}
	cltOpts := options.Client()
	cltOpts = cltOpts.ApplyURI(url)
	cltOpts = cltOpts.SetSocketTimeout(opts.commTimeoutMS)
	connTimeOutMS := opts.commTimeoutMS * 2
	cltOpts = cltOpts.SetConnectTimeout(connTimeOutMS)

	clt, err := mongo.NewClient(cltOpts)
	if err != nil {
		return nil, normalizeError(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), (connTimeOutMS / 1000) * time.Second)
	defer cancel()
	err = clt.Connect(ctx)
	if err != nil {
		return nil, normalizeError(err)
	}

	client := Client{
		clt,
		&opts,
	}
	return &client, nil
}

//FIX func (clt *Client) Create(entity, key string, val interface{}) (*map[string]interface{}, error) {
// Create or insert a new entry into the collection entity
func (clt *Client) Create(entity, keyValue string, valueEntry map[string]interface{}) (*map[string]interface{}, error) {
	//FIX valMap := val.(map[string]interface{})
	if _, ok := valueEntry[keyFieldName]; !ok {
		valueEntry[keyFieldName] = keyValue
	}

	coll := clt.client.Database(clt.opts.dbName).Collection(entity)
	res, err := coll.InsertOne(context.Background(), valueEntry)
	if err != nil {
		return nil, normalizeError(err)
	}

	result := make(map[string]interface{})
	result["_id"] = res.InsertedID
	result[keyFieldName] = keyValue

	return &result, nil
}

func (clt *Client) Read(entity, keyValue string) (*map[string]interface{}, error) {

	coll := clt.client.Database(clt.opts.dbName).Collection(entity)
	if coll == nil {
		errMsg := nerrorMap[errNoFindEntKey] + entity
		return nil, errors.New(errMsg)
	}

	opts := options.FindOne().SetSort(bson.D{{keyFieldName, 1}}) // sort on key values
	sr := coll.FindOne(context.Background(), bson.D{{keyFieldName, keyValue}}, opts)
	if sr == nil {
		errMsg := nerrorMap[errNoFindKeyKey] + keyValue
		return nil, errors.New(errMsg)
	}

	result := make(map[string]interface{})
	err := sr.Decode(&result)
	if err != nil {
		return &result, normalizeError(err)
	}

	for key, value := range result {
		v := convertToNative(value)
		result[key] = v
	}

	return &result, nil
}


func convertToNative(value interface{}) interface{} {

	if av, ok := value.(primitive.A); ok {
		// convert into generic array
		pa := make([]interface{}, len(av))
		for i, pav := range av {
			pa[i] = convertToNative(pav)
		}
		return pa
	} else if mv, ok := value.(primitive.M); ok {
		pm := make(map[string]interface{})
		for k, v := range mv {
			pm[k] = convertToNative(v)
		}
		return pm
	} else if dv, ok := value.(primitive.D); ok {
		return convertToNative(dv.Map())
	} else if dtv, ok := value.(primitive.DateTime); ok {
		return dtv.Time()
	} else if ev, ok := value.(primitive.E); ok {
		emap := map[string]interface{}{ev.Key: ev.Value}
		return emap
	} else if oidv, ok := value.(primitive.ObjectID); ok {
		return oidv.String()
	} else if sv, ok := value.(primitive.Symbol); ok {
		s := string(sv)
		return s
	}

	return value
}


func normalizeError(err error) error {
	if err == nil {
		return nil
	}

	for key, val := range nerrorMap {
		//if strings.Contains(err.Error(), "no documents in result") {
			//return errors.New("not_found")
		//}
		if strings.Contains(err.Error(), key) {
			return errors.New(val)
		}
	}

	return err
}

