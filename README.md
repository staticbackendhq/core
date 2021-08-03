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

I'm personally using it to build SaaS.

## How it works / dev workflow

The main idea is that StaticBackend is your backend API for your frontend apps.

It needs to have access to MongoDB and Redis servers. Once you have your instance 
running you create accounts for your applications.

An account has its own database, file storage, etc.

I think `app` might have been a better name instead of `account`. Naming things 
is hard.

A StaticBackend account(app) can have multiple user accounts and each user 
accounts may have multiple users.

From there each users can create database documents that are by default Read/Write 
for the owner (the user) and Read for its parent account. You may customize 
permission for each of your collection (see that later in the documentation).

You have the basic building blocks to create a typical web 
application. You have all your CRUD and data query operations covered, file 
storage and websocket-like capabilities.

We have a [JavaScript library](https://www.npmjs.com/package/@staticbackend/js) to 
get started quickly. We have also server-side libraries for Node and Go atm.

Why would you need server-side libraries, was it not supposed to be a backend 
for client-side applications.

Yes, but, there's always a but. Sometimes your application will need to 
perform tasks on behalf of users or public user that do not have access to 
perform CRUD from the client-side.

Let's imagine we're building an invoicing app. Here's the major entities 
we have for this examples:

* A StaticBackend account (our app inside our SB instance)
* An account with 2 users (this would be your customer)
* An invoices collection (where your customer create invoice)
* A clients collection (Your customer send invoice to their clients)

Now let's imagine our customer (our app user) sends an invoice to their Client.

Their client does not have any user account, but they need to see their invoice 
when they click on the unique link on their email they received.

This can be achieve via a backend function. Couple of ways:

* The email the client received can be directly a unique URL pointing to a 
function as a service hosted somewhere. (We will have functions soon). That 
function returns the invoice HTML.
* Or it could be pointing to your client-side app and you perform a call to 
a serverless function you're hosting somewhere. That function uses the 
server-side library to perform a `GET` on behalf of the creator of the invoice 
document, which is our customer.

The function will be able to perform a Read operation using a special `Root Token`.

This `Root Token` allows your system to do anything on the server-side.

I hope I did not lost the majority of people here ;)

This is one example of your typical day-to-day workflow using StaticBackend.

## Get started with the self-hosted version

Please refer to this [guide here](https://staticbackend.com/getting-started/self-hosting/).

If you have Docker & Docker Compose ready, here's how you can have your server 
up and running in dev mode in 30 seconds:

```shell
$> git clone git@github.com:staticbackendhq/core.git
$> cd core
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