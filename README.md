# autosr: Automated Scheduled Recordings

![Imgur](https://i.imgur.com/umULMjL.png)

[Installing on Windows 10](#installing-on-windows-10)

[Installing on Mac OS X](#installing-on-mac-os-x)

[Installing on Linux](#installing-on-linux)

autosr is like DVR for live streams.

autosr tracks streamers you are interested in and records them when they become live.

autosr is a command-line application.

An external program such as streamlink or livestreamer is required for downloading. Instructions on how to install them can be found below.

Currently only SHOWROOM is supported.

Contributors would be greatly appreciated!

## Quick start

- Write a list of streamer urls using 'autosr track'
- Run 'autosr'
- Watch videos as they complete

# Installing on Windows 10

Please note I do not use Windows. I'm open to alternative install methods.

Quick overview:

```
Set-ExecutionPolicy RemoteSigned -scope CurrentUser
iex (new-object net.webclient).downloadstring('https://get.scoop.sh')
scoop bucket add extras
scoop bucket add bobbytrapz https://github.com/bobbytrapz/scoop-bucket
scoop install cmder streamlink autosr
```

If you are reinstalling:

```
scoop update
scoop uninstall autosr
scoop install autosr
```

When a new version is released and you want to update:

```
scoop update autosr
```

Open cmder and [start using autosr](#how-to-use)

## Guide

Open powershell:

![Imgur](https://i.imgur.com/VxBlI2m.png)

Install scoop:

```
Set-ExecutionPolicy RemoteSigned -scope CurrentUser
```

![Imgur](https://i.imgur.com/ui54FHw.png)

```
iex (new-object net.webclient).downloadstring('https://get.scoop.sh')
```

![Imgur](https://i.imgur.com/UdGogLo.png?1)

Now you can run each of these commands one at a time to finish installing autosr and everything it needs.

```
scoop bucket add extras
scoop bucket add bobbytrapz https://github.com/bobbytrapz/scoop-bucket
scoop install cmder streamlink autosr
```

Open cmder and [start using autosr](#how-to-use)

## Why install scoop?

It is not easy to install command-line applications on Windows. [scoop](https://scoop.sh/) makes installing them simple.

Choosing scoop over Chocolatey was the result of personal preference.

## Why install cmder?

[cmder](http://cmder.net/) is a popular alternative console emulator.

Neither powershell nor powershell_ise will display our interface properly.

You can choose any other console emulator you like.

autosr will still record no matter where you run it.

## Why install streamlink?

autosr uses [streamlink](https://streamlink.github.io/) by default.
You can use any stream downloader you'd like as long as you change the options.

Use 'autosr options' to make changes.

# Installing on Mac OS X

If your computer contains an Apple silicon chip such as the Apple M1:
Save [autosr-osx-arm64](https://github.com/bobbytrapz/autosr/releases/latest) to your home directory.
Otherwise, save [autosr-osx](https://github.com/bobbytrapz/autosr/releases/latest) to your home directory.

Then open Terminal and run the following commands one at a time.

Install autosr (if Apple silicon):

```
chmod +x autosr-osx-arm64
sudo mv autosr-osx-arm64 /usr/local/bin/autosr
```

Install autosr (if not Apple silicon)

```
chmod +x autosr-osx
sudo mv autosr-osx /usr/local/bin/autosr
```

Install streamlink:

```
easy_install -U streamlink
```

On newer Macs (>=12.3) you will need to install [Homebrew](https://brew.sh/) then:

```
brew install streamlink
```

When a new version is released, and you want to update:

```
sudo autosr update
```

[How to use](#how-to-use)

# Installing on Linux

Download [autosr](https://github.com/bobbytrapz/autosr/releases/latest)

Install autosr:

```
chmod +x autosr
sudo mv autosr /usr/local/bin/autosr
```

Install streamlink:

```
sudo pip install streamlink
```

When a new version is released and you want to update:

```
sudo autosr update
```

# How to use

## Track streamers

Add streamers to track:

```
autosr track
```

A text editor will open.
Put the url of the streamers you are interested in one line at a time.

Save the file to apply changes.

The changes are applied without restarting.
For example,

```
# a comment to help me organize
https://www.showroom-live.com/MY_FAVORITE_ROOM
https://www.showroom-live.com/ANOTHER_ROOM

# others
https://www.showroom-live.com/OTHER_ROOM
```

Blank lines and lines that start with '#' are ignored.

To stop tracking someone just remove them from the list or add a '#' to comment them out. Then save the file.

There is no need to restart. autosr will stop tracking them immediately.

## Start recording

Simply run:

```
autosr
```

You should see the autosr dashboard.

On Windows and OS X you may see a warning popup about your firewall.

You can just accept the default options and allow access.

autosr needs network access to work.

The dashboard shows you how long someone has been streaming and how long until streams start.

To exit press 'q'.

Even if you exit, autosr will still track and record in the background.

To stop all tracking and recording run:

```
autosr stop
```

## Watching videos

If anything is recorded, by default they can be found in your home directory in a 'autosr' directory.

On Windows this would be C:\Users\YOUR_USER_NAME\autosr

The files will be in ts format.

You can encode them however you like or just watch them as is.

Video quality will depend on the connection quality between you and the streamer.

For playing media I recommend [mpv](https://mpv.io)

## Customize options

The default options should be fine.

If you want to change them. For example, to use a different stream downloader or change the dashboard's appearance:

```
autosr options
```

A configuration file should open for you to edit.

Save the file to apply changes.

The changes are applied without restarting.

## Help

To see help or dashboard controls:

```
autosr help
```

## Resource Usage

autosr makes modest use of memory even including multiple instances of streamlink.

Memory use grows as more people are recorded at the same time.

For what it's worth, I track around 100 people without problems.

Streams that last around an hour take about 1GB of space each.

The bandwidth usage can be very high depending on how many people you track.

If you have a data cap, you probably should not be running autosr. A safer solution would be to just run [streamlink](https://streamlink.github.io/) manually whenever you need to.

## Known Issues

On Windows, it seems streamlink is sometimes not closed properly when tracking is canceled.

## Alternatives

I first wrote this for myself a year ago and so I am not aware of any alternative solutions. If you know any please let me know.

## Bugs

Please report bugs on Github or contact me [@pibisubukebe](https://twitter.com/pibisubukebe) on Twitter.

## License

This software is released under the GPL v3 license. See LICENSE.
