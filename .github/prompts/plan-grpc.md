I need to setup a gRPC service. I need the following

## cli command

add a server command.
The command should do the following

- use the db-uri cli option to init the db connection
- init the database connection using the pool in db/postgres and add telemetry
- setup the webserver using the connectrpc api
- the webserver should be instrumented with open telemetry

## service layer

the grpc services should be located in the directory services. each service should be done in its own package.
