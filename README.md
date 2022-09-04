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

p.s. If you'd like to contribute to an active Go project, you've found a nice 
one in my biased opinion.

# StaticBackend - simple backend for your apps

[StaticBackend](https://staticbackend.com) is a simple backend API that handles 
user management, database, file storage, forms, real-time experiences via 
channel/topic-based communication, and server-side functions for web and mobile 
applications.

You can think of it as a lightweight Firebase replacement you may self-host. Less 
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
* [help](#help)


## What can you build

I built StaticBackend with the mindset of someone tired of writing the same code 
over and over on the backend. If your application needs one or all of 
user management, database, file storage, real-time interactions, it should be 
a good fit.

I'm personally using it to build SaaS:

[En Pyjama - an online course platform for kids](https://enpyjama.com)

Abandoned projects:

* [Vivid - Automatic video clips for podcasts](https://vivid.fm)
* [Tangara - one page checkout for creators](https://tangara.io)

It can be used from client-side and/or server-side.

## How it works / dev workflow

The main idea is that StaticBackend is your backend API for your applications. 
A performant free and open-source self-hosted Firebase alternative.

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

StaticBackend provides commonly used building blocks for web applications.

You may use server-side libraries for Node, Python and Go or use an HTTP client 
and use your preferred language.

## Get started with the self-hosted version

### Deploy buttons

**Heroku**: Deploy an instance to your Heroku account.

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/staticbackendhq/core)

**Render**: Deploy an instance to your Render account

[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy)

### Docker or manual setup


[![Get started with self-hosted version](https://img.youtube.com/vi/vQjfaMxidx4/0.jpg)](https://www.youtube.com/watch?v=vQjfaMxidx4)

_Click on the image above to see a video showing how to get started with the 
self-hosted version_.

Please refer to this [guide here](https://staticbackend.com/getting-started/self-hosting/).

We also have this 
[blog post](https://staticbackend.com/blog/get-started-self-hosted-version/) 
that includes the above video.

If you have Docker & Docker Compose ready, here's how you can have your server 
up and running in dev mode in 30 seconds:

```shell
$> git clone git@github.com:staticbackendhq/core.git
$> cd core
$> cp .demo.env .env
$> docker build . -t staticbackend:latest
$> docker-compose -f docker-compose-demo.yml up
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
your app. This email contains all the keys and your super user account 
information.

## Documentation

We're trying to have the best experience possible reading our documentation.

Please help us improve if you have any feedback.

**Documentation with example using our libraries or curl**:

* [Introduction and authentication](https://staticbackend.com/docs/)
* [User management](https://staticbackend.com/docs/users/)
* [Social logins (beta)](https://staticbackend.com/docs/social-logins/)
* [Database](https://staticbackend.com/docs/database/)
* [Real-time communication](https://staticbackend.com/docs/websocket/)
* [File storage](https://staticbackend.com/docs/storage/)
* [Server-side functions](https://staticbackend.com/docs/functions/)
* [Send emails](https://staticbackend.com/docs/sendmail/)
* [Caching](https://staticbackend.com/docs/cache/)
* [Forms](https://staticbackend.com/docs/forms/)
* [Root token](https://staticbackend.com/docs/root-token/)

## Librairies & CLI

We [provide a CLI](https://staticbackend.com/getting-started/) for local 
development if you want to get things started without any infrastructure and 
for prototyping / testing.

You can use the CLI to manage your database, form submissions, and deploy 
server-side-functions. We have an alpha Web UI as well to manage your resources.

We have a page listing our 
[client-side and server-side libraries](https://staticbackend.com/docs/libraries/).

## Examples

If you'd like to see specific examples please let us know via the 
[Discussions](https://github.com/staticbackendhq/core/discussions) tab.

Here's the examples we have created so far:

* [To-do list example](https://staticbackend.com/getting-started/)
* [Realtime collaboration](https://staticbackend.com/blog/realtime-collaboration-example/)
* [Live chat using server-side function & real-time component](https://staticbackend.com/blog/server-side-functions-task-scheduler-example/)
* [Jamstack Bostom talk](https://www.youtube.com/watch?v=Uf-K6io9p7w)

## Deploying in production

We've not written anything yet regarding deploying, but once you have the 
core` built into a binary and have access to either PostgreSQL or MongoDB, and 
Redis in production you should be able to deploy it like any other Go server.

We'll have documentation and an example soon for deploying to DigitalOcean.

## Feedback & contributing

If you have any feedback (good or bad) we'd be more than happy to talk. Please 
use the [Discussions](https://github.com/staticbackendhq/core/discussions) tab.

Same for contributing. The easiest is to get in touch first. We're working 
to make it easier to contribute code. If you'd like to work on something 
precise let us know.

Here are videos made specifically for people wanting to contribute:

* [Intro, setup, running tests, project structure](https://youtu.be/uTj7UEbg0p4)

Check the [contributing file](CONTRIBUTING.md) for details.


## Help

If you're looking to help the project, here are some ways:

* Use it and share your experiences.
* Sponsor the development via GitHub sponsors.
* Spread the words, a tweet, a blog post, any mention is helpful.
* Join the [Discord](https://discord.gg/vgh2PTp9ZB) server.
