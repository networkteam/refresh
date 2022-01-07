# Refresh

## Rebuild and re-run your Go applications when files change.

> This project is based on [https://github.com/markbates/refresh](https://github.com/markbates/refresh).
We used the tool for quite some time (thx for making and maintaining it so long!) but hit some harder issues with process management (clean shutdown) and
issues with `fsnotify/fsnotify` on recent macOS versions. So this is once again a new re-iteration ;)

This simple command line application will watch your files, trigger a build of your Go binary and restart the application for you.

## Installation

```
$ go install github.com/networkteam/refresh@latest
```

## Getting Started

First you'll want to create a `refresh.yml` configuration file:

```
$ refresh init
```

If you want the config file in a different directory:

```
$ refresh init -c path/to/config.yml
```

Set it up the way you want, but I believe the defaults really speak for themselves, and will probably work for 90% of the use cases out there.

## Usage

Once you have your configuration all set up, all you need to do is run it:

```
$ refresh run
```

That's it! Now, as you change your code the binary will be re-built and re-started for you.

## Configuration Settings

```yml
# The root of your application relative to your configuration file.
app_root: .
# List of folders you don't want to watch. The more folders you ignore, the 
# faster things will be.
ignored_folders:
  - vendor
  - log
  - tmp
# List of file extensions you want to watch for changes.
included_extensions:
  - .go
# The directory you want to build your binary in.
build_path: /tmp
# `notify` can trigger many events at once when you change files. To minimize
# unnecessary builds, a delay is used to ignore extra events until the delay passes after the first event.
build_delay: 200ms
# If you have a specific sub-directory of your project you want to build.
build_target_path : "./cmd/cli"
# What you would like to name the built binary.
binary_name: refresh-build
# Extra command line flags you want passed to the built binary when running it.
command_flags: ["--env", "development"]
# Extra environment variables you want defined when the built binary is run.
command_env: ["PORT=1234"]
# If you want colors to be used when printing out log messages.
enable_colors: true
```
