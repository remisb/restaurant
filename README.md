# Restaurant service API

## Prerequisites

* Docker is required to run this software on your local machine.
* aaa


### Installing Docker

This project requires the use of Docker since images are created and run in a Docker-Compose environment. 
[Installing Docker](https://docs.docker.com/get-docker/)


### Building the project

A makefile has also been provide to allow building, running and testing the software easier.

### Running the project


### Stopping the project

You can hit C in the terminal window running make up. 
Once that shutdown sequence is complete, it is important to run the make down command.

```bash
$ <ctrl>C
$ make down
```

Running make down will properly stop and terminate the Docker Compose session.

## Making Requests

Once the project is running, you will want to hit the sales-api endpoints. 
The following is required in order to have success web API access.

### Seeding The Database

To do anything the database needs to be defined and seeded with data. This will also create the initial user.

```bash
$ cd $GOPATH/src/github.com/ardanlabs/service
$ make seed
```

### Authenticating

Before any requests can be sent you must acquire an auth token. 
Make a request using HTTP Basic auth with the test user email and password to get the token.

```bash
$ curl --user "admin@example.com:gophers" http://localhost:3000/v1/users/token
```

I suggest putting the resulting token in an environment variable like $TOKEN.

```bash
$ export TOKEN="COPY TOKEN STRING FROM LAST CALL"
```

This will create a user with email admin@example.com and password gophers.

### Authenticated Requests

To make authenticated requests put the token in the Authorization header with the Bearer prefix.

```bash
$ curl -H "Authorization: Bearer ${TOKEN}" http://localhost:3000/v1/users
``

