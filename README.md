# autosr: Automated Schelduled Recordings

autosr tracks streamers you are interested in.
When they go live autosr records the stream.

An external program such as streamlink or livestreamer is required.

Currently only SHOWROOM is supported.
Contributors would be greatly appreciated!

## Installing on Linux or OS X

autosr is a command-line application.
The following commands should be types into a terminal.

Download the right version

```
# Linux
curl -O https://raw.githubusercontent.com/bobbytrapz/autosr/dist/linux/autosr
```

```
# OS X
curl -O https://raw.githubusercontent.com/bobbytrapz/autosr/dist/osx/autosr
```

Install streamlink and autosr

```
python -m pip install --user streamlink
sudo mv autosr /usr/local/bin
```

## Installing on Windows

Open Powershell

Press Windows key + R to open Run menu
Type: powershell
Hit enter and powershell should open.

From this point on all the instructions should be run in powershell.

First we need to install scoop.
According to their [their website](https://scoop.sh) we can do this:

```
iex (new-object net.webclient).downloadstring('https://get.scoop.sh')
```

For help refer to [documentation](https://github.com/lukesampson/scoop/wiki/Quick-Start)

After scoop is installed we can install autosr.

```
scoop bucket add bobbytrapz https://github.com/bobbytrapz/scoop-bucket
scoop install autosr streamlink
```

## Track streamers

Add streamers to track

```
autosr track
```

A text editor will open.
Put the url of the streamer you are interested in one line at a time.
For example,

```
https://www.showroom-live.com/MY_FAVORITE_STREAMER
```

## Start recording

Simply run:

```
autosr
```

You should see the autosr dashboard.
To exit press 'q'.

Even if you exit autosr is running in the background.

To stop it run:

```
autosr stop
```

## Watching videos

If anything is recorded, by default they can be found in your home directory in a 'autosr' directory.

The files will be in ts format. You can encode them however you like or just watch them.
I recommend using [mpv](https://mpv.io)

## Set custom options

The default options should be fine.
If you want to change them. For example, to use a different stream ripper:

```
autosr options
```

A configuration file should open for you to edit.

If autosr is running in the background you can have it reload the options using:

```
autosr reload
```
