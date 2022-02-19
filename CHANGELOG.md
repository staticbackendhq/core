# Changelog for StaticBackend

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
