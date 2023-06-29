# <img src="https://staticbackend.com/img/logo-sb-no-text.png" style="height: 35px" alt="StaticBackend logo" /> StaticBackend

![build badge](https://github.com/staticbackendhq/core/actions/workflows/go.yml/badge.svg)
[![GoReportCard](https://goreportcard.com/badge/github.com/staticbackendhq/core)](https://goreportcard.com/report/github.com/staticbackendhq/core)
[![Go Reference](https://pkg.go.dev/badge/github.com/staticbackendhq/core/backend.svg)](https://pkg.go.dev/github.com/staticbackendhq/core/backend)
<a href="https://discord.gg/vgh2PTp9ZB">
	<img src="https://img.shields.io/discord/872035652944928838?logoColor=%23DD0000">
</a>
<a href="https://twitter.com/staticbackend">
	<img src="https://img.shields.io/twitter/follow/staticbackend?color=DD0000&style=social">
</a>

[StaticBackend](https://staticbackend.com) is a simple backend server API that 
helps you create web applications. It handles most of the building blocks you'll 
need on the backend.

- [x] Authentication ([docs →](https://staticbackend.com/docs))
- [x] Account->users management ([docs →](https://staticbackend.com/docs/users))
- [x] Database CRUD, queries and full-text search ([docs →](https://staticbackend.com/docs/database))
- [x] Realtime/Websockets ([docs →](https://staticbackend.com/docs/websocket))
- [x] File storage ([docs →](https://staticbackend.com/docs/storage))
- [x] server-side functions ([docs →](https://staticbackend.com/docs/functions))
- [x] Schedule jobs
- [x] Send mails/sms ([docs →](https://staticbackend.com/docs/sendmail))
- [x] Caching ([docs →](https://staticbackend.com/docs/cache))
- [x] Handle forms ([docs →](https://staticbackend.com/docs/forms))
- [x] Resize images & convert URL to PDF ([docs →](https://staticbackend.com/docs/extras/))


## Table of content

* [Install](#install)
	* [Local development](#local-development)
	* [Frontend client](#frontend-client)
	* [Backend clients](#backend-clients)
	* [Go package](#go-package)
* [Usage](#usage)
	* [JavaScript example](#javascript-example)
	* [Go client example](#go-client-example)
	* [Go package example](#go-package-example)
* [Documentation](#documentation)
* [Deployment](#deployment)
	* [Render](#render)
	* [Heroku](#heroku)
	* [Docker](#docker)
* [Get support](#get-support)
* [Contributing](#contributing)
* [How you can help](#how-you-can-help)

## Install

You'll want to install different pieces depending on what you want to build. 
Here's what you can install:

### Local development

Our [CLI](https://github.com/staticbackendhq/cli) includes a fully functional 
development server. You don't need to install anything else.

```sh
$ npm install -g @staticbackend/cli
```

*You may 
[install the CLI manually](https://staticbackend.com/getting-started/cli) as 
well.*

This will install as the `backend` program. Start the development server with:

```sh
$ backend server
```

This command creates a new application and an admin user for you. You'll 
receive a PublicKey and a RootToken.

All HTTP request to the API requires a public key. The root token allows you 
to sign in to the dashboard for this application as the owner.

### Frontend client

Add the library to your dependencies:

```sh
$ npm install @staticbackend/js
```

Inside your module:

```javascript
import { Backend } from "@staticbackend/js";
const bkn = new Backend("dev_memory_pk", "dev");
```

**dev_memory_pk** is the default local development public key and **dev** is the 
default region / host for the instance you're targetting.

You may also include the library inside a `<script` tag if you're not using 
a module system:

```html
<script src="https://cdn.jsdelivr.net/npm/@staticbackend/js@1.5.0/dist/backend.min.js"></script>
<script>
	const bkn = new sb.Backend("dev_memory_pk", "dev");
</script>
```

### Backend clients

We've pre-built backend client libraries you may use directly:

**Node**:

```sh
$ npm install @staticbackend/backend
```

**Go**:

```sh
$ go get github.com/staticbackendhq/backend-go
```

[View the Go package documentation](https://pkg.go.dev/github.com/staticbackendhq/backend-go)

**Python**:

```sh
$ pip install staticbackend
```

### Go package

You can import a Go package directly into your Go program and build your 
application with the same building blocks without hosting the API separately.

```sh
$ go get github.com/staticbackendhq/core/backend
```

[View the Go package document](https://pkg.go.dev/github.com/staticbackendhq/core/backend)

## Usage

You may build web and mobile applications using StaticBackend as your main 
backend API.

StaticBackend is a multi-tenant platform allowing you to host multiple isolated 
applications.

You need an instance of the backend API running via the CLI for local 
development or running as a normal process with required dependencies.

You create your first application before you can start.

Using the CLI:

```sh
$ backend server
```

Using the source code:

```sh
$ git clone https://github.com/staticbackendhq/core
$ cd core
$ cp .local.env .env
$ make start
```

Visit [http://localhost:8099](http://localhost:8099) and create an application.

### Javascript example

*Note that the Nodejs client library has the same API / function names as the 
JavaScript library.*

```javascript
import { Backend } from "@staticbackend/js";

const bkn = new Backend("dev_memory_pk", "dev");

let token = "";

login = async () => {
	const res = await bkn.register("email@test.com", "password");
	if (!res.ok) {
		console.error(res.content);
		return;
	}
	token = res.content;

	createTask();
}

createTask = async () => {
	const task = {
		desc: "Do something for XYZ",
		done: false
	};

	const res = bkn.create(token, "tasks", task);
	if (!res.ok) {
		console.error(res.content);
		return;
	}
	console.log(res.content);
}
```

### Go client example

```go
package main

import (
	"fmt"
	"github.com/staticbackendhq/backend-go"
)

func main() {
	backend.PublicKey = "dev_memory_pk"
	backend.Region = "dev"

	token, err := backend.Login("admin@dev.com", "devpw1234")
	// no err handling in example

	task := new(struct{
		ID string `json:"id"`
		AccountID string `json:"accountId"`
		Title string `json:"title"`
		Done bool `json:"done"`
	})
	task.Title = "A todo item"
	err = backend.Create(token, "tasks", task, &task)
	// task.ID and task.AccountID would be filled with proper values
}
```

### Go package example

```go
// using the cache & pub/sub
backend.Cache.Set("key", "value")

msg := model.Command{Type: "chan_out", Channel: "#lobby", Data: "hello world"}
backend.Cache.Publish(msg)

// use the generic Collection for strongly-typed CRUD and querying
type Task struct {
	ID string `json:"id"`
	Title string `json:"title"`
}
// auth is the currently authenticated user performing the action.
// base is the current tenant's database to execute action
// "tasks" is the collection name
tasks := backend.Collection[Task](auth, base, "tasks")
newTask, err := tasks.Create(Task{Title: "testing"})
// newTask.ID is filled with the unique ID of the created task in DB
```

View a 
[full example in the doc](https://pkg.go.dev/github.com/staticbackendhq/core/backend#example-package).

## Documentation

We're trying to have the best experience possible reading our documentation.

Please help us improve if you have any feedback.

* [Documentation with code samples for client libraries and CURL](https://staticbackend.com/docs)
* [Go client library package](https://pkg.go.dev/github.com/staticbackendhq/backend-go)
* [Go importable package](https://pkg.go.dev/github.com/staticbackendhq/core/backend)
* [Self-host guide](https://staticbackend.com/getting-started/self-hosting)
* [Install the CLI](https://staticbackend.com/getting-started/cli)

**Examples**:

* [To-do list example](https://staticbackend.com/getting-started/)
* [Realtime collaboration](https://staticbackend.com/blog/realtime-collaboration-example/)
* [Live chat using server-side function & real-time component](https://staticbackend.com/blog/server-side-functions-task-scheduler-example/)
* [Jamstack Bostom talk](https://www.youtube.com/watch?v=Uf-K6io9p7w)

## Deployment

To deploy StaticBackend you'll need the following:

* Either PostgreSQL or MongoDB
* Redis

StaticBackend is a single file binary you can run as a `systemd` daemon.

Here's some quick way to deploy an instance.

### Render

[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy)

### Heroku

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/staticbackendhq/core)

### Docker

If you have Docker and Docker Compose ready, here's how to run StaticBackend:

```shell
$ git clone https://github.com/staticbackendhq/core
$ cd core
$ cp .demo.env .env
$ docker build . -t staticbackend:latest
$ docker-compose -f docker-compose-demo.yml up
```

Open a browser at [http://localhost:8099](http://localhost:8099) to create 
your first application.

For production, you'll want to configure environment variables found in `.env` 
file.

* [Self-hosting guide](https://staticbackend.com/getting-started/self-hosting/)
* [Video showing how to self-host](https://www.youtube.com/watch?v=vQjfaMxidx4)
* [Detailed blog post on how to self-host](https://staticbackend.com/blog/get-started-self-hosted-version/)

## Get support

You may use the following channels to get help and support.

* [Discord](https://discord.gg/vgh2PTp9ZB): for any help and joining the conversation.
* [GitHub issues](https://github.com/staticbackendhq/core/issues): To report bugs / contributing code.
* [GitHub Discussions](https://github.com/staticbackendhq/core/discussions): For ideas, feature requests and general discussions.

## Contributing

If you have any feedback (good or bad) we'd be more than happy to talk. Please 
use the [Discussions](https://github.com/staticbackendhq/core/discussions) tab.

Same for contributing. The easiest is to get in touch first. We're working 
to make it easier to contribute code. If you'd like to work on something 
precise let us know.

Here are videos made specifically for people wanting to contribute:

* [Intro, setup, running tests, project structure](https://youtu.be/uTj7UEbg0p4)
* [backend package and v1.4.1 refactor and changes](https://youtu.be/oWxk2g2yp_g)

Check the [contributing file](CONTRIBUTING.md) for details.


## How you can help

If you're looking to help the project, here are multiple ways:

* Use it and share your experiences.
* Sponsor the development via GitHub sponsors.
* Spread the words, a tweet, a blog post, any mention is helpful.
* Join the [Discord](https://discord.gg/vgh2PTp9ZB) server.