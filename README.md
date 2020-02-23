# mongo-clt-go-api
Client API for accessing mongodb

mongo-clt-go-api is a golang implementation for accessing mongodb.

## Installation
Install mongo-clt-go-api with go tool:
```
    go get github.com/bluelamar/mongo-clt-go-api
```

## Usage
To use mongo-clt-go-api, you need import the package and create a new client
with options. This client uses SCRAM authentication with mongodb.
```go
import "github.com/bluelemar/mongo-clt-go-api"

clt, err := mongoclt.NewClient(
	mongoclt.ClientHostPort("127.0.0.1:7000"),
	mongoclt.ClientHostPort("127.0.0.1:7001"),
	mongoclt.ClientDbUser("my_mongo_user"),
	mongoclt.ClientDbPasswd("my_mongo_password"),
	mongoclt.ClientDbName("my_database"),
	mongoclt.ClientDbAuthDbName("my_database"),
	mongoclt.ClientCommTimeout(50 * time.Millisecond)
)
```

## Run Test
Run mongodb and initialize with your users and database.

Start mongodb then copy paste the mongo db commands.

Start the server and connect to it with the mongo command shell:
```
scripts/initdb.sh
```

Copy and paste the command sets within the command shell:

First copy/paste the Admin user setup:
```
use testdb
db.createUser(
  {
    user: "myadmin",
    pwd: "mypassword",
    roles: [
       { role: "read", db: "admin" },
       { role: "readWrite", db: "testdb" }
    ]
  }
)
```

Next copy/paste the database index setup for a collection called "rooms":
```
use testdb
db.rooms.createIndex( { "key": 1 }, { unique: true } )
```

Lastly, copy/paste some test data:
```
db.rooms.insertOne(
{
key: "306",
RoomNum:"306",
BedSize:"Twin",
NumBeds:"1",
RateClass:"STANDARD ROOM"
})
```

## Run mongodb with kubernetes
FIX TODO how to init the db with the user+passwd


