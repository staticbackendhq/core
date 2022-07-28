# How to contribute

I'm really happy you're considering contributing to StaticBackend. There's so 
much to do from tiny things to large features, every help is more than welcome.

If you haven't join our [Discord server](https://discord.gg/vgh2PTp9ZB), please do.

I'll start by saying, don't hesitate to ask any questions. I'm personally 
always happy to help, especially if it's your first time contributing to an 
open-source project.

Here are what we're using to maintain the project:

[Discord](https://discord.gg/vgh2PTp9ZB) for general and real-time discussions.

[GitHub discussions](https://github.com/staticbackendhq/core/discussions) for 
questions, new ideas, and such.

[GitHub issues](https://github.com/staticbackendhq/core/issues) for bugs and 
features that has been discussed and approved.

## Run the tests

Here's what you'll need to run the tests:

* Go 1.18+
* Either Docker or local PostgreSQL, Mongo, and Redis
* Environment variables in an `.env` file

Here's my dev `.env` file:

```
APP_SECRET=a-very-long-key-should-be-32long
APP_ENV=dev
APP_URL=http://localhost:8099
DATABASE_URL=user=postgres password=postgres dbname=postgres sslmode=disable
DATA_STORE=pg
# DATABASE_URL=mongodb://localhost:27017
#DATA_STORE=mongo
JWT_SECRET=tiaAvfn
FROM_EMAIL=host@dev.com
FROM_NAME=StaticBackend
REDIS_HOST=localhost:6379
REDIS_PASSWORD=
LOCAL_STORAGE_URL=http://localhost:8099
```

I personally use `docker-compose` to load services dependencies (PostgreSQL, 
MongoDB, and Redis) and have a local Go compiler to run tests.

```sh
$ docker-compose -f docker-compose-unittest.yml up
```

I use `make` to run tests, refer the the `Makefile` for the commands if you 
don't have `make` available.

```sh
$ make alltest
```

I often changes the `DATA_STORE` between `pg` and `mongo`. There's also specific 
make entry for all database providers.

## Submit Pull Requests

Here's how you'd submit changes you've made:

1. Fork the repo
2. Work on the `main` branch or create specifics branches on your repo.
3. Push to your fork
4. Create a pull request when you're ready

Some guideline:

1. Make sure you have tests with your code changes
2. Please add clear commit log messages
3. Keep your commits scope isolated as much as possible
4. When creating your pull request please be as detailed as you can

## Code conventions

1. We use `tabs` as indentation, mainly for accessibility reasons
2. Please use the go format tool before committing your changes

Thanks