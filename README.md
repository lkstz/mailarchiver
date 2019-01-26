# mailarchiver

Automatically sort your mails into a year-month folder structure.

If you are an inbox-zero user for years, your `Archive` folder will contain tens of thousands of mails, which causes problems with some mail user agents.
This tool will sort your mails into a year-month folder structure, which has several advantages:

- All mail user agents are capable of processing the folders
- Search will be a lot faster, if you remember the year or the month of the mail you search
- A much tidier and structured mail archive

**Example folder structure**

```
├── INBOX
├── Drafts
├── Sent
├── Archive
├── Junk
├── Trash
└── Archives
    ├── 2016
    │   ├── 01
    │   ├── 02
    │   ├── 03
    │   ├── ...
    │   └── 12
    ├── 2017
    |   ├── 01
    |   ├── 02
    |   ├── 03
    |   ├── ...
    |   └── 12
    └── ...
```

In this case, `Archive` is the archive folder used for inbox zero, and `Archives` contains the archive organized by mailarchiver.

## Installation

### Source

```
$ go get -u github.com/kolletzki/mailarchiver
$ cd $GOPATH/src/github.com/kolletzki/mailarchiver
$ go install .
```

## Usage

mailarchiver is intended to be used from CLI.

### Example

```
$ mailarchiver \
	--host imap.example.com \
	--port 993 \
	--user myemail@example.com \
	--archive Archives \
	--mbox Inbox \
	--rmbox Archives \
	--imbox Archives/Special \
	--skip-current \
	--dry
```

This will promt you for the IMAP password and then connect to `imap.example.com` on port `993` with the user `myemail@example.com` and the prompted password.
It will then process the mailbox `Inbox` and `Archives` including all of it's nested mailboxes, except all mailboxes beginning with `Archives/Special`.
No mails from the current month will be touched. For safety, this will be a dry run, meaning mailarchiver does not modify anything, however behaves as it'd do so.

### Options

```
$ mailarchiver --help
NAME:
   Mailarchiver - Automatically sort your emails into a year-month folder structure

USAGE:
   mailarchiver.exe [global options] command [command options] [arguments...]

VERSION:
   0.9.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --host value, -H value        IMAP host
   --port value, -p value        IMAP port (default: 0)
   --user value, -u value        IMAP user
   --password value, --pw value  IMAP password (will be promted, if omitted)
   --archive value, -a value     Main archive folder
   --mbox value                  Mailboxes to process
   --rmbox value                 Mailboxes to process recursive
   --imbox value                 Mailboxes to ignore (overrides mbox and rmbox)
   --skip-current                Skip mails from current month
   --dry                         Performs a dry run, nothing will be changed on the IMAP server
   --help, -h                    show help
   --version, -v                 print the version
```

`host`, `port`, `user` and `archive` are mandatory options.
In addition, you have to pass at least one `mbox` or `rmbox` option.

## ToDo

- [ ] enhance documentation
- [ ] default values for config (port, archive, ...)
- [x] fallback for [IMAP `MOVE` extension](https://tools.ietf.org/html/rfc6851)
- [x] add support for [IMAP `ID` extension](https://tools.ietf.org/html/rfc2971)
- [ ] allow password to be read from file
- [ ] add support for a config file instead of CLI args
- [ ] add support for non-ssl IMAP connections
- [ ] add releases with binaries
- [ ] enhance moving logic
- [ ] enhance fetch logic
- [ ] add tests

### Gmail

Gmail classifies raw IMAP as "insecure access" by default. 
If you want to use mailarchiver with Gmail, you have to allow "less secure apps" in your Gmail settings.

Since I'm not using Gmail, I did not yet implement a better Gmail support to mailarchiver.
If you want to add nice Gmail support, feel free to open a pull request. :)

## License

GNU General Public License v3.0

See [LICENSE](https://github.com/kolletzki/mailarchiver/blob/master/LICENSE) for full license text.
