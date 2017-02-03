# Orcale DB Interfacer
Oracle Databases are not an easy thing to talk to. So I made a Interfacer to make it easier.

## Demo
```
POST: http://hostname:8800/creds
Header: Content-Type = application/json
Body: {"connection":"username/password@DBURL:PORT"}

Response: {"key":"a5c44a8f-bc74-4e49-6fec-f5701548a1e1"}

```
```
POST: http://hostname:8800/query
Header: Content-Type = application/json
Body: {"key":"a5c44a8f-bc74-4e49-6fec-f5701548a1e1", "query":"select * mytable"}

Response:
{
  "data": [
    {"column1": "tim", "column2": "cook"},
    {"column1": "joe", "column2": "cool"}
  ],
  "qkey": "9e0902b9-d693-468d-4312-4427f02482dd"
}
```
```
GET: http://hostname:8800/query/9e0902b9-d693-468d-4312-4427f02482dd

Response:
[
  {"column1": "tim", "column2": 1022},
  {"column1": "joe", "column2": 1011}
]
```

## Build
For this build you will need the Oracle Instant Client installed on your system.

### Oracle Instant Client Install Guide
Go to the [Download Page](http://www.oracle.com/technetwork/database/features/instant-client/index.html) and download these 3 packages
- instantclient-basic-linux.x64-12.1.0.2.0.zip
- instantclient-sdk-linux.x64-12.1.0.2.0.zip
- instantclient-sqlplus-linux.x64-12.1.0.2.0.zip

Unpack all the contents of these zips into one file called instantclient_12_1.
- Make symlink libclntsh.so -> libclntsh.so.12.1
- Make symlink libocci.so -> libocci.so.12.1

Then set oic needed env variable.
```
export LD_LIBRARY_PATH=/opt/oic/instantclient_12_1
```

### Build Go Code
Use $glide up; to install the Go deps into a folder called vendor.

Next set you oic8.pc file to match your OIC install. I installed it in /opt/oic.
```
includedir=/opt/oic/instantclient_12_1/sdk/include
libdir=/opt/oic/instantclient_12_1

Name: oci8
Description: Oracle Instant Client
Version: 12.1
Cflags: -I${includedir}
Libs: -L${libdir} -lclntsh
```

Then set mattn-oci8 needed env variable.
```
export PKG_CONFIG_PATH=$HOME/src/sql_keys_oracle/vendor/github.com/mattn/go-oci8/
go build
```
