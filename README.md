<p align="center">
	<img src="https://staticbackend.com/img/logo-sb-no-text.png"  alt="StaticBackend logo">
	<br />
	<a href="https://discord.gg/vgh2PTp9ZB">
		<img src="https://img.shields.io/discord/872035652944928838?logoColor=%23DD0000">
	</a>
	<a href="https://twitter.com/staticbackend">
		<img src="https://img.shields.io/twitter/follow/staticbackend?color=DD0000&style=social">
	</a>
	
</p>

# StaticBackend - simple backend for your apps

[StaticBackend](https://staticbackend.com) is a simple backend that handles 
user management, database, file storage, forms, and real-time experiences via 
channel/topic-based communication for web and mobile applications.

You can think of it as a lightweight Firebase replacement you may self-host. No 
vendor lock-in, and your data stays in your control.

### Table of content

* [What can you build](#what-can-you-build)
* [How it works / dev workflow](#how-it-works--dev-workflow)
* [Get started with the self-hosted version](#get-started-with-the-self-hosted-version)
* [Documentation](#documentation)
* [Librairies & CLI](#librairies--cli)
* [Examples](#examples)
* [Deploying in production](#deploying-in-production)
* [Feedback & contributing](#feedback--contributing)
* [Open source, sponsors, paid SaaS](#open-source-sponsors-paid-saas)
* [Spread the words](#spread-the-words)


## What can you build

I built StaticBackend with the mindset of someone tired of writing the same code 
over and over on the backend. If your application needs one or all of 
user management, database, file storage, real-time interactions, it should be 
a good fit.

I'm personally using it to build SaaS:

* [Tangara - one page checkout for creators](https://tangara.io)

## How it works / dev workflow

The main idea is that StaticBackend is your backend API for your frontend apps. 
A performant free and open-source Firebase alternative.

_Note that it can also be used from your backend code as well._

Once you have an instance running and your first app created, you may install 
the JavaScript client-side library:

```shell
$> npm install @staticbackend/js
```

Let's create a user account and get a session `token` and create a `task` 
document in the `tasks` collection:

```javascript
import { Backend } from "@staticbackend/js";

const bkn = new Backend("your_public-key", "dev");

let token = "";

login = async () => {
	const res = await bkn.login("email@test.com", "password");
	if (!res.ok) {
		console.error(res.content);
		return;
	}
	token = res.content();

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

The last `console.log` prints

```json
{
	"id": "123456-unique-id",
	"accountId": "aaa-bbb-unique-account-id",
	"desc": "Do something for XYZ",
	"done": false
}
```

From there you build your application using the 
[database](https://staticbackend.com/docs/database/) CRUD and query functions, 
the [real-time component](https://staticbackend.com/docs/websocket/),
the [storage API](https://staticbackend.com/docs/storage/), etc.

You may use server-side libraries for Node and Go or use an HTTP client 
and use your preferred language.

## Get started with the self-hosted version

[![Get started with self-hosted version](https://img.youtube.com/vi/vQjfaMxidx4/0.jpg)](https://www.youtube.com/watch?v=vQjfaMxidx4)

_Click on the image above to see a video showing how to get started with the 
self-hosted version_.

Please refer to this [guide here](https://staticbackend.com/getting-started/self-hosting/).

We also have this 
[blog post](https://staticbackend.com/blog/get-started-self-hosted-version/) 
that also includes the above video.

If you have Docker & Docker Compose ready, here's how you can have your server 
up and running in dev mode in 30 seconds:

```shell
$> git clone git@github.com:staticbackendhq/core.git
$> cd core
$> cp .demo.env .env
$> docker build . -t staticbackend:latest
$> docker-compuse -f docker-compose-demo.yml up
```

Test your instance:

```shell
$> curl -v http://localhost:8099/db/test
```

You should get an error as follow:

```shell
< HTTP/1.1 401 Unauthorized
< Content-Type: text/plain; charset=utf-8
< Vary: Origin
< Vary: Access-Control-Request-Method
< Vary: Access-Control-Request-Headers
< X-Content-Type-Options: nosniff
< Date: Tue, 03 Aug 2021 11:40:15 GMT
< Content-Length: 33
< 
invalid StaticBackend public key
```

This is normal, as you're trying to request protected API, but you're all set.

The next step is to visit [http://localhost:8099](http://localhost:8099) and 
create your first app. Please note that in dev mode you'll have to look at your 
docker compose output terminal to see the content of the email after creating 
your app.

## Documentation

We're trying to have the best experience possible reading our documentation.

Please help us improve if you have any feedback.

**Documentation with example using our libraries or curl**:

* [Introduction and authentication](https://staticbackend.com/docs/)
* [User management](https://staticbackend.com/docs/users/)
* [Database](https://staticbackend.com/docs/database/)
* [Real-time communication](https://staticbackend.com/docs/websocket/)
* [File storage](https://staticbackend.com/docs/storage/)
* [Forms](https://staticbackend.com/docs/forms/)

## Librairies & CLI

We [provide a CLI](https://staticbackend.com/getting-started/) for local 
development if you want to get things started without any infrastructure and 
for prototyping. 

You can use the CLI to manage your database and form submission. This is the 
only interface we currently have to interact with your database, other than via 
code. There will be a web UI available before v1.0 is released.

We have a page listing our 
[client-side and server-side libraries](https://staticbackend.com/docs/libraries/).

## Examples

If you'd like to see specific examples please let us know via the 
[Discussions](https://github.com/staticbackendhq/core/discussions) tab.

Here's the examples we have created so far:

* [To-do list example](https://staticbackend.com/getting-started/)
* [Realtime collaboration](https://staticbackend.com/blog/realtime-collaboration-example/)
* [Live chat using server-side function & real-time component](https://staticbackend.com/blog/server-side-functions-task-scheduler-example/)

## Deploying in production

We've not written anything yet regarding deploying, but once you have the 
core` built into a binary and have access to MongoDB and Redis in production you 
should be able to deploy it like any other Go server.

We'll have documentation and an example soon for deploying to DigitalOcean.

## Feedback & contributing

If you have any feedback (good or bad) we'd be more than happy to talk. Please 
use the [Discussions](https://github.com/staticbackendhq/core/discussions) tab.

Same for contributing. The easiest is to get in touch first. We're working 
to make it easier to contribute code. If you'd like to work on something 
precise let us know.


## Open source, sponsors, paid SaaS

You may read here 
[why we've decided to open source StaticBackend](https://staticbackend.com/blog/open-source-backend-as-a-service/).

Hopefully we can start getting sponsorship so the open source version development 
and future is secure.

We're also offering paid subscription for a 
[fully managed](https://staticbackend.com/blog/open-source-backend-as-a-service/) 
version of SB.

## Spread the words

It would means the world to us if you could help us spread the words about 
StaticBackend. A tweet, a blog post, any visibility is helpful and I (Dominic) 
personally thanks you for this.

I've failed at getting any kind of traction with StaticBackend on its closed 
source form. I think developer tools like this need to be open source.

I believe in the product, it solves a pain I have for so long, but I'm hoping 
others will also get value out of it and will be excited about the project.