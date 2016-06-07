# Gofer
A slave that runs tasks on the basis of every file/directory change.

# What does it do?
Simply put, it runs an executable (shell script, or a binary) every time 
a file's/directory's content changes.

# How does it know something has changed?
It is based on inotify (on Linux), and FSEvents (on Mac), and uses the 
excellent [fsnotify](https://github.com/howeyc/fsnotify) package under 
the hood.

# What platforms does it run on?
It runs well on Linux (Ubuntu has been tested), and Mac. Haven't checked 
on Windows, but since [fsnotify](https://github.com/howeyc/fsnotify) is 
a cross-platform library, there's no reason why it shouldn't run on 
Windows.

# Show me how to run this thing!
1. Download the binary, or download the source code and build your own 
binary.
2. `gofer {MENTION YOUR CONFIG FILE HERE}`

# How does the Config file look like?
A Gofer config file is a valid JSON file that consists of the following 
parameters:
```
[
	{
		"source" : "{{SourceJSON File location}}",
		"log" : "{{Log file where the output needs to be written}}"
	},
	{
		"source" : "{{SourceJSON File location}}",
		 "log" : "{{Log file where the output needs to be written}}"
	}
]
```

# OK, but what's a SourceJSON file?
A SourceJSON file is another valid JSON file that consists of all the 
files and folders to monitor, as well as the scripts that are supposed 
to run when they change. The structure of each SourceJSON file is as 
follows:

```
{
	"paths" : [
		{
			"dir" : "{{Absolute Path name of the file or directory to 
			monitor}}",
			"exec" : "{{Absolute Path name of the executable that 
			needs to be run when a change has occurred}}",
			"path" : "absolute"
		},
		{
			"dir" : "{{Relative Path name of the file or directory 
			to monitor}}",
			"exec" : "{{Relative Path name of the executable that 
			needs to be run a change has occurred}}",
			"path" : "relative"
		}
	],
	"selfobserve" : bool
}
```
`selfobserve` is a boolean (true/false) flag that directs gofer to 
reload the SourceJSON file in case there's a change in it. This is so 
that another program could probably keep appending to the SourceJSON 
file, and Gofer would automatically adjust.

`paths` is an array that consists of the rules for a particular source 
file/directory (`dir`), an `exec` string that holds the path of the 
executable that is fired when there's a modification in `dir`, and a 
`path` parameter that mentions if the `dir` and the `exec` paths are 
relative or absolute. If `path` is `relative`, then the `exec` path is 
resolved relative to the `dir` path. If `path` is `absolute`, the `exec` 
path is resolved as the absolute path.

# What's the difference between a Gofer Config file and a SourceJSON file?
A Gofer Config file consists of all INDEPENDENT SourceJSON files that 
need to be observed. It's kinda like configuring Nginx/Apache to serve 
multiple websites on the same server. It lends the flexibility of 
monitoring multiple SourceJSON files, thus making organization and 
maintenance a lot cleaner. 

A SourceJSON file would consist of all the rules for a particular 
logical project.

# Why would anyone need Gofer?
Gofer would make a developer's life easier, by running a particular 
script. For e.g., if a source file has changed, Gofer could be 
configured to create a new build immediately. Or, it could be configured 
to send a POST request whenever a directory has been modified. Or, if 
Gofer is monitoring a database (such as PostgreSQL), it can be 
configured to sync the WAL file with a remote server. The possibilities 
are endless. 

# Wouldn't it be using a lot of CPU cycles?
Not really. On Linux (the main targeted platform), it is based on 
inotify, and hence, is very light. Gofer would only run the `exec` 
script when there's a change. Will update this with more data once 
comprehensive tests have run.

# What is the current version?
Gofer is currently pre-alpha. It would be safe to say that it isn't 
ready for production yet. But, it should fly when used for personal 
tasks.