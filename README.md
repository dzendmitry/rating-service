Rating Service
========================
**Rating service** - it's just a simple composition of microservices is used as a service to keep notes.
It's written in Golang, using Mongo database as persistent storage and Redis kv storage as cache.
Mongo database and redis cache are replicated in master-slave mode for ensuring service availability.
Parsers are presented as separated microservices and discovered by rating service using multicast network.
Another one microservice is for users authentication. It communicates only with Mongo db storage.
Sessions mechanism is implemented using Mongodb TTL indexes.
... and of course Docker for testing and deployment.
![Scheme](https://github.com/dzendmitry/rating-service/blob/master/scheme.png)
