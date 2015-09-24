# Overview
[![Twitter](https://img.shields.io/badge/author-%40MachielMolenaar-blue.svg)](https://twitter.com/MachielMolenaar)

This is the source of StrangerBot, the bot on Telegram that matches two random
users and allows them to chat with each other.

# License
StrangerBot is licensed under the Apache 2.0 license.

# Installation
Currently there are no binaries available as direct download yet, you should
build it yourself using Go.

If you have go installed, you can install strangerbot like this:

`go get -u github.com/Machiel/strangerbot`

# Usage

Make sure you have MySQL installed, and retrieved an API key from Telegram.

## Example

Make sure you have the following environment variables set:

```
MYSQL_USER
MYSQL_PASSWORD
MYSQL_DATABASE
TELEGRAM_BOT_KEY
```

You can then run start StrangerBot by running `strangerbot` in your terminal.
