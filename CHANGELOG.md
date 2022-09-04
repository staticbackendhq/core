# Changelog for StaticBackend

### Sep 04, 2022 v1.4.0

Features:

* Dev cache impl now handle pub/sub (thanks @rostikts)
* New server-side runtime function allowing to make HTTP requests (thanks @ksankeerth)
* Magic link to login without entering password
* Clean up log output via zerolog (thanks @VladPetriv)
* Ability to update multiple documents from a query. (thanks @rostikts)
* UI list uploaded files. (thanks @VladPetriv)
* Added `/me` endpoint that returns current user with their role.
* Creation of the `config` package to holw all configuration variables
* In memory datastore and cache (for local dev / CLI)

Bug fixes:

* Fixed bug when calling List/Query when collection did not exists (thanks @rostikts)
* Fixed a bug with PostgreSQL read/write permission
* Fixed issue when serving local store files.
* Fixed UI displaying internal SB collections
* Fixed Go version in DockerFile (thanks @MyWay)
* Fixed issue with PostgreSQL schema name not starting with a letter


### Mar 16, 2022 v1.3.0

* Feature: resize image when uploading
* Feature: convert URL to PDF or PNG
* Feature: send SMS text messages
* Added possibility to create database indexes
* Fixed an issue with Mongo's read permissions for list and query functions
* Added a new environment variable for Redis: REDIS_URL
* Fixed an issue with MongoDB server-side functions schema

### Feb 22, 2022 v1.2.1

* Fixed issue with form submission (thanks c-nv-s)
* Fixed issue with the `query` function inside the server-side runtime 
function execution.
* Added PostgreSQL indexes when creating base tables on the `account_id` and 
added indexes for the `sb` schema.

### Feb 19, 2022 v1.2.0

*  Created a data persistance interface to support different data store.
* Added support for PostgreSQL.
* Database tests for PostgreSQL and MongoDB.
* Default Docker Compose for demo use PostgreSQL.

### Jan 1, 2022

* Added atomic worker queue

### Nov 17, 2021

* Added graceful shutdown

### Oct 31, 2021 v1.1.0

* Added reset password flow and made the reset code generation avail from backend.
* Added bulk create function to insert lots of documents reliably
* Added an increment function to inc/dec a specific field atomically

### Aug 23, 2021 v1.0.1

* Server-side function runtime allows to run JavaScript code on event/schedule
* Task scheduler allows to run function on specifics interval

### Aug 12, 2021

* Updated the realtime broker to handle distribution by having all messages 
using Redis's PubSub

### Aug 3, 2021

* Added Dockerfile and made it easier to use Docker Compose to start an 
instance quickly.

### ### Aug 2, 2021

* Added form submission list/view to the web UI

### Jul 31, 2021

* Huge database refactor to make it easier to share with UI
* Created first basic web UI to make it easier to get started with new instance

### ### Jul 29, 2021

* Removed AWS requirements by provider local implementation for storage and 
email

### Jul 27, 2021

* Created interface for sending email
* Created interface for storage operations
* Binary release 1.0.0-alpha1

### Jul 26, 2021

* Refactored lots of code into sub-packages

* 

### Jul 2021

* Added possibilities to delete files
* Added possibilities to send email (still not on client library)
* Released as open source

### May 2021

* New realtime implementation using SSE, websocket was causing lots of issues.
* Added the MIT LICENSE, preparing for open source release

### Jan 2021

Started the websocket implementation

### Dec 2020

After almost 1 year of in and out, the first production version is deployed.

### Jan 2020

First commit to GitHub, when the project got real and rewritten in Go.

### Oct 2019

Project started, first version was written in Node.
