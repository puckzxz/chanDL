# chanDL

An easy way to download all images in a 4chan thread.

## Compiling

Run `go build`

## How to use

chanDL has 2 arguments you can pass to it

* `-thread` The full URL to the 4chan thread. **Required**
* `-path` The optional location to download to, defaults to the current directory

Example: `chandl -thread https://boards.4channel.org/a/thread/123`