DailyMile.com Command Line Tool
===============================


INSTALL
-------

Extract the zip to a directory and add that directory to your PATH.


BUILD FROM SOURCE
-----------------

This project is written in Google's Go language.  Go is free and open source.  See http://golang.org for instructions on installing the Go compiler.

To build from source:

git clone http://github.com/dgnorton/dm

cd dm && go get && go build

If you wish to cross-compile, see http://dave.cheney.net/2013/07/09/an-introduction-to-cross-compilation-with-go-1-1.  Once you've successfully run through Dave's instructions, the all.bash script in the project can be used to generate Windows, Linux, and Mac binaries (32 & 64 bit).


USAGE
-----

Use the command line help for a complete list of commands...

    dm help
    dm help command

The first time running it, the user needs to be set using this...

    dm -u <username> user <username>

After the first time, the default user can be changed using...

    dm user <username>

Pull default user's entries from dailymile.com...

    dm sync

The initial sync will probably take minutes depending on number of entries.  Future syncs will be incremental and should only take a few seconds.  It'll tell you if it was "Already up-to-date" or how many new entries it pulled down.  One thing it does NOT handle are deletes.  If you sync and then delete an entry on the website it will remain in your local copy of the data unless you delete and do a full sync. 

If you're on some flavor of unix, your data should be stored in ~/.dailymile_cli/<username>/entries.json.  You can use your favorite browser plugin or editor to view the JSON pretty-printed.  Or, if you have python 2.6+ installed...

    cat ~/.dailymile_cli/<username>/entries.json | python -mjson.tool | less

Basic search & formatting capabilities...

    dm find [-s start date] [-e end date] [natural language dates] [-p regex pattern]
            [-f template file]

Basic natural language examples:

    dm find week
    dm find month
    dm find year
    dm find last week
    dm find last month
    dm find last month last year
    dm find next month last year

The 'dm find' command can format output using user-defined templates.
Simple example templates (entries.csv, entries.tsv (default), entries.html) are provided
with this source.  See http://golang.org/pkg/text/template for information
on the Go language template rules.

All of this year's entries in CSV format:

    dm find -s 14/1/1 -f entries.csv

All of this year's entries in a column layout (Linux):

    dm find -s 14/1/1 -f entries.csv | column -t -s,

Case-insensitive search for the word "interval":

    dm find -s 13/1/1 -e 13/12/31 -p "(?i)interval"
Removing the "(?i)" in the above pattern will make the search case-sensitive.

Search for patterns like "8 x 400", "10x800", "10 x 1600m":

    dm find -s 13/1/1 -e 13/12/31 -p "(?i)\d{1,2} *x *\d{3,4}"

Count ALL of your entries on Linux:

    dm find -f entries.csv | wc -l

Format matching entries as HTML:

    dm find -s 14/1/1 -f entries.html

Format matching entries as JSON:

   dm find last week -f json
