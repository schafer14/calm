# Calm

Calm is a app to manage my relationship with time. The goal is to frame
my thoughts about the past, present and future in the most productive manor.

When I think about the *present* I want to frame my thoughts along the questions:

- Am I intentionally choosing the activity I am doing?
- How does this activity relate to my broader goals or values?
- How is this activity contributing to my mental health?

When I think about the *future* I want to frame my thoughts along the questions:

- Why would I do this activity?
- What do I hope to learn?
- How do I know if/when I have succeeded?

When I think about the *past* I want to frame my thoughts along the questions:

- How did an activity contribute to my broader goals or values?
- What factors contributed to the outcome of the goals this activity was related to?

# Project status

The app is broken up into three sections (past, present and future). The past
and future sections have not been started.

The present section is in a very alpha state. I am continually tweaking the UI
and UX based on my experience using the application. 

There are a list of short term todo items in the TODO file.

# Running a development server

Prerequisites for running include Elm 1.19 (with default tool chain),
Go 1.24 (with default tool chain), inotifywait and a unix-like shell.
Run `cd api && go mod tidy` to install Go dependencies.

Register a user:

```
cd api
go run ./cmd/create_user banner "Banner B. Schafer"
go run ./cmd/add_role banner "super_admin"
```

When you start the dev server navigate to `/register/$REGISTRATION_TOKEN`.

NOTE: The database has lock protection on it so you will need to stop any servers
before you run a one-off command.

Then to start the dev server run `./dev.sh`.

# Running a production server

`./build_prod.sh` creates a self-contained binary that can run on your machine
(just make sure you have gcc or some equivalent installed.)

If you need to build for an architecture/os that is different than the machine
you are building on you will need to change the build script.

When running the binary you will need to configure webauthn relying parties.
You should use the `--help` flags for instrction on how to set this up.

# Technical details

A main goal of this project is to provide production level features without
relying on third parties to provide those features. This is an overview on
the techincial features and how they compare against alternatives.

## Languages: Go and Elm.

## Database: bbolt

Perhaps the most dubious claim of this entire project is that the database
can be considered production ready. Bbolt is a single file database and so
without modification it can not be considered production ready.

At a bare minimum a production-ready database requires static and active
backups. For active backups Bbolt could be extended to create an active failover
model without to much work. However the better solution is probably to migrate
to a more established database. However, I wouldn't care if the data in my
production system was lost, so Bbolt suits my requirements perfectly.

Another considerations for a system with many users would be indexing and
sharding. These requirements vary wildly based on a number of factors. For
my system (which has two users) neither better indexes nor sharding would
provide much value.

## Authentication: webauthn

This application uses webauthn for authentication. Each end user must negotiate
with the device they are using a suitable authentication method to login. For
example I use a Yubikey mini to login on computers, and my Android fingerprint
thing on my phone. 

Each authentication device is responsible for storing an ID and private key
for each account it can login with. This allows a user to login without provding
a username/email. If a user has multiple accounts registered on the same device
the device should prompt him/her for the account to login when logging in.

## Authorisation: Casbin (rest based)

Casbin provides role based access control. I have configured the resources and
actions to match HTTP standards. So a permission might look like `super_admin /users POST`.
Which allows me to authorize routes without describing any details of how.

The downside of this method is that the permissions don't fully describe what
they apply to, only the HTTP methods they allow.

## PWA

This app is a PWA so users can install it as an native app on their mobile or
desktop devices. This has allowed me to develop the application with very
little thought to what platform users will use.

The app currently looks best on mobile (because that's what I use most). But
is useable on a computer.

# Not on usability

I use this application. It helps me draw attention to the activity I am doing
in a given moment. If it's useful to me it might be useful to you. If you
want to get involved please ask. If you want to use the application
you can set up your own server easily, or maybe use mine.

# Getting access to calm.bannerschafer.com

Alpha access to a hosted service is available by request. Features and interfaces
change at a daily rate.
