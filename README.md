# StaticBackend - simple backend for web & mobile apps

[StaticBackend](https://staticbackend.com) is a simple backend that handles 
user management, database, file storage, and real-time experiences via  
channel/topic-based communication for web and mobile applications.

You can think of it as a lightweight Firebase replacement you may self-host. No 
vendor lock-in, and your data stays in your control. You may contribute to the 
product.

We've decided to open the core of the product and have a permissive license 
with the MIT license.

We [provide a CLI](https://staticbackend.com/getting-started/) for local 
development if you want to get things started without any infrastructure and 
for prototyping. 

### Run locally

You'll need docker (or access to a MongoDB and Redis instances).

1. Clone this repository

```shell
$> git clone git@github.com/staticbackendhq/core.git
```

2. In a terminal start the docker services

```shell
$> docker-compuse up
```

3. Create a file named `.env` with the following environment variables:

```
APP_ENV=dev
DATABASE_URL=localhost
FROM_EMAIL=you@domain.com
FROM_NAME=your-name
JWT_SECRET=something-here
AWS_ACCESS_KEY_ID=your-aws-key
AWS_SECRET_ACCESS_KEY=your-aws-secret
AWS_SECRET_KEY=your-aws-key
AWS_SES_ENDPOINT=https://email.us-east-1.amazonaws.com
```

3. Compile and run the API server

```shell
$> make start
```

### usage

To start using the backend you'll need to create an account on your local 
instance.

You'll need to [install our CLI](https://staticbackend.com/getting-started/) and 
have it running in local mode so it will talk to your local backend instance.

```shell
$> backend account create you@domain.com
```

Make sure you use a real domain and make sure you're all set sending email 
via your AWS account.

In `dev` mode emails are printed to the stdout so you will see the account 
information for your new database account.

Once you have those info you're ready to start calling the API from client-side 
or server-side application.

Refer to [our main documentation](https://staticbackend.com/docs/) for more 
information.

### Contributing

This is still pre-v1 and API _might_ change. All contribution highly appreciated, 
please make sure to discuss before starting anything.
